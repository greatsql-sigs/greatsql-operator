package kube

import (
	"context"
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-22 23:34:49
 * @file: pod.go
 * @description: kubernetes pod operation
 */

func NewContainers(name string, cr *v1alpha1.PodSpec, ordinal int, isStatefulSet bool) []corev1.Container {

	var volumeMounts []corev1.VolumeMount

	configVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%s", name, consts.Config),
		MountPath: consts.ConfigDir + consts.ConfigFile,
		SubPath:   consts.ConfigFile,
	}

	dbVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%s", name, consts.DB),
		MountPath: consts.DataDir,
	}

	if isStatefulSet {
		configVolumeMount.Name = fmt.Sprintf("%s-%s-%d", name, consts.Config, ordinal)
		dbVolumeMount.Name = fmt.Sprintf("%s-%s-%d", name, consts.DB, ordinal)
	}

	volumeMounts = append(volumeMounts, configVolumeMount, dbVolumeMount)

	return []corev1.Container{
		{
			Name:            cr.Containers[0].Name,
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

func NewPod(name, namespace, configMapName string, cr *v1alpha1.PodSpec, ordinal int) corev1.Pod {

	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-manager",
			Namespace: namespace,
			Labels: map[string]string{
				consts.AppKubernetesName:     name,
				consts.AppKubernetesInstance: name,
			},
		},
		Spec: corev1.PodSpec{
			Containers:                    NewContainers(name, cr, ordinal, false),
			TerminationGracePeriodSeconds: cr.TerminationGracePeriodSeconds,
			SchedulerName:                 cr.SchedulerName,
			ServiceAccountName:            cr.ServiceAccountName,
			SecurityContext:               cr.PodSecurityContext,
			NodeSelector:                  cr.NodeSelector,
			Tolerations:                   cr.Tolerations,
			Volumes: []corev1.Volume{
				{
					Name: fmt.Sprintf("%s-%s", name, consts.Config),
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
					Name: name + consts.DB,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: name + consts.DB,
						},
					},
				},
			},
			//DNSPolicy: cr.Spec.ClusterSpec.DnsPolicy,
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

func IsPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning || pod.DeletionTimestamp != nil {
		return false
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.ContainersReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func PodsByLabels(ctx context.Context, cl client.Reader, l map[string]string, namespace string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(l),
		Namespace:     namespace,
	}
	if err := cl.List(ctx, podList, opts); err != nil {
		return nil, err
	}

	return podList.Items, nil
}
