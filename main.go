package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/k8s-interface/k8sinterface"
	"github.com/kubescape/operator/config"
	"github.com/kubescape/operator/crdhandler"
	cs "github.com/kubescape/operator/continuousscanning"
	"github.com/kubescape/operator/mainhandler"
	"github.com/kubescape/operator/notificationhandler"
	"github.com/kubescape/operator/restapihandler"
	"github.com/kubescape/operator/utils"
	restclient "k8s.io/client-go/rest"

	"github.com/armosec/utils-k8s-go/probes"
	beUtils "github.com/kubescape/backend/pkg/utils"
	logger "github.com/kubescape/go-logger"
)

//go:generate swagger generate spec -o ./docs/swagger.yaml
func main() {
	ctx := context.Background()

	flag.Parse()

	isReadinessReady := false
	go probes.InitReadinessV1(&isReadinessReady)

	displayBuildTag()

	clusterConfig, err := config.LoadClusterConfig()
	if err != nil {
		logger.L().Ctx(ctx).Fatal("load clusterData error", helpers.Error(err))
	}

	components, err := config.LoadCapabilitiesConfig("/etc/config")
	if err != nil {
		logger.L().Ctx(ctx).Fatal("load components error", helpers.Error(err))
	}
	logger.L().Debug("loaded config for components", helpers.Interface("components", components))

	var credentials *beUtils.Credentials
	if credentials, err = beUtils.LoadCredentialsFromFile("/etc/credentials"); err != nil {
		logger.L().Ctx(ctx).Error("failed to load credentials", helpers.Error(err))
		credentials = &beUtils.Credentials{}
	} else {
		logger.L().Info("credentials loaded",
			helpers.Int("accessKeyLength", len(credentials.AccessKey)),
			helpers.Int("accountLength", len(credentials.Account)))
	}

	var eventReceiverRestURL string
	if components.Components.ServiceDiscovery.Enabled {
		services, err := config.GetServiceURLs("/etc/config/services.json")
		if err != nil {
			logger.L().Ctx(ctx).Fatal("failed discovering urls", helpers.Error(err))
		}

		eventReceiverRestURL = services.GetReportReceiverHttpUrl()
		logger.L().Debug("setting eventReceiverRestURL", helpers.String("url", eventReceiverRestURL))
	}

	cfg, err := config.LoadConfig("/etc/config")
	if err != nil {
		logger.L().Ctx(ctx).Fatal("load config error", helpers.Error(err))
	}

	// wrapper for all configs
	operatorConfig := config.NewOperatorConfig(components, clusterConfig, credentials, eventReceiverRestURL, cfg)
	if err := config.ValidateConfig(operatorConfig); err != nil {
		logger.L().Ctx(ctx).Error("validate config error", helpers.Error(err))
	}

	// to enable otel, set OTEL_COLLECTOR_SVC=otel-collector:4317
	if otelHost, present := os.LookupEnv("OTEL_COLLECTOR_SVC"); present && components.Components.OtelCollector.Enabled {
		ctx = logger.InitOtel("operator",
			os.Getenv("RELEASE"),
			operatorConfig.AccountID(),
			operatorConfig.ClusterName(),
			url.URL{Host: otelHost})
		defer logger.ShutdownOtel(ctx)
	}

	initHttpHandlers(operatorConfig)
	k8sApi := k8sinterface.NewKubernetesApi()
	restclient.SetDefaultWarningHandler(restclient.NoWarnings{})

	// setup main handler
	mainHandler := mainhandler.NewMainHandler(operatorConfig, k8sApi)

	if components.Components.Gateway.Enabled {
		go func() { // open websocket connection to notification server
			notificationHandler := notificationhandler.NewNotificationHandler(mainHandler.EventWorkerPool(), operatorConfig)
			if err := notificationHandler.WebsocketConnection(ctx); err != nil {
				logger.L().Ctx(ctx).Fatal(err.Error(), helpers.Error(err))
			}
		}()
	}

	go func() { // open a REST API connection listener
		restAPIHandler := restapihandler.NewHTTPHandler(mainHandler.EventWorkerPool(), operatorConfig)
		if err := restAPIHandler.SetupHTTPListener(cfg.RestAPIPort); err != nil {
			logger.L().Ctx(ctx).Fatal(err.Error(), helpers.Error(err))
		}
	}()

	if operatorConfig.Components().Gateway.Enabled {
		logger.L().Debug("triggering a full kubescapeScan on startup")
		go mainHandler.StartupTriggerActions(ctx, mainhandler.GetStartupActions(operatorConfig))
	}

	isReadinessReady = true

	// wait for requests to come from the websocket or from the REST API
	go mainHandler.HandleCommandResponse(ctx)
	if operatorConfig.Components().Kubevuln.Enabled {
		mainHandler.HandleWatchers(ctx)
	}

	if operatorConfig.ContinuousScanEnabled() {
		go func(mh *mainhandler.MainHandler) {
			err := mh.SetupContinuousScanning(ctx, cs.DefaultQueueSize, cfg.EventDeduplicationInterval)
			logger.L().Ctx(ctx).Info("set up cont scanning service")
			if err != nil {
				logger.L().Ctx(ctx).Fatal(err.Error(), helpers.Error(err))
			}
		}(mainHandler)
	}

	crdHandler := crdhandler.NewCrdHandler(k8sApi)

	go crdHandler.HandleCrds()

	<-ctx.Done()
}

func displayBuildTag() {
	logger.L().Info(fmt.Sprintf("Image version: %s", os.Getenv("RELEASE")))
}

func initHttpHandlers(config config.IConfig) {
	mainhandler.KubescapeHttpClient = utils.InitHttpClient(config.KubescapeURL())
	mainhandler.VulnScanHttpClient = utils.InitHttpClient(config.KubevulnURL())
	utils.ReporterHttpClient = utils.InitHttpClient(config.EventReceiverURL())
}
