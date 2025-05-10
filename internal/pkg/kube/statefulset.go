package kube

import (
	"fmt"
	"reflect"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type VolumeBuilderFunc func(cr interface{}) ([]corev1.Volume, error)
type VolumeMountBuilderFunc func(cr interface{}) ([]corev1.VolumeMount, error)

// BuildStatefulSet 通用的 StatefulSet
func BuildStatefulSet(
	cr interface{}, configMapName, serviceName string, ordinal int,
	volumeBuilder VolumeBuilderFunc,
	volumeMountBuilder VolumeMountBuilderFunc,
) (*appsv1.StatefulSet, error) {

	crValue := reflect.ValueOf(cr)
	crType := reflect.TypeOf(cr)

	if crType.Kind() != reflect.Ptr || crValue.IsNil() {
		return nil, fmt.Errorf("cr must be a non-nil pointer")
	}

	crElem := crValue.Elem()
	//crElemType := crElem.Type()

	// 获取 metadata.name 和 metadata.namespace
	metadataField := crElem.FieldByName("ObjectMeta")
	if !metadataField.IsValid() {
		return nil, fmt.Errorf("ObjectMeta field not found in CR")
	}
	metadata := metadataField.Interface().(metav1.ObjectMeta)
	name := metadata.Name
	namespace := metadata.Namespace

	// 获取 replicas 数量
	specField := crElem.FieldByName("Spec")
	if !specField.IsValid() {
		return nil, fmt.Errorf("spec field not found in CR")
	}

	replicas := int32(0)
	specValue := specField
	memberField := specValue.FieldByName("Member")
	if memberField.IsValid() && memberField.Kind() == reflect.Slice {
		for i := 0; i < memberField.Len(); i++ {
			member := memberField.Index(i)
			sizeField := member.FieldByName("Size")
			if sizeField.IsValid() && !sizeField.IsNil() {
				size := sizeField.Elem().Int()
				replicas += int32(size)
			}
		}
	}

	labels := map[string]string{
		"app.kubernetes.io/name":     name,
		"app.kubernetes.io/instance": name,
	}

	volumes, err := volumeBuilder(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to build volumes: %v", err)
	}

	volumeMounts, err := volumeMountBuilder(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to build volumeMounts: %v", err)
	}

	// 构建 StatefulSet
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: crType.String(),
					Kind:       crType.String(),
					Name:       name,
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: serviceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           crElem.FieldByName("Image").String(),
							ImagePullPolicy: corev1.PullPolicy(crElem.FieldByName("ImagePullPolicy").String()),
							Env:             crElem.FieldByName("Env").Interface().([]corev1.EnvVar),
							VolumeMounts:    volumeMounts,
							Resources:       crElem.FieldByName("Resources").Interface().(corev1.ResourceRequirements),
							StartupProbe:    crElem.FieldByName("StartupProbe").Interface().(*corev1.Probe),
							ReadinessProbe:  crElem.FieldByName("ReadinessProbe").Interface().(*corev1.Probe),
							LivenessProbe:   crElem.FieldByName("LivenessProbe").Interface().(*corev1.Probe),
							SecurityContext: crElem.FieldByName("SecurityContext").Interface().(*corev1.SecurityContext),
						},
					},
					TerminationGracePeriodSeconds: &[]int64{int64(crElem.FieldByName("TerminationGracePeriodSeconds").Int())}[0],
					SchedulerName:                 crElem.FieldByName("SchedulerName").String(),
					ServiceAccountName:            crElem.FieldByName("ServiceAccountName").String(),
					SecurityContext:               crElem.FieldByName("SecurityContext").Interface().(*corev1.PodSecurityContext),
					NodeSelector:                  crElem.FieldByName("NodeSelector").Interface().(map[string]string),
					Tolerations:                   crElem.FieldByName("Tolerations").Interface().([]corev1.Toleration),
					Volumes:                       volumes,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name + "-data-" + strconv.Itoa(ordinal),
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(DefaultPersistentVolumeClaimSize),
							},
						},
						StorageClassName: func() *string {
							s := crElem.FieldByName("StorageClassName").String()
							return &s
						}(),
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: func() *int32 { p := int32(0); return &p }(),
					MaxUnavailable: func() *intstr.IntOrString {
						val := intstr.FromInt(1)
						return &val
					}(),
				},
			},
		},
	}

	return sts, nil
}
