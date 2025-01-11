package kube

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-10-13 16:01:02
 * @file: util.go
 * @description: util
 */

// handleReplicaChange 处理副本数变化的通用逻辑
func handleReplicaChange[T any](oldObj, newObj interface{}, resourceType string) {
	oldReplicas := getReplicas(oldObj)
	newReplicas := getReplicas(newObj)

	if oldReplicas != newReplicas {
		fmt.Printf("%s %s has been updated. Replicas: %d -> %d\n", resourceType, getName(newObj), oldReplicas, newReplicas)
	}
}

// getReplicas 从对象中获取副本数
func getReplicas(obj interface{}) int32 {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return v.Status.Replicas
	case *appsv1.StatefulSet:
		return v.Status.Replicas
	default:
		return 0
	}
}

// getName 从对象中获取名称
func getName(obj interface{}) string {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return v.Name
	case *appsv1.StatefulSet:
		return v.Name
	default:
		return ""
	}
}

// getPodByLabels 根据标签获取 Pod
func getPodByLabels(cli client.Reader, labelSelector string) ([]corev1.Pod, error) {
	sel, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}
	opts := &client.ListOptions{
		LabelSelector: sel,
		// Namespace:     namespace,
	}
	err = cli.List(context.Background(), podList, opts)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// Retry 重试函数
func Retry(step func() error, retries int, delay time.Duration) error {
	for i := 0; i < retries; i++ {
		if err := step(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("step failed after %d retries", retries)
}
