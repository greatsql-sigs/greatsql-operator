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
// log is for logging in this package.
var groupreplicationclusterlog = logf.Log.WithName("groupreplicationcluster-resource")

// SetupGroupReplicationClusterWebhookWithManager registers the webhook for GroupReplicationCluster in the manager.
func SetupGroupReplicationClusterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &databasev1alpha1.GroupReplicationCluster{}).
		WithValidator(&GroupReplicationClusterCustomValidator{}).
		WithDefaulter(&GroupReplicationClusterCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-database-greatsql-cn-v1alpha1-groupreplicationcluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=groupreplicationclusters,verbs=create;update,versions=v1alpha1,name=mgroupreplicationcluster-v1alpha1.kb.io,admissionReviewVersions=v1

// GroupReplicationClusterCustomDefaulter 为 GroupReplicationCluster 设置默认值。
type GroupReplicationClusterCustomDefaulter struct{}

var _ admission.Defaulter[*databasev1alpha1.GroupReplicationCluster] = &GroupReplicationClusterCustomDefaulter{}

// Default 在 create/update 时为未填字段设置默认值。
func (d *GroupReplicationClusterCustomDefaulter) Default(ctx context.Context, c *databasev1alpha1.GroupReplicationCluster) error {
	groupreplicationclusterlog.Info("defaulting GroupReplicationCluster", "name", c.GetName())

	spec := &c.Spec
	if spec.Mode == "" {
		spec.Mode = databasev1alpha1.ClusterModeSingle
	}
	for i := range spec.Member {
		m := &spec.Member[i]
		if m.Size == nil {
			one := int32(1)
			m.Size = &one
		}
		if *m.Size <= 0 {
			one := int32(1)
			m.Size = &one
		}
	}
	if spec.Pod == nil {
		spec.Pod = &databasev1alpha1.Pod{}
	}
	DefaultPodSpec(spec.Pod)
	if spec.UpdateStrategy == nil {
		spec.UpdateStrategy = &databasev1alpha1.UpdateStrategy{}
	}
	DefaultUpdateStrategy(spec.UpdateStrategy)
	if spec.Service == nil {
		spec.Service = &databasev1alpha1.Service{}
	}
	DefaultService(spec.Service)
	return nil
}

// +kubebuilder:webhook:path=/validate-database-greatsql-cn-v1alpha1-groupreplicationcluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=groupreplicationclusters,verbs=create;update,versions=v1alpha1,name=vgroupreplicationcluster-v1alpha1.kb.io,admissionReviewVersions=v1

// GroupReplicationClusterCustomValidator 校验 GroupReplicationCluster 的 create/update。
type GroupReplicationClusterCustomValidator struct{}

var _ admission.Validator[*databasev1alpha1.GroupReplicationCluster] = &GroupReplicationClusterCustomValidator{}

// validateGroupReplicationClusterSpec 校验 spec：成员非空、角色合法、单主/多主约束。
// 注意：spec.Pod/container（name、image 等）不在此处校验，由 controller 在运行时使用默认镜像等逻辑处理。
func validateGroupReplicationClusterSpec(spec *databasev1alpha1.GroupReplicationClusterSpec) error {
	if len(spec.Member) == 0 {
		return fmt.Errorf("spec.member must have at least one member")
	}
	validRoles := map[databasev1alpha1.MemberRole]struct{}{
		databasev1alpha1.PrimaryRole:    {},
		databasev1alpha1.SecondaryRole:  {},
		databasev1alpha1.ArbitratorRole: {},
	}
	for i, m := range spec.Member {
		if m.Role == "" {
			return fmt.Errorf("spec.member[%d].role is required", i)
		}
		if _, ok := validRoles[m.Role]; !ok {
			return fmt.Errorf("spec.member[%d].role must be one of primary, secondary, arbitrator, got %q", i, m.Role)
		}
		if m.Size != nil && *m.Size <= 0 {
			return fmt.Errorf("spec.member[%d].size must be positive, got %d", i, *m.Size)
		}
	}
	// 按角色统计：单主/多主均要求 primary 数量按 Role 校验
	primaryCount := len(spec.GetPrimaryMembers())
	if spec.IsSingleMode() {
		// 单主模式：必须恰好 1 个 Role=primary 的成员（不依赖 GetPrimaryMembers 对单主“取第一个”的语义）
		var byRole int
		for _, m := range spec.Member {
			if m.Role == databasev1alpha1.PrimaryRole {
				byRole++
			}
		}
		if byRole != 1 {
			return fmt.Errorf("single mode cluster must have exactly 1 primary member")
		}
		if len(spec.GetArbitratorMembers()) > 1 {
			return fmt.Errorf("single mode cluster must have at most 1 arbitrator member")
		}
	} else if spec.IsMultipleMode() {
		if spec.GetTotalMembers() < 3 {
			return fmt.Errorf("multiple mode cluster must have at least 3 members")
		}
		if primaryCount < 1 {
			return fmt.Errorf("multiple mode cluster must have at least 1 primary member")
		}
		if len(spec.GetArbitratorMembers()) > 1 {
			return fmt.Errorf("multiple mode cluster must have at most 1 arbitrator member")
		}
	} else {
		return fmt.Errorf("invalid cluster mode: %s (must be single or multiple)", spec.Mode)
	}
	return nil
}

// ValidateCreate 在创建时校验 spec。
func (v *GroupReplicationClusterCustomValidator) ValidateCreate(ctx context.Context, c *databasev1alpha1.GroupReplicationCluster) (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate create", "name", c.GetName())
	return nil, validateGroupReplicationClusterSpec(&c.Spec)
}

// ValidateUpdate 在更新时校验 spec，并禁止修改 spec.mode。
func (v *GroupReplicationClusterCustomValidator) ValidateUpdate(ctx context.Context, oldC, newC *databasev1alpha1.GroupReplicationCluster) (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate update", "name", newC.GetName())
	if oldC.Spec.Mode != newC.Spec.Mode {
		return nil, fmt.Errorf("spec.mode is immutable (was %q)", oldC.Spec.Mode)
	}
	return nil, validateGroupReplicationClusterSpec(&newC.Spec)
}

// ValidateDelete 删除时不拒绝，仅打日志。
func (v *GroupReplicationClusterCustomValidator) ValidateDelete(ctx context.Context, c *databasev1alpha1.GroupReplicationCluster) (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate delete", "name", c.GetName())
	return nil, nil
}
