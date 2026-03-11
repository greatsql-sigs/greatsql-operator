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

const commandTemplate = `
# 从 hostname 提取 ordinal
ORDINAL=${HOSTNAME##*-}
CONFIG_FILE="/configs/config-${ORDINAL}/my.cnf"

# 检查配置文件是否存在
if [ -f "$CONFIG_FILE" ]; then
	echo "Using config for ordinal ${ORDINAL}"
	cp "$CONFIG_FILE" /etc/my.cnf
else
	echo "Config file not found: $CONFIG_FILE"
	exit 1
fi
`

// BuildStatefulSet 构建 StatefulSet
func BuildStatefulSet(cr v1alpha1.Pod,
	replicas *int32,
	name,
	namespace,
	configMapName string,
) (*appsv1.StatefulSet, error) {
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

	// init container 需要根据 Pod 名称动态选择正确的 ConfigMap
	// Pod 名称格式：<statefulset-name>-<ordinal>
	// 从 Pod 名称提取 ordinal，然后从对应的 ConfigMap 复制配置
	initContainers := []corev1.Container{
		{
			Name:  "init",
			Image: "busybox:1.36",
			Command: []string{
				"sh", "-c",
				commandTemplate,
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "configs", MountPath: "/configs"},
				{Name: initVolumeName, MountPath: "/etc"},
			},
			Env: []corev1.EnvVar{
				{
					Name: "HOSTNAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
		},
	}

	// 创建一个 projected volume，包含所有的 ConfigMap
	// 每个 ConfigMap 挂载到 /configs/config-<ordinal>/
	// 从 configMapName 中提取前缀 (例如: greatsql-mgr-config-0 -> greatsql-mgr)
	configMapPrefix := name + "-config"

	var projections []corev1.VolumeProjection
	if replicas != nil {
		// 为每个 replica 创建一个 ConfigMap projection
		for i := int32(0); i < *replicas; i++ {
			optional := true
			projections = append(projections, corev1.VolumeProjection{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-%d", configMapPrefix, i),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "my.cnf",
							Path: fmt.Sprintf("config-%d/my.cnf", i),
						},
					},
					Optional: &optional,
				},
			})
		}
	}

	volumes := []corev1.Volume{
		{
			Name: "configs",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: projections,
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
		StartupProbe:    cr.Container.StartupProbe,
		ReadinessProbe:  cr.Container.ReadinessProbe,
		LivenessProbe:   cr.Container.LivenessProbe,
		SecurityContext: cr.Container.SecurityContext,
		Lifecycle:       cr.Container.Lifecycle,
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    replicas,
			ServiceName: fmt.Sprintf("%s-headless", name),
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					InitContainers:                initContainers,
					Volumes:                       volumes,
					Containers:                    []corev1.Container{mainContainer},
					TerminationGracePeriodSeconds: cr.TerminationGracePeriodSeconds,
					SchedulerName:                 cr.SchedulerName,
					ServiceAccountName:            cr.ServiceAccountName,
					SecurityContext:               cr.PodSecurityContext,
					NodeSelector:                  cr.NodeSelector,
					PriorityClassName:             getPriorityClassName(cr.PriorityClassName),
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
					MaxUnavailable: func() *intstr.IntOrString { v := intstr.FromInt32(1); return &v }(),
				},
			},
		},
	}

	return sts, nil
}

func getPriorityClassName(priorityClassName *string) string {
	if priorityClassName == nil {
		return ""
	}
	return *priorityClassName
}
