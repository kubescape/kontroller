package mainhandler

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	core1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/operator/config"
	cs "github.com/kubescape/operator/continuousscanning"
	"github.com/kubescape/operator/utils"
	"github.com/kubescape/operator/watcher"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"

	"github.com/armosec/armoapi-go/identifiers"
	"github.com/armosec/utils-go/boolutils"
	"github.com/armosec/utils-go/httputils"

	"github.com/armosec/armoapi-go/apis"

	uuid "github.com/google/uuid"
	v1 "github.com/kubescape/opa-utils/httpserver/apis/v1"
	utilsmetav1 "github.com/kubescape/opa-utils/httpserver/meta/v1"

	pkgwlid "github.com/armosec/utils-k8s-go/wlid"
	beClientV1 "github.com/kubescape/backend/pkg/client/v1"
	"github.com/kubescape/backend/pkg/server/v1/systemreports"
	instanceidhandlerv1 "github.com/kubescape/k8s-interface/instanceidhandler/v1"
	"github.com/kubescape/k8s-interface/k8sinterface"
	kssc "github.com/kubescape/storage/pkg/generated/clientset/versioned"
)

type MainHandler struct {
	eventWorkerPool        *ants.PoolWithFunc
	k8sAPI                 *k8sinterface.KubernetesApi
	commandResponseChannel *commandResponseChannelData
	config                 config.IConfig
	sendReport             bool
}

type ActionHandler struct {
	command                apis.Command
	reporter               beClientV1.IReportSender
	config                 config.IConfig
	k8sAPI                 *k8sinterface.KubernetesApi
	commandResponseChannel *commandResponseChannelData
	wlid                   string
	sendReport             bool
}

type waitFunc func(clusterConfig config.IConfig)

var k8sNamesRegex *regexp.Regexp
var actionNeedToBeWaitOnStartUp = map[apis.NotificationPolicyType]waitFunc{}
var KubescapeHttpClient httputils.IHttpClient
var VulnScanHttpClient httputils.IHttpClient

func init() {
	var err error
	k8sNamesRegex, err = regexp.Compile("[^A-Za-z0-9-]+")
	if err != nil {
		logger.L().Fatal(err.Error(), helpers.Error(err))
	}

	actionNeedToBeWaitOnStartUp[apis.TypeScanImages] = waitForVulnScanReady
	actionNeedToBeWaitOnStartUp[apis.TypeRunKubescape] = waitForKubescapeReady
}

// CreateWebSocketHandler Create ws-handler obj
func NewMainHandler(config config.IConfig, k8sAPI *k8sinterface.KubernetesApi) *MainHandler {

	commandResponseChannel := make(chan *CommandResponseData, 100)
	limitedGoRoutinesCommandResponseChannel := make(chan *timerData, 10)
	mainHandler := &MainHandler{
		k8sAPI:                 k8sAPI,
		commandResponseChannel: &commandResponseChannelData{commandResponseChannel: &commandResponseChannel, limitedGoRoutinesCommandResponseChannel: &limitedGoRoutinesCommandResponseChannel},
		config:                 config,
		sendReport:             config.EventReceiverURL() != "",
	}
	pool, _ := ants.NewPoolWithFunc(config.ConcurrencyWorkers(), func(i interface{}) {
		j, ok := i.(utils.Job)
		if !ok {
			logger.L().Error("failed to cast job", helpers.Interface("job", i))
			return
		}
		mainHandler.handleRequest(j)
	})
	mainHandler.eventWorkerPool = pool
	return mainHandler
}

// CreateWebSocketHandler Create ws-handler obj
func NewActionHandler(config config.IConfig, k8sAPI *k8sinterface.KubernetesApi, sessionObj *utils.SessionObj, commandResponseChannel *commandResponseChannelData) *ActionHandler {
	return &ActionHandler{
		reporter:               sessionObj.Reporter,
		command:                sessionObj.Command,
		k8sAPI:                 k8sAPI,
		commandResponseChannel: commandResponseChannel,
		config:                 config,
		sendReport:             config.EventReceiverURL() != "",
	}
}

