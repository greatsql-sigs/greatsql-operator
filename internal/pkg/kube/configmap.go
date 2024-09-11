package kube

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 17:09:59
 * @file: configmap.go
 * @description: kubenetes configmap operation
 */

// ConfigMap returns a ConfigMap object
func NewConfigMap(name, namespace, key, value string) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			key: value,
		},
	}
}
