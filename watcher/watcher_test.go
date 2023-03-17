package watcher

import (
	"context"
	_ "embed"
	"reflect"
	"sync"
	"testing"

	"github.com/armosec/armoapi-go/apis"
	"github.com/kubescape/operator/utils"
	spdxv1beta1 "github.com/kubescape/storage/pkg/apis/softwarecomposition/v1beta1"
	"github.com/stretchr/testify/assert"
	core1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	// Kubescape storage client
	kssfake "github.com/kubescape/storage/pkg/generated/clientset/versioned/fake"
)

func NewWatchHandlerMock() *WatchHandler {
	return &WatchHandler{
		imagesIDToWlidsMap:                make(map[string][]string),
		wlidsToContainerToImageIDMap:      make(map[string]map[string]string),
		imageIDsMapMutex:                  &sync.RWMutex{},
		wlidsToContainerToImageIDMapMutex: &sync.RWMutex{},
		instanceIDsMutex:                  &sync.RWMutex{},
	}
}

func TestNewWatchHandlerProducesValidResult(t *testing.T) {
	ctx := context.TODO()
	k8sClient := k8sfake.NewSimpleClientset()
	k8sAPI := utils.NewK8sInterfaceFake(k8sClient)
	storageClient := kssfake.NewSimpleClientset()

	wh, err := NewWatchHandler(ctx, k8sAPI, storageClient)

	assert.NoErrorf(t, err, "Constructing should produce no errors")
	assert.NotNilf(t, wh, "Constructing should create a non-nil object")
}

func TestHandleSBOMProducesMatchingCommands(t *testing.T) {
	tt := []struct {
		name          string
		sbomNamespace string
		sboms         []spdxv1beta1.SBOMSPDXv2p3
		wlidMap       map[string][]string
	}{
		{
			name:          "Valid SBOM produces matching command",
			sbomNamespace: "sbom-test-ns",
			sboms: []spdxv1beta1.SBOMSPDXv2p3{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "0acbac6272564700d30edebaf7d546330836f8e0065b26cd2789b83b912e049d",
						Namespace: "sbom-test-ns",
					},
				},
			},
			wlidMap: map[string][]string{
				"0acbac6272564700d30edebaf7d546330836f8e0065b26cd2789b83b912e049d": {
					"wlid://test-wlid",
				},
			},
		},
		{
			"Two valid SBOMs produce matching commands",
			"sbom-test-ns",
			[]spdxv1beta1.SBOMSPDXv2p3{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "0acbac6272564700d30edebaf7d546330836f8e0065b26cd2789b83b912e049d",
						Namespace: "sbom-test-ns",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
						Namespace: "sbom-test-ns",
					},
				},
			},
			map[string][]string{
				"0acbac6272564700d30edebaf7d546330836f8e0065b26cd2789b83b912e049d": {
					"wlid://test-wlid",
				},
				"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08": {
					"wlid://test-wlid-02",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()

			k8sClient := k8sfake.NewSimpleClientset()
			ksStorageClient := kssfake.NewSimpleClientset()

			k8sAPI := utils.NewK8sInterfaceFake(k8sClient)
			wh, _ := NewWatchHandler(ctx, k8sAPI, ksStorageClient)

			sessionObjCh := make(chan utils.SessionObj)

			sbomWatcher, _ := ksStorageClient.SpdxV1beta1().SBOMSPDXv2p3s("").Watch(ctx, v1.ListOptions{})
			sbomEvents := sbomWatcher.ResultChan()

			go wh.HandleSBOMEvents(sbomEvents, sessionObjCh)

			// Handling the event is expected to transform
			// incloming imageID in the SBOM name to a valid WLID
			wh.imagesIDToWlidsMap = tc.wlidMap
			expectedWlidsCounter := map[string]int{}

			for _, sbom := range tc.sboms {
				ksStorageClient.SpdxV1beta1().SBOMSPDXv2p3s(tc.sbomNamespace).Create(ctx, &sbom, v1.CreateOptions{})
				expectedSbomWlid := tc.wlidMap[sbom.ObjectMeta.Name][0]
				expectedWlidsCounter[expectedSbomWlid] += 1
			}
			sbomWatcher.Stop()

			actualWlids := map[string]int{}
			for range tc.sboms {
				sessionObj := <- sessionObjCh
				assert.Equalf(t, apis.TypeScanImages, sessionObj.Command.CommandName, "Should produce Scan commands")

				actualWlids[sessionObj.Command.Wlid] += 1
			}

			assert.Equalf(t, expectedWlidsCounter, actualWlids, "Produced WLIDs should match the expected.")
		},
		)
	}
}

