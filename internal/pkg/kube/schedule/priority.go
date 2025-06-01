package schedule

import (
	"context"
	"fmt"

	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PriorityClassConfig 定义创建所需配置
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
