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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-17 18:32:59
 * @file: single_types.go
 * @description: SingleInstance types
 */

// SingleInstance defines the desired state of SingleInstance
type SingleInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// //+kubebuilder:validation:Enum=Sinlge;GroupReplicationCluster
	// Category   GreatSqlType                  `json:"category,omitempty"`
	// Role           MemberRole                    `json:"role,omitempty"`
	Size           *int32                        `json:"size,omitempty"`
	PodSpec        PodSpec                       `json:"podSpec,omitempty"`
	Ports          []corev1.ServicePort          `json:"ports,omitempty"`
	Type           corev1.ServiceType            `json:"type,omitempty"`
	DnsPolicy      corev1.DNSPolicy              `json:"dnsPolicy,omitempty"`
	UpgradeOptions UpgradeOptions                `json:"upgradeOptions,omitempty"`
	UpdateStrategy appsv1.DeploymentStrategyType `json:"updateStrategy,omitempty"`
}

// GetSize returns the size of the SingleInstance
func (s *SingleInstanceSpec) GetSize() int32 {
	if s.Size != nil {
		return *s.Size
	}
	return 1
}

// SingleInstanceStatus defines the observed state of SingleInstance
type SingleInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AccessPoint             string `json:"accessPoint,omitempty"`
	Size                    int32  `json:"size,omitempty"`
	Ready                   int32  `json:"ready,omitempty"`
	Age                     string `json:"age,omitempty"`
	appsv1.DeploymentStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="AccessPoint",type="string",JSONPath=".status.accessPoint",description="The access point of the SingleInstance"
//+kubebuilder:printcolumn:name="Size",type="integer",JSONPath=".spec.size",description="The size of the SingleInstance"
//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.ready",description="The ready of the SingleInstance"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="The age of the SingleInstance"

// SingleInstance is the Schema for the singles API
type SingleInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SingleInstanceSpec   `json:"spec,omitempty"`
	Status SingleInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SingleInstanceList contains a list of SingleInstance
type SingleInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SingleInstance `json:"items"`
}

func (s *SingleInstanceList) Finalizer() []string {
	return []string{"finalizer.SingleInstance.greatsql.cn"}
}

func init() {
	SchemeBuilder.Register(&SingleInstance{}, &SingleInstanceList{})
}
