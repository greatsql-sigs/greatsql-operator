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

/**
 * @author: HuaiAn xu
 * @date: 2024-03-17 18:32:59
 * @file: standalone_types.go
 * @description: Standalone types
 */

// StandaloneSpec 单节点配置
type StandaloneSpec struct {
	Size           *int32 `json:"size,omitempty"`
	Pod            `json:",omitempty"`
	Upgrade        Upgrade        `json:"upgrade,omitempty"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	Service        Service        `json:"service,omitempty"`
}

// StandaloneStatus 状态
type StandaloneStatus struct {
	Status `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Standalone is the Schema for the singles API
type Standalone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneSpec   `json:"spec,omitempty"`
	Status StandaloneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StandaloneList contains a list of Standalone
type StandaloneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Standalone `json:"items"`
}

func (s *StandaloneList) Finalizer() []string {
	return []string{"finalizer.standalone.database.greatsql.cn"}
}

func init() {
	SchemeBuilder.Register(&Standalone{}, &StandaloneList{})
}
