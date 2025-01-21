package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/registryx/registryclients"
	"github.com/kubescape/backend/pkg/command"
	"github.com/kubescape/backend/pkg/command/types/v1alpha1"
	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/k8s-interface/k8sinterface"
	"github.com/kubescape/operator/config"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	registryCronjobTemplate = "registry-scan-cronjob-template"
	cronjobTemplateName     = "cronjobTemplate"
)

var (
	secretGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}

	configMapGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	cronJobGVR = schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "cronjobs",
	}

	jobGVR = schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}
)

type RegistryCommandsHandler struct {
	ctx             context.Context
	k8sAPI          *k8sinterface.KubernetesApi
	commands        chan v1alpha1.OperatorCommand
	commandsWatcher *CommandWatchHandler
	config          config.IConfig
}

func NewRegistryCommandsHandler(ctx context.Context, k8sAPI *k8sinterface.KubernetesApi, commandsWatcher *CommandWatchHandler, config config.IConfig) *RegistryCommandsHandler {
	return &RegistryCommandsHandler{
		ctx:             ctx,
		k8sAPI:          k8sAPI,
		commands:        make(chan v1alpha1.OperatorCommand, 100),
		commandsWatcher: commandsWatcher,
		config:          config,
	}
}

func (ch *RegistryCommandsHandler) Start() {
	logger.L().Info("starting RegistryCommandsHandler")
	ch.commandsWatcher.RegisterForCommands(ch.commands)

	for {
		select {
		case cmd := <-ch.commands:
			if !isRegistryCommand(cmd.Spec.CommandType) {
				logger.L().Debug("not a registry command" + cmd.Spec.CommandType)
				continue
			}
			ctx, span := otel.Tracer("").Start(context.Background(), "actionHandler.scanRegistriesV2")
			status := v1alpha1.OperatorCommandStatus{
				Executer:  "operator",
				Started:   true,
				StartedAt: &metav1.Time{Time: time.Now()},
			}
			var err error
			var payload []byte

			logger.L().Ctx(ctx).Info(fmt.Sprintf("handling %s command", cmd.Spec.CommandType))
			switch cmd.Spec.CommandType {
			case string(command.OperatorCommandTypeCreateRegistry):
				err = ch.upsertRegistry(ctx, cmd, true)
			case string(command.OperatorCommandTypeUpdateRegistry):
				err = ch.upsertRegistry(ctx, cmd, false)
			case string(command.OperatorCommandTypeDeleteRegistry):
				err = ch.deleteRegistry(ctx, cmd)
			case string(command.OperatorCommandTypeCheckRegistry):
				payload, err = ch.checkRegistry(ctx, cmd)
			}

			status.Completed = true
			status.CompletedAt = &metav1.Time{Time: time.Now()}

			if err != nil {
				status.Error = &v1alpha1.OperatorCommandStatusError{Message: err.Error()}
			} else if len(payload) > 0 {
				status.Payload = payload
			}

			logger.L().Ctx(ctx).Info(fmt.Sprintf("finished handling %s command ", cmd.Spec.CommandType))
			ch.patchCommandStatus(ctx, &cmd, status)
			span.End()
		case <-ch.ctx.Done():
			logger.L().Ctx(ch.ctx).Info("RegistryCommandsHandler: context done")
			return
		}
	}
}

func (ch *RegistryCommandsHandler) patchCommandStatus(ctx context.Context, command *v1alpha1.OperatorCommand, status v1alpha1.OperatorCommandStatus) {
	logger.L().Ctx(ctx).Debug("patchCommandStatus - updating operator command status", helpers.String("command", command.Spec.CommandType), helpers.Interface("status", status))
	patchBytes, err := json.Marshal(map[string]v1alpha1.OperatorCommandStatus{"status": status})
	if err != nil {
		logger.L().Ctx(ctx).Error("patchCommandStatus - failed to marshal status patch", helpers.Error(err))
		return
	}

	_, err = ch.k8sAPI.GetDynamicClient().Resource(v1alpha1.SchemaGroupVersionResource).Namespace(command.Namespace).Patch(
		ch.ctx,
		command.Name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)
	if err != nil {
		logger.L().Ctx(ctx).Error("patchCommandStatus - failed to patch command status", helpers.Error(err))
	}
	logger.L().Ctx(ctx).Info("patchCommandStatus: command status patched successfully")
}

