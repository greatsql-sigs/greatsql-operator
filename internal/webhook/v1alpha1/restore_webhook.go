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
var restorelog = logf.Log.WithName("restore-resource")

// SetupRestoreWebhookWithManager registers the webhook for Restore in the manager.
func SetupRestoreWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &databasev1alpha1.Restore{}).
		WithValidator(&RestoreCustomValidator{}).
		WithDefaulter(&RestoreCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-database-greatsql-cn-v1alpha1-restore,mutating=true,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=restores,verbs=create;update,versions=v1alpha1,name=mrestore-v1alpha1.kb.io,admissionReviewVersions=v1

// RestoreCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Restore when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type RestoreCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ admission.Defaulter[*databasev1alpha1.Restore] = &RestoreCustomDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the Kind Restore.
func (d *RestoreCustomDefaulter) Default(ctx context.Context, restore *databasev1alpha1.Restore) error {
	restorelog.Info("Defaulting for Restore", "name", restore.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-database-greatsql-cn-v1alpha1-restore,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.greatsql.cn,resources=restores,verbs=create;update,versions=v1alpha1,name=vrestore-v1alpha1.kb.io,admissionReviewVersions=v1

// RestoreCustomValidator struct is responsible for validating the Restore resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type RestoreCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ admission.Validator[*databasev1alpha1.Restore] = &RestoreCustomValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type Restore.
func (v *RestoreCustomValidator) ValidateCreate(ctx context.Context, restore *databasev1alpha1.Restore) (admission.Warnings, error) {
	restorelog.Info("Validation for Restore upon creation", "name", restore.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type Restore.
func (v *RestoreCustomValidator) ValidateUpdate(ctx context.Context, _, newObj *databasev1alpha1.Restore) (admission.Warnings, error) {
	restorelog.Info("Validation for Restore upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type Restore.
func (v *RestoreCustomValidator) ValidateDelete(ctx context.Context, restore *databasev1alpha1.Restore) (admission.Warnings, error) {
	restorelog.Info("Validation for Restore upon deletion", "name", restore.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
