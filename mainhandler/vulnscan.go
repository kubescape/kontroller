package mainhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/operator/utils"
	"github.com/kubescape/operator/watcher"
	"go.opentelemetry.io/otel"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"

	uuid "github.com/google/uuid"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/armoapi-go/armotypes"
	apitypes "github.com/armosec/armoapi-go/armotypes"
	reporterlib "github.com/armosec/logger-go/system-reports/datastructures"
	"github.com/armosec/utils-go/httputils"
	"github.com/kubescape/k8s-interface/cloudsupport"
	"github.com/kubescape/k8s-interface/k8sinterface"
)

const (
	dockerPullableURN        = "docker-pullable://"
	cmDefaultMode     cmMode = "default"
	cmLoadedMode      cmMode = "loaded"
)

type cmMode string

type testRegistryConnectivityStatus string

const (
	testRegistryInformationStatus    testRegistryConnectivityStatus = "registryInformation"
	testRegistryAuthenticationStatus testRegistryConnectivityStatus = "registryAuthentication"
	testRegistryRetrieveReposStatus  testRegistryConnectivityStatus = "retrieveRepositories"
	testRegistryRetrieveTagsStatus   testRegistryConnectivityStatus = "retrieveTags"
	containerToImageIdsArg                                          = "containerToImageIDs"
)

func getVulnScanURL() *url.URL {
	vulnURL := url.URL{}
	vulnURL.Scheme = "http"
	vulnURL.Host = utils.ClusterConfig.KubevulnURL
	vulnURL.Path = fmt.Sprintf("%s/%s", apis.WebsocketScanCommandVersion, apis.WebsocketScanCommandPath)
	return &vulnURL
}
func sendAllImagesToVulnScan(ctx context.Context, webSocketScanCMDList []*apis.WebsocketScanCommand) error {
	var err error
	errs := make([]error, 0)
	for _, webSocketScanCMD := range webSocketScanCMDList {
		err = sendWorkloadToVulnerabilityScanner(ctx, webSocketScanCMD)
		if err != nil {
			logger.L().Ctx(ctx).Error("sendWorkloadToVulnerabilityScanner failed", helpers.Error(err))
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		err = fmt.Errorf("sendAllImagesToVulnScan errors: ")
		for errIdx := range errs {
			err = fmt.Errorf("%s; %w", err, errs[errIdx])
		}
		return err
	}
	return nil
}

func convertImagesToWebsocketScanCommand(registry *registryScan, sessionObj *utils.SessionObj) []*apis.WebsocketScanCommand {
	images := registry.mapImageToTags

	webSocketScanCMDList := make([]*apis.WebsocketScanCommand, 0)
	for repository, tags := range images {
		// registry/project/repo --> repo
		repositoryName := strings.Replace(repository, registry.registry.hostname+"/", "", -1)
		if registry.registry.projectID != "" {
			repositoryName = strings.Replace(repositoryName, registry.registry.projectID+"/", "", -1)
		}
		for _, tag := range tags {
			websocketScanCommand := &apis.WebsocketScanCommand{
				ParentJobID: sessionObj.Reporter.GetJobID(),
				JobID:       uuid.NewString(),
				ImageTag:    repository + ":" + tag,
				Session:     apis.SessionChain{ActionTitle: "vulnerability-scan", JobIDs: make([]string, 0), Timestamp: sessionObj.Reporter.GetTimestamp()},
				Args: map[string]interface{}{
					apitypes.AttributeRegistryName:  registry.registry.hostname + "/" + registry.registry.projectID,
					apitypes.AttributeRepository:    repositoryName,
					apitypes.AttributeTag:           tag,
					apitypes.AttributeUseHTTP:       !*registry.registryInfo.IsHTTPS,
					apitypes.AttributeSkipTLSVerify: registry.registryInfo.SkipTLSVerify,
					apitypes.AttributeSensor:        utils.ClusterConfig.ClusterName,
				},
			}
			// Check if auth is empty (used for public registries)
			authConfig := registry.authConfig()
			if authConfig != nil {
				websocketScanCommand.Credentialslist = append(websocketScanCommand.Credentialslist, *authConfig)
			}
			webSocketScanCMDList = append(webSocketScanCMDList, websocketScanCommand)
		}
	}

	return webSocketScanCMDList
}

func (actionHandler *ActionHandler) scanRegistries(ctx context.Context, sessionObj *utils.SessionObj) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.scanRegistries")
	defer span.End()

	registryScan, err := actionHandler.loadRegistryScan(ctx, sessionObj)
	if err != nil {
		logger.L().Ctx(ctx).Error("in parseRegistryCommand", helpers.Error(err))
		sessionObj.Reporter.SetDetails("loadRegistryScan")
		return fmt.Errorf("scanRegistries failed with err %v", err)
	}

	err = registryScan.validateRegistryScanInformation()
	if err != nil {
		logger.L().Ctx(ctx).Error("in parseRegistryCommand", helpers.Error(err))
		sessionObj.Reporter.SetDetails("validateRegistryScanInformation")
		return fmt.Errorf("scanRegistries failed with err %v", err)
	}

	return actionHandler.scanRegistry(ctx, registryScan, sessionObj)
}