func (ch *RegistryCommandsHandler) checkRegistry(ctx context.Context, cmd v1alpha1.OperatorCommand) ([]byte, error) {
	registry, err := armotypes.UnmarshalRegistry(cmd.Spec.Body)
	if err != nil {
		logger.L().Ctx(ctx).Error("checkRegistry - failed to unmarshal command payload", helpers.Error(err))
		return nil, err
	}
	client, err := registryclients.GetRegistryClient(registry)
	if err != nil {
		logger.L().Ctx(ctx).Error("checkRegistry - failed to get registry client", helpers.Error(err))
		return nil, err
	}

	repositories, err := client.GetAllRepositories(ctx)
	if err != nil {
		logger.L().Ctx(ctx).Error("checkRegistry - failed to get registry repositories", helpers.Error(err))
		return nil, err
	}
	logger.L().Ctx(ctx).Debug(fmt.Sprintf("checkRegistry - found %d repositories", len(repositories)))

	payload, err := json.Marshal(repositories)
	if err != nil {
		logger.L().Ctx(ctx).Error("checkRegistry - failed to marshal repositories", helpers.Error(err))
		return nil, err
	}

	return payload, nil
}

func (ch *RegistryCommandsHandler) deleteRegistry(ctx context.Context, cmd v1alpha1.OperatorCommand) error {
	registry, err := armotypes.UnmarshalRegistry(cmd.Spec.Body)
	if err != nil {
		logger.L().Ctx(ctx).Error("deleteRegistry - failed to unmarshal command payload", helpers.Error(err))
		return err
	}
	resourceName := registry.GetBase().ResourceName
	err = ch.k8sAPI.KubernetesClient.BatchV1().CronJobs(ch.config.Namespace()).Delete(context.Background(), resourceName, metav1.DeleteOptions{})
	if err != nil {
		logger.L().Ctx(ctx).Error("deleteRegistry - failed to delete cronjob resource", helpers.Error(err))
		return err
	}
	logger.L().Ctx(ctx).Debug("deleteRegistry - successfully deleted cronjob")

	err = ch.k8sAPI.KubernetesClient.CoreV1().Secrets(ch.config.Namespace()).Delete(context.Background(), resourceName, metav1.DeleteOptions{})
	if err != nil {
		logger.L().Ctx(ctx).Error("deleteRegistry - failed to delete secret resource", helpers.Error(err))
		return err
	}
	logger.L().Ctx(ctx).Debug("deleteRegistry - successfully deleted secret")

	err = ch.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(ch.config.Namespace()).Delete(context.Background(), resourceName, metav1.DeleteOptions{})
	if err != nil {
		logger.L().Ctx(ctx).Error("deleteRegistry - failed to delete configmap resource", helpers.Error(err))
		return err
	}
	logger.L().Ctx(ctx).Debug("deleteRegistry - successfully deleted configmap")

	logger.L().Ctx(ctx).Info("deleteRegistry: registry deleted successfully")
	return nil
}

