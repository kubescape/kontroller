package crdhandler

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/k8s-interface/k8sinterface"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGithubRepositoryHandler(k8sAPI *k8sinterface.KubernetesApi) *GithubRepositoryHandler {
	return &GithubRepositoryHandler{
		k8sAPI: k8sAPI,
	}
}

func (grh *GithubRepositoryHandler) InitRepository() { // err ? have
	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/kubescape/regolibrary/tags", nil)
	if err != nil {
		logger.L().Fatal(err.Error())
	}
	req.Header = http.Header{
		"If-Modified-Since": {"Fri, 08 Aug 2023 09:42:43 GMT"},
	}

	res, err := client.Do(req)
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	if res == nil {
		logger.L().Info("No new updates\n")
	}

	defer res.Body.Close()

	var githubResp []GithubRelease
	json.NewDecoder(res.Body).Decode(&githubResp)

	zipUrl := githubResp[0].ZipballUrl

	req, err = http.NewRequest("GET", zipUrl, nil)
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	res, err = client.Do(req)
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	filename := "artifacts.zip"
	out, _ := os.Create(filename)
	defer out.Close()
	io.Copy(out, res.Body)

	reader, _ := zip.OpenReader(filename)
	defer reader.Close()
	unzip(filename, "")
	os.Rename(reader.File[0].Name, "artifacts")

	logger.L().Success("Github Repository Initialized")
}

func (grh *GithubRepositoryHandler) GetFrameworks() []*Framework {
	files, err := os.ReadDir("artifacts/frameworks/")
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	var frameworks []*Framework

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		data, _ := os.ReadFile("artifacts/frameworks/" + file.Name())
		var frameworkJson FrameworkJson
		_ = json.Unmarshal([]byte(data), &frameworkJson)

		var controlRefs []FrameworkControlRef
		for _, activeControl := range frameworkJson.ActiveControls {
			controlRefs = append(controlRefs, FrameworkControlRef{
				Name:       activeControl.Patch.Name,
				ControlID:  activeControl.ControlID,
				Kind:       "Control",
				ApiVersion: "v1alpha1",
			})
		}

		framework := &Framework{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Framework",
				APIVersion: "spdx.softwarecomposition.kubescape.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(frameworkJson.Name),
			},
			ControlsRef: controlRefs,
			Payload:     frameworkJson,
		}

		frameworks = append(frameworks, framework)
	}
	logger.L().Success("All Github Frameworks Extracted")
	return frameworks
}

func (grh *GithubRepositoryHandler) GetControls() []*Control {
	files, err := os.ReadDir("artifacts/controls/")
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	var controls []*Control

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		data, _ := os.ReadFile("artifacts/controls/" + file.Name())
		var controlJson ControlJson
		_ = json.Unmarshal([]byte(data), &controlJson)

		var ruleRefs []ControlRuleRef
		for _, ruleName := range controlJson.RulesNames {
			ruleRefs = append(ruleRefs, ControlRuleRef{
				Name:       ruleName,
				Kind:       "Rule",
				ApiVersion: "v1alpha1",
			})
		}

		control := &Control{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Control",
				APIVersion: "spdx.softwarecomposition.kubescape.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(controlJson.ControlID),
			},
			RulesRef: ruleRefs,
			Payload:  controlJson,
		}

		controls = append(controls, control)
	}
	logger.L().Success("All Github Controls Extracted")
	return controls
}

func (grh *GithubRepositoryHandler) GetRules() []*Rule {
	ruleDirs, err := os.ReadDir("artifacts/rules/")
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	var rules []*Rule

	for _, ruleDir := range ruleDirs {
		if !ruleDir.IsDir() || strings.HasPrefix(ruleDir.Name(), ".") {
			continue
		}
		ruleFiles, err := os.ReadDir("artifacts/rules/" + ruleDir.Name())
		if err != nil {
			logger.L().Fatal(err.Error())
		}
		var ruleRego string
		var ruleFilter string
		var ruleJson RuleJson
		for _, ruleFile := range ruleFiles {
			if ruleFile.Name() == "raw.rego" {
				ruleRegoBytes, err := os.ReadFile("artifacts/rules/" + ruleDir.Name() + "/" + ruleFile.Name())
				if err != nil {
					logger.L().Fatal(err.Error())
				}
				ruleRego = string(ruleRegoBytes)
			} else if ruleFile.Name() == "filter.rego" {
				ruleFilterBytes, err := os.ReadFile("artifacts/rules/" + ruleDir.Name() + "/" + ruleFile.Name())
				if err != nil {
					logger.L().Fatal(err.Error())
				}
				ruleFilter = string(ruleFilterBytes)
			}else {
				data, _ := os.ReadFile("artifacts/rules/" + ruleDir.Name() + "/" + ruleFile.Name())
				_ = json.Unmarshal([]byte(data), &ruleJson)
			}
		}

		ruleJson.Rule = ruleRego
		ruleJson.ResourceEnumerator = ruleFilter

		rule := &Rule{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rule",
				APIVersion: "spdx.softwarecomposition.kubescape.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(ruleJson.Name), " ", "-"), "_", "-"),
			},
			Rego:    ruleRego,
			Payload: ruleJson,
		}

		rules = append(rules, rule)
	}
	logger.L().Success("All Github Rules Extracted")
	return rules
}

func (grh *GithubRepositoryHandler) GetExceptions() []*Exception {
	files, err := os.ReadDir("artifacts/exceptions/")
	if err != nil {
		logger.L().Fatal(err.Error())
	}

	var exceptions []*Exception

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		data, _ := os.ReadFile("artifacts/exceptions/" + file.Name())
		var exceptionJson interface{}
		_ = json.Unmarshal([]byte(data), &exceptionJson)

		exception := &Exception{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Exception",
				APIVersion: "spdx.softwarecomposition.kubescape.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(strings.ReplaceAll(file.Name(), ".json", "")),
			},
			Payload: exceptionJson,
		}

		exceptions = append(exceptions, exception)
	}
	logger.L().Success("All Github Exceptions Extracted")
	return exceptions
}

func (grh *GithubRepositoryHandler) GetControlConfigurations() []*ControlConfiguration {
	data, _ := os.ReadFile("artifacts/default-config-inputs.json")

	var controlConfigurations []*ControlConfiguration

	var controlConfigurationJson interface{}
	_ = json.Unmarshal([]byte(data), &controlConfigurationJson)

	controlConfiguration := &ControlConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ControlConfiguration",
			APIVersion: "spdx.softwarecomposition.kubescape.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-config-inputs",
		},
		Payload: controlConfigurationJson,
	}

	controlConfigurations = append(controlConfigurations, controlConfiguration)

	logger.L().Success("All Github Control Configurations Extracted")
	return controlConfigurations
}

func (grh *GithubRepositoryHandler) CleanRepository() {
	if err := os.RemoveAll("artifacts/"); err != nil {
		logger.L().Warning(err.Error())
	}
	if err := os.Remove("artifacts.zip"); err != nil {
		logger.L().Warning(err.Error())
	}
	logger.L().Success("Github Repository Cleaned")
}
