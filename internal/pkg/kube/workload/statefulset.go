package workload

import (
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildStatefulSet 构建 StatefulSet
func BuildStatefulSet(cr v1alpha1.Pod, replicas *int32, name, namespace, configMapName string) (*appsv1.StatefulSet, error) {
	labels := map[string]string{
		"app.kubernetes.io/name":     name,
		"app.kubernetes.io/instance": name,
	}

	size := cr.Storages.Size
	if size == nil || *size == "" {
		val := "10Gi"
		cr.Storages.Size = &val
	}

	// 构造 initContainer 和共享 volume
	initVolumeName := fmt.Sprintf("%s-config", name)
	configVolumeName := "config"

	initContainers := []corev1.Container{
		{
			Name:  "init",
			Image: "busybox:1.36",
			Command: []string{
				"sh", "-c", "cp /tmp/conf/my.cnf /etc/my.cnf",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: configVolumeName, MountPath: "/tmp/conf"},
				{Name: initVolumeName, MountPath: "/etc"},
			},
		},
	}

	volumes := []corev1.Volume{
		{
			Name: configVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				},
			},
		},
		{
			Name: initVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// 主容器
	mainContainer := corev1.Container{
		Name:            name,
		Image:           cr.Container.Image,
		ImagePullPolicy: cr.Container.ImagePullPolicy,
		Env:             cr.Container.Envs,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data/GreatSQL",
			},
			{
				Name:      initVolumeName,
				MountPath: "/etc/my.cnf",
				SubPath:   "my.cnf",
			},
		},
		Resources:       cr.Container.Resources,
		StartupProbe:    &cr.Container.StartupProbe,
		ReadinessProbe:  &cr.Container.ReadinessProbe,
		LivenessProbe:   &cr.Container.LivenessProbe,
		SecurityContext: cr.Container.SecurityContext,
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    replicas,
			ServiceName: cr.ServiceName,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Volumes: append(volumes, corev1.Volume{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}),
					Containers:                    []corev1.Container{mainContainer},
					TerminationGracePeriodSeconds: cr.TerminationGracePeriodSeconds,
					SchedulerName:                 cr.SchedulerName,
					ServiceAccountName:            cr.ServiceAccountName,
					SecurityContext:               cr.PodSecurityContext,
					NodeSelector:                  cr.NodeSelector,
					PriorityClassName:             *cr.PriorityClassName,
					Tolerations:                   cr.Tolerations,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "data", Labels: labels},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(*size)},
						},
						StorageClassName: cr.Storages.StorageClassName,
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition:      func() *int32 { p := int32(0); return &p }(),
					MaxUnavailable: func() *intstr.IntOrString { v := intstr.FromInt(1); return &v }(),
				},
			},
		},
	}

	return sts, nil
}