func (actionHandler *ActionHandler) loadRegistryScan(ctx context.Context, sessionObj *utils.SessionObj) (*registryScan, error) {
	registryScan := NewRegistryScan(actionHandler.k8sAPI)
	var err error
	if regName := actionHandler.parseRegistryName(sessionObj); regName != "" {
		registryScan.setRegistryName(regName)
	}

	// for scan triggered by cronjob, we get the secret name
	if sessionObj.Command.CommandName == apis.TypeScanRegistry {
		secretName := actionHandler.parseSecretName(sessionObj)
		registryScan.setSecretName(secretName)
	}

	err = registryScan.parseRegistry(ctx, sessionObj)
	if err != nil {
		return nil, err
	}

	return &registryScan, nil
}

func (actionHandler *ActionHandler) testRegistryConnectivity(ctx context.Context, sessionObj *utils.SessionObj) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.testRegistryConnectivity")
	defer span.End()

	registryScan, err := actionHandler.loadRegistryScan(ctx, sessionObj)
	if err != nil {
		sessionObj.Reporter.SetDetails("loadRegistryScan")
		logger.L().Ctx(ctx).Error("in testRegistryConnectivity: loadRegistryScan failed", helpers.Error(err))
		return err
	}

	err = registryScan.validateRegistryScanInformation()
	if err != nil {
		sessionObj.Reporter.SetDetails(string(testRegistryInformationStatus))
		logger.L().Ctx(ctx).Error("in testRegistryConnectivity: validateRegistryScanInformation failed", helpers.Error(err))
		return err
	}

	err = actionHandler.testRegistryConnect(ctx, registryScan, sessionObj)
	if err != nil {
		logger.L().Ctx(ctx).Error("in testRegistryConnectivity: testRegistryConnect failed", helpers.Error(err))
		return err
	}

	return nil
}

func (actionHandler *ActionHandler) parseSecretName(sessionObj *utils.SessionObj) string {
	registryInfo, ok := sessionObj.Command.Args[armotypes.RegistryInfoArgKey].(map[string]interface{})
	if !ok {
		return ""
	}
	secretName, _ := registryInfo[secretNameField].(string)
	return secretName
}

func (actionHandler *ActionHandler) parseRegistryName(sessionObj *utils.SessionObj) string {
	registryInfo, ok := sessionObj.Command.Args[armotypes.RegistryInfoArgKey].(map[string]interface{})
	if !ok {
		return ""
	}
	registryName, ok := registryInfo[registryNameField].(string)
	if !ok {
		return ""
	}

	sessionObj.Reporter.SetTarget(fmt.Sprintf("%s: %s", apitypes.AttributeRegistryName,
		registryName))
	sessionObj.Reporter.SendDetails(fmt.Sprintf("registryInfo parsed: %v", registryInfo), true, sessionObj.ErrChan)
	return registryName
}