func (ch *RegistryCommandsHandler) upsertRegistry(ctx context.Context, cmd v1alpha1.OperatorCommand, triggerNow bool) error {
	registry, err := armotypes.UnmarshalRegistry(cmd.Spec.Body)
	if err != nil {
		logger.L().Ctx(ctx).Error("upsertRegistry - failed to unmarshal command payload", helpers.Error(err))
		return err
	}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		secret, err := ch.generateSecretObject(registry)
		if err != nil {
			logger.L().Ctx(ctx).Error("upsertRegistry - failed to create secret resource", helpers.Error(err))
			return err
		}
		if err = ch.upsertResource(secret, secretGVR, secret.Name); err != nil {
			logger.L().Ctx(ctx).Error("upsertRegistry - failed to upsert secret resource", helpers.Error(err))
			return err
		}
		logger.L().Ctx(ctx).Debug("upsertRegistry - successfully upserted secret")
		return nil
	})

	errGroup.Go(func() error {
		configMap, err := ch.generateConfigMapObject(registry)
		if err != nil {
			logger.L().Ctx(ctx).Error("upsertRegistry - failed to create config map resource", helpers.Error(err))
			return err
		}
		if err = ch.upsertResource(configMap, configMapGVR, configMap.Name); err != nil {
			logger.L().Ctx(ctx).Error("upsertRegistry - failed to upsert config map resource", helpers.Error(err))
			return err
		}
		logger.L().Ctx(ctx).Debug("upsertRegistry - successfully upserted configmap")
		return nil
	})

	errGroup.Go(func() error {
		if triggerNow {
			job, err := ch.generateJobObject(registry)
			if err != nil {
				logger.L().Ctx(ctx).Error("upsertRegistry - failed to create job resource", helpers.Error(err))
				return err
			}
			if err = ch.upsertResource(job, jobGVR, registry.GetBase().ResourceName); err != nil {
				logger.L().Ctx(ctx).Error("upsertRegistry - failed to upsert job resource", helpers.Error(err))
				return err
			}
			logger.L().Ctx(ctx).Debug("upsertRegistry - successfully upserted job")
		}
		if registry.GetBase().ScanFrequency != "" {
			cronjob, err := ch.generateCronJobObject(registry)
			if err != nil {
				logger.L().Ctx(ctx).Error("upsertRegistry - failed to create cron job resource", helpers.Error(err))
				return err

			}
			if err = ch.upsertResource(cronjob, cronJobGVR, registry.GetBase().ResourceName); err != nil {
				logger.L().Ctx(ctx).Error("upsertRegistry - failed to upsert job resource", helpers.Error(err))
				return err
			}
			logger.L().Ctx(ctx).Debug("upsertRegistry - successfully upserted cronjob")
		}
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		logger.L().Ctx(ctx).Error("upsertRegistry - failed to upsert registry", helpers.Error(err))
		return err
	}

	logger.L().Ctx(ctx).Info("upsertRegistry: registry upserted successfully")
	return nil
}

func (ch *RegistryCommandsHandler) upsertResource(resource interface{}, gvr schema.GroupVersionResource, name string) error {
	applyOpts := metav1.ApplyOptions{
		FieldManager: "application/apply-patch",
		Force:        true,
	}
	unstructuredResource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
	if err != nil {
		return err
	}
	_, err = ch.k8sAPI.DynamicClient.Resource(gvr).Namespace(ch.config.Namespace()).Apply(ch.ctx, name, &unstructured.Unstructured{Object: unstructuredResource}, applyOpts)
	return err
}

func (ch *RegistryCommandsHandler) generateCronJobObject(registry armotypes.ContainerImageRegistry) (*batchv1.CronJob, error) {
	template, err := ch.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(ch.config.Namespace()).Get(context.Background(), registryCronjobTemplate, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	jobTemplateStr, ok := template.Data[cronjobTemplateName]
	if !ok {
		return nil, fmt.Errorf("getCronJobTemplate: jobTemplate not found")
	}
	cronjob := &batchv1.CronJob{}
	if err := yaml.Unmarshal([]byte(jobTemplateStr), cronjob); err != nil {
		return nil, err
	}
	cronjob.Name = registry.GetBase().ResourceName
	cronjob.Spec.Schedule = registry.GetBase().ScanFrequency
	cronjob.Spec.TimeZone = ptr.To("Etc/UTC")
	for i, v := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes {
		if v.Name == armotypes.RegistryRequestVolumeName {
			if cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].ConfigMap != nil {
				cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].ConfigMap.Name = registry.GetBase().ResourceName
			}
		}
	}
	cronjob.ObjectMeta.Labels = map[string]string{"app": registry.GetBase().ResourceName}
	return cronjob, nil
}

