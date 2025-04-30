package kube

import (
	"github.com/pkg/errors"

	"github.com/go-logr/logr"
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
	Log logr.Logger
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

// PodInformer Pod Informer
func (e *EventHandlerFunc) PodInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[corev1.Pod](factory.Core().V1().Pods().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			if oldPod.Status.ContainerStatuses[0].RestartCount != newPod.Status.ContainerStatuses[0].RestartCount {
				e.Log.Info("Pod %s has been restarted. Restart count: %d", newPod.Name, newPod.Status.ContainerStatuses[0].RestartCount)
			}
		},
	})
}

// DeploymentInformer Deployment Informer
func (e *EventHandlerFunc) DeploymentInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[appsv1.Deployment](factory.Apps().V1().Deployments().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			handleReplicaChange[appsv1.Deployment](oldObj, newObj, "Deployment")
		},
	})
}

// StatefulSetInformer StatefulSet Informer
func (e *EventHandlerFunc) StatefulSetInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[appsv1.StatefulSet](factory.Apps().V1().StatefulSets().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			handleReplicaChange[appsv1.StatefulSet](oldObj, newObj, "StatefulSet")
		},
	})
}

// SecretInformer Secret Informer
func (e *EventHandlerFunc) SecretInformer(factory informers.SharedInformerFactory, onUpdate func(oldSecret, newSecret *corev1.Secret)) (cache.SharedIndexInformer, error) {
	return informerUpdate[corev1.Secret](factory.Core().V1().Secrets().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSecret := oldObj.(*corev1.Secret)
			newSecret := newObj.(*corev1.Secret)

			oldKey, err := cache.MetaNamespaceKeyFunc(oldSecret)
			if err != nil {
				errors.Wrapf(err, "error getting key for old secret")
				return
			}
			newKey, err := cache.MetaNamespaceKeyFunc(newSecret)
			if err != nil {
				errors.Wrapf(err, "error getting key for new secret")
				return
			}

			if oldKey != newKey {
				e.Log.Info("Secret key has been changed from %s to %s", oldKey, newKey)
			}

			if oldSecret.ResourceVersion != newSecret.ResourceVersion {
				onUpdate(oldSecret, newSecret)
				e.Log.Info("Secret %s has been updated. ResourceVersion: %s", newSecret.Name, newSecret.ResourceVersion)
			}
		},
	}), nil
}

// PersistentVolumeClaimInformer PersistentVolumeClaim Informer
func (e *EventHandlerFunc) PersistentVolumeClaimInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
	return informerUpdate[corev1.PersistentVolumeClaim](factory.Core().V1().PersistentVolumeClaims().Informer, EventHandlerFunc{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPVC := oldObj.(*corev1.PersistentVolumeClaim)
			newPVC := newObj.(*corev1.PersistentVolumeClaim)

			if oldPVC.Status.Phase != newPVC.Status.Phase {
				e.Log.Info("PVC %s has been updated. Phase: %s", newPVC.Name, newPVC.Status.Phase)
			}
		},
	})
}

// StandaloneInformer Standalone Informer
// Custom Resource Definition Informer
// func StandaloneInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
// 	return informerUpdate[v1.Standalone](factory.Core().V1().Pods().Informer, EventHandlerFunc{
// 		UpdateFunc: func(oldObj, newObj interface{}) {
// 			oldPod := oldObj.(*v1.Standalone)
// 			newPod := newObj.(*v1.Standalone)
// 			if oldPod.Status.Replicas != newPod.Status.Replicas {
// 				fmt.Printf("Instance %s has been restarted. Restart count: %d\n", newPod.Name, newPod.Status.Replicas)
// 			}
// 		},
// 	})
// }

// GroupReplicationClusterInformer GroupReplicationCluster Informer
// Custom Resource Definition Informer
// func GroupReplicationClusterInformer(factory informers.SharedInformerFactory) cache.SharedIndexInformer {
// 	return informerUpdate[v1.GroupReplicationCluster](factory.Core().V1().Pods().Informer, EventHandlerFunc{
// 		UpdateFunc: func(oldObj, newObj interface{}) {
// 			oldPod := oldObj.(*v1.GroupReplicationCluster)
// 			newPod := newObj.(*v1.GroupReplicationCluster)
// 			if oldPod.Status.Replicas != newPod.Status.Replicas {
// 				fmt.Printf("Cluster instance %s has been restarted. Restart count: %d\n", newPod.Name, newPod.Status.Replicas)
// 			}
// 		},
// 	})
// }
