package kube

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-10-12 10:50:48
 * @file: resource.go
 * @description: resource operation
 */

type ResourceOperation struct {
	Log    logr.Logger
	Client client.Client
}

func NewResourceOperation(log logr.Logger, cli client.Client) *ResourceOperation {
	return &ResourceOperation{
		Log:    log,
		Client: cli,
	}
}

func (r *ResourceOperation) CreateResource(ctx context.Context, obj client.Object, name, namespace, kind string) error {
	err := r.Client.Create(ctx, obj)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			r.Log.Info("Resource already exists, skip creation or update it.", "kind", kind, "name", name, "namespace", namespace)
			// 如果需要更新，可以在此处执行 Update
			return nil
		}
		r.Log.Error(err, "Could not create resource", "kind", kind, "name", name)
		return fmt.Errorf("could not create %s %s/%s: %w", kind, namespace, name, err)
	}
	r.Log.Info("Create resource successful", "kind", kind, "name", name, "namespace", namespace)
	return nil
}

func (r *ResourceOperation) UpdateResource(ctx context.Context, resource client.Object, name, namespace, kind string) error {
	if err := r.Client.Update(ctx, resource); err != nil {
		return fmt.Errorf("could not update Name: %s namespace: %s", name, namespace)
	}
	return nil
}

func (r *ResourceOperation) DeleteReource(ctx context.Context, obj client.Object) error {
	if err := r.Client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("could not delete Name: %s namespace: %s", obj.GetName(), obj.GetNamespace())
	}
	r.Log.Info("Delete resource successful",
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace())
	return nil
}

func (r *ResourceOperation) PatchResource(ctx context.Context, obj client.Object, patch client.Patch) error {

	if err := r.Client.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("could not patch Name: %s namespace: %s", obj.GetName(), obj.GetNamespace())
	}
	r.Log.Info("Patch %s is successful, Name %s, Namespace %s", obj.GetObjectKind(), obj.GetName(), obj.GetNamespace())
	return nil

}

func (r *ResourceOperation) ListResource(ctx context.Context, objList client.ObjectList) error {

	if err := r.Client.List(ctx, objList); err != nil {
		return fmt.Errorf("could not list resources")
	}
	return nil
}

func (r *ResourceOperation) GetResource(ctx context.Context, cli client.Client, req ctrl.Request, obj client.Object) error {

	if err := cli.Get(ctx, types.NamespacedName{Namespace: req.Name, Name: req.Namespace}, obj); err != nil {
		if client.IgnoreNotFound(err) != nil {
			r.Log.Info("Object resource %s not found. Ignoring since object must be deleted", req.Name)
			return nil
		}
		return client.IgnoreNotFound(err)
	}

	return nil
}

func (r *ResourceOperation) UpdateResourceStatus(ctx context.Context, obj client.Object) error {

	if err := r.Client.Status().Update(ctx, obj); err != nil {
		return fmt.Errorf("could not update status Name: %s namespace: %s", obj.GetName(), obj.GetNamespace())
	}
	r.Log.Info("Update status %s is successful, Name %s, Namespace %s", obj.GetObjectKind(), obj.GetName(), obj.GetNamespace())
	return nil
}

func (r *ResourceOperation) ResourceExists(ctx context.Context, name, namespace string, obj client.Object) (bool, error) {
	err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
