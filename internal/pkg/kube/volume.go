package kube

import (
	greatsqlv1 "github.com/gagraler/greatsql-operator/api/v1"
	"github.com/gagraler/greatsql-operator/internal/consts"
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

// NewPersistentVolumeClaim returns a new persistent volume claim
func NewPersistentVolumeClaim(name, namespace string, cr *greatsqlv1.PodSpec) *corev1.PersistentVolumeClaim {

	defaultStorage := *setDefaultStorage(cr)
	storageQuantity := defaultStorage.String()

	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + consts.DB,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageQuantity),
				},
			},
			StorageClassName: cr.PersistentVolumeClaimTemplate.StorageClassName,
			VolumeMode:       setDefaultPersistentVolumeMode(),
		},
	}
}

// setDefaultStorage set default storage
func setDefaultStorage(cr *greatsqlv1.PodSpec) *resource.Quantity {
	storageQuantity := resource.NewQuantity(cr.PersistentVolumeClaimTemplate.Resources.Requests.Storage().Value(), resource.BinarySI)
	if storageQuantity == nil || storageQuantity.Value() <= 0 {
		storageQuantity = resource.NewQuantity(5, resource.BinarySI)
	}
	return storageQuantity
}

// setDefaultPersistentVolumeMode set default persistent volume mode
func setDefaultPersistentVolumeMode() *corev1.PersistentVolumeMode {
	mode := corev1.PersistentVolumeFilesystem
	return &mode
}
