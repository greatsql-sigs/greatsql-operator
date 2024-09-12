package kube

import (
	greatsqlv1 "github.com/greatsql-sigs/greatsql-operator/api/v1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 18:02:46
 * @file: deployment.go
 * @description: kubernetes deployment operation
 */

// NewDeployment returns a new deployment
func NewDeployment(configMapName string, cr *greatsqlv1.SingleInstance, ordinal int) *appsv1.Deployment {
	labels := map[string]string{
		consts.AppKubernetesName:     cr.Name,
		consts.AppKubernetesInstance: cr.Name,
	}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	var affinity *corev1.Affinity

	if cr.Spec.PodSpec.Affinity == nil {
		cr.Spec.PodSpec.Affinity = nil
	} else {
		affinity = cr.PodAffinity(labels)
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   greatsqlv1.GroupVersion.Group,
					Version: greatsqlv1.GroupVersion.Version,
					Kind:    consts.SingleInstance,
				}),
			},
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: cr.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers:                    NewContainers(cr.Name, &cr.Spec.PodSpec, ordinal, false),
					TerminationGracePeriodSeconds: cr.Spec.PodSpec.TerminationGracePeriodSeconds,
					SchedulerName:                 cr.Spec.PodSpec.SchedulerName,
					Affinity:                      affinity,
					ServiceAccountName:            cr.Spec.PodSpec.ServiceAccountName,
					SecurityContext:               cr.Spec.PodSpec.PodSecurityContext,
					NodeSelector:                  cr.Spec.PodSpec.NodeSelector,
					Tolerations:                   cr.Spec.PodSpec.Tolerations,
					Volumes: []corev1.Volume{
						{
							Name: cr.Name + consts.Config,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
									DefaultMode: &[]int32{0664}[0],
								},
							},
						},
						{
							Name: cr.Name + consts.DB,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: cr.Name + consts.DB,
								},
							},
						},
					},
					DNSPolicy: cr.Spec.DnsPolicy,
				},
			},
			Selector: selector,
			Strategy: appsv1.DeploymentStrategy{
				Type: cr.Spec.UpdateStrategy,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{IntVal: 1},
					MaxSurge:       &intstr.IntOrString{IntVal: 1},
				},
			},
		},
	}
}
