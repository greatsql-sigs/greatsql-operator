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

// ClusterMode 集群模式
type ClusterMode string

const (
	ClusterModeSingle   ClusterMode = "single"
	ClusterModeMultiple ClusterMode = "multiple"
)

// Member 描述一个角色下要起多少个 Pod
// 比如：
//   - role: primary
//     size: 1
//   - role: secondary
//     size: 2
type Member struct {
	// 成员角色
	// +optional
	Role MemberRole `json:"role,omitempty"`

	// 该角色下的实例数，默认 1
	// +optional
	Size *int32 `json:"size,omitempty"`
}

// GroupReplicationClusterSpec defines the desired state of GroupReplicationCluster
type GroupReplicationClusterSpec struct {
	// 集群模式：single / multiple
	// +optional
	Mode ClusterMode `json:"mode,omitempty"`

	// 成员描述
	// +optional
	Member []Member `json:"member,omitempty"`

	// Pod / 容器 / 调度等通用配置
	*Pod `json:",inline"`

	// +optional
	Upgrade Upgrade `json:"upgrade,omitempty"`

	// +optional
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`

	// +optional
	Service *Service `json:"service,omitempty"`
}

// GroupReplicationClusterStatus defines the observed state of GroupReplicationCluster
type GroupReplicationClusterStatus struct {
	Status Status `json:"status,omitempty"`

	// 初始化是否完
	Bootstrapped bool `json:"bootstrapped,omitempty"`

	// 哪个节点负责初始化的（ <cr-name>-0）
	InitNode string `json:"initNode,omitempty"`

	// 初始化完成时间
	InitAt *metav1.Time `json:"initAt,omitempty"`

	// 当前感知到的成员视图
	Members []MemberStatus `json:"members,omitempty"`

	// Conditions 当前状态条件列表
	Conditions []Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=mgr
//nolint:lll
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.status.phase",description="The current phase of the group replication cluster"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.status.ready",description="The number of ready members"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.status.message",description="The status message"
// +kubebuilder:printcolumn:name="Reason",type="string",priority=1,JSONPath=".status.status.reason",description="The status reason"

type GroupReplicationCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupReplicationClusterSpec   `json:"spec,omitempty"`
	Status GroupReplicationClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type GroupReplicationClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupReplicationCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GroupReplicationCluster{}, &GroupReplicationClusterList{})
}

// --- helper methods ---

// getSize 返回这个成员要起几个实例：不写就 1，写 0 也按 1 处理
func (m *Member) getSize() int32 {
	if m == nil || m.Size == nil {
		return 1
	}
	if *m.Size <= 0 {
		return 1
	}
	return *m.Size
}

// GetTotalMembers 获取集群总成员数
func (s *GroupReplicationClusterSpec) GetTotalMembers() int32 {
	var total int32
	for _, member := range s.Member {
		total += member.getSize()
	}
	return total
}

// IsSingleMode 判断是否为单主模式
func (s *GroupReplicationClusterSpec) IsSingleMode() bool {
	return s.Mode == ClusterModeSingle
}

// IsMultipleMode 判断是否为多主模式
func (s *GroupReplicationClusterSpec) IsMultipleMode() bool {
	return s.Mode == ClusterModeMultiple
}

// GetPrimaryMembers 获取主节点成员列表
func (s *GroupReplicationClusterSpec) GetPrimaryMembers() []Member {
	if s.IsSingleMode() {
		if len(s.Member) == 0 {
			return nil
		}
		return []Member{s.Member[0]}
	}

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

// GetMemberByOrdinal 根据序号获取成员信息（和你 controller 里的一致）
//
// 例如：
// spec.member = [
//
//	{role: primary, size: 1},   // ordinal: 0
//	{role: secondary, size: 2}, // ordinal: 1,2
//	{role: arbitrator, size: 1} // ordinal: 3
//
// ]
// GetMemberByOrdinal(2) → secondary
func (s *GroupReplicationClusterSpec) GetMemberByOrdinal(ordinal int32) *Member {
	var currentOrdinal int32
	for idx := range s.Member {
		size := s.Member[idx].getSize()
		if ordinal >= currentOrdinal && ordinal < currentOrdinal+size {
			return &s.Member[idx]
		}
		currentOrdinal += size
	}
	return nil
}