// SetupContinuousScanning sets up the continuous cluster scanning function
func (mainHandler *MainHandler) SetupContinuousScanning(ctx context.Context, queueSize int, eventCooldown time.Duration) error {
	ksStorageClient, err := kssc.NewForConfig(k8sinterface.GetK8sConfig())
	if err != nil {
		logger.L().Ctx(ctx).Fatal(fmt.Sprintf("Unable to initialize the storage client: %v", err))
	}

	triggeringHandler := cs.NewTriggeringHandler(mainHandler.eventWorkerPool, mainHandler.config)
	deletingHandler := cs.NewDeletedCleanerHandler(mainHandler.eventWorkerPool, mainHandler.config, ksStorageClient)

	rulesFilename := mainHandler.config.MatchingRulesFilename()
	rulesReader, err := os.Open(rulesFilename)
	if err != nil {
		return err
	}

	fetcher := cs.NewFileFetcher(rulesReader)
	loader := cs.NewTargetLoader(fetcher)

	dynClient := mainHandler.k8sAPI.DynamicClient
	svc := cs.NewContinuousScanningService(dynClient, loader, queueSize, eventCooldown, triggeringHandler, deletingHandler)
	svc.Launch(ctx)

	return nil
}

func (mainHandler *MainHandler) HandleWatchers(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Fatal("recover in HandleWatchers", helpers.Interface("error", err))
		}
	}()

	ksStorageClient, err := kssc.NewForConfig(k8sinterface.GetK8sConfig())
	if err != nil {
		logger.L().Ctx(ctx).Fatal(fmt.Sprintf("Unable to initialize the storage client: %v", err))
	}
	eventQueue := watcher.NewCooldownQueue(watcher.DefaultQueueSize, watcher.DefaultTTL)
	watchHandler := watcher.NewWatchHandler(ctx, mainHandler.config, mainHandler.k8sAPI, ksStorageClient, eventQueue)

	// wait for the kubevuln component to be ready
	logger.L().Ctx(ctx).Info("Waiting for vuln scan to be ready")
	waitFunc := isActionNeedToWait(apis.Command{CommandName: apis.TypeScanImages})
	waitFunc(mainHandler.config)

	// start watching
	go watchHandler.PodWatch(ctx, mainHandler.eventWorkerPool)
	go watchHandler.SBOMFilteredWatch(ctx, mainHandler.eventWorkerPool)
}

func (h *MainHandler) StartContinuousScanning(ctx context.Context) error {
	return nil
}

// HandlePostmanRequest Parse received commands and run the command
func (mainHandler *MainHandler) handleRequest(j utils.Job) {

	ctx := j.Context()
	sessionObj := j.Obj()

	// recover
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Fatal("recover in HandleRequest", helpers.Interface("error", err))
		}
	}()

	ctx, span := otel.Tracer("").Start(ctx, string(sessionObj.Command.CommandName))

	// the all user experience depends on this line(the user/backend must get the action name in order to understand the job report)
	sessionObj.Reporter.SetActionName(string(sessionObj.Command.CommandName))

	isToItemizeScopeCommand := sessionObj.Command.WildWlid != "" || sessionObj.Command.WildSid != "" || len(sessionObj.Command.Designators) > 0
	switch sessionObj.Command.CommandName {
	case apis.TypeRunKubescape, apis.TypeRunKubescapeJob, apis.TypeSetKubescapeCronJob, apis.TypeDeleteKubescapeCronJob, apis.TypeUpdateKubescapeCronJob:
		isToItemizeScopeCommand = false

	case apis.TypeSetVulnScanCronJob, apis.TypeDeleteVulnScanCronJob, apis.TypeUpdateVulnScanCronJob:
		isToItemizeScopeCommand = false
	}

	if isToItemizeScopeCommand {
		if sessionObj.Command.CommandName == apis.TypeScanImages {
			mainHandler.HandleImageScanningScopedRequest(ctx, &sessionObj)
		} else {
			// TODO: handle scope request
			// I'm not sure when we will need this case
			mainHandler.HandleScopedRequest(ctx, &sessionObj) // this might be a heavy action, do not send to a goroutine
		}
	} else {
		// handle requests
		if err := mainHandler.HandleSingleRequest(ctx, &sessionObj); err != nil {
			logger.L().Ctx(ctx).Error("failed to complete action", helpers.String("command", string(sessionObj.Command.CommandName)), helpers.String("wlid", sessionObj.Command.GetID()), helpers.Error(err))
			sessionObj.Reporter.SendError(err, mainHandler.sendReport, true)
		} else {
			sessionObj.Reporter.SendStatus(systemreports.JobDone, mainHandler.sendReport)
			logger.L().Ctx(ctx).Info("action completed successfully", helpers.String("command", string(sessionObj.Command.CommandName)), helpers.String("wlid", sessionObj.Command.GetID()))
		}
	}
	span.End()
}