func (actionHandler *ActionHandler) testRegistryConnect(ctx context.Context, registry *registryScan, sessionObj *utils.SessionObj) error {
	repos, err := registry.enumerateRepos(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unauthorized") || strings.Contains(strings.ToLower(err.Error()), "DENIED") || strings.Contains(strings.ToLower(err.Error()), "authentication") || strings.Contains(strings.ToLower(err.Error()), "empty token") {
			// registry info is good, but authentication failed
			sessionObj.Reporter.SetDetails(string(testRegistryInformationStatus))
			sessionObj.Reporter.SendStatus(reporterlib.JobSuccess, true, sessionObj.ErrChan)
			sessionObj.Reporter.SetDetails(string(testRegistryAuthenticationStatus))
			return fmt.Errorf("failed to retrieve repositories: authentication error: %v", err)
		} else {
			sessionObj.Reporter.SetDetails(string(testRegistryInformationStatus))
			return fmt.Errorf("testRegistryConnect failed with error:  %v", err)
		}
	}

	sessionObj.Reporter.SetDetails(string(testRegistryInformationStatus))
	sessionObj.Reporter.SendStatus(reporterlib.JobSuccess, true, sessionObj.ErrChan)
	sessionObj.Reporter.SetDetails(string(testRegistryAuthenticationStatus))
	sessionObj.Reporter.SendStatus(reporterlib.JobSuccess, true, sessionObj.ErrChan)

	if len(repos) == 0 {
		sessionObj.Reporter.SetDetails(fmt.Sprintf("%v failed with err %v", testRegistryRetrieveReposStatus, err))
		return fmt.Errorf("failed to retrieve repositories: got empty list of repositories")
	}

	sessionObj.Reporter.SetDetails(string(testRegistryRetrieveReposStatus))
	sessionObj.Reporter.SendStatus(reporterlib.JobSuccess, true, sessionObj.ErrChan)

	// check that we can pull tags. One is enough
	if len(repos) > 0 {
		reposToTags := make(chan map[string][]string, 1)
		if err := registry.setImageToTagsMap(ctx, repos[0], sessionObj.Reporter, reposToTags); err != nil {
			sessionObj.Reporter.SetDetails(string(testRegistryRetrieveTagsStatus))
			return fmt.Errorf("setImageToTagsMap failed with err %v", err)
		}
	}

	sessionObj.Reporter.SetDetails(string(testRegistryRetrieveTagsStatus))
	sessionObj.Reporter.SendStatus(reporterlib.JobSuccess, true, sessionObj.ErrChan)

	var repositories []armotypes.Repository
	for _, repo := range repos {
		repositories = append(repositories, armotypes.Repository{
			RepositoryName: repo,
		})
	}

	params := RepositoriesAndTagsParams{
		RegistryName: registry.registryInfo.RegistryName,
		CustomerGUID: sessionObj.Reporter.GetCustomerGUID(),
		JobID:        sessionObj.Reporter.GetJobID(),
		Repositories: repositories,
	}

	err = registry.SendRepositoriesAndTags(params)
	if err != nil {
		return err
	}
	return nil
}

func (actionHandler *ActionHandler) scanRegistry(ctx context.Context, registry *registryScan, sessionObj *utils.SessionObj) error {
	err := registry.getImagesForScanning(ctx, actionHandler.reporter)
	if err != nil {
		return fmt.Errorf("GetImagesForScanning failed with err %v", err)
	}
	webSocketScanCMDList := convertImagesToWebsocketScanCommand(registry, sessionObj)
	sessionObj.Reporter.SendDetails(fmt.Sprintf("sending %d images from registry %v to vuln scan", len(webSocketScanCMDList), registry.registry), true, sessionObj.ErrChan)

	return sendAllImagesToVulnScan(ctx, webSocketScanCMDList)
}

