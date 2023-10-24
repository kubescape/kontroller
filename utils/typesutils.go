package utils

import (
	"context"
	"fmt"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/armoapi-go/identifiers"

	"github.com/armosec/utils-go/httputils"
	utilsmetadata "github.com/armosec/utils-k8s-go/armometadata"
	"github.com/google/uuid"
	beClientV1 "github.com/kubescape/backend/pkg/client/v1"
	"github.com/kubescape/backend/pkg/server/v1/systemreports"
	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
)

var ReporterHttpClient httputils.IHttpClient

func NewSessionObj(ctx context.Context, eventReceiverRestURL string, clusterConfig utilsmetadata.ClusterConfig, command *apis.Command, message, parentID, jobID string, actionNumber int) *SessionObj {
	var reporter beClientV1.IReportSender
	if eventReceiverRestURL == "" {
		reporter = newDummyReportSender()
	} else {
		report := systemreports.NewBaseReport(clusterConfig.AccountID, message)
		target := command.GetID()
		if target == identifiers.DesignatorsToken {
			target = fmt.Sprintf("wlid://cluster-%s/", clusterConfig.ClusterName)
		}
		if target == "" {
			target = fmt.Sprintf("%v", command.Args)
		}
		report.SetTarget(target)

		if jobID == "" {
			jobID = uuid.NewString()
		}
		report.SetJobID(jobID)
		report.SetParentAction(parentID)
		report.SetActionIDN(actionNumber)
		if command.CommandName != "" {
			report.SetActionName(string(command.CommandName))
		}

		noHeaders := map[string]string{}
		reporter = beClientV1.NewBaseReportSender(eventReceiverRestURL, ReporterHttpClient, noHeaders, report)
	}

	sessionObj := SessionObj{
		Command:  *command,
		Reporter: reporter,
		ErrChan:  make(chan error),
	}
	go sessionObj.WatchErrors(ctx)

	sessionObj.Reporter.SendAsRoutine(true)
	return &sessionObj
}

func (sessionObj *SessionObj) WatchErrors(ctx context.Context) {
	for err := range sessionObj.ErrChan {
		if err != nil {
			logger.L().Ctx(ctx).Error("failed to send job report", helpers.Error(err))
		}
	}
}

func NewJobTracking(reporter systemreports.IReporter) *apis.JobTracking {
	return &apis.JobTracking{
		JobID:            reporter.GetJobID(),
		ParentID:         reporter.GetParentAction(),
		LastActionNumber: reporter.GetActionIDN() + 1,
	}
}
