/*
Copyright 2024 greatsql.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	databasev1alpha1 "github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
)

// nolint:unused
var standalonelog = logf.Log.WithName("standalone-resource")

// SetupStandaloneWebhookWithManager 向 manager 注册 Standalone 的 webhook。
func SetupStandaloneWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &databasev1alpha1.Standalone{}).
		WithValidator(&StandaloneCustomValidator{}).
		WithDefaulter(&StandaloneCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-database-greatsql-cn-v1alpha1-standalone,mutating=true,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=standalones,verbs=create;update,versions=v1alpha1,name=mstandalone-v1alpha1.kb.io,admissionReviewVersions=v1

// StandaloneCustomDefaulter 为 Standalone 设置默认值。
type StandaloneCustomDefaulter struct{}

var _ admission.Defaulter[*databasev1alpha1.Standalone] = &StandaloneCustomDefaulter{}

// Default 在 create/update 时为未填字段设置默认值。
func (d *StandaloneCustomDefaulter) Default(ctx context.Context, c *databasev1alpha1.Standalone) error {
	standalonelog.Info("defaulting Standalone", "name", c.GetName())

	if c.Spec.Size == nil {
		one := int32(1)
		c.Spec.Size = &one
	}
	if *c.Spec.Size <= 0 {
		one := int32(1)
		c.Spec.Size = &one
	}
	if c.Spec.Pod == nil {
		c.Spec.Pod = &databasev1alpha1.Pod{}
	}
	DefaultPodSpec(c.Spec.Pod)
	DefaultUpdateStrategy(&c.Spec.UpdateStrategy)
	DefaultService(&c.Spec.Service)
	return nil
}

// +kubebuilder:webhook:path=/validate-database-greatsql-cn-v1alpha1-standalone,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=standalones,verbs=create;update,versions=v1alpha1,name=vstandalone-v1alpha1.kb.io,admissionReviewVersions=v1

// StandaloneCustomValidator 校验 Standalone 的 create/update。
type StandaloneCustomValidator struct{}

var _ admission.Validator[*databasev1alpha1.Standalone] = &StandaloneCustomValidator{}

// validateStandaloneSpec 校验 spec：size 为正，容器 name/image 必填。
func validateStandaloneSpec(spec *databasev1alpha1.StandaloneSpec) error {
	if spec.Size == nil || *spec.Size <= 0 {
		return fmt.Errorf("spec.size must be positive, got %v", spec.Size)
	}
	if spec.Pod == nil {
		return fmt.Errorf("spec.pod/container configuration is required (container.name and container.image must be set)")
	}
	container := spec.Pod.Container
	if container.Name == "" {
		return fmt.Errorf("spec.container.name is required")
	}
	if container.Image == "" {
		return fmt.Errorf("spec.container.image is required")
	}
	return nil
}

// ValidateCreate 在创建时校验 spec。
func (v *StandaloneCustomValidator) ValidateCreate(ctx context.Context, c *databasev1alpha1.Standalone) (admission.Warnings, error) {
	standalonelog.Info("validate create", "name", c.GetName())
	return nil, validateStandaloneSpec(&c.Spec)
}

// ValidateUpdate 在更新时校验 spec。
func (v *StandaloneCustomValidator) ValidateUpdate(ctx context.Context, _ *databasev1alpha1.Standalone, newObj *databasev1alpha1.Standalone) (admission.Warnings, error) {
	standalonelog.Info("validate update", "name", newObj.GetName())
	return nil, validateStandaloneSpec(&newObj.Spec)
}

// ValidateDelete 删除时不拒绝，仅打日志。
func (v *StandaloneCustomValidator) ValidateDelete(ctx context.Context, c *databasev1alpha1.Standalone) (admission.Warnings, error) {
	standalonelog.Info("validate delete", "name", c.GetName())
	return nil, nil
}
