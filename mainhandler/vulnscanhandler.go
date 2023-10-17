package mainhandler

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/kubescape/go-logger"
	"go.opentelemetry.io/otel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const VulnScanCronjobTemplateName = "kubevuln-cronjob-template"

func (actionHandler *ActionHandler) setVulnScanCronJob(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.setVulnScanCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubevulnScheduler.Enabled {
		return errors.New("KubevulnScheduler is not enabled")
	}

	req := getVulnScanRequest(&actionHandler.command)

	name := fixK8sCronJobNameLimit(fmt.Sprintf("%s-%d", "kubevuln-schedule", rand.NewSource(time.Now().UnixNano()).Int63()))

	if err := createConfigMapForTriggerRequest(actionHandler.k8sAPI, actionHandler.config.Namespace(), name, req); err != nil {
		return err
	}

	jobTemplateObj, err := getCronJobTemplate(actionHandler.k8sAPI, VulnScanCronjobTemplateName, actionHandler.config.Namespace())
	if err != nil {
		return err
	}

	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.CronTabSchedule == "" {
		return fmt.Errorf("setVulnScanCronJob: CronTabSchedule not found")
	}
	setCronJobForTriggerRequest(jobTemplateObj, name, scanJobParams.CronTabSchedule, actionHandler.command.JobTracking.JobID)

	// add namespace annotation
	namespace := getNamespaceFromVulnScanCommand(&actionHandler.command)
	logger.L().Info(fmt.Sprintf("setVulnScanCronJob: command namespace - '%s'", namespace))
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armotypes.CronJobTemplateAnnotationNamespaceKeyDeprecated] = namespace // deprecated
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armotypes.CronJobTemplateAnnotationNamespaceKey] = namespace

	if _, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Create(context.Background(), jobTemplateObj, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (actionHandler *ActionHandler) updateVulnScanCronJob(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.updateVulnScanCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubevulnScheduler.Enabled {
		return errors.New("KubevulnScheduler is not enabled")
	}

	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.CronTabSchedule == "" {
		return fmt.Errorf("updateVulnScanCronJob: CronTabSchedule not found")
	}
	if scanJobParams.JobName == "" {
		return fmt.Errorf("updateVulnScanCronJob: jobName not found")
	}

	jobTemplateObj, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Get(context.Background(), scanJobParams.JobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	jobTemplateObj.Spec.Schedule = scanJobParams.CronTabSchedule
	if jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations == nil {
		jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations = make(map[string]string)
	}
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armotypes.CronJobTemplateAnnotationUpdateJobIDDeprecated] = actionHandler.command.JobTracking.JobID // deprecated
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armotypes.CronJobTemplateAnnotationUpdateJobID] = actionHandler.command.JobTracking.JobID

	_, err = actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Update(context.Background(), jobTemplateObj, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (actionHandler *ActionHandler) deleteVulnScanCronJob(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.deleteVulnScanCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubevulnScheduler.Enabled {
		return errors.New("KubevulnScheduler is not enabled")
	}

	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.JobName == "" {
		return fmt.Errorf("deleteVulnScanCronJob: CronTabSchedule not found")
	}

	return actionHandler.deleteCronjob(scanJobParams.JobName, actionHandler.config.Namespace())

}

func (actionHandler *ActionHandler) deleteCronjob(name, namespace string) error {
	if err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := actionHandler.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil

}
