package utils

import (
	"context"

	"github.com/armosec/armoapi-go/apis"
	beClientV1 "github.com/kubescape/backend/pkg/client/v1"
)

// Commands list of commands received from websocket
type SessionObj struct {
	Command  apis.Command             `json:"command"`
	Reporter beClientV1.IReportSender `json:"reporter"`
}

// CredStruct holds the various credentials needed to do login into CA BE
type CredStruct struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Customer string `json:"customer"`
}

type Job struct {
	ctx        context.Context
	sessionObj SessionObj
}

func (j *Job) Context() context.Context {
	return j.ctx
}

func (j *Job) Obj() SessionObj {
	return j.sessionObj
}

func (j *Job) SetContext(ctx context.Context) {
	j.ctx = ctx
}

func (j *Job) SetObj(sessionObj SessionObj) {
	j.sessionObj = sessionObj
}
