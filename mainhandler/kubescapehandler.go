package mainhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/operator/config"
	"github.com/kubescape/operator/utils"
	"go.opentelemetry.io/otel"

	armoapi "github.com/armosec/armoapi-go/apis"
	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/utils-go/httputils"
	utilsapisv1 "github.com/kubescape/opa-utils/httpserver/apis/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WaitTimeForKubescapeScanResponse = 40
	KubescapeCronJobTemplateName     = "kubescape-cronjob-template"
)

type kubescapeResponseData struct {
	scanID     string
	sessionObj *utils.SessionObj
}

func (actionHandler *ActionHandler) deleteKubescapeCronJob(ctx context.Context) error {
	_, span := otel.Tracer("").Start(ctx, "actionHandler.deleteKubescapeCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubescapeScheduler.Enabled {
		return errors.New("KubescapeScheduler is not enabled")
	}

	kubescapeJobParams := getKubescapeJobParams(actionHandler.sessionObj.Command)
	if kubescapeJobParams == nil {
		return fmt.Errorf("failed to convert kubescapeJobParams list to KubescapeJobParams")
	}

	if err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Delete(context.Background(), kubescapeJobParams.JobName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := actionHandler.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(actionHandler.config.Namespace()).Delete(context.Background(), kubescapeJobParams.JobName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (actionHandler *ActionHandler) updateKubescapeCronJob(ctx context.Context) error {
	_, span := otel.Tracer("").Start(ctx, "actionHandler.updateKubescapeCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubescapeScheduler.Enabled {
		return errors.New("KubescapeScheduler is not enabled")
	}

	jobParams := getKubescapeJobParams(actionHandler.sessionObj.Command)
	if jobParams == nil {
		return fmt.Errorf("failed to convert kubescapeJobParams list to KubescapeJobParams")
	}

	jobTemplateObj, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Get(context.Background(), jobParams.JobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	jobTemplateObj.Spec.Schedule = getCronTabSchedule(actionHandler.sessionObj.Command)
	if jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations == nil {
		jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations = make(map[string]string)
	}
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armotypes.CronJobTemplateAnnotationUpdateJobID] = actionHandler.sessionObj.Command.JobTracking.JobID

	_, err = actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Update(context.Background(), jobTemplateObj, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (actionHandler *ActionHandler) setKubescapeCronJob(ctx context.Context) error {
	_, span := otel.Tracer("").Start(ctx, "actionHandler.setKubescapeCronJob")
	defer span.End()

	if !actionHandler.config.Components().KubescapeScheduler.Enabled {
		return errors.New("KubescapeScheduler is not enabled")
	}

	req, err := getKubescapeRequest(actionHandler.sessionObj.Command.Args)
	if err != nil {
		return err
	}

	for i := range req.TargetNames {
		name := fixK8sCronJobNameLimit(fmt.Sprintf("%s-%s-%d", "ks-scheduled-scan", req.TargetNames[i], rand.NewSource(time.Now().UnixNano()).Int63()))

		// create config map
		if err := createTriggerRequestConfigMap(actionHandler.k8sAPI, actionHandler.config.Namespace(), name, req); err != nil {
			return err
		}

		jobTemplateObj, err := getCronJobTemplate(actionHandler.k8sAPI, KubescapeCronJobTemplateName, actionHandler.config.Namespace())
		if err != nil {
			return err
		}

		setCronJobTemplate(jobTemplateObj, name, getCronTabSchedule(actionHandler.sessionObj.Command), actionHandler.sessionObj.Command.JobTracking.JobID, req.TargetNames[i], req.TargetType, req.HostScanner)

		// create cronJob
		if _, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(actionHandler.config.Namespace()).Create(context.Background(), jobTemplateObj, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func HandleKubescapeResponse(ctx context.Context, config config.IConfig, payload interface{}) (bool, *time.Duration) {
	data := payload.(*kubescapeResponseData)
	logger.L().Info(fmt.Sprintf("handle kubescape response for scan id %s", data.scanID))

	resp, err := httputils.HttpGetWithContext(ctx, KubescapeHttpClient, getKubescapeV1ScanStatusURL(config, data.scanID).String(), nil)
	if err != nil {
		data.sessionObj.SetOperatorCommandStatus(ctx, utils.WithError(fmt.Errorf("get scanID job status with scanID '%s' returned an error: %s", data.scanID, err.Error())),
			utils.WithPayload([]byte(data.scanID)))
		logger.L().Ctx(ctx).Error("get scanID job status returned an error", helpers.String("scanID", data.scanID), helpers.Error(err))
		return false, nil
	}

	response, err := readKubescapeV1ScanResponse(resp)
	if err != nil {
		data.sessionObj.SetOperatorCommandStatus(ctx,
			utils.WithError(fmt.Errorf("parse scanID job status with scanID '%s' returned an error: %s", data.scanID, err.Error())),
			utils.WithPayload([]byte(data.scanID)))
		logger.L().Ctx(ctx).Error("parse scanID job status returned an error", helpers.String("scanID", data.scanID), helpers.Error(err))
		return false, nil
	}

	if response.Type == utilsapisv1.BusyScanResponseType {
		nextTimeRehandled := time.Duration(WaitTimeForKubescapeScanResponse * time.Second)
		logger.L().Info(fmt.Sprintf("Kubescape get job status for scanID '%s' is %s next handle time is %s", data.scanID, utilsapisv1.BusyScanResponseType, nextTimeRehandled.String()))
		return true, &nextTimeRehandled
	}

	logger.L().Info(fmt.Sprintf("Kubescape get job status scanID '%s' finished successfully", data.scanID))
	data.sessionObj.SetOperatorCommandStatus(ctx, utils.WithSuccess(), utils.WithPayload([]byte(data.scanID)))
	return false, nil
}

func (actionHandler *ActionHandler) kubescapeScan(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "actionHandler.kubescapeScan")
	defer span.End()

	if !actionHandler.config.Components().Kubescape.Enabled {
		return errors.New("kubescape is not enabled")
	}

	request, err := getKubescapeV1ScanRequest(actionHandler.sessionObj.Command.Args)
	if err != nil {
		return err
	}

	// append security framework if TriggerSecurityFramework is true
	if actionHandler.config.TriggerSecurityFramework() {
		appendSecurityFramework(request)
	}

	body, err := json.Marshal(*request)
	if err != nil {
		return err
	}
	resp, err := httputils.HttpPostWithContext(ctx, KubescapeHttpClient, getKubescapeV1ScanURL(actionHandler.config).String(), nil, body, -1, func(resp *http.Response) bool { return true })
	if err != nil {
		return err
	}
	response, err := readKubescapeV1ScanResponse(resp)
	if err != nil {
		return err
	}

	if response.Type == utilsapisv1.ErrorScanResponseType {
		err := fmt.Errorf("%s", response.Response)
		logger.L().Info("Kubescape scan returned an error", helpers.String("scanID", response.ID), helpers.Error(err))
		actionHandler.sessionObj.SetOperatorCommandStatus(ctx, utils.WithError(err), utils.WithPayload([]byte(response.ID)))
	} else {
		logger.L().Info("Kubescape scan triggered successfully", helpers.String("scanID", response.ID))
		// sessionObj.SetOperatorCommandStatus(ctx, utils.WithSuccess(), utils.WithPayload([]byte(response.ID)))
	}

	data := &kubescapeResponseData{
		scanID:     response.ID,
		sessionObj: actionHandler.sessionObj,
	}

	if actionHandler.sessionObj.ParentCommandDetails != nil {
		nextHandledTime := WaitTimeForKubescapeScanResponse * time.Second
		commandResponseData := createNewCommandResponseData(KubescapeResponse, HandleKubescapeResponse, data, &nextHandledTime)
		insertNewCommandResponseData(actionHandler.commandResponseChannel, commandResponseData)
	}

	return nil
}

func getCronTabSchedule(command *armoapi.Command) string {
	if kubescapeJobParams := getKubescapeJobParams(command); kubescapeJobParams != nil {
		return kubescapeJobParams.CronTabSchedule
	}
	if schedule, ok := command.Args["cronTabSchedule"]; ok {
		if s, k := schedule.(string); k {
			return s
		}
	}
	if len(command.Designators) > 0 {
		if schedule, ok := command.Designators[0].Attributes["cronTabSchedule"]; ok {
			return schedule
		}
	}

	return ""
}

func getKubescapeJobParams(command *armoapi.Command) *armoapi.CronJobParams {

	if jobParams := command.GetCronJobParams(); jobParams != nil {
		return jobParams
	}

	// fallback
	if jobParams, ok := command.Args["kubescapeJobParams"]; ok {
		if kubescapeJobParams, ok := jobParams.(armoapi.CronJobParams); ok {
			return &kubescapeJobParams
		}
		b, err := json.Marshal(jobParams)
		if err != nil {
			return nil
		}
		kubescapeJobParams := &armoapi.CronJobParams{}
		if err = json.Unmarshal(b, kubescapeJobParams); err != nil {
			return nil
		}
		return kubescapeJobParams
	}
	return nil
}
