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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SchedulingBackupSpec defines the desired state of SchedulingBackup (XtraBackup physical backup).
type SchedulingBackupSpec struct {
	// ClusterRef is the cluster to back up (GroupReplicationCluster or Standalone).
	ClusterRef ClusterRef `json:"clusterRef"`

	// SourcePod is the specific pod to backup from (e.g. greatsql-mgr-0). If empty, controller chooses primary or first pod.
	// +optional
	SourcePod *string `json:"sourcePod,omitempty"`

	// Storage is where to store the backup (PVC or S3).
	Storage BackupStorage `json:"storage"`

	// ContainerOptions override env/args for xtrabackup tools.
	// +optional
	ContainerOptions *ContainerOptions `json:"containerOptions,omitempty"`
}

// SchedulingBackupStatus defines the observed state of SchedulingBackup.
type SchedulingBackupStatus struct {
	// Phase of the backup (Pending, Running, Completed, Failed).
	// +optional
	Phase BackupPhase `json:"phase,omitempty"`

	// Message human-readable message.
	// +optional
	Message string `json:"message,omitempty"`

	// Reason short reason for current phase.
	// +optional
	Reason string `json:"reason,omitempty"`

	// StartedAt when the backup job started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt when the backup job completed.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// JobName is the name of the backup Job.
	// +optional
	JobName string `json:"jobName,omitempty"`

	// BackupPath is the path or S3 prefix where backup was stored (for Restore to reference).
	// +optional
	BackupPath string `json:"backupPath,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=sb
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Backup phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.message",description="Status message"

// SchedulingBackup is the Schema for the schedulingbackups API (XtraBackup physical backup).
type SchedulingBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulingBackupSpec   `json:"spec,omitempty"`
	Status SchedulingBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SchedulingBackupList contains a list of SchedulingBackup.
type SchedulingBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SchedulingBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SchedulingBackup{}, &SchedulingBackupList{})
}