func (mainHandler *MainHandler) HandleSingleRequest(ctx context.Context, sessionObj *utils.SessionObj) error {
	ctx, span := otel.Tracer("").Start(ctx, "mainHandler.HandleSingleRequest")
	defer span.End()

	actionHandler := NewActionHandler(mainHandler.config, mainHandler.k8sAPI, sessionObj, mainHandler.commandResponseChannel)
	actionHandler.reporter.SetActionName(string(sessionObj.Command.CommandName))
	actionHandler.reporter.SendDetails("Handling single request", mainHandler.sendReport)

	return actionHandler.runCommand(ctx, sessionObj)

}

func (actionHandler *ActionHandler) runCommand(ctx context.Context, sessionObj *utils.SessionObj) error {
	c := sessionObj.Command
	if pkgwlid.IsWlid(c.GetID()) {
		actionHandler.wlid = c.GetID()
	}

	switch c.CommandName {
	case apis.TypeScanImages:
		return actionHandler.scanImage(ctx, sessionObj)
	case utils.CommandScanFilteredSBOM:
		actionHandler.scanFilteredSBOM(ctx, sessionObj)
	case apis.TypeRunKubescape, apis.TypeRunKubescapeJob:
		return actionHandler.kubescapeScan(ctx)
	case apis.TypeSetKubescapeCronJob:
		return actionHandler.setKubescapeCronJob(ctx)
	case apis.TypeUpdateKubescapeCronJob:
		return actionHandler.updateKubescapeCronJob(ctx)
	case apis.TypeDeleteKubescapeCronJob:
		return actionHandler.deleteKubescapeCronJob(ctx)
	case apis.TypeSetVulnScanCronJob:
		return actionHandler.setVulnScanCronJob(ctx)
	case apis.TypeUpdateVulnScanCronJob:
		return actionHandler.updateVulnScanCronJob(ctx)
	case apis.TypeDeleteVulnScanCronJob:
		return actionHandler.deleteVulnScanCronJob(ctx)
	case apis.TypeSetRegistryScanCronJob:
		return actionHandler.setRegistryScanCronJob(ctx, sessionObj)
	case apis.TypeScanRegistry:
		return actionHandler.scanRegistries(ctx, sessionObj)
	case apis.TypeTestRegistryConnectivity:
		return actionHandler.testRegistryConnectivity(ctx, sessionObj)
	case apis.TypeUpdateRegistryScanCronJob:
		return actionHandler.updateRegistryScanCronJob(ctx, sessionObj)
	case apis.TypeDeleteRegistryScanCronJob:
		return actionHandler.deleteRegistryScanCronJob(ctx)
	default:
		logger.L().Ctx(ctx).Error(fmt.Sprintf("Command %s not found", c.CommandName))
	}
	return nil
}

// HandleScopedRequest handle a request of a scope e.g. all workloads in a namespace
func (mainHandler *MainHandler) HandleScopedRequest(ctx context.Context, sessionObj *utils.SessionObj) {
	ctx, span := otel.Tracer("").Start(ctx, "mainHandler.HandleScopedRequest")
	defer span.End()

	if sessionObj.Command.GetID() == "" {
		logger.L().Ctx(ctx).Error("Received empty id")
		return
	}

	namespaces := make([]string, 0, 1)
	namespaces = append(namespaces, pkgwlid.GetNamespaceFromWlid(sessionObj.Command.GetID()))
	labels := sessionObj.Command.GetLabels()
	fields := sessionObj.Command.GetFieldSelector()
	if len(sessionObj.Command.Designators) > 0 {
		namespaces = make([]string, 0, 3)
		for desiIdx := range sessionObj.Command.Designators {
			if ns, ok := sessionObj.Command.Designators[desiIdx].Attributes[identifiers.AttributeNamespace]; ok {
				namespaces = append(namespaces, ns)
			}
		}
	}
	if len(namespaces) == 0 {
		namespaces = append(namespaces, "")
	}

	info := fmt.Sprintf("%s: id: '%s', namespaces: '%v', labels: '%v', fieldSelector: '%v'", sessionObj.Command.CommandName, sessionObj.Command.GetID(), namespaces, labels, fields)
	logger.L().Info(info)
	sessionObj.Reporter.SendDetails(info, mainHandler.sendReport)

	ids, errs := mainHandler.getIDs(namespaces, labels, fields, []string{"pods"})
	for i := range errs {
		logger.L().Ctx(ctx).Warning(errs[i].Error())
		sessionObj.Reporter.SendError(errs[i], mainHandler.sendReport, true)
	}

	sessionObj.Reporter.SendStatus(systemreports.JobSuccess, mainHandler.sendReport)

	logger.L().Info(fmt.Sprintf("ids found: '%v'", ids))

	for i := range ids {
		cmd := sessionObj.Command.DeepCopy()

		var err error
		if pkgwlid.IsWlid(ids[i]) {
			cmd.Wlid = ids[i]
			err = pkgwlid.IsWlidValid(cmd.Wlid)
		} else {
			err = fmt.Errorf("unknown id")
		}

		// clean all scope request parameters
		cmd.WildWlid = ""
		cmd.Designators = make([]identifiers.PortalDesignator, 0)

		// send specific command to the channel
		newSessionObj := utils.NewSessionObj(ctx, mainHandler.config, cmd, "Websocket", sessionObj.Reporter.GetJobID(), "", 1)

		if err != nil {
			err := fmt.Errorf("invalid: %s, id: '%s'", err.Error(), newSessionObj.Command.GetID())
			logger.L().Ctx(ctx).Error(err.Error())
			sessionObj.Reporter.SendError(err, mainHandler.sendReport, true)
			continue
		}

		logger.L().Info("triggering", helpers.String("id", newSessionObj.Command.GetID()))
		if err := mainHandler.HandleSingleRequest(ctx, newSessionObj); err != nil {
			logger.L().Ctx(ctx).Error("failed to complete action", helpers.String("command", string(sessionObj.Command.CommandName)), helpers.String("wlid", sessionObj.Command.GetID()), helpers.Error(err))
			sessionObj.Reporter.SendError(err, mainHandler.sendReport, true)
			continue
		}
		sessionObj.Reporter.SendStatus(systemreports.JobDone, mainHandler.sendReport)
		logger.L().Ctx(ctx).Info("action completed successfully", helpers.String("command", string(sessionObj.Command.CommandName)), helpers.String("wlid", sessionObj.Command.GetID()))
	}
}