func (ch *RegistryCommandsHandler) generateJobObject(registry armotypes.ContainerImageRegistry) (*batchv1.Job, error) {
	template, err := ch.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(ch.config.Namespace()).Get(context.Background(), registryCronjobTemplate, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	jobTemplateStr, ok := template.Data[cronjobTemplateName]
	if !ok {
		return nil, fmt.Errorf("getJobTemplate: jobTemplate not found")
	}
	cronjob := &batchv1.CronJob{}
	if err := yaml.Unmarshal([]byte(jobTemplateStr), cronjob); err != nil {
		return nil, err
	}
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      registry.GetBase().ResourceName,
			Namespace: cronjob.Namespace,
			Labels:    map[string]string{"app": registry.GetBase().ResourceName},
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
	}
	for i, v := range job.Spec.Template.Spec.Volumes {
		if v.Name == armotypes.RegistryRequestVolumeName {
			if job.Spec.Template.Spec.Volumes[i].ConfigMap != nil {
				job.Spec.Template.Spec.Volumes[i].ConfigMap.Name = registry.GetBase().ResourceName
			}
		}
	}

	return job, nil
}

func (ch *RegistryCommandsHandler) generateSecretObject(registry armotypes.ContainerImageRegistry) (*v1.Secret, error) {
	secret := v1.Secret{}
	secret.Name = registry.GetBase().ResourceName
	secret.Kind = armotypes.K8sKindSecret
	secret.APIVersion = armotypes.K8sApiVersionV1
	secret.Type = v1.SecretTypeOpaque
	secret.Namespace = ch.config.Namespace()
	secret.StringData = make(map[string]string)
	registryAuthBytes, err := json.Marshal(registry.ExtractSecret())
	if err != nil {
		return nil, err
	}
	secret.StringData[armotypes.RegistryAuthFieldInSecret] = string(registryAuthBytes)

	return &secret, err
}

func (ch *RegistryCommandsHandler) generateConfigMapObject(registry armotypes.ContainerImageRegistry) (*v1.ConfigMap, error) {
	configMap := v1.ConfigMap{}
	configMap.Name = registry.GetBase().ResourceName
	configMap.Kind = armotypes.K8sKindConfigMap
	configMap.APIVersion = armotypes.K8sApiVersionV1
	configMap.Namespace = ch.config.Namespace()
	configMap.Labels = map[string]string{"app": registry.GetBase().ResourceName}
	cmd, err := getCommandForConfigMap(registry, registry.GetBase().ResourceName)
	if err != nil {
		return nil, err
	}
	configMap.Data = map[string]string{armotypes.RegistryCommandBody: cmd}

	return &configMap, nil
}

func getCommandForConfigMap(imageRegistry armotypes.ContainerImageRegistry, resourceName string) (string, error) {
	scanRegistryCommand := apis.Command{}
	scanRegistryCommand.CommandName = apis.TypeScanRegistryV2
	scanRegistryCommand.Args = map[string]interface{}{}
	scanRegistryCommand.Args[armotypes.RegistryInfoArgKey] = *imageRegistry.GetBase()
	scanRegistryCommand.Args[armotypes.RegistrySecretNameArgKey] = resourceName

	scanRegistryCommands := apis.Commands{}
	scanRegistryCommands.Commands = append(scanRegistryCommands.Commands, scanRegistryCommand)

	scanV1Bytes, err := json.Marshal(scanRegistryCommands)
	if err != nil {
		return "", err
	}

	return string(scanV1Bytes), nil
}

var registryCommands = []string{
	string(command.OperatorCommandTypeCreateRegistry),
	string(command.OperatorCommandTypeUpdateRegistry),
	string(command.OperatorCommandTypeDeleteRegistry),
	string(command.OperatorCommandTypeCheckRegistry),
}

func isRegistryCommand(commandType string) bool {
	return slices.Contains(registryCommands, commandType)
}
