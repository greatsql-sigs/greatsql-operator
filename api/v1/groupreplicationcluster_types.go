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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GroupReplicationClusterSpec defines the desired state of GroupReplicationCluster
type GroupReplicationClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	Member            []Member                      `json:"member,omitempty"`
	ClusterSpec       *MySQLGroupReplicationCluster `json:"clusterSpec,omitempty"`
	ProxySpec         *Proxy                        `json:"proxy,omitempty"`
	SchedulerBuckup   *SchedulerBuckup              `json:"schedulerBuckup,omitempty"`
	MetricsCollection *MetricsCollection            `json:"metricsCollection,omitempty"`
}

type Member struct {
	Role MemberRole `json:"role,omitempty"`
	Size *int32     `json:"size,omitempty"`
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
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	AccessPoint              string `json:"accessPoint,omitempty"`
	Size                     int32  `json:"size,omitempty"`
	Ready                    int32  `json:"ready,omitempty"`
	Age                      string `json:"age,omitempty"`
	appsv1.StatefulSetStatus `json:",inline"`
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