// HandleScopedRequest handle a request of a scope e.g. all workloads in a namespace
func (mainHandler *MainHandler) HandleImageScanningScopedRequest(ctx context.Context, sessionObj *utils.SessionObj) {
	ctx, span := otel.Tracer("").Start(ctx, "mainHandler.HandleImageScanningScopedRequest")
	defer span.End()

	if sessionObj.Command.GetID() == "" {
		logger.L().Ctx(ctx).Error("Received empty id")
		return
	}

	namespaces := make([]string, 0, 1)
	namespaces = append(namespaces, pkgwlid.GetNamespaceFromWlid(sessionObj.Command.GetID()))
	labels := sessionObj.Command.GetLabels()
	fields := sessionObj.Command.GetFieldSelector()
	if len(sessionObj.Command.Designators) > 0 {
		namespaces = make([]string, 0, 3)
		for desiIdx := range sessionObj.Command.Designators {
			if ns, ok := sessionObj.Command.Designators[desiIdx].Attributes[identifiers.AttributeNamespace]; ok {
				namespaces = append(namespaces, ns)
			}
		}
	}
	if len(namespaces) == 0 {
		namespaces = append(namespaces, "")
	}

	info := fmt.Sprintf("%s: id: '%s', namespaces: '%v', labels: '%v', fieldSelector: '%v'", sessionObj.Command.CommandName, sessionObj.Command.GetID(), namespaces, labels, fields)
	logger.L().Info(info)
	sessionObj.Reporter.SendDetails(info, mainHandler.sendReport)

	listOptions := metav1.ListOptions{
		LabelSelector: k8sinterface.SelectorToString(labels),
		FieldSelector: k8sinterface.SelectorToString(fields),
	}

	sessionObj.Reporter.SendStatus(systemreports.JobSuccess, mainHandler.sendReport)

	slugs := map[string]bool{}

	for _, ns := range namespaces {
		pods, err := mainHandler.k8sAPI.KubernetesClient.CoreV1().Pods(ns).List(ctx, listOptions)
		if err != nil {
			logger.L().Ctx(ctx).Error("failed to list pods", helpers.String("namespace", ns), helpers.Error(err))
			sessionObj.Reporter.SendError(err, mainHandler.sendReport, true)
			continue
		}
		for i := range pods.Items {
			pod := pods.Items[i]
			if pod.Status.Phase != core1.PodRunning {
				// skip non-running pods, for some reason the list includes non-running pods
				continue
			}
			pod.APIVersion = "v1"
			pod.Kind = "Pod"

			// get pod instanceIDs
			instanceIDs, err := instanceidhandlerv1.GenerateInstanceIDFromPod(&pod)
			if err != nil {
				logger.L().Ctx(ctx).Error("failed to generate instance ID for pod", helpers.String("pod", pod.GetName()), helpers.String("namespace", pod.GetNamespace()), helpers.Error(err))
				continue
			}

			for _, instanceID := range instanceIDs {
				s, _ := instanceID.GetSlug()
				if ok := slugs[s]; ok {
					// slug already scanned, there is no need to scan again in this request
					continue
				}

				// get container data
				containerData, err := utils.PodToContainerData(mainHandler.k8sAPI, &pod, instanceID, mainHandler.config.ClusterName())
				if err != nil {
					// if pod is not running, we can't get the image id
					continue
				}

				// set scanning command
				cmd := &apis.Command{
					Wlid:        containerData.Wlid,
					CommandName: apis.TypeScanImages,
					Args: map[string]interface{}{
						utils.ArgsContainerData: containerData,
						utils.ArgsPod:           &pod,
					},
				}

				// send specific command to the channel
				newSessionObj := utils.NewSessionObj(ctx, mainHandler.config, cmd, "Websocket", sessionObj.Reporter.GetJobID(), "", 1)

				logger.L().Info("triggering", helpers.String("id", newSessionObj.Command.GetID()), helpers.String("slug", s), helpers.String("containerName", containerData.ContainerName), helpers.String("imageTag", containerData.ImageTag), helpers.String("imageID", containerData.ImageID))
				if err := mainHandler.HandleSingleRequest(ctx, newSessionObj); err != nil {
					logger.L().Info("failed to complete action", helpers.String("id", newSessionObj.Command.GetID()), helpers.String("slug", s), helpers.String("containerName", containerData.ContainerName), helpers.String("imageTag", containerData.ImageTag), helpers.String("imageID", containerData.ImageID))
					newSessionObj.Reporter.SendError(err, mainHandler.sendReport, true)
					continue
				}
				newSessionObj.Reporter.SendStatus(systemreports.JobDone, mainHandler.sendReport)
				logger.L().Info("action completed successfully", helpers.String("id", newSessionObj.Command.GetID()), helpers.String("slug", s), helpers.String("containerName", containerData.ContainerName), helpers.String("imageTag", containerData.ImageTag), helpers.String("imageID", containerData.ImageID))
				slugs[s] = true
			}
		}
	}
}

