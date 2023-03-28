package utils


import (
	"net/http"

	"github.com/armosec/utils-go/httputils"
)

const KubescapeScanV1 = "scanV1"
const KubescapeRequestPathV1 = "v1/scan"
const KubescapeRequestStatusV1 = "v1/status"

func MapToString(m map[string]interface{}) []string {
	s := []string{}
	for i := range m {
		s = append(s, i)
	}
	return s
}

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       http.NoBody}, nil
}

func InitKubescapeHttpClient() httputils.IHttpClient {
	// If the KubescapeURL not configured, then the HttpClient defined as a mock
	if ClusterConfig.KubescapeURL == "" {
		return &ClientMock{}
	}
	return &http.Client{}
}
func InitVulnScanHttpClient() httputils.IHttpClient {
	// If the VulnScan URL not configured, then the HttpClient defined as a mock
	if ClusterConfig.KubevulnURL == "" {
		return &ClientMock{}
	}
	return &http.Client{}
}
func InitReporterHttpClient() httputils.IHttpClient {
	// If the EventReceiverREST not configured, then the HttpClient defined as a mock
	if ClusterConfig.EventReceiverRestURL == "" {
		return &ClientMock{}
	}
	return &http.Client{}
}
