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
	corev1 "k8s.io/api/core/v1"
)

// BackupPhase is the phase of a SchedulingBackup.
type BackupPhase string

const (
	BackupPhasePending   BackupPhase = "Pending"
	BackupPhaseRunning   BackupPhase = "Running"
	BackupPhaseCompleted BackupPhase = "Completed"
	BackupPhaseFailed    BackupPhase = "Failed"
)

// ClusterRef references a GroupReplicationCluster or Standalone to backup/restore.
type ClusterRef struct {
	// APIVersion of the target resource (e.g. database.greatsql.cn/v1alpha1).
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind of the target resource (GroupReplicationCluster or Standalone).
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name of the target resource.
	Name string `json:"name"`

	// Namespace of the target resource. Defaults to the same namespace as the Backup/Restore CR.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// BackupStoragePVC defines PVC storage for backup.
type BackupStoragePVC struct {
	// StorageClassName for the backup PVC.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Size of the backup PVC (e.g. "10Gi").
	// +optional
	Size *string `json:"size,omitempty"`

	// ExistingClaim name to use instead of creating a new PVC.
	// +optional
	ExistingClaim *string `json:"existingClaim,omitempty"`
}

// BackupStorageS3 defines S3-compatible storage for backup.
type BackupStorageS3 struct {
	// Bucket name.
	Bucket string `json:"bucket"`

	// Region (required for AWS and S3-compatible storages).
	Region string `json:"region"`

	// Endpoint URL for S3-compatible storage (e.g. MinIO). Omit for AWS.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// CredentialsSecret is a secret containing AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.
	CredentialsSecret string `json:"credentialsSecret"`

	// VerifyTLS enables TLS verification for the storage endpoint.
	// +optional
	VerifyTLS *bool `json:"verifyTLS,omitempty"`
}

// BackupStorage defines where to store the backup (PVC or S3).
type BackupStorage struct {
	// PVC storage. Exactly one of PVC or S3 should be set.
	// +optional
	PVC *BackupStoragePVC `json:"pvc,omitempty"`

	// S3-compatible storage. Exactly one of PVC or S3 should be set.
	// +optional
	S3 *BackupStorageS3 `json:"s3,omitempty"`
}

// ContainerOptions for xtrabackup/xbcloud/xbstream (env and args).
type ContainerOptions struct {
	// Env for the backup/restore container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// XtrabackupArgs extra arguments for xtrabackup.
	// +optional
	XtrabackupArgs []string `json:"xtrabackupArgs,omitempty"`

	// XbcloudArgs extra arguments for xbcloud (S3).
	// +optional
	XbcloudArgs []string `json:"xbcloudArgs,omitempty"`

	// XbstreamArgs extra arguments for xbstream.
	// +optional
	XbstreamArgs []string `json:"xbstreamArgs,omitempty"`
}
