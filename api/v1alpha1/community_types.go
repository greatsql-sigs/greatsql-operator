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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-17 18:34:07
 * @file: community_types.go
 * @description: common types
 */
// MemberRole defines the role of the member
type MemberRole string

const (
	PrimaryRole    MemberRole = "primary"
	SecondaryRole  MemberRole = "secondary"
	ArbitratorRole MemberRole = "arbitrator"
)

// State defines the state of the member
type State string

const (
	StateInitializing State = "initializing"
	StateRunning      State = "running"
	StateStoping      State = "stopping"
	StateReady        State = "ready"
	StateError        State = "error"
	StatePaused       State = "paused"
)

func (s State) String() string {
	return cases.Title(language.English).String(string(s))
}

type MySQLGroupReplicationCluster struct {
	PodSpec *Pod `json:"podSpec,omitempty"`
	// UpgradeOptions UpgradeOptions                 `json:"upgradeOptions,omitempty"`
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`
	// Partition      *int32                         `json:"partition,omitempty"`
	// MaxUnavailable *intstr.IntOrString            `json:"maxUnavailable,omitempty"`
}

// MySQLRouterSpec defines the desired state of MySQLRouter
// TODO:MySQLRouter is not implemented yet
type Proxy struct {
	Enabled bool `json:"enable,omitempty"`
	Pod     `json:",inline"`
}

// SchedulerBuckup defines the desired state of SchedulerBuckup
// TODO: SchedulerBuckup is not implemented yet
type SchedulerBuckup struct {
	//+kube:validation:Enum=true, false
	Enable *bool `json:"enable,omitempty"`
}

// MetricsCollection greatsql metrics collection, define the desired state of MetricsCollection
// TODO: MetricsCollection is not implemented
type MetricsCollection struct {
	//+kube:validation:Enum=true, false
	Enable *bool `json:"enable,omitempty"`
}

// Service 配置
type Service struct {
	Type  corev1.ServiceType   `json:"type,omitempty"`
	Ports []corev1.ServicePort `json:"ports,omitempty"`
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
	// +optional
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Storage 存储配置
type Storage struct {
	StorageClassName *string                             `json:"storageClassName,omitempty"`
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	VolumeMode       *corev1.PersistentVolumeMode        `json:"volumeMode,omitempty"`
	Size             *string                             `json:"size,omitempty"` // e.g. "10Gi"
}

// Scheduling 调度策略
type Scheduling struct {
	Affinity                      *corev1.Affinity           `json:"affinity,omitempty"`
	NodeSelector                  map[string]string          `json:"nodeSelector,omitempty"`
	Tolerations                   []corev1.Toleration        `json:"tolerations,omitempty"`
	SchedulerName                 string                     `json:"schedulerName,omitempty"`
	TerminationGracePeriodSeconds *int64                     `json:"terminationGracePeriodSeconds,omitempty"`
	PodSecurityContext            *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	DNSPolicy                     corev1.DNSPolicy           `json:"dnsPolicy,omitempty"`
}

// Upgrade 升级策略
type Upgrade struct {
	VersionServiceEndpoint string `json:"versionServiceEndpoint,omitempty"`
	Apply                  string `json:"apply,omitempty"`
}

// UpdateStrategy 更新策略
type UpdateStrategy struct {
	Type          appsv1.StatefulSetUpdateStrategyType `json:"type,omitempty"`
	RollingUpdate *RollingUpdate                       `json:"rollingUpdate,omitempty"`
}

// RollingUpdate 滚动更新
type RollingUpdate struct {
	Partition      *int32              `json:"partition,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// Status 状态
type Status struct {
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Age     string `json:"age,omitempty"`
	Ready   int32  `json:"ready,omitempty"`
}

// Container 容器配置
type Container struct {
	Name             string                        `json:"name"`                       // Name of the container
	Image            string                        `json:"image"`                      // Image of the container
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`  // Image pull policy
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"` // Image pull secrets
	Resources        corev1.ResourceRequirements   `json:"resources,omitempty"`        // Resource requirements
	StartupProbe     corev1.Probe                  `json:"startupProbe,omitempty"`     // Startup probe
	ReadinessProbe   corev1.Probe                  `json:"readinessProbe,omitempty"`   // Readiness probe
	LivenessProbe    corev1.Probe                  `json:"livenessProbe,omitempty"`    // Liveness probe
	SecurityContext  *corev1.SecurityContext       `json:"securityContext,omitempty"`  // Security context for the container
	Envs             []corev1.EnvVar               `json:"env,omitempty"`              // Environment variables
}

// Pod 基础配置
type Pod struct {
	// 基础配置
	Version                       string                     `json:"version,omitempty"`                       // 版本信息
	ServiceAccountName            string                     `json:"serviceAccountName,omitempty"`            // ServiceAccount 名称
	ServiceName                   string                     `json:"serviceName,omitempty"`                   // Service 名称
	Containers                    []Container                `json:"containers,omitempty"`                    // 容器配置列表
	Storages                      []Storage                  `json:"storages,omitempty"`                      // 存储配置列表
	Affinity                      *Affinity                  `json:"affinity,omitempty"`                      // Pod 亲和性配置
	NodeSelector                  map[string]string          `json:"nodeSelector,omitempty"`                  // 节点选择器
	Tolerations                   []corev1.Toleration        `json:"tolerations,omitempty"`                   // 容忍配置
	SchedulerName                 string                     `json:"schedulerName,omitempty"`                 // 调度器名称
	TerminationGracePeriodSeconds *int64                     `json:"terminationGracePeriodSeconds,omitempty"` // 终止宽限期
	PodSecurityContext            *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`            // Pod 安全上下文
	DnsPolicy                     corev1.DNSPolicy           `json:"dnsPolicy,omitempty"`                     // DNS 策略
	RestartPolicy                 corev1.RestartPolicy       `json:"restartPolicy,omitempty"`                 // 重启策略
}

// Affinity defines the affinity/anti-affinity rules for the pod.
type Affinity struct {
	//+builder:default="kubernetes.io/hostname"
	//+Optional
	TopologyKey *string `json:"antiAffinityTopologyKey,omitempty"`
}

// PodAffinity returns the Standalone pod affinity of the resource
func (s *Standalone) PodAffinity(labels map[string]string) *corev1.Affinity {
	return SetPodAffinity(s.Spec, labels)
}

// PodAffinity returns the Standalone primary group cluster pod affinity of the resource
func (s *GroupReplicationCluster) PodAffinity(labels map[string]string) *corev1.Affinity {
	return SetPodAffinity(s.Spec, labels)
}

// SetPodAffinity sets the pod affinity of the resource
func SetPodAffinity(spec interface{}, labels map[string]string) *corev1.Affinity {
	var topologyKey *string

	switch spec := spec.(type) {
	case Standalone:
		topologyKey = spec.Spec.Pod.Affinity.TopologyKey
	case GroupReplicationCluster:
		topologyKey = spec.Spec.ClusterSpec.PodSpec.Affinity.TopologyKey
	default:
		return nil
	}

	if topologyKey == nil {
		return nil
	}

	return &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					TopologyKey: *topologyKey,
				},
			},
		},
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					TopologyKey: *topologyKey,
				},
			},
		},
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      *topologyKey,
								Operator: corev1.NodeSelectorOpNotIn,
								Values:   []string{""},
							},
						},
					},
				},
			},
		},
	}
}