// func TestBuildImageIDsToWlidsMap(t *testing.T) {
// 	tests := []struct {
// 		name                string
// 		podList             core1.PodList
// 		expectedImageIDsMap map[string][]string
// 	}{
// 		{
// 			name: "remove prefix docker-pullable://",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					}}},
// 			expectedImageIDsMap: map[string][]string{
// 				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
// 			},
// 		},
// 		{
// 			name: "image id without docker-pullable:// prefix",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					}}},
// 			expectedImageIDsMap: map[string][]string{
// 				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
// 			},
// 		},
// 		{
// 			name: "two wlids for the same image id",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					},
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test2",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container2",
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 			expectedImageIDsMap: map[string][]string{
// 				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test"), pkgwlid.GetWLID("", "default", "pod", "test2")},
// 			},
// 		},
// 		{
// 			name: "two wlids two image ids",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					},
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test2",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:2",
// 									Name:    "container2",
// 								},
// 							},
// 						},
// 					}}},
// 			expectedImageIDsMap: map[string][]string{
// 				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
// 				"alpine@sha256:2": {pkgwlid.GetWLID("", "default", "pod", "test2")},
// 			},
// 		},
// 		{
// 			name: "one wlid two image ids",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "test",
// 							Namespace: "default",
// 						},
// 						TypeMeta: v1.TypeMeta{
// 							Kind: "pod",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:2",
// 									Name:    "container2",
// 								},
// 							},
// 						},
// 					}}},
// 			expectedImageIDsMap: map[string][]string{
// 				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
// 				"alpine@sha256:2": {pkgwlid.GetWLID("", "default", "pod", "test")},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		wh := NewWatchHandlerMock()
// 		t.Run(tt.name, func(t *testing.T) {
// 			wh.buildIDs(context.TODO(), &tt.podList)
// 			assert.True(t, reflect.DeepEqual(wh.getImagesIDsToWlidMap(), tt.expectedImageIDsMap))
// 		})
// 	}
// }

// func TestBuildWlidsToContainerToImageIDMap(t *testing.T) {
// 	tests := []struct {
// 		name                                 string
// 		podList                              core1.PodList
// 		expectedwlidsToContainerToImageIDMap WlidsToContainerToImageIDMap
// 	}{
// 		{
// 			name: "imageID with docker-pullable prefix",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "pod1",
// 							Namespace: "namespace1",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					}},
// 			},
// 			expectedwlidsToContainerToImageIDMap: WlidsToContainerToImageIDMap{
// 				pkgwlid.GetWLID("", "namespace1", "pod", "pod1"): {
// 					"container1": "alpine@sha256:1",
// 				},
// 			},
// 		},
// 		{
// 			name: "imageID without docker-pullable prefix",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "pod1",
// 							Namespace: "namespace1",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "alpine@sha256:1",
// 									Name:    "container1",
// 								},
// 							},
// 						},
// 					}},
// 			},
// 			expectedwlidsToContainerToImageIDMap: WlidsToContainerToImageIDMap{
// 				pkgwlid.GetWLID("", "namespace1", "pod", "pod1"): {
// 					"container1": "alpine@sha256:1",
// 				},
// 			},
// 		},
// 		{
// 			name: "two containers for same wlid",
// 			podList: core1.PodList{
// 				Items: []core1.Pod{
// 					{
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "pod3",
// 							Namespace: "namespace3",
// 						},
// 						Status: core1.PodStatus{
// 							ContainerStatuses: []core1.ContainerStatus{
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:3",
// 									Name:    "container3",
// 								},
// 								{
// 									ImageID: "docker-pullable://alpine@sha256:4",
// 									Name:    "container4",
// 								},
// 							},
// 						},
// 					},
// 				}},
// 			expectedwlidsToContainerToImageIDMap: WlidsToContainerToImageIDMap{
// 				pkgwlid.GetWLID("", "namespace3", "pod", "pod3"): {
// 					"container3": "alpine@sha256:3",
// 					"container4": "alpine@sha256:4",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		wh := NewWatchHandlerMock()
// 		t.Run(tt.name, func(t *testing.T) {
// 			wh.buildIDs(context.TODO(), &tt.podList)
// 			got := wh.getWlidsToContainerToImageIDMap()
// 			assert.True(t, reflect.DeepEqual(got, tt.expectedwlidsToContainerToImageIDMap))
// 		})
// 	}
// }

func TestAddToImageIDToWlidsMap(t *testing.T) {
	wh := NewWatchHandlerMock()

	wh.addToImageIDToWlidsMap("alpine@sha256:1", "wlid1")
	wh.addToImageIDToWlidsMap("alpine@sha256:2", "wlid2")
	// add the new wlid to the same imageID
	wh.addToImageIDToWlidsMap("alpine@sha256:1", "wlid3")

	assert.True(t, reflect.DeepEqual(wh.getImagesIDsToWlidMap(), map[string][]string{
		"alpine@sha256:1": {"wlid1", "wlid3"},
		"alpine@sha256:2": {"wlid2"},
	}))
}

func TestAddTowlidsToContainerToImageIDMap(t *testing.T) {
	wh := NewWatchHandlerMock()

	wh.addToWlidsToContainerToImageIDMap("wlid1", "container1", "alpine@sha256:1")
	wh.addToWlidsToContainerToImageIDMap("wlid2", "container2", "alpine@sha256:2")

	assert.True(t, reflect.DeepEqual(wh.getWlidsToContainerToImageIDMap(), WlidsToContainerToImageIDMap{
		"wlid1": {
			"container1": "alpine@sha256:1",
		},
		"wlid2": {
			"container2": "alpine@sha256:2",
		},
	}))
}

func TestGetNewImageIDsToContainerFromPod(t *testing.T) {
	wh := NewWatchHandlerMock()

	wh.imagesIDToWlidsMap = map[string][]string{
		"alpine@sha256:1": {"wlid"},
		"alpine@sha256:2": {"wlid"},
		"alpine@sha256:3": {"wlid"},
	}

	tests := []struct {
		name     string
		pod      *core1.Pod
		expected map[string]string
	}{
		{
			name: "no new images",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod1",
					Namespace: "namespace1",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							ImageID: "docker-pullable://alpine@sha256:2",
							Name:    "container2",
						},
					},
				},
			},
			expected: map[string]string{},
		},
		{
			name: "one new image",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod2",
					Namespace: "namespace2",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							ImageID: "docker-pullable://alpine@sha256:4",
							Name:    "container4",
						},
					},
				},
			},
			expected: map[string]string{
				"container4": "alpine@sha256:4",
			},
		},
		{
			name: "two new images",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod3",
					Namespace: "namespace3",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:4",
							Name:    "container4",
						},
						{
							ImageID: "docker-pullable://alpine@sha256:5",
							Name:    "container5",
						},
					},
				},
			},
			expected: map[string]string{
				"container4": "alpine@sha256:4",
				"container5": "alpine@sha256:5",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(wh.getNewContainerToImageIDsFromPod(tt.pod), tt.expected))
		})
	}
}

