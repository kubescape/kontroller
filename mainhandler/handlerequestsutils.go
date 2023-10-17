package mainhandler

import (
	"fmt"
	"time"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/operator/config"
	"github.com/kubescape/operator/utils"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/utils-go/httputils"
	"github.com/armosec/utils-k8s-go/probes"
	pkgwlid "github.com/armosec/utils-k8s-go/wlid"

	"github.com/kubescape/k8s-interface/k8sinterface"
)

func (mainHandler *MainHandler) listWorkloads(namespaces []string, resource string, labels, fields map[string]string) ([]k8sinterface.IWorkload, error) {
	groupVersionResource, err := k8sinterface.GetGroupVersionResource(resource)
	if err != nil {
		return nil, err
	}
	res := make([]k8sinterface.IWorkload, 0, 1)
	for nsIdx := range namespaces {
		iwls, err := mainHandler.k8sAPI.ListWorkloads(&groupVersionResource, namespaces[nsIdx], labels, fields)
		if err != nil {
			return res, err
		}
		res = append(res, iwls...)
	}
	return res, nil
}
func (mainHandler *MainHandler) getResourcesIDs(workloads []k8sinterface.IWorkload) ([]string, []error) {
	errs := []error{}
	idMap := make(map[string]interface{})
	for i := range workloads {
		switch workloads[i].GetKind() {
		case "Namespace":
			idMap[pkgwlid.GetWLID(mainHandler.config.ClusterName(), workloads[i].GetName(), "namespace", workloads[i].GetName())] = true
		default:
			// find wlid
			kind, name, err := mainHandler.k8sAPI.CalculateWorkloadParentRecursive(workloads[i])
			if err != nil {
				errs = append(errs, fmt.Errorf("CalculateWorkloadParentRecursive: namespace: %s, pod name: %s, error: %s", workloads[i].GetNamespace(), workloads[i].GetName(), err.Error()))
			}

			// skip cronjobs
			if kind == "CronJob" {
				continue
			}

			wlid := pkgwlid.GetWLID(mainHandler.config.ClusterName(), workloads[i].GetNamespace(), kind, name)
			if wlid != "" {
				idMap[wlid] = true
			}
		}
	}
	return utils.MapToString(idMap), errs
}

func notWaitAtAll(_ config.IConfig) {
}

func isActionNeedToWait(action apis.Command) waitFunc {
	if f, ok := actionNeedToBeWaitOnStartUp[action.CommandName]; ok {
		return f
	}
	return notWaitAtAll
}

func waitForVulnScanReady(config config.IConfig) {
	fullURL := getVulnScanURL(config)
	// replace path
	fullURL.Path = fmt.Sprintf("v1/%s", probes.ReadinessPath)

	timer := time.NewTimer(time.Duration(1) * time.Minute)

	for {
		timer.Reset(time.Duration(1) * time.Second)
		<-timer.C
		resp, err := httputils.HttpGet(VulnScanHttpClient, fullURL.String(), nil)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode <= 203 {
			logger.L().Info("image vulnerability scanning is available")
			break
		}

	}
}

func waitForKubescapeReady(config config.IConfig) {
	fullURL := getKubescapeV1ScanURL(config)
	fullURL.Path = "readyz"
	timer := time.NewTimer(time.Duration(1) * time.Minute)

	for {
		timer.Reset(time.Duration(1) * time.Second)
		<-timer.C
		resp, err := httputils.HttpHead(KubescapeHttpClient, fullURL.String(), nil)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode <= 203 {
			logger.L().Info("kubescape service is ready")
			break
		}

	}
}
