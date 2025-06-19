package storage

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 22:30:23
 * @file: volume.go
 * @description: persistent volume
 */

// DefaultPersistentVolumeClaimSize 默认PersistentVolumeClaim大小
const DefaultPersistentVolumeClaimSize = "10Gi"

// BuildPersistentVolumeClaim 单个 PVC 通用规范
func BuildPersistentVolumeClaim(cr any,
	mode corev1.PersistentVolumeAccessMode,
	size string,
	storageClassName *string,
) (corev1.PersistentVolumeClaim, error) {
	if size == "" {
		size = DefaultPersistentVolumeClaimSize
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return corev1.PersistentVolumeClaim{}, fmt.Errorf("invalid storage size format: %v", err)
	}

	ObjectMeta := metav1.ObjectMeta{
		Name:      cr.(metav1.Object).GetName(),
		Namespace: cr.(metav1.Object).GetNamespace(),
	}

	persistentVolumeClaim := corev1.PersistentVolumeClaim{
		ObjectMeta: ObjectMeta,
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				mode,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
			StorageClassName: storageClassName,
		},
	}

	return persistentVolumeClaim, nil
}

// BuildPersistentVolumeClaims 多个 PVC 的通用规范
func BuildPersistentVolumeClaims(cr any,
	accessModes []corev1.PersistentVolumeAccessMode,
	size string,
	storageClassName *string,
	count int,
) ([]corev1.PersistentVolumeClaim, error) {
	if size == "" {
		size = DefaultPersistentVolumeClaimSize
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("invalid storage size format: %v", err)
	}

	var pvcs []corev1.PersistentVolumeClaim
	for i := range count {
		ObjectMeta := metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", cr.(metav1.Object).GetName(), i),
			Namespace: cr.(metav1.Object).GetNamespace(),
		}

		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: ObjectMeta,
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: quantity,
					},
				},
				StorageClassName: storageClassName,
			},
		}
		pvcs = append(pvcs, pvc)
	}

	return pvcs, nil
}

// SetPersistentVolumeConfig 设置 PersistentVolume 的通用配置
func SetPersistentVolumeConfig(size string,
	mode *corev1.PersistentVolumeMode,
) (*resource.Quantity, *corev1.PersistentVolumeMode, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid storage size format: %v", err)
	}

	if mode == nil {
		defaultMode := corev1.PersistentVolumeFilesystem
		mode = &defaultMode
	}

	return &quantity, mode, nil
}
