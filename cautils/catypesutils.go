package cautils

import (
	"github.com/armosec/capacketsgo/apis"
	reporterlib "github.com/armosec/capacketsgo/system-reports/datastructures"
	reportutils "github.com/armosec/capacketsgo/system-reports/utilities"
)

func NewSessionObj(command *apis.Command, message, jobID string, actionNumber int) *SessionObj {
	reporter := reporterlib.NewBaseReport(CA_CUSTOMER_GUID, message)
	reporter.SetTarget(command.GetID())
	reporter.SetJobID(jobID)
	reporter.SetActionIDN(actionNumber)
	reporter.SendAsRoutine(reportutils.EmptyString, true)

	sessionObj := SessionObj{
		Command:  *command,
		Reporter: reporter,
	}
	return &sessionObj
}