func (actionHandler *ActionHandler) scanWorkload(ctx context.Context, sessionObj *utils.SessionObj) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.scanWorkload")
	defer span.End()

	workload, err := actionHandler.k8sAPI.GetWorkloadByWlid(actionHandler.wlid)
	if err != nil {
		return fmt.Errorf("failed to get workload %s with err %v", actionHandler.wlid, err)
	}

	pod := actionHandler.getPodByWLID(ctx, workload)
	if pod == nil {
		logger.L().Info(fmt.Sprintf("workload %s has no podSpec, skipping", actionHandler.wlid))
		return nil
	}
	mapContainerToImageID := make(map[string]string) // map of container name to image ID. Container name is unique per pod

	// look for container to imageID map in the command args. If not found, look for it in the wl. If not found, get it from the pod
	if val, ok := actionHandler.command.Args[containerToImageIdsArg].(map[string]string); !ok {
		// get from pod
		if len(pod.Status.ContainerStatuses) == 0 {
			mapContainerToImageID, err = actionHandler.getContainerToImageIDsMap(workload)
			if err != nil {
				logger.L().Ctx(ctx).Error(fmt.Sprintf("failed to get container to image ID map for workload %s with err %v", actionHandler.wlid, err))
			}
		} else {
			// get from wl
			for contIdx := range pod.Status.ContainerStatuses {
				imageID := pod.Status.ContainerStatuses[contIdx].ImageID
				mapContainerToImageID[pod.Status.ContainerStatuses[contIdx].Name] = watcher.ExtractImageID(imageID)
			}
		}
	} else {
		// get from args
		mapContainerToImageID = val
	}

	// get all images of workload
	errs := ""
	containers, err := listWorkloadImages(workload)
	if err != nil {
		return fmt.Errorf("failed to get workloads from k8s, wlid: %s, reason: %s", actionHandler.wlid, err.Error())
	}

	for i := range containers {
		imgID := ""
		if val, ok := mapContainerToImageID[containers[i].container]; !ok {
			logger.L().Ctx(ctx).Error(fmt.Sprintf("failed to get image ID for container %s", containers[i].container))
			continue
		} else {
			imgID = val
		}
		websocketScanCommand := &apis.WebsocketScanCommand{
			Wlid:          actionHandler.wlid,
			ImageTag:      containers[i].image,
			ContainerName: containers[i].container,
			Session:       apis.SessionChain{ActionTitle: "vulnerability-scan", JobIDs: make([]string, 0), Timestamp: sessionObj.Reporter.GetTimestamp()},
			ImageHash:     imgID,
		}

		if actionHandler.reporter != nil {

			prepareSessionChain(sessionObj, websocketScanCommand, actionHandler)

			logger.L().Info(fmt.Sprintf("wlid: %s, container: %s, image: %s, jobIDs: %s/%s/%s", websocketScanCommand.Wlid, websocketScanCommand.ContainerName, websocketScanCommand.ImageTag, actionHandler.reporter.GetParentAction(), websocketScanCommand.ParentJobID, websocketScanCommand.JobID))
		}
		if pod != nil {
			secrets, err := cloudsupport.GetImageRegistryCredentials(websocketScanCommand.ImageTag, pod)
			if err != nil {
				logger.L().Ctx(ctx).Error(err.Error(), helpers.Error(err))
			} else if len(secrets) > 0 {
				for secretName := range secrets {
					websocketScanCommand.Credentialslist = append(websocketScanCommand.Credentialslist, secrets[secretName])
				}

				/*
					the websocketScanCommand.Credentials is deprecated, still use it for backward computability
				*/
				if len(websocketScanCommand.Credentialslist) != 0 {
					websocketScanCommand.Credentials = &websocketScanCommand.Credentialslist[0]
				}
			}
		}
		if err := sendWorkloadToVulnerabilityScanner(ctx, websocketScanCommand); err != nil {
			logger.L().Ctx(ctx).Error("scanning failed", helpers.String("image", websocketScanCommand.ImageTag), helpers.Error(err))
			errs += fmt.Sprintf("failed scanning, wlid: '%s', image: '%s', container: %s, reason: %s", actionHandler.wlid, containers[i].image, containers[i].container, err.Error())

		}

	}
	if errs != "" {
		return fmt.Errorf(errs)
	}
	return nil
}

func prepareSessionChain(sessionObj *utils.SessionObj, websocketScanCommand *apis.WebsocketScanCommand, actionHandler *ActionHandler) {
	sessionParentJobId := sessionObj.Reporter.GetParentAction()
	if sessionParentJobId != "" {
		websocketScanCommand.Session.JobIDs = append(websocketScanCommand.Session.JobIDs, sessionParentJobId)
		websocketScanCommand.Session.RootJobID = sessionParentJobId
	}
	sessionJobID := sessionObj.Reporter.GetJobID()
	if websocketScanCommand.Session.RootJobID == "" {
		websocketScanCommand.Session.RootJobID = sessionJobID
	}
	websocketScanCommand.Session.JobIDs = append(websocketScanCommand.Session.JobIDs, sessionJobID)

	if actionHandler.reporter.GetParentAction() != "" && !slices.Contains(websocketScanCommand.Session.JobIDs, actionHandler.reporter.GetParentAction()) {
		websocketScanCommand.Session.JobIDs = append(websocketScanCommand.Session.JobIDs, actionHandler.reporter.GetParentAction())
	}

	if actionHandler.reporter.GetJobID() != "" && !slices.Contains(websocketScanCommand.Session.JobIDs, actionHandler.reporter.GetJobID()) {
		websocketScanCommand.Session.JobIDs = append(websocketScanCommand.Session.JobIDs, actionHandler.reporter.GetJobID())
	}

	websocketScanCommand.ParentJobID = actionHandler.reporter.GetJobID()
	websocketScanCommand.LastAction = actionHandler.reporter.GetActionIDN()
	websocketScanCommand.JobID = uuid.NewString()
	websocketScanCommand.Session.JobIDs = append(websocketScanCommand.Session.JobIDs, websocketScanCommand.JobID)
}

