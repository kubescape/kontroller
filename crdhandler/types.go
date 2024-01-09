package crdhandler

import (
	"github.com/kubescape/k8s-interface/k8sinterface"
	crdhandler "github.com/kubescape/operator/crdhandler/github"
)

// Repository Handler
type CrdHandler struct {
	k8sAPI *k8sinterface.KubernetesApi
}

// Repository Interface
type repositoryHandler interface { // private / public (Repository naming convention)
	InitRepository()
	GetFrameworks() []*crdhandler.Framework
	GetControls() []*crdhandler.Control
	GetRules() []*crdhandler.Rule
	GetExceptions() []*crdhandler.Exception
	GetControlConfigurations() []*crdhandler.ControlConfiguration
	CleanRepository()
}
