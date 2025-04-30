package kube

import (
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 17:06:11
 * @file: service.go
 * @description: kubenetes service operation
 */

func NewService(name, nameSpace string, service v1alpha1.ServiceExpose) *corev1.Service {

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
			Annotations: service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:              service.Type,
			Ports:             service.Ports,
			Selector:          service.Selector,
			LoadBalancerClass: service.LoadBalancerClass,
		},
	}
}
