package kube

import (
	"strconv"

	greatsqlv1 "github.com/gagraler/greatsql-operator/api/v1"
	"github.com/gagraler/greatsql-operator/internal/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 22:43:46
 * @file: statefulset.go
 * @description: statefulset operation
 */

func NewStatefulSet(configMapName, serviceName string, cr *greatsqlv1.GroupReplicationCluster, ordinal int) *appsv1.StatefulSet {

	labels := map[string]string{
		consts.AppKubernetesName:     cr.Name,
		consts.AppKubernetesInstance: cr.Name,
	}
	affinity := cr.PodAffinity(labels)

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   greatsqlv1.GroupVersion.Group,
					Version: greatsqlv1.GroupVersion.Version,
					Kind:    consts.GroupReplicationCluster,
				}),
			},
			Labels: labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    cr.Spec.Member[0].Size,
			ServiceName: serviceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.AppKubernetesName: cr.Name,
					},
				},
				Spec: corev1.PodSpec{
					// InitContainers:                NewInitContainers(cr.Name, cr.Spec.ClusterSpec.PodSpec, cr.Spec.ClusterSpec.Ports),
					Containers:                    NewContainers(cr.Name, cr.Spec.ClusterSpec.PodSpec, ordinal, true),
					TerminationGracePeriodSeconds: cr.Spec.ClusterSpec.PodSpec.TerminationGracePeriodSeconds,
					SchedulerName:                 cr.Spec.ClusterSpec.PodSpec.SchedulerName,
					Affinity:                      affinity,
					ServiceAccountName:            cr.Spec.ClusterSpec.PodSpec.ServiceAccountName,
					SecurityContext:               cr.Spec.ClusterSpec.PodSpec.PodSecurityContext,
					NodeSelector:                  cr.Spec.ClusterSpec.PodSpec.NodeSelector,
					Tolerations:                   cr.Spec.ClusterSpec.PodSpec.Tolerations,
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
							//Name: cr.Name + consts.DB,
							// VolumeSource: corev1.VolumeSource{
							// 	PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							// 		ClaimName: cr.Name + consts.DB + strconv.Itoa(ordinal),
							// 	},
							// },
						},
					},
					DNSPolicy: cr.Spec.ClusterSpec.DnsPolicy,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   cr.Name + consts.DB + strconv.Itoa(ordinal),
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources:        cr.Spec.ClusterSpec.PodSpec.PersistentVolumeClaimTemplate.Resources,
						StorageClassName: cr.Spec.ClusterSpec.PodSpec.PersistentVolumeClaimTemplate.StorageClassName,
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: cr.Spec.ClusterSpec.UpdateStrategy.Type,
				// Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition:      cr.Spec.ClusterSpec.UpdateStrategy.RolelingUpdate.Partition,
					MaxUnavailable: cr.Spec.ClusterSpec.UpdateStrategy.RolelingUpdate.MaxUnavailable,
				},
			},
		},
	}
}
