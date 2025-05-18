package schedule

import (
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetPodAffinity sets the pod affinity of the resource
func SetPodAffinity(cr any, labels map[string]string) *corev1.Affinity {
	var topologyKey *string

	switch spec := cr.(type) {
	case v1alpha1.Standalone:
		topologyKey = spec.Spec.Pod.Affinity.TopologyKey
	case v1alpha1.GroupReplicationCluster:
		topologyKey = spec.Spec.Affinity.TopologyKey
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

// SetPodAntiAffinity 设置Pod的反亲和性配置
func SetPodAntiAffinity(spec v1alpha1.StandaloneSpec, labels map[string]string) *corev1.Affinity {
	if spec.Pod.Affinity == nil {
		return nil
	}

	// 设置Pod反亲和性
	podAntiAffinity := &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					TopologyKey: *spec.Pod.Affinity.TopologyKey,
				},
			},
		},
	}

	return &corev1.Affinity{
		PodAntiAffinity: podAntiAffinity,
	}
}
