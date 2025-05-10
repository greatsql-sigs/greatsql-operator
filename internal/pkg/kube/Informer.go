package kube

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// InformerFactory encapsulates shared informer factories for both typed and dynamic clients
type InformerFactory struct {
	TypedFactory   informers.SharedInformerFactory
	DynamicFactory dynamicinformer.DynamicSharedInformerFactory
	StopCh         chan struct{}
}

// NewInformerFactory initializes a new InformerFactory
func NewInformerFactory(config *rest.Config, resyncPeriod time.Duration) (*InformerFactory, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &InformerFactory{
		TypedFactory:   informers.NewSharedInformerFactory(clientset, resyncPeriod),
		DynamicFactory: dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, resyncPeriod),
		StopCh:         make(chan struct{}),
	}, nil
}

// Start begins the informer factories
func (f *InformerFactory) Start() {
	f.TypedFactory.Start(f.StopCh)
	f.DynamicFactory.Start(f.StopCh)
}

// Stop stops the informer factories
func (f *InformerFactory) Stop() {
	close(f.StopCh)
}

// GetTypedInformer returns a typed informer for the specified resource
func (f *InformerFactory) GetTypedInformer(resource string) cache.SharedIndexInformer {
	switch resource {
	case "pods":
		return f.TypedFactory.Core().V1().Pods().Informer()
	case "statefulsets":
		return f.TypedFactory.Apps().V1().StatefulSets().Informer()
	case "services":
		return f.TypedFactory.Core().V1().Services().Informer()
	case "configmaps":
		return f.TypedFactory.Core().V1().ConfigMaps().Informer()
	case "secrets":
		return f.TypedFactory.Core().V1().Secrets().Informer()
	case "pv":
		return f.TypedFactory.Core().V1().PersistentVolumes().Informer()
	case "pvc":
		return f.TypedFactory.Core().V1().PersistentVolumeClaims().Informer()
	case "cronjobs":
		return f.TypedFactory.Batch().V1beta1().CronJobs().Informer()
	case "jobs":
		return f.TypedFactory.Batch().V1().Jobs().Informer()
	default:
		return nil
	}
}

// GetDynamicInformer returns a dynamic informer for the specified GroupVersionResource
func (f *InformerFactory) GetDynamicInformer(gvr schema.GroupVersionResource) cache.SharedIndexInformer {
	return f.DynamicFactory.ForResource(gvr).Informer()
}
