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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestorePhase is the phase of a Restore.
type RestorePhase string

const (
	RestorePhasePending   RestorePhase = "Pending"
	RestorePhaseRunning   RestorePhase = "Running"
	RestorePhaseCompleted RestorePhase = "Completed"
	RestorePhaseFailed    RestorePhase = "Failed"
)

// RestoreBackupSource defines external backup source (S3 or PVC) for restore.
type RestoreBackupSource struct {
	// PVC claim name in the same namespace where the backup is stored.
	// +optional
	PVC *RestoreBackupSourcePVC `json:"pvc,omitempty"`

	// S3-compatible source.
	// +optional
	S3 *RestoreBackupSourceS3 `json:"s3,omitempty"`
}

// RestoreBackupSourcePVC is PVC source for restore.
type RestoreBackupSourcePVC struct {
	// ClaimName is the name of the PVC containing the backup.
	ClaimName string `json:"claimName"`

	// Path inside the PVC (e.g. /backup). Optional, defaults to root.
	// +optional
	Path *string `json:"path,omitempty"`
}

// RestoreBackupSourceS3 is S3 source for restore.
type RestoreBackupSourceS3 struct {
	// Bucket name.
	Bucket string `json:"bucket"`

	// Region.
	Region string `json:"region"`

	// Path/prefix inside the bucket (e.g. my-backup/2024-01-15).
	// +optional
	Path *string `json:"path,omitempty"`

	// Endpoint URL for S3-compatible storage. Omit for AWS.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// CredentialsSecret name (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY).
	CredentialsSecret string `json:"credentialsSecret"`

	// +optional
	VerifyTLS *bool `json:"verifyTLS,omitempty"`
}

// RestoreSpec defines the desired state of Restore (XtraBackup physical restore).
type RestoreSpec struct {
	// ClusterRef is the cluster to restore to (GroupReplicationCluster or Standalone).
	ClusterRef ClusterRef `json:"clusterRef"`

	// BackupRef is the name of a SchedulingBackup in the same namespace to restore from. Mutually exclusive with BackupSource.
	// +optional
	BackupRef *string `json:"backupRef,omitempty"`

	// BackupSource is the external backup location (PVC or S3). Mutually exclusive with BackupRef.
	// +optional
	BackupSource *RestoreBackupSource `json:"backupSource,omitempty"`

	// ContainerOptions override env/args for xtrabackup tools.
	// +optional
	ContainerOptions *ContainerOptions `json:"containerOptions,omitempty"`

	// Resources for the restore Job container.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RestoreStatus defines the observed state of Restore.
type RestoreStatus struct {
	// Phase of the restore (Pending, Running, Completed, Failed).
	// +optional
	Phase RestorePhase `json:"phase,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// +optional
	JobName string `json:"jobName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=gr
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Restore phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.message",description="Status message"

// Restore is the Schema for the restores API (XtraBackup physical restore).
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestoreSpec   `json:"spec,omitempty"`
	Status RestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RestoreList contains a list of Restore.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Restore{}, &RestoreList{})
}
