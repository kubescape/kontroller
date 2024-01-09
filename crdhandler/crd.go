package crdhandler

import (
	"context"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/k8s-interface/k8sinterface"
	crdhandler "github.com/kubescape/operator/crdhandler/github"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCrdHandler(k8sAPI *k8sinterface.KubernetesApi) *CrdHandler {
	return &CrdHandler{
		k8sAPI: k8sAPI,
	}
}

func (ch *CrdHandler) HandleCrds() {
	githubRepositoryHandler := crdhandler.NewGithubRepositoryHandler(ch.k8sAPI)

	repositoryHandlers := []repositoryHandler{
		githubRepositoryHandler,
	}

	var allFrameworks []*crdhandler.Framework
	var allControls []*crdhandler.Control
	var allRules []*crdhandler.Rule
	var allExceptions []*crdhandler.Exception
	var allControlConfigurations []*crdhandler.ControlConfiguration

	for _, repositoryHandler := range repositoryHandlers {
		repositoryHandler.InitRepository()

		frameworks := repositoryHandler.GetFrameworks()
		allFrameworks = append(allFrameworks, frameworks...)

		controls := repositoryHandler.GetControls()
		allControls = append(allControls, controls...)

		rules := repositoryHandler.GetRules()
		allRules = append(allRules, rules...)

		exceptions := repositoryHandler.GetExceptions()
		allExceptions = append(allExceptions, exceptions...)

		controlConfigurations := repositoryHandler.GetControlConfigurations()
		allControlConfigurations = append(allControlConfigurations, controlConfigurations...)

		repositoryHandler.CleanRepository()
	}

	ch.createFrameworks(allFrameworks)
	ch.createControls(allControls)
	ch.createRules(allRules)
	ch.createExceptions(allExceptions)
	ch.createControlConfiguration(allControlConfigurations)
}

func (ch *CrdHandler) createFrameworks(frameworks []*crdhandler.Framework) {
	for _, framework := range frameworks {
		frameworkRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(framework)
		_, err := ch.k8sAPI.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "spdx.softwarecomposition.kubescape.io",
			Version:  "v1alpha1",
			Resource: "frameworks",
		}).Create(context.TODO(), &unstructured.Unstructured{Object: frameworkRaw}, metav1.CreateOptions{})
		if err != nil {
			logger.L().Fatal(err.Error())
		}
	}
	logger.L().Success("All Frameworks Created")
}

func (ch *CrdHandler) createControls(controls []*crdhandler.Control) {
	for _, control := range controls {
		controlRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(control)
		_, err := ch.k8sAPI.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "spdx.softwarecomposition.kubescape.io",
			Version:  "v1alpha1",
			Resource: "controls",
		}).Create(context.TODO(), &unstructured.Unstructured{Object: controlRaw}, metav1.CreateOptions{})
		if err != nil {
			logger.L().Fatal(err.Error())
		}
	}
	logger.L().Success("All Controls Created")
}

func (ch *CrdHandler) createRules(rules []*crdhandler.Rule) {
	for _, rule := range rules {
		ruleRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(rule)
		_, err := ch.k8sAPI.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "spdx.softwarecomposition.kubescape.io",
			Version:  "v1alpha1",
			Resource: "rules",
		}).Create(context.TODO(), &unstructured.Unstructured{Object: ruleRaw}, metav1.CreateOptions{})
		if err != nil {
			logger.L().Fatal(err.Error())
		}
	}
	logger.L().Success("All Rules Created")
}

func (ch *CrdHandler) createExceptions(exceptions []*crdhandler.Exception) {
	for _, exception := range exceptions {
		exceptionRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(exception)
		_, err := ch.k8sAPI.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "spdx.softwarecomposition.kubescape.io",
			Version:  "v1alpha1",
			Resource: "exceptions",
		}).Create(context.TODO(), &unstructured.Unstructured{Object: exceptionRaw}, metav1.CreateOptions{})
		if err != nil {
			logger.L().Fatal(err.Error())
		}
	}
	logger.L().Success("All Exceptions Added")
}

func (ch *CrdHandler) createControlConfiguration(controlConfigurations []*crdhandler.ControlConfiguration) {
	for _, controlConfiguration := range controlConfigurations {
		controlConfigurationRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(controlConfiguration)
		_, err := ch.k8sAPI.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "spdx.softwarecomposition.kubescape.io",
			Version:  "v1alpha1",
			Resource: "controlconfigurations",
		}).Create(context.TODO(), &unstructured.Unstructured{Object: controlConfigurationRaw}, metav1.CreateOptions{})
		if err != nil {
			logger.L().Fatal(err.Error())
		}
	}
	logger.L().Success("All Control Configurations Added")
}

// STORAGE COMPONENT
/*
	Take in any type, interface{}, type agnostic
	CRUD, watchers

	Add type definitions -> Store/Access them (easy)

	1) Rename/Refactor/Clean code

	2) Finalize type definitions (v1alpha1) (PRIORITY)
		2.1) Ensure NO INTERMEDIATRIES ARE PUBLIC or violate contract
		2.2) Define types in storage

	3) Integrate with Kubescape
*/
