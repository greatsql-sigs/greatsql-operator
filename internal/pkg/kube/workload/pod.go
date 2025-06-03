package workload

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodStatus 获取 Pod 的状态
func GetPodStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "Unknown"
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		return "Pending"
	case corev1.PodRunning:
		return "Running"
	case corev1.PodSucceeded:
		return "Succeeded"
	case corev1.PodFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// IsPodReady 检查 Pod 是否就绪
func IsPodReady(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// GetPodAge 计算 Pod 的创建时间
func GetPodAge(pod *corev1.Pod) string {
	if pod == nil {
		return "Unknown"
	}

	age := time.Since(pod.CreationTimestamp.Time)
	return age.Round(time.Second).String()
}

// GetPodContainerStatus 获取 Pod 中指定容器的状态
func GetPodContainerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	if pod == nil {
		return nil
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			return &status
		}
	}
	return nil
}

// GetPodMetrics 获取 Pod 的指标数据
func GetPodMetrics(ctx context.Context, client client.Client, pod *corev1.Pod) (map[string]string, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod is nil")
	}

	// TODO: 这里需要实现与 metrics-server 的集成
	// 暂时返回空数据
	return make(map[string]string), nil
}

// GetPodHealth 获取 Pod 的健康状态
func GetPodHealth(pod *corev1.Pod) string {
	if pod == nil {
		return "Unknown"
	}

	// 检查所有容器的就绪状态
	for _, container := range pod.Status.ContainerStatuses {
		if !container.Ready {
			return "NotReady"
		}
	}

	// 检查 Pod 的就绪状态
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return "Healthy"
		}
	}

	return "NotHealthy"
}

// GetPodIP returns the pod ip of the pod
func GetPodIP(pod *corev1.Pod) string {
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
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
