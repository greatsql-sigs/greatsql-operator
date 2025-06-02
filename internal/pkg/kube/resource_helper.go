package kube

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ResourceHelper struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Recorder        events.FakeRecorder
	Log             logr.Logger
	Mapper          meta.RESTMapper
	InformerFactory *InformerFactory
}

func NewResourceHelper(mgr manager.Manager, log logr.Logger) *ResourceHelper {
	return &ResourceHelper{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    log.WithName("ResourceHelper"),
		Mapper: mgr.GetRESTMapper(),
	}
}

// CreateOrUpdateWithOwner 创建或更新资源，并设置 ownerReference
func (r *ResourceHelper) CreateOrUpdateWithOwner(ctx context.Context, owner client.Object, obj client.Object) error {
	log := r.Log.WithValues("namespace", obj.GetNamespace(), "name", obj.GetName())

	if err := controllerutil.SetControllerReference(owner, obj, r.Scheme); err != nil {
		log.Error(err, "failed to set controller reference")
		return err
	}

	gvk, err := r.getGVK(obj)
	if err != nil {
		log.Error(err, "failed to get GVK")
		return err
	}
	log = log.WithValues("gvk", gvk.String())

	key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	existing := obj.DeepCopyObject().(client.Object)

	err = r.Client.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating new resource")
			if err := r.Client.Create(ctx, obj); err != nil {
				log.Error(err, "failed to create resource")
				return err
			}
			return nil
		}
		log.Error(err, "failed to get resource")
		return err
	}

	// 特殊处理 PVC
	if gvk.Kind == "PersistentVolumeClaim" {
		newPVC := obj.(*corev1.PersistentVolumeClaim)
		existingPVC := existing.(*corev1.PersistentVolumeClaim)

		if newPVC.Spec.Resources.Requests != nil {
			existingPVC.Spec.Resources.Requests = newPVC.Spec.Resources.Requests
		}
		existingPVC.SetLabels(newPVC.GetLabels())
		existingPVC.SetAnnotations(newPVC.GetAnnotations())
		existingPVC.SetOwnerReferences(newPVC.GetOwnerReferences())

		log.Info("Updating existing PVC")
		return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return r.Client.Update(ctx, existingPVC)
		})
	}

	// 默认资源更新，冲突重试
	log.Info("Updating existing resource")
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latest := obj.DeepCopyObject().(client.Object)
		if err := r.Client.Get(ctx, key, latest); err != nil {
			return err
		}
		latest.SetLabels(obj.GetLabels())
		latest.SetAnnotations(obj.GetAnnotations())
		latest.SetOwnerReferences(obj.GetOwnerReferences())
		return r.Client.Update(ctx, latest)
	})
}

// DeleteResource 删除资源
func (r *ResourceHelper) DeleteResource(ctx context.Context, namespace string, name string, obj client.Object) error {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := r.Client.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.Client.Delete(ctx, obj)
}

// DeleteResourceWithFinalizer 删除资源，并设置 finalizer
func (r *ResourceHelper) DeleteResourceWithFinalizer(ctx context.Context, namespace string, name string, obj client.Object, finalizer string) error {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := r.Client.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	controllerutil.RemoveFinalizer(obj, finalizer)
	return r.Client.Update(ctx, obj)
}

// DeleteResourceWithFinalizerAndOwner 删除资源，并设置 finalizer，并设置 ownerReference
func (r *ResourceHelper) DeleteResourceWithFinalizerAndOwner(ctx context.Context, owner client.Object, namespace string, name string, obj client.Object, finalizer string) error {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := r.Client.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	controllerutil.RemoveFinalizer(obj, finalizer)
	return r.Client.Update(ctx, obj)
}

// ListResource 列出资源
// 采用informer 方式列出资源，防止因频繁查询资源导致性能问题
func (r *ResourceHelper) ListResource(ctx context.Context, namespace string, obj client.Object) ([]client.Object, error) {
	gvk, err := r.getGVK(obj)
	if err != nil {
		return nil, err
	}
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: pluralizeResourceName(gvk.Kind), // 实现 pluralizeResourceName 函数以获取资源的复数形式
	}

	informer := r.InformerFactory.GetDynamicInformer(gvr)
	if informer == nil {
		return nil, fmt.Errorf("no informer found for GVR: %v", gvr)
	}

	indexer := informer.GetIndexer()
	var objs []interface{}

	if namespace == "" {
		objs = indexer.List()
	} else {
		objs, err = indexer.ByIndex(cache.NamespaceIndex, namespace)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %v", err)
	}

	// 类型断言并转换为 client.Object 切片
	result := make([]client.Object, 0, len(objs))
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		if cObj, ok := obj.(client.Object); ok {
			result = append(result, cObj)
		}
	}

	return result, nil
}

// ResourceExists 检查资源是否存在
func (r *ResourceHelper) ResourceExists(ctx context.Context, namespace string, name string, obj client.Object) (bool, error) {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := r.Client.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// pluralizeResourceName 将资源的 Kind 转换为复数形式的资源名
func pluralizeResourceName(kind string) string {
	// 特殊规则映射
	specialCases := map[string]string{
		"ConfigMap":             "configmaps",
		"Service":               "services",
		"StatefulSet":           "statefulsets",
		"Deployment":            "deployments",
		"DaemonSet":             "daemonsets",
		"ReplicaSet":            "replicasets",
		"Pod":                   "pods",
		"Namespace":             "namespaces",
		"PersistentVolumeClaim": "persistentvolumeclaims",
		"PersistentVolume":      "persistentvolumes",
		"Secret":                "secrets",
	}

	// 检查特殊规则
	if plural, ok := specialCases[kind]; ok {
		return plural
	}

	// 通用规则
	lastChar := kind[len(kind)-1]
	switch lastChar {
	case 'y':
		return strings.ToLower(kind[:len(kind)-1]) + "ies"
	case 's', 'x', 'z', 'h':
		return strings.ToLower(kind) + "es"
	default:
		return strings.ToLower(kind) + "s"
	}
}

// getGVK 返回对象的 GVK
func (r *ResourceHelper) getGVK(obj client.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := r.Scheme.ObjectKinds(obj)
	if err != nil || len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("failed to get GVK: %w", err)
	}
	return gvks[0], nil
}
