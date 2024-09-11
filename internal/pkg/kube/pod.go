package kube

import (
	"fmt"

	greatsqlv1 "github.com/gagraler/greatsql-operator/api/v1"
	"github.com/gagraler/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-22 23:34:49
 * @file: pod.go
 * @description: kubernetes pod operation
 */

// NewContainers returns a new container
func NewContainers(name string, cr *greatsqlv1.PodSpec, ordinal int, isStatefulSet bool) []corev1.Container {

	var volumeMounts []corev1.VolumeMount

	configVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%s", name, consts.Config),
		MountPath: consts.ConfigDir + consts.ConfigFile,
		SubPath:   consts.ConfigFile,
	}

	dbVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%s", name, consts.DB),
		MountPath: consts.DB,
	}

	if isStatefulSet {
		configVolumeMount.Name = fmt.Sprintf("%s-%s-%d", name, consts.Config, ordinal)
		dbVolumeMount.Name = fmt.Sprintf("%s-%s-%d", name, consts.DB, ordinal)
	}

	volumeMounts = append(volumeMounts, configVolumeMount, dbVolumeMount)

	return []corev1.Container{
		{
			Name:            name,
			Image:           cr.Containers[0].Image,
			Resources:       cr.Containers[0].Resources,
			StartupProbe:    &cr.Containers[0].StartupProbe,
			ReadinessProbe:  &cr.Containers[0].ReadinessProbe,
			LivenessProbe:   &cr.Containers[0].LivenessProbe,
			SecurityContext: cr.Containers[0].SecurityContext,
			Ports: []corev1.ContainerPort{
				{
					Name:          consts.MySQLPortName,
					ContainerPort: consts.MySQLPort,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			ImagePullPolicy: cr.Containers[0].ImagePullPolicy,
			Env:             cr.Containers[0].Envs,
			VolumeMounts:    volumeMounts,
		},
	}
}

func NewPod(configMapName string, cr *greatsqlv1.GroupReplicationCluster, ordinal int) corev1.Pod {

	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-manager",
			Namespace: cr.Name,
			Labels: map[string]string{
				consts.AppKubernetesName:     cr.Name,
				consts.AppKubernetesInstance: cr.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers:                    NewContainers(cr.Name, cr.Spec.ClusterSpec.PodSpec, ordinal, false),
			TerminationGracePeriodSeconds: cr.Spec.ClusterSpec.PodSpec.TerminationGracePeriodSeconds,
			SchedulerName:                 cr.Spec.ClusterSpec.PodSpec.SchedulerName,
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
					Name: cr.Name + consts.DB,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: cr.Name + consts.DB,
						},
					},
				},
			},
			DNSPolicy: cr.Spec.ClusterSpec.DnsPolicy,
		},
	}
}

// GetNodeName returns the node name of the pod
func GetNodeName(pod *corev1.Pod) string {
	if pod.Spec.NodeName != "" {
		return pod.Spec.NodeName
	}
	return ""
}

// GetPodDNS returns the pod dns.
func GetPodDNS(pod *corev1.Pod) string {
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
}

// GetPodIP returns the pod ip of the pod
func GetPodIP(pod *corev1.Pod) string {
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
}
