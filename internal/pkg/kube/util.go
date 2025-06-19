package kube

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodByLabels 根据标签获取 Pod
func GetPodByLabels(cli client.Reader, labelSelector string) ([]corev1.Pod, error) {
	sel, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}
	opts := &client.ListOptions{
		LabelSelector: sel,
		// Namespace:     namespace,
	}
	err = cli.List(context.Background(), podList, opts)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// Retry 重试函数
func Retry(step func() error, retries int, delay time.Duration) error {
	for range retries {
		if err := step(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("step failed after %d retries", retries)
}

// UpdateStatusWithRetry 更新对象状态，使用重试机制
func UpdateStatusWithRetry(ctx context.Context, c client.Client, obj client.Object, newStatus any) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// 获取最新对象
		latest := obj.DeepCopyObject().(client.Object)
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}, latest); err != nil {
			return fmt.Errorf("get latest object: %w", err)
		}

		// 反射设置 status 字段
		// accessor := meta.NewAccessor()
		accessorInterface, ok := latest.(runtime.Object)
		if !ok {
			return fmt.Errorf("object does not implement runtime.Object")
		}
		if err := setStatus(accessorInterface, newStatus); err != nil {
			return fmt.Errorf("set status: %w", err)
		}

		// 调用 status 更新
		return c.Status().Update(ctx, accessorInterface.(client.Object))
	})
}

// setStatus 使用反射设置 Status 字段
func setStatus(obj runtime.Object, status any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("obj must be a non-nil pointer")
	}
	elem := v.Elem()
	field := elem.FieldByName("Status")
	if !field.IsValid() || !field.CanSet() {
		return fmt.Errorf("cannot set Status field")
	}

	statusVal := reflect.ValueOf(status)
	if statusVal.Type() != field.Type() {
		return fmt.Errorf("status type mismatch: expected %s, got %s", field.Type(), statusVal.Type())
	}

	field.Set(statusVal)
	return nil
}
