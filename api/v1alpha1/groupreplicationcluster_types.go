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

// MemberRole 成员角色
type MemberRole string

const (
	PrimaryRole    MemberRole = "primary"
	SecondaryRole  MemberRole = "secondary"
	ArbitratorRole MemberRole = "arbitrator"
)

// ClusterType 集群类型
type ClusterType string

const (
	ClusterTypeSingle   ClusterType = "single"
	ClusterTypeMultiple ClusterType = "multiple"
)

type Member struct {
	Role MemberRole `json:"role,omitempty"`
	Size *int32     `json:"size,omitempty"`
}

// GroupReplicationClusterSpec defines the desired state of GroupReplicationCluster
type GroupReplicationClusterSpec struct {
	ClusterType    ClusterType `json:"clusterType,omitempty"`
	Member         []Member    `json:"member,omitempty"`
	*Pod           `json:",inline"`
	Upgrade        Upgrade         `json:"upgrade,omitempty"`
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`
	Service        *Service        `json:"service,omitempty"`
}

func (m *Member) GetSize() int32 {
	count := int32(0)
	if m.Size != nil {
		count++
	}
	return count
}

// GroupReplicationClusterStatus defines the observed state of GroupReplicationCluster
type GroupReplicationClusterStatus struct {
	Status Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GroupReplicationCluster is the Schema for the GroupReplicationClusters API
type GroupReplicationCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupReplicationClusterSpec   `json:"spec,omitempty"`
	Status GroupReplicationClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GroupReplicationClusterList contains a list of GroupReplicationCluster
type GroupReplicationClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupReplicationCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GroupReplicationCluster{}, &GroupReplicationClusterList{})
}
