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

package v1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var groupreplicationclusterlog = logf.Log.WithName("groupreplicationcluster-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *GroupReplicationCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-greatsql-greatsql-cn-v1-groupreplicationcluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=greatsql.greatsql.cn,resources=groupreplicationclusters,verbs=create;update,versions=v1,name=mgroupreplicationcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &GroupReplicationCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *GroupReplicationCluster) Default() {
	groupreplicationclusterlog.Info("default", "name", r.Name)

	// Set default values for GroupReplicationCluster
	if r.Spec.ClusterSpec.PodSpec.Containers[0].Image == "" {
		r.Spec.ClusterSpec.PodSpec.Containers[0].Image = "greatsql:latest"
	}

	// Ensure we have at least 3 members for high availability
	if len(r.Spec.Member) > 0 {
		for i := range r.Spec.Member {
			if r.Spec.Member[i].Size == nil {
				defaultSize := int32(3)
				r.Spec.Member[i].Size = &defaultSize
			}
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-greatsql-greatsql-cn-v1-groupreplicationcluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=greatsql.greatsql.cn,resources=groupreplicationclusters,verbs=create;update,versions=v1,name=vgroupreplicationcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &GroupReplicationCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *GroupReplicationCluster) ValidateCreate() (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate create", "name", r.Name)

	if err := r.validateCommon(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *GroupReplicationCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate update", "name", r.Name)

	if err := r.validateCommon(); err != nil {
		return nil, err
	}

	// Add specific update validation if needed
	oldGRC := old.(*GroupReplicationCluster)
	if len(r.Spec.Member) < len(oldGRC.Spec.Member) {
		return nil, errors.New("reducing member count is not allowed")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *GroupReplicationCluster) ValidateDelete() (admission.Warnings, error) {
	groupreplicationclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

// validateCommon implements common validation logic
func (r *GroupReplicationCluster) validateCommon() error {
	// Validate member configuration
	if len(r.Spec.Member) == 0 {
		return errors.New("at least one member configuration is required")
	}

	// Validate member size
	for i, member := range r.Spec.Member {
		if member.Size != nil && *member.Size < 3 {
			return errors.New("member size must be at least 3 for high availability")
		}
		if member.Role == "" {
			return errors.New("member name cannot be empty")
		}
		// Check for duplicate member names
		for j := i + 1; j < len(r.Spec.Member); j++ {
			if member.Role == r.Spec.Member[j].Role {
				return errors.New("duplicate member names are not allowed")
			}
		}
	}

	if err := r.validateArbitratorSize(); err != nil {
		return err
	}

	return nil
}

func (r *GroupReplicationCluster) validateArbitratorSize() error {
	arbitratorCount := 0
	for _, member := range r.Spec.Member {
		if member.Role == ArbitratorRole {
			if member.Size != nil {
				arbitratorCount += int(*member.Size)
			}
		}
	}
	if arbitratorCount > 1 {
		return errors.New("only one arbitrator is allowed")
	}
	return nil
}
