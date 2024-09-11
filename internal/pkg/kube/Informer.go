package kube

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-07-27 20:29:25
 * @file: Informer.go
 * @description: Informer
 *
 */

// EventHandlerFunc 事件处理函数
type EventHandlerFunc struct {
	UpdateFunc func(oldObj, newObj interface{})
	// DeleteFunc func(obj interface{})
}

// informerUpdate Informer更新
func informerUpdate[T any](informerFunc func() cache.SharedIndexInformer,
	eventHandlers EventHandlerFunc) cache.SharedIndexInformer {

	informer := informerFunc()
	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: eventHandlers.UpdateFunc,
	}); err != nil {
		panic(err)
	}
	return informer
}

// informerDelete Informer删除
// func informerDelete[T any](informerFunc func() cache.SharedIndexInformer,
// 	eventHandlers EventHandlerFunc) cache.SharedIndexInformer {

// 	informer := informerFunc()
// 	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		DeleteFunc: eventHandlers.DeleteFunc,
// 	}); err != nil {
// 		panic(err)
// 	}
// 	return informer
// }

// PodInformer Pod Informer
func PodInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[corev1.Pod](factory.Core().V1().Pods().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			if oldPod.Status.ContainerStatuses[0].RestartCount != newPod.Status.ContainerStatuses[0].RestartCount {
				fmt.Printf("Pod %s has been restarted. Restart count: %d\n", newPod.Name, newPod.Status.ContainerStatuses[0].RestartCount)
			}
		},
	})
}

// DeploymentInformer Deployment Informer
func DeploymentInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[appsv1.Deployment](factory.Apps().V1().Deployments().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeploy := oldObj.(*appsv1.Deployment)
			newDeploy := newObj.(*appsv1.Deployment)
			if oldDeploy.Status.Replicas != newDeploy.Status.Replicas {
				fmt.Printf("Deployment %s has been updated. Replicas: %d\n", newDeploy.Name, newDeploy.Status.Replicas)
			}
		},
	})
}

// StatefulSetInformer StatefulSet Informer
func StatefulSetInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[appsv1.StatefulSet](factory.Apps().V1().StatefulSets().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSts := oldObj.(*appsv1.StatefulSet)
			newSts := newObj.(*appsv1.StatefulSet)
			if oldSts.Status.Replicas != newSts.Status.Replicas {
				fmt.Printf("StatefulSet %s has been updated. Replicas: %d\n", newSts.Name, newSts.Status.Replicas)
			}
		},
	})
}

// SecretInformer Secret Informer
func SecretInformer(factory informers.SharedInformerFactory, onUpdate func(oldSecret, newSecret *corev1.Secret)) cache.SharedIndexInformer {
	return informerUpdate[corev1.Secret](factory.Core().V1().Secrets().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSecret := oldObj.(*corev1.Secret)
			newSecret := newObj.(*corev1.Secret)

			oldKey, err := cache.MetaNamespaceKeyFunc(oldSecret)
			if err != nil {
				fmt.Printf("Error getting key for old secret: %v\n", err)
				return
			}
			newKey, err := cache.MetaNamespaceKeyFunc(newSecret)
			if err != nil {
				fmt.Printf("Error getting key for new secret: %v\n", err)
				return
			}

			if oldKey != newKey {
				fmt.Printf("Secret key has been changed from %s to %s\n", oldKey, newKey)
			}

			if oldSecret.ResourceVersion != newSecret.ResourceVersion {
				onUpdate(oldSecret, newSecret)
				fmt.Printf("Secret %s has been updated. ResourceVersion: %s\n", newSecret.Name, newSecret.ResourceVersion)
			}
		},
	})
}

// PersistentVolumeClaimInformer PersistentVolumeClaim Informer
func PersistentVolumeClaimInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[corev1.PersistentVolumeClaim](factory.Core().V1().PersistentVolumeClaims().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPVC := oldObj.(*corev1.PersistentVolumeClaim)
			newPVC := newObj.(*corev1.PersistentVolumeClaim)

			if oldPVC.Status.Phase != newPVC.Status.Phase {
				fmt.Printf("PVC %s has been updated. Phase: %s\n", newPVC.Name, newPVC.Status.Phase)
			}
		},
	})
}
