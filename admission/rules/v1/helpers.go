package rules

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/kubernetes"
)

func GetParentWorkloadDetails(event admission.Attributes, clientset kubernetes.Interface) (string, string, string, error) {
	podName, namespace := event.GetName(), event.GetNamespace()

	if podName == "" || namespace == "" {
		return "", "", "", fmt.Errorf("invalid pod details from admission event")
	}

	pod, err := GetPodDetails(clientset, podName, namespace)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get pod details: %v", err)
	}

	workloadKind, workloadName, workloadNamespace := ExtractPodInformation(pod)

	return workloadKind, workloadName, workloadNamespace, nil
}

func GetPodDetails(clientset kubernetes.Interface, podName, namespace string) (*v1.Pod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %v", err)
	}
	return pod, nil
}

func GetNodeName(event admission.Attributes, clientset kubernetes.Interface) (string, error) {
	podName, namespace := event.GetName(), event.GetNamespace()

	if podName == "" || namespace == "" {
		return "", fmt.Errorf("invalid pod details from admission event")
	}

	pod, err := GetPodDetails(clientset, podName, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get pod details: %v", err)
	}

	return pod.Spec.NodeName, nil
}

func ExtractPodInformation(pod *v1.Pod) (string, string, string) {
	var workloadKind, workloadName, workloadNamespace string

	for _, ownerRef := range pod.OwnerReferences {
		// Consider common workload controllers
		if ownerRef.Kind == "ReplicaSet" || ownerRef.Kind == "StatefulSet" || ownerRef.Kind == "DaemonSet" || ownerRef.Kind == "Job" {
			workloadKind = ownerRef.Kind
			workloadName = ownerRef.Name
			workloadNamespace = pod.Namespace
			break
		}
	}

	return workloadKind, workloadName, workloadNamespace
}