func sendWorkloadToVulnerabilityScanner(ctx context.Context, websocketScanCommand *apis.WebsocketScanCommand) error {

	jsonScannerC, err := json.Marshal(websocketScanCommand)
	if err != nil {
		return err
	}
	vulnURL := getVulnScanURL()

	creds := websocketScanCommand.Credentials
	credsList := websocketScanCommand.Credentialslist
	hasCreds := creds != nil && len(creds.Username) > 0 && len(creds.Password) > 0 || len(credsList) > 0
	logger.L().Info(fmt.Sprintf("requesting scan. url: %s wlid: %s image: %s with credentials: %v", vulnURL.String(), websocketScanCommand.Wlid, websocketScanCommand.ImageTag, hasCreds))

	resp, err := httputils.HttpPost(VulnScanHttpClient, vulnURL.String(), map[string]string{"Content-Type": "application/json"}, jsonScannerC)
	refusedNum := 0
	for ; refusedNum < 5 && err != nil && strings.Contains(err.Error(), "connection refused"); resp, err = httputils.HttpPost(VulnScanHttpClient, vulnURL.String(), map[string]string{"Content-Type": "application/json"}, jsonScannerC) {
		logger.L().Ctx(ctx).Error("failed posting to vulnerability scanner", helpers.String("query", websocketScanCommand.ImageTag), helpers.Error(err))
		time.Sleep(5 * time.Second)
		refusedNum++
	}
	if err != nil {
		return fmt.Errorf("failed posting to vulnerability scanner. query: '%s', reason: %s", websocketScanCommand.ImageTag, err.Error())
	}
	if resp == nil {
		return fmt.Errorf("failed posting to vulnerability scanner. query: '%s', reason: 'empty response'", websocketScanCommand.ImageTag)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode > 203 {
		return fmt.Errorf("failed posting to vulnerability scanner. query: '%s', reason: 'received bad status code: %d'", websocketScanCommand.ImageTag, resp.StatusCode)
	}
	return nil
}

func (actionHandler *ActionHandler) getPodByWLID(ctx context.Context, workload k8sinterface.IWorkload) *corev1.Pod {
	var err error

	podspec, err := workload.GetPodSpec()
	if err != nil {
		return nil
	}
	podObj := &corev1.Pod{Spec: *podspec}
	if workload.GetKind() == "Pod" {
		status, err := workload.GetPodStatus()
		if err != nil {
			logger.L().Ctx(ctx).Error("failed getting pod status", helpers.String("wlid", actionHandler.wlid), helpers.Error(err))
		} else {
			podObj.Status = *status
		}
	}

	podObj.ObjectMeta.Namespace = workload.GetNamespace()
	return podObj
}

// get a workload, retrieves its pod and returns a map of container name to image id
func (actionHandler *ActionHandler) getContainerToImageIDsMap(workload k8sinterface.IWorkload) (map[string]string, error) {
	var mapContainerToImageID = make(map[string]string)

	labels := workload.GetPodLabels()

	pods, err := actionHandler.k8sAPI.ListPods(workload.GetNamespace(), labels)
	if err != nil {
		return nil, fmt.Errorf("failed listing pods for workload %s", workload.GetName())
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for workload %s", workload.GetName())
	}
	pod := pods.Items[0]

	containerStatuses := pod.Status.ContainerStatuses
	if len(containerStatuses) == 0 {
		return nil, fmt.Errorf("no containers found for workload %s", workload.GetName())
	}

	for containerStatus := range containerStatuses {
		imageID := pod.Status.ContainerStatuses[containerStatus].ImageID
		mapContainerToImageID[pod.Status.ContainerStatuses[containerStatus].Name] = watcher.ExtractImageID(imageID)
	}

	return mapContainerToImageID, nil
}
