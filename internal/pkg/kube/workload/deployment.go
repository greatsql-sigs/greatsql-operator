package workload

import (
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/kube/schedule"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
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

// stringPtr 返回字符串的指针
func stringPtr(s string) *string {
	return &s
}

// NewDeployment returns a new deployment
func NewDeployment(configMapName string, cr *v1alpha1.Standalone, ordinal int) *appsv1.Deployment {
	labels := map[string]string{
		consts.AppKubernetesName:     cr.Name,
		consts.AppKubernetesInstance: cr.Name,
	}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	var affinity *corev1.Affinity

	if cr.Spec.Pod.Affinity == nil {
		cr.Spec.Pod.Affinity = &v1alpha1.Affinity{
			TopologyKey: stringPtr("kubernetes.io/hostname"),
		}
	}
	affinity = schedule.SetPodAntiAffinity(cr.Spec, labels)

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
					Group:   v1alpha1.GroupVersion.Group,
					Version: v1alpha1.GroupVersion.Version,
					Kind:    consts.Standalone,
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
					Containers:                    NewContainers(cr.Name, &cr.Spec.Pod, ordinal, false),
					TerminationGracePeriodSeconds: cr.Spec.Pod.TerminationGracePeriodSeconds,
					SchedulerName:                 cr.Spec.Pod.SchedulerName,
					Affinity:                      affinity,
					ServiceAccountName:            cr.Spec.Pod.ServiceAccountName,
					SecurityContext:               cr.Spec.Pod.PodSecurityContext,
					NodeSelector:                  cr.Spec.Pod.NodeSelector,
					Tolerations:                   cr.Spec.Pod.Tolerations,
					Volumes: []corev1.Volume{
						{
							Name: fmt.Sprintf("%s-%s", cr.Name, consts.Config),
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
							Name: fmt.Sprintf("%s-%s", cr.Name, consts.DB),
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-%s", cr.Name, consts.DB),
								},
							},
						},
					},
					DNSPolicy: cr.Spec.Pod.DnsPolicy,
				},
			},
			Selector: selector,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyType(cr.Spec.UpdateStrategy.Type),
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{IntVal: 1},
					MaxSurge:       &intstr.IntOrString{IntVal: 1},
				},
			},
		},
	}
}
