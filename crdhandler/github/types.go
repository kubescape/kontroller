package crdhandler

import (
	"github.com/kubescape/k8s-interface/k8sinterface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Github Artifact Handler
type GithubRepositoryHandler struct {
	k8sAPI *k8sinterface.KubernetesApi
}

// Github Release
type GithubRelease struct { // intermediatary (no reason to be public)
	Name       string       `json:"name"`
	ZipballUrl string       `json:"zipball_url"`
	TarballUrl string       `json:"tarball_url"`
	NodeId     string       `json:"node_id"`
	Commit     GithubCommit `json:"commit"`
}

type GithubCommit struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Framework JSON
type FrameworkJson struct {
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Attributes     FrameworkAttributes      `json:"attributes"`
	ScanningScope  FrameworkScanningScope   `json:"scanningScope"`
	TypeTags       []string                 `json:"typeTags"`
	ActiveControls []FrameworkActiveControl `json:"activeControls"`
}

type FrameworkAttributes struct {
	ArmoBuiltin bool `json:"armoBuiltin"`
}

type FrameworkScanningScope struct {
	Matches []string `json:"matches"`
}

type FrameworkActiveControl struct {
	ControlID string `json:"controlID"`
	Patch     Patch  `json:"patch"`
}

type Patch struct {
	Name string `json:"name"`
}

/////////////  IMPORT FROM STORAGE COMPONENT
// Framework CRD
type Framework struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ControlsRef       []FrameworkControlRef `json:"controlsRef,omitempty"`
	Payload           interface{}           `json:"payload"`
}

type FrameworkControlRef struct {
	Name       string `json:"name"`
	ControlID  string `json:"controlID"`
	Kind       string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
}

// Control Json struct
type ControlJson struct {
	Name            string               `json:"name"`
	Attributes      ControlAttributes    `json:"attributes"`
	Description     string               `json:"description"`
	Remediation     string               `json:"remediation"`
	RulesNames      []string             `json:"rulesNames"`
	LongDescription string               `json:"long_description"`
	Test            string               `json:"test"`
	ControlID       string               `json:"controlID"`
	BaseScore       float64              `json:"baseScore"`
	Example         string               `json:"example"`
	Category        ControlCategory      `json:"category"`
	ScanningScope   ControlScanningScope `json:"scanningScope"`
}

type ControlAttributes struct {
	ArmoBuiltin           bool     `json:"armoBuiltin"`
	MicrosoftMitreColumns []string `json:"microsoftMitreColumns"`
	ControlTypeTags       []string `json:"controlTypeTags"`
	ActionRequired        string   `json:"actionRequired"`
}

type ControlCategory struct {
	Name string `json:"name"`
}

type ControlScanningScope struct {
	Matches []string `json:"matches"`
}

// Control CRD
type Control struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	RulesRef          []ControlRuleRef `json:"rulesRef,omitempty"`
	Payload           interface{}      `json:"payload"`
}

type ControlRuleRef struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
}

// Rule Json
type RuleJson struct {
	Name               string         `json:"name"`
	Attributes         RuleAttributes `json:"attributes"`
	RuleLanguage       string         `json:"ruleLanguage"`
	Match              []RuleMatch    `json:"match"`
	RuleDependencies   []string       `json:"ruleDependencies"`
	Description        string         `json:"description"`
	Remediation        string         `json:"remediation"`
	RuleQuery          string         `json:"ruleQuery"`
	ResourceCount      string         `json:"resourceCount"`
	Rule               string         `json:"rule"`
	ResourceEnumerator string         `json:"resourceEnumerator"`
}

type RuleAttributes struct {
	MK8sThreatMatrix         string `json:"m$K8sThreatMatrix"`
	ArmoBuiltin              bool   `json:"armoBuiltin"`
	UseUntilKubescapeVersion string `json:"useUntilKubescapeVersion"`
}

type RuleMatch struct {
	ApiGroups   []string `json:"apiGroups"`
	ApiVersions []string `json:"apiVersions"`
	Resources   []string `json:"resources"`
}

// Rule CRD
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Rego              string      `json:"rego,omitempty"`
	Payload           interface{} `json:"payload"`
}

// Exception CRD
type Exception struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Payload           interface{} `json:"payload"`
}

// ControlConfiguration CRD
type ControlConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Payload           interface{} `json:"payload"`
}