func TestExtractImageIDsToContainersFromPod(t *testing.T) {
	tests := []struct {
		name     string
		pod      *core1.Pod
		expected map[string]string
	}{
		{
			name: "one container",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod1",
					Namespace: "namespace1",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
					},
				},
			},
			expected: map[string]string{
				"alpine@sha256:1": "container1",
			},
		},
		{
			name: "two containers",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod2",
					Namespace: "namespace2",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							ImageID: "docker-pullable://alpine@sha256:2",
							Name:    "container2",
						},
					},
				},
			},
			expected: map[string]string{
				"alpine@sha256:1": "container1",
				"alpine@sha256:2": "container2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(extractImageIDsToContainersFromPod(tt.pod), tt.expected))
		})
	}
}

func TestExtractImageIDsFromPod(t *testing.T) {
	tests := []struct {
		name     string
		pod      *core1.Pod
		expected []string
	}{
		{
			name: "one container",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod1",
					Namespace: "namespace1",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
					},
				},
			},
			expected: []string{"alpine@sha256:1"},
		},
		{
			name: "two containers",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod2",
					Namespace: "namespace2",
				},
				Status: core1.PodStatus{
					ContainerStatuses: []core1.ContainerStatus{
						{
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							ImageID: "docker-pullable://alpine@sha256:2",
							Name:    "container2",
						},
					},
				},
			},
			expected: []string{"alpine@sha256:1", "alpine@sha256:2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(extractImageIDsFromPod(tt.pod), tt.expected))
		})
	}
}

func TestCleanUpImagesIDToWlidsMap(t *testing.T) {
	wh := NewWatchHandlerMock()
	wh.imagesIDToWlidsMap = map[string][]string{
		"alpine@sha256:1": {"pod1"},
		"alpine@sha256:2": {"pod2"},
		"alpine@sha256:3": {"pod3"},
	}
	wh.cleanUpImagesIDToWlidsMap()

	assert.Equal(t, len(wh.imagesIDToWlidsMap), 0)
}

func TestCleanUpWlidsToContainerToImageIDMap(t *testing.T) {
	wh := NewWatchHandlerMock()
	wh.wlidsToContainerToImageIDMap = map[string]map[string]string{
		"pod1": {"container1": "alpine@sha256:1"},
		"pod2": {"container2": "alpine@sha256:2"},
		"pod3": {"container3": "alpine@sha256:3"},
	}
	wh.cleanUpWlidsToContainerToImageIDMap()

	assert.Equal(t, len(wh.wlidsToContainerToImageIDMap), 0)
}

func Test_cleanUpIDs(t *testing.T) {
	wh := NewWatchHandlerMock()
	wh.imagesIDToWlidsMap = map[string][]string{
		"alpine@sha256:1": {"pod1"},
		"alpine@sha256:2": {"pod2"},
		"alpine@sha256:3": {"pod3"},
	}
	wh.wlidsToContainerToImageIDMap = map[string]map[string]string{
		"pod1": {"container1": "alpine@sha256:1"},
		"pod2": {"container2": "alpine@sha256:2"},
		"pod3": {"container3": "alpine@sha256:3"},
	}
	wh.cleanUpIDs()

	assert.Equal(t, len(wh.imagesIDToWlidsMap), 0)
	assert.Equal(t, len(wh.wlidsToContainerToImageIDMap), 0)
}

//go:embed testdata/deployment-two-containers.json
var deploymentTwoContainersJson []byte

//go:embed testdata/deployment.json
var deploymentJson []byte