func (mainHandler *MainHandler) getIDs(namespaces []string, labels, fields map[string]string, resources []string) ([]string, []error) {
	ids := []string{}
	errs := []error{}
	for _, resource := range resources {
		workloads, err := mainHandler.listWorkloads(namespaces, resource, labels, fields)
		if err != nil {
			errs = append(errs, err)
		}
		if len(workloads) == 0 {
			continue
		}
		w, e := mainHandler.getResourcesIDs(workloads)
		if len(e) != 0 {
			errs = append(errs, e...)
		}
		if len(w) == 0 {
			err := fmt.Errorf("resource: '%s', failed to calculate workloadIDs. namespaces: '%v', labels: '%v'", resource, namespaces, labels)
			errs = append(errs, err)
		}
		ids = append(ids, w...)
	}

	return ids, errs
}

// HandlePostmanRequest Parse received commands and run the command
func (mainHandler *MainHandler) StartupTriggerActions(ctx context.Context, actions []apis.Command) {

	for i := range actions {
		go func(index int) {
			waitFunc := isActionNeedToWait(actions[index])
			waitFunc(mainHandler.config)
			sessionObj := utils.NewSessionObj(ctx, mainHandler.config, &actions[index], "Websocket", "", uuid.NewString(), 1)
			l := utils.Job{}
			l.SetContext(ctx)
			l.SetObj(*sessionObj)
			if err := mainHandler.eventWorkerPool.Invoke(l); err != nil {
				logger.L().Ctx(ctx).Error("failed to invoke job", helpers.String("wlid", actions[index].GetID()), helpers.String("command", fmt.Sprintf("%v", actions[index].CommandName)), helpers.String("args", fmt.Sprintf("%v", actions[index].Args)), helpers.Error(err))
			}
		}(i)
	}
}

func (mainHandler *MainHandler) EventWorkerPool() *ants.PoolWithFunc {
	return mainHandler.eventWorkerPool
}

func GetStartupActions(config config.IConfig) []apis.Command {
	return []apis.Command{
		{
			CommandName: apis.TypeRunKubescape,
			WildWlid:    pkgwlid.GetK8sWLID(config.ClusterName(), "", "", ""),
			Args: map[string]interface{}{
				utils.KubescapeScanV1: utilsmetav1.PostScanRequest{
					HostScanner: boolutils.BoolPointer(false),
					TargetType:  v1.KindFramework,
					TargetNames: []string{"allcontrols", "nsa", "mitre"},
				},
			},
		},
	}
}
