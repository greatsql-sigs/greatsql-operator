package kube

import (
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 17:06:11
 * @file: service.go
 * @description: kubenetes service operation
 */

func NewService(name, nameSpace, kind string, objectMeta metav1.Object, port []corev1.ServicePort, svcType corev1.ServiceType) *corev1.Service {

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       corev1.SchemeGroupVersion.WithKind("Service").Kind,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(objectMeta, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    kind,
				}),
			},
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  svcType,
			Ports: port,
			Selector: map[string]string{
				consts.AppKubernetesName: name,
			},
		},
	}
}
