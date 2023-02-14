package watcher

import (
	"context"
	"reflect"
	"testing"

	pkgwlid "github.com/armosec/utils-k8s-go/wlid"
	"github.com/stretchr/testify/assert"
	core1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildImageIDsToWlidsToContainerToImageIDMap(t *testing.T) {

	tests := []struct {
		name                string
		podList             core1.PodList
		expectedImageIDsMap map[string][]string
	}{
		{
			name: "remove prefix docker-pullable://",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "docker-pullable://alpine@sha256:1",
									Name:    "container1",
								},
							},
						},
					}}},
			expectedImageIDsMap: map[string][]string{
				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
			},
		},
		{
			name: "image id without docker-pullable:// prefix",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "alpine@sha256:1",
									Name:    "container1",
								},
							},
						},
					}}},
			expectedImageIDsMap: map[string][]string{
				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
			},
		},
		{
			name: "two wlids for the same image id",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
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
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test2",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "docker-pullable://alpine@sha256:1",
									Name:    "container2",
								},
							},
						},
					},
				},
			},
			expectedImageIDsMap: map[string][]string{
				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test"), pkgwlid.GetWLID("", "default", "pod", "test2")},
			},
		},
		{
			name: "two wlids two image ids",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
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
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test2",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "docker-pullable://alpine@sha256:2",
									Name:    "container2",
								},
							},
						},
					}}},
			expectedImageIDsMap: map[string][]string{
				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
				"alpine@sha256:2": {pkgwlid.GetWLID("", "default", "pod", "test2")},
			},
		},
		{
			name: "one wlid two image ids",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						TypeMeta: v1.TypeMeta{
							Kind: "pod",
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
					}}},
			expectedImageIDsMap: map[string][]string{
				"alpine@sha256:1": {pkgwlid.GetWLID("", "default", "pod", "test")},
				"alpine@sha256:2": {pkgwlid.GetWLID("", "default", "pod", "test")},
			},
		},
	}

	for _, tt := range tests {
		wh := NewWatchHandler()
		t.Run(tt.name, func(t *testing.T) {
			wh.buildImageIDsToWlidsToContainerToImageIDMap(context.TODO(), &tt.podList)
			assert.True(t, reflect.DeepEqual(wh.GetImagesIDsToWlidMap(), tt.expectedImageIDsMap))
		})
	}
}

func TestBuildWlidsToContainerToImageIDMap(t *testing.T) {

	tests := []struct {
		name                                 string
		podList                              core1.PodList
		expectedwlidsToContainerToImageIDMap map[string]map[string]string
	}{
		{
			name: "imageID with docker-pullable prefix",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
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
					}},
			},
			expectedwlidsToContainerToImageIDMap: map[string]map[string]string{
				pkgwlid.GetWLID("", "namespace1", "pod", "pod1"): {
					"container1": "alpine@sha256:1",
				},
			},
		},
		{
			name: "imageID without docker-pullable prefix",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "pod1",
							Namespace: "namespace1",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "alpine@sha256:1",
									Name:    "container1",
								},
							},
						},
					}},
			},
			expectedwlidsToContainerToImageIDMap: map[string]map[string]string{
				pkgwlid.GetWLID("", "namespace1", "pod", "pod1"): {
					"container1": "alpine@sha256:1",
				},
			},
		},
		{
			name: "two containers for same wlid",
			podList: core1.PodList{
				Items: []core1.Pod{
					{
						ObjectMeta: v1.ObjectMeta{
							Name:      "pod3",
							Namespace: "namespace3",
						},
						Status: core1.PodStatus{
							ContainerStatuses: []core1.ContainerStatus{
								{
									ImageID: "docker-pullable://alpine@sha256:3",
									Name:    "container3",
								},
								{
									ImageID: "docker-pullable://alpine@sha256:4",
									Name:    "container4",
								},
							},
						},
					},
				}},
			expectedwlidsToContainerToImageIDMap: map[string]map[string]string{
				pkgwlid.GetWLID("", "namespace3", "pod", "pod3"): {
					"container3": "alpine@sha256:3",
					"container4": "alpine@sha256:4",
				},
			},
		},
	}

	for _, tt := range tests {
		wh := NewWatchHandler()
		t.Run(tt.name, func(t *testing.T) {
			wh.buildWlidsToContainerToImageIDMap(context.TODO(), &tt.podList)
			assert.True(t, reflect.DeepEqual(wh.GetWlidsToContainerToImageIDMap(), tt.expectedwlidsToContainerToImageIDMap))
		})
	}
}

func TestAddToImageIDToWlidsToContainerToImageIDMap(t *testing.T) {
	wh := NewWatchHandler()

	wh.addToImageIDToWlidsToContainerToImageIDMap("alpine@sha256:1", "wlid1")
	wh.addToImageIDToWlidsToContainerToImageIDMap("alpine@sha256:2", "wlid2")
	// add the new wlid to the same imageID
	wh.addToImageIDToWlidsToContainerToImageIDMap("alpine@sha256:1", "wlid3")

	assert.True(t, reflect.DeepEqual(wh.GetImagesIDsToWlidMap(), map[string][]string{
		"alpine@sha256:1": {"wlid1", "wlid3"},
		"alpine@sha256:2": {"wlid2"},
	}))
}

func TestAddTowlidsToContainerToImageIDMap(t *testing.T) {
	wh := NewWatchHandler()

	wh.addToWlidsToContainerToImageIDMap("wlid1", "container1", "alpine@sha256:1")
	wh.addToWlidsToContainerToImageIDMap("wlid2", "container2", "alpine@sha256:2")

	assert.True(t, reflect.DeepEqual(wh.GetWlidsToContainerToImageIDMap(), map[string]map[string]string{
		"wlid1": {
			"container1": "alpine@sha256:1",
		},
		"wlid2": {
			"container2": "alpine@sha256:2",
		},
	}))
}

func TestGetNewImageIDsToContainerFromPod(t *testing.T) {
	wh := NewWatchHandler()
	wh.imagesIDToWlidsToContainerToImageIDMap = map[string][]string{
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
			assert.True(t, reflect.DeepEqual(wh.getNewImageIDsToContainerFromPod(tt.pod), tt.expected))
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
