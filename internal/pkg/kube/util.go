package kube

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
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
