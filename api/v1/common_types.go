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
	"k8s.io/apimachinery/pkg/util/intstr"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-17 18:34:07
 * @file: community_types.go
 * @description: common types
 */

// Category defines the type of the GreatSql
// Supported values are "SingleInstance" "GroupReplicationCluster"
// SingleInstance: SingleInstance instance of GreatSql
// ReplicaofCluster: Master-slave replication cluster
// TODO: GroupReplicationCluster: GroupReplicationCluster of a GreatSql cluster(MGR)
// type Category string

// const (
// 	SingleInstanceCategory          Category = "SingleInstance"
// 	ReplicaofClusterCategory        Category = "ReplicaofCluster"
// 	GroupReplicationClusterCategory Category = "GroupReplicationCluster"
// )

type MemberRole string

const (
	PrimaryRole    MemberRole = "primary"
	SencondaryRole MemberRole = "sencondary"
	ArbitratorRole MemberRole = "arbitrator"
)

type MySQLGroupReplicationCluster struct {
	PodSpec        *PodSpec                       `json:"podSpec,omitempty"`
	Ports          []corev1.ServicePort           `json:"ports,omitempty"`
	Type           corev1.ServiceType             `json:"type,omitempty"`
	DnsPolicy      corev1.DNSPolicy               `json:"dnsPolicy,omitempty"`
	UpgradeOptions UpgradeOptions                 `json:"upgradeOptions,omitempty"`
	UpdateStrategy *StatefulSetUpdateStrategyType `json:"updateStrategy,omitempty"`
	// Partition      *int32                         `json:"partition,omitempty"`
	// MaxUnavailable *intstr.IntOrString            `json:"maxUnavailable,omitempty"`
}

type StatefulSetUpdateStrategyType struct {
	Type           appsv1.StatefulSetUpdateStrategyType `json:"type,omitempty"`
	RolelingUpdate *RolelingUpdate                      `json:"rolelingUpdate,omitempty"`
}

type RolelingUpdate struct {
	Partition      *int32              `json:"partition,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// MySQLRouterSpec defines the desired state of MySQLRouter
// TODO:MySQLRouter is not implemented yet
type Proxy struct {
	Enabled bool `json:"enable,omitempty"`
	PodSpec `json:",inline"`
	Expose  ServiceExpose `json:"expose,omitempty"`
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

// PodSpec defines the desired state of Pod
type PodSpec struct {
	Affinity                      *PodAffinity               `json:"affinity,omitempty"` // pod affinity(pod亲和性)
	Annotation                    map[string]string          `json:"annotation,omitempty"`
	Labels                        map[string]string          `json:"labels,omitempty"`
	NodeSelector                  map[string]string          `json:"nodeSelector,omitempty"`
	Tolerations                   []corev1.Toleration        `json:"tolerations,omitempty"`                   //schedule tolerations
	TerminationGracePeriodSeconds *int64                     `json:"terminationGracePeriodSeconds,omitempty"` // 在规定时间内停止pod，俗称 优雅停机
	SchedulerName                 string                     `json:"schedulerName,omitempty"`
	PodSecurityContext            *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	ServiceAccountName            string                     `json:"serviceAccountName,omitempty"`
	ServiceName                   string                     `json:"serviceName,omitempty"`
	Version                       string                     `json:"version,omitempty"`
	//+Optional
	Containers                    []ContainerSpec                   `json:"containers,omitempty"` // container spec
	PersistentVolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaimTemplate,omitempty"`
	// Storage    *Storage        `json:"storage,omitempty"`
}

// TODO: not implemented yet
// PersistentVolumeClaimTemplate 创建pvc后会自动关联创建pv，
// PersistentVolumeSource 是用于定义PV的持久卷的资源，暂时不考虑支持创建PV，只支持PVC
type Storage struct {
	Type                          string                            `json:"type,omitempty"`
	PersistentVolumeSource        *corev1.PersistentVolumeSource    `json:"persistentVolumeSource,omitempty"`
	PersistentVolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaimTemplate,omitempty"`
}

// PodAffinity defines the affinity/anti-affinity rules for the pod.
type PodAffinity struct {
	//+builder:default="kubernetes.io/hostname"
	//+Optional
	TopologyKey *string `json:"antiAffinityTopologyKey,omitempty"`
}

// ContainerSpec defines the desired state of the container
type ContainerSpec struct {
	Image            string                        `json:"image"`                      // Image of the container
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`  // Image pull policy
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"` // Image pull secrets
	Resources        corev1.ResourceRequirements   `json:"resources,omitempty"`        // Resource requirements
	StartupProbe     corev1.Probe                  `json:"startupProbe,omitempty"`     // Startup probe
	ReadinessProbe   corev1.Probe                  `json:"readinessProbe,omitempty"`   // Readiness probe
	LivenessProbe    corev1.Probe                  `json:"livenessProbe,omitempty"`    // Liveness probe
	SecurityContext  *corev1.SecurityContext       `json:"securityContext,omitempty"`  // Security context for the container
	Envs             []corev1.EnvVar               `json:"envs,omitempty"`             // Environment variables
}

// UpgradeOptions defines the desired state of UpgradeOptions
type UpgradeOptions struct {
	VersionServiceEndpoint string `json:"versionServiceEndpoint,omitempty"`
	Apply                  string `json:"apply,omitempty"`
}

// ServiceExpose defines the desired state of ServiceExpose
type ServiceExpose struct {
	Enabled                  bool                                    `json:"enabled,omitempty"`
	Type                     corev1.ServiceType                      `json:"type,omitempty"`
	LoadBalancerSourceRanges []string                                `json:"loadBalancerSourceRanges,omitempty"`
	LoadBalancerIP           string                                  `json:"loadBalancerIP,omitempty"`
	Annotations              map[string]string                       `json:"annotations,omitempty"`
	Labels                   map[string]string                       `json:"labels,omitempty"`
	ExternalTrafficPolicy    corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy    corev1.ServiceInternalTrafficPolicy     `json:"internalTrafficPolicy,omitempty"`

	// Deprecated: Use ExternalTrafficPolicy instead
	TrafficPolicy corev1.ServiceExternalTrafficPolicyType `json:"trafficPolicy,omitempty"`
}

// PodAffinity returns the SingleInstance pod affinity of the resource
func (s *SingleInstance) PodAffinity(labels map[string]string) *corev1.Affinity {
	return SetPodAffinity(s.Spec, labels)
}

// PodAffinity returns the SingleInstance primary group cluster pod affinity of the resource
func (s *GroupReplicationCluster) PodAffinity(labels map[string]string) *corev1.Affinity {
	return SetPodAffinity(s.Spec, labels)
}

// SetPodAffinity sets the pod affinity of the resource
func SetPodAffinity(spec interface{}, labels map[string]string) *corev1.Affinity {
	var topologyKey *string

	switch spec := spec.(type) {
	case SingleInstance:
		topologyKey = spec.Spec.PodSpec.Affinity.TopologyKey
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
