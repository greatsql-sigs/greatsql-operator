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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemberRole 成员角色
type MemberRole string

const (
	PrimaryRole    MemberRole = "primary"
	SecondaryRole  MemberRole = "secondary"
	ArbitratorRole MemberRole = "arbitrator"
)

// ClusterMode 集群模式
type ClusterMode string

const (
	ClusterModeSingle   ClusterMode = "single"
	ClusterModeMultiple ClusterMode = "multiple"
)

type Member struct {
	Role MemberRole `json:"role,omitempty"`
	Size *int32     `json:"size,omitempty"`
}

// GroupReplicationClusterSpec defines the desired state of GroupReplicationCluster
type GroupReplicationClusterSpec struct {
	Mode           ClusterMode `json:"mode,omitempty"`
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

// GetTotalMembers 获取集群总成员数
func (s *GroupReplicationClusterSpec) GetTotalMembers() int32 {
	var total int32
	for _, member := range s.Member {
		total += member.GetSize()
	}
	return total
}

// IsSingleMode 判断是否为单节点模式
func (s *GroupReplicationClusterSpec) IsSingleMode() bool {
	return s.Mode == ClusterModeSingle
}

// IsMultipleMode 判断是否为多节点模式
func (s *GroupReplicationClusterSpec) IsMultipleMode() bool {
	return s.Mode == ClusterModeMultiple
}

// GetPrimaryMembers 获取主节点成员列表
func (s *GroupReplicationClusterSpec) GetPrimaryMembers() []Member {
	var primaries []Member
	for _, member := range s.Member {
		if member.Role == PrimaryRole {
			primaries = append(primaries, member)
		}
	}
	return primaries
}

// GetSecondaryMembers 获取从节点成员列表
func (s *GroupReplicationClusterSpec) GetSecondaryMembers() []Member {
	var secondaries []Member
	for _, member := range s.Member {
		if member.Role == SecondaryRole {
			secondaries = append(secondaries, member)
		}
	}
	return secondaries
}

// GetArbitratorMembers 获取仲裁节点成员列表
func (s *GroupReplicationClusterSpec) GetArbitratorMembers() []Member {
	var arbitrators []Member
	for _, member := range s.Member {
		if member.Role == ArbitratorRole {
			arbitrators = append(arbitrators, member)
		}
	}
	return arbitrators
}

// ValidateClusterSpec 验证集群配置是否有效
func (s *GroupReplicationClusterSpec) ValidateClusterSpec() error {
	if s.IsSingleMode() {
		// 单节点模式验证
		if s.GetTotalMembers() != 1 {
			return fmt.Errorf("single mode cluster must have exactly one member")
		}
		if len(s.GetPrimaryMembers()) != 1 {
			return fmt.Errorf("single mode cluster must have exactly one primary member")
		}
	} else if s.IsMultipleMode() {
		// 多节点模式验证
		if s.GetTotalMembers() < 3 {
			return fmt.Errorf("multiple mode cluster must have at least 3 members")
		}
		if len(s.GetPrimaryMembers()) != 1 {
			return fmt.Errorf("multiple mode cluster must have exactly one primary member")
		}
		if len(s.GetSecondaryMembers()) < 1 {
			return fmt.Errorf("multiple mode cluster must have at least one secondary member")
		}
	} else {
		return fmt.Errorf("invalid cluster mode: %s", s.Mode)
	}
	return nil
}

// GetMemberByOrdinal 根据序号获取成员信息
func (s *GroupReplicationClusterSpec) GetMemberByOrdinal(ordinal int32) *Member {
	var currentOrdinal int32
	for _, member := range s.Member {
		size := member.GetSize()
		if ordinal >= currentOrdinal && ordinal < currentOrdinal+size {
			return &member
		}
		currentOrdinal += size
	}
	return nil
}
