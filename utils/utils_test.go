package utils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	core1 "k8s.io/api/core/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractContainersToImageIDsFromPod(t *testing.T) {
	tests := []struct {
		pod      *core1.Pod
		expected map[string]string
		name     string
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
							State: core1.ContainerState{
								Running: &core1.ContainerStateRunning{},
							},
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
					},
				},
			},
			expected: map[string]string{
				"container1": "alpine@sha256:1",
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
							State: core1.ContainerState{
								Running: &core1.ContainerStateRunning{},
							},
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							State: core1.ContainerState{
								Running: &core1.ContainerStateRunning{},
							},
							ImageID: "docker-pullable://alpine@sha256:2",
							Name:    "container2",
						},
					},
				},
			},
			expected: map[string]string{
				"container1": "alpine@sha256:1",
				"container2": "alpine@sha256:2",
			},
		},
		{
			name: "init container",
			pod: &core1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "pod2",
					Namespace: "namespace2",
				},
				Status: core1.PodStatus{
					InitContainerStatuses: []core1.ContainerStatus{
						{
							State: core1.ContainerState{
								Running: &core1.ContainerStateRunning{},
							},
							ImageID: "docker-pullable://alpine@sha256:1",
							Name:    "container1",
						},
						{
							State: core1.ContainerState{
								Running: &core1.ContainerStateRunning{},
							},
							ImageID: "docker-pullable://alpine@sha256:2",
							Name:    "container2",
						},
					},
				},
			},
			expected: map[string]string{
				"container1": "alpine@sha256:1",
				"container2": "alpine@sha256:2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(ExtractContainersToImageIDsFromPod(tt.pod), tt.expected))
		})
	}
}
