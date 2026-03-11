package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Phase defines the state of the member / cluster
type Phase string

const (
	PhaseInitializing Phase = "initializing"
	PhaseRunning      Phase = "running"
	PhaseStopping     Phase = "stopping" // fixed: was "stoping"
	PhaseReady        Phase = "ready"
	PhaseError        Phase = "error"
	PhasePaused       Phase = "paused"
)

// Proxy defines the desired state of MySQLRouter
// TODO: MySQLRouter is not implemented yet
type Proxy struct {
	Enabled bool `json:"enable,omitempty"`
	Pod     `json:",inline"`
}

// MetricsCollection greatsql metrics collection
// TODO: MetricsCollection is not implemented
type MetricsCollection struct {
	// +kubebuilder:validation:Optional
	Enable *bool `json:"enable,omitempty"`
}

// Service 配置
type Service struct {
	// Service 类型，默认 ClusterIP
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// Selector 通常不用用户写，会由 controller 注入
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
	StorageClassName *string `json:"storageClassName,omitempty"`

	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// +optional
	VolumeMode *corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`

	// e.g. "10Gi"
	// +optional
	Size *string `json:"size,omitempty"`
}

// Scheduling 调度策略
type Scheduling struct {
	// +optional
	Affinity *Affinity `json:"affinity,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`

	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`
}

// Upgrade 升级策略
type Upgrade struct {
	// 对接的版本服务地址
	// +optional
	VersionServiceEndpoint string `json:"versionServiceEndpoint,omitempty"`

	// 升级策略：auto / manual / ...
	// +optional
	Apply string `json:"apply,omitempty"`
}

// UpdateStrategy 更新策略
type UpdateStrategy struct {
	// +optional
	Type appsv1.StatefulSetUpdateStrategyType `json:"type,omitempty"`

	// +optional
	RollingUpdate *RollingUpdate `json:"rollingUpdate,omitempty"`
}

// RollingUpdate 滚动更新
type RollingUpdate struct {
	// +optional
	Partition *int32 `json:"partition,omitempty"`

	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// MemberStatus 成员状态
type MemberStatus struct {
	// 成员名称，如 greatsql-mgr-0
	Name string `json:"name,omitempty"`

	// 成员角色
	Role MemberRole `json:"role,omitempty"`

	// 成员状态，如 ONLINE, RECOVERING, OFFLINE
	State string `json:"state,omitempty"`

	// 是否就绪
	Ready bool `json:"ready,omitempty"`
}

// Status 通用状态
type Status struct {
	Role    MemberRole `json:"role,omitempty"`
	Phase   Phase      `json:"phase,omitempty"`
	Message string     `json:"message,omitempty"`
	Reason  string     `json:"reason,omitempty"`
	Age     string     `json:"age,omitempty"`
	Ready   int32      `json:"ready,omitempty"`
}

type Condition struct {
	Type               string                 `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
}

const (
	ConditionTypeAvailable   = "Available"
	ConditionTypeProgressing = "Progressing"
	ConditionTypeDegraded    = "Degraded"
)

// Container 容器配置
type Container struct {
	// Name of the container
	Name string `json:"name"`

	// Image of the container
	Image string `json:"image"`

	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// 这三个改成可选，避免 CRD 写起来太啰嗦
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`

	// +optional
	Envs []corev1.EnvVar `json:"env,omitempty"`
}

// Pod 基础配置
type Pod struct {
	// +optional
	Version string `json:"version,omitempty"`

	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// 容器配置（目前只放一个，够用）
	// +optional
	Container Container `json:"container,omitempty"`

	// 调度相关字段内联
	*Scheduling `json:",inline"`

	// +optional
	DnsPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// +optional
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`

	// 存储
	// +optional
	Storages Storage `json:"storages,omitempty"`
}

// Affinity defines the affinity/anti-affinity rules for the pod.
type Affinity struct {
	// +optional
	TopologyKey *string `json:"antiAffinityTopologyKey,omitempty"`
}
