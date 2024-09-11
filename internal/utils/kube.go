package utils

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-07-21 15:41:31
 * @file: kube.go
 * @description: kubernetes util
 */

// GetServiceAccessPoint returns the access point for a given service
func GetServiceAccessPoint(svc corev1.Service) string {
	var accessPoint string
	switch svc.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		accessPoint = fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
	case corev1.ServiceTypeNodePort:
		accessPoint = fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].NodePort)
	case corev1.ServiceTypeLoadBalancer:
		// fix bug: 当svc.Status.LoadBalancer.Ingress为空时，会导致数组越界
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			accessPoint = fmt.Sprintf("%s:%s", svc.Status.LoadBalancer.Ingress[0].IP, strconv.Itoa(int(svc.Spec.Ports[0].Port)))
		}
	}
	return accessPoint
}

// CreateOwnerReference creates an owner reference
// for the given owner and group version kind
// with controller and block owner deletion set to true
// and returns the owner reference
func CreateOwnerReference(owner metav1.Object, gvk schema.GroupVersionKind) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		Controller:         ptr.To(true),
		BlockOwnerDeletion: ptr.To(true),
	}
}
