package schedule

import (
	"context"
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PriorityClassFields 定义创建所需配置
type PriorityClassFields struct {
	Name          string
	Value         int32
	GlobalDefault bool
	Description   string
}

// EnsurePriorityClass 确保 PriorityClass 存在（无则创建，有则更新）
func EnsurePriorityClass(client client.Client, field PriorityClassFields) error {
	ctx := context.Background()
	pc := &schedulingv1.PriorityClass{}
	key := types.NamespacedName{Name: field.Name}

	err := client.Get(ctx, key, pc)
	if apierrors.IsNotFound(err) {
		// 不存在，创建
		newPC := &schedulingv1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: field.Name,
			},
			Value:         field.Value,
			GlobalDefault: field.GlobalDefault,
			Description:   field.Description,
		}
		if err := client.Create(ctx, newPC); err != nil {
			return fmt.Errorf("failed to create PriorityClass: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get PriorityClass: %w", err)
	}

	// 检查是否需要更新
	needUpdate := pc.Value != field.Value ||
		pc.GlobalDefault != field.GlobalDefault ||
		pc.Description != field.Description

	if needUpdate {
		pc.Value = field.Value
		pc.GlobalDefault = field.GlobalDefault
		pc.Description = field.Description

		if err := client.Update(ctx, pc); err != nil {
			return fmt.Errorf("failed to update PriorityClass: %w", err)
		}
	}

	return nil
}

// CreatePriorityClass 创建 PriorityClass
func CreatePriorityClass(ctx context.Context, cr *v1alpha1.Pod, cli client.Client) error {
	if cr.PriorityClassName == nil {
		// PriorityClassName 未设置，无需创建，直接返回
		return nil
	}

	priorityClass := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: *cr.PriorityClassName,
		},
		Value: 1000000, // 设置一个较高的优先级值
	}

	// 检查 PriorityClass 是否已存在
	existing := &schedulingv1.PriorityClass{}
	err := cli.Get(ctx, client.ObjectKey{Name: priorityClass.Name}, existing)
	if err == nil {
		// PriorityClass 已存在，无需创建
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	// 创建 PriorityClass
	if err := cli.Create(ctx, priorityClass); err != nil {
		return fmt.Errorf("failed to create PriorityClass %s: %w", priorityClass.Name, err)
	}

	return nil
}
