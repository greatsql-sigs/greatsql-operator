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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-17 18:34:07
 * @file: community_types.go
 * @description: common types
 */

// Phase defines the state of the member
type Phase string

const (
	PhaseInitializing Phase = "initializing"
	PhaseRunning      Phase = "running"
	PhaseStoping      Phase = "stopping"
	PhaseReady        Phase = "ready"
	PhaseError        Phase = "error"
	PhasePaused       Phase = "paused"
)

// Proxy defines the desired state of MySQLRouter
// TODO:MySQLRouter is not implemented yet
type Proxy struct {
	Enabled bool `json:"enable,omitempty"`
	Pod     `json:",inline"`
}

// SchedulingBackup defines the desired state of SchedulingBackup
// TODO: SchedulingBackup is not implemented yet
type SchedulingBackup struct {
	//+kube:validation:Enum=true, false
	Enable *bool `json:"enable,omitempty"`
}

// Restore defines the desired state of Restore
// TODO: Restore is not implemented yet
type Restore struct {
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
	Type corev1.ServiceType `json:"type,omitempty"`

	// 指定外部流量策略，
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
	// +optional
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Storage 存储配置
type Storage struct {
	// +optional
	StorageClassName *string                             `json:"storageClassName,omitempty"`
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	VolumeMode       *corev1.PersistentVolumeMode        `json:"volumeMode,omitempty"`
	Size             *string                             `json:"size,omitempty"` // e.g. "10Gi"
}

// Scheduling 调度策略
type Scheduling struct {
	// +optional
	Affinity *Affinity `json:"affinity,omitempty"` // 亲和性和反亲和性规则
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"` // 节点选择器
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"` // 容忍规则
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"` // 调度器名称
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"` // 容器终止宽限期，单位为秒
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"` // Pod 安全上下文
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"` // Pod 优先级类名称
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

// MemberStatus 成员状态
type MemberStatus struct {
	Name  string     `json:"name,omitempty"`  // 成员名称，如 greatsql-mgr-0
	Role  MemberRole `json:"role,omitempty"`  // 成员角色
	State string     `json:"state,omitempty"` // 成员状态，如 ONLINE, RECOVERING, OFFLINE
	Ready bool       `json:"ready,omitempty"` // 是否就绪
}

// Status 状态
type Status struct {
	Role    MemberRole `json:"role,omitempty"`
	Phase   Phase      `json:"phase,omitempty"`
	Message string     `json:"message,omitempty"`
	Reason  string     `json:"reason,omitempty"`
	Age     string     `json:"age,omitempty"`
	Ready   int32      `json:"ready,omitempty"`
}

// Container 容器配置
type Container struct {
	Name             string                        `json:"name"`                       // Name of the container
	Image            string                        `json:"image"`                      // Image of the container
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`  // Image pull policy
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"` // Image pull secrets
	Resources        corev1.ResourceRequirements   `json:"resources,omitempty"`        // Resource requirements
	StartupProbe     corev1.Probe                  `json:"startupProbe"`               // Startup probe
	ReadinessProbe   corev1.Probe                  `json:"readinessProbe"`             // Readiness probe
	LivenessProbe    corev1.Probe                  `json:"livenessProbe"`              // Liveness probe
	SecurityContext  *corev1.SecurityContext       `json:"securityContext,omitempty"`  //nolint:lll    // Security context for the container
	Envs             []corev1.EnvVar               `json:"env,omitempty"`              // Environment variables
}

// Pod 基础配置
type Pod struct {
	// 基础配置
	Version            string `json:"version,omitempty"`            // 版本信息
	ServiceAccountName string `json:"serviceAccountName,omitempty"` // ServiceAccount 名称
	ServiceName        string `json:"serviceName,omitempty"`        // Service 名称
	// TODO: 需要支持多个容器
	Container   Container        `json:"container,omitempty"` // 容器配置列表
	*Scheduling `json:",inline"` // 调度配置
	// +optional
	DnsPolicy     corev1.DNSPolicy     `json:"dnsPolicy,omitempty"`     // DNS 策略
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"` // 重启策略
	Storages      Storage              `json:"storages,omitempty"`      // 存储配置列表
}

// Affinity defines the affinity/anti-affinity rules for the pod.
type Affinity struct {
	// +builder:default="kubernetes.io/hostname"
	// +Optional
	TopologyKey *string `json:"antiAffinityTopologyKey,omitempty"`
}
