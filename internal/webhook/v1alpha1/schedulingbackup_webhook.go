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

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	databasev1alpha1 "github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var schedulingbackuplog = logf.Log.WithName("schedulingbackup-resource")

// SetupSchedulingBackupWebhookWithManager registers the webhook for SchedulingBackup in the manager.
func SetupSchedulingBackupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &databasev1alpha1.SchedulingBackup{}).
		WithValidator(&SchedulingBackupCustomValidator{}).
		WithDefaulter(&SchedulingBackupCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-database-greatsql-cn-v1alpha1-schedulingbackup,mutating=true,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=schedulingbackups,verbs=create;update,versions=v1alpha1,name=mschedulingbackup-v1alpha1.kb.io,admissionReviewVersions=v1

// SchedulingBackupCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind SchedulingBackup when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type SchedulingBackupCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ admission.Defaulter[*databasev1alpha1.SchedulingBackup] = &SchedulingBackupCustomDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the Kind SchedulingBackup.
func (d *SchedulingBackupCustomDefaulter) Default(ctx context.Context, schedulingbackup *databasev1alpha1.SchedulingBackup) error {
	schedulingbackuplog.Info("Defaulting for SchedulingBackup", "name", schedulingbackup.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-database-greatsql-cn-v1alpha1-schedulingbackup,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=schedulingbackups,verbs=create;update,versions=v1alpha1,name=vschedulingbackup-v1alpha1.kb.io,admissionReviewVersions=v1

// SchedulingBackupCustomValidator struct is responsible for validating the SchedulingBackup resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SchedulingBackupCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ admission.Validator[*databasev1alpha1.SchedulingBackup] = &SchedulingBackupCustomValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type SchedulingBackup.
func (v *SchedulingBackupCustomValidator) ValidateCreate(ctx context.Context, schedulingbackup *databasev1alpha1.SchedulingBackup) (admission.Warnings, error) {
	schedulingbackuplog.Info("Validation for SchedulingBackup upon creation", "name", schedulingbackup.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type SchedulingBackup.
func (v *SchedulingBackupCustomValidator) ValidateUpdate(ctx context.Context, _, newObj *databasev1alpha1.SchedulingBackup) (admission.Warnings, error) {
	schedulingbackuplog.Info("Validation for SchedulingBackup upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type SchedulingBackup.
func (v *SchedulingBackupCustomValidator) ValidateDelete(ctx context.Context, schedulingbackup *databasev1alpha1.SchedulingBackup) (admission.Warnings, error) {
	schedulingbackuplog.Info("Validation for SchedulingBackup upon deletion", "name", schedulingbackup.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
