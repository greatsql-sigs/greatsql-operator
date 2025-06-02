package workload

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNodeName returns the node name of the pod
func GetNodeName(pod *corev1.Pod) string {
	if pod.Spec.NodeName != "" {
		return pod.Spec.NodeName
	}
	return ""
}

// GetPodDNS returns the pod dns.
func GetPodDNS(pod *corev1.Pod) string {
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
}

// GetPodIP returns the pod ip of the pod
func GetPodIP(pod *corev1.Pod) string {
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
}

func IsPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning || pod.DeletionTimestamp != nil {
		return false
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.ContainersReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func PodsByLabels(ctx context.Context, cl client.Reader, l map[string]string, namespace string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(l),
		Namespace:     namespace,
	}
	if err := cl.List(ctx, podList, opts); err != nil {
		return nil, err
	}

	return podList.Items, nil
}
