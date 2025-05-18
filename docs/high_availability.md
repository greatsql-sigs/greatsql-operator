# GreatSQL Operator 高可用实现

## 1. Operator 高可用

### 1.1 架构设计
在实际业务中，数据库需要具备高可用性。GreatSQL Operator 作为数据库的控制器，其高可用性设计体现在两个层面：Operator Controller 和 Database Instance.

#### Operator Controller
- 使用 Kubernetes 的 Leader Election 机制，确保控制器具备高可用能力。
- 默认部署 3 个副本，Pod 分布在不同节点上。
- 配置 Pod 反亲和性，确保同一时间仅一个控制器实例在运行。

#### Database Instance
- 使用 Pod 反亲和性，确保数据库实例不集中于单节点。
- 支持 Affinity & Anti-Affinity（亲和性与反亲和性）策略。
- 支持 Taints & Tolerations（污点与容忍）策略。
- 支持自定义调度器。
- 支持自定义资源限制，如 CPU、内存、存储等。
- 支持自定义网络策略，如 VIP、负载均衡等。

### 1.2 配置示例
Operator Controller 的配置示例，主要通过Pod反亲和性来实现高可用，将pod分布在不同节点上，避免单点故障，同时使用Leader Election机制，确保控制器具备高可用能力。
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: greatsql-controller-manager
  namespace: greatsql-system
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          # 使用preferredDuringSchedulingIgnoredDuringExecution策略，这意味着调度器会尽量满足反亲和性要求，但如果无法满足也不会阻止Pod调度
          # 权重设置为100，表示这是一个重要的调度要求
          # 使用control-plane: greatsql-controller-manager标签来匹配operator的Pod
          # 使用kubernetes.io/hostname作为拓扑键，确保Pod分布在不同节点上
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  control-plane: greatsql-controller-manager
              topologyKey: kubernetes.io/hostname
      containers:
      - name: manager
        args:
        - --leader-elect  # 启用 Leader 选举
        env:
        # 控制operator的协调（reconciliation）循环的时间间隔，10s
        - name: RECONCILIATION_INTERVAL
          value: "10s"
```

### 1.3 高可用特性
- **Leader 选举**：通过 `--leader-elect` 参数启用，确保同一时间仅有一个控制器实例在运行。
- **Pod 反亲和性**：确保控制器 Pod 分布在不同节点上。
- **自动故障转移**：当 Leader Pod 故障时，其他 Pod 自动发起选举，选出新 Leader。
- **状态同步**：所有 Pod 共享状态信息，确保一致性。

## 2. CRD 实例高可用配置
CRD 实例的高可用配置，主要通过Affinity & Anti-Affinity（亲和性和反亲和性）策略来实现。
- GreatSQl Operator 在开发设计之初，为方便用户使用，将Affinity & Anti-Affinity（亲和性和反亲和性）策略封装为一个key，用户只需要配置topologyKey即可。在绝大多数场景下，这种方式可以满足大部分用户需求。
- 在特殊场景下，用户可以自定义调度器，如：使用自定义调度器来实现高可用。

```go
// SetAffinity sets the pod affinity of the resource
func SetAffinity(cr any, labels map[string]string) *corev1.Affinity {
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
```
> 由于单机实例只有一个Pod，所以不需要配置Affinity & Anti-Affinity（亲和性和反亲和性）策略。

### 2.2 集群模式高可用配置
GreatSQL Operator支持两种集群模式：单主模式(Single Primary)和多主模式(Multiple Primary)。每种模式都支持高可用配置。

#### 2.2.1 多主模式配置
```yaml
apiVersion: database.greatsql.cn/v1alpha1
kind: GroupReplicationCluster
metadata:
  name: greatsql-cluster
spec:
  mode: "multiple"  # 多主模式
  member:
    - role: primary
      size: 3
    - role: secondary
      size: 3
    - role: arbitrator
      size: 1
  pod:
    affinity:
      topologyKey: "kubernetes.io/hostname"  # 确保Pod分布在不同节点
    # 可选：节点选择器
    nodeSelector:
      kubernetes.io/os: linux
    # 可选：容忍配置
    tolerations:
    - key: "key1"
      operator: "Equal"
      value: "value1"
      effect: "NoSchedule"
    # 可选：调度器名称
    schedulerName: "custom-scheduler"
    # 可选：终止宽限期
    terminationGracePeriodSeconds: 30
    # 可选：Pod安全上下文
    podSecurityContext:
      runAsUser: 1000
      runAsGroup: 1000
      fsGroup: 1000
```

#### 2.2.2 单主模式配置
```yaml
apiVersion: database.greatsql.cn/v1alpha1
kind: GroupReplicationCluster
metadata:
  name: greatsql-cluster
spec:
  mode: "single"  # 单主模式
  member:
    - role: primary
      size: 3
    - role: arbitrator
      size: 1
  pod:
    affinity:
      topologyKey: "kubernetes.io/hostname"  # 确保Pod分布在不同节点
    # 其他配置与多主模式相同
```

#### 2.2.3 高可用特性说明
1. **Pod分布**
   - 通过`topologyKey: "kubernetes.io/hostname"`确保Pod分布在不同节点
   - 支持自定义拓扑键，如`zone`、`region`等
   - 使用`RequiredDuringSchedulingIgnoredDuringExecution`策略确保强制分布

2. **节点选择**
   - 通过`nodeSelector`指定节点标签
   - 支持多标签组合选择
   - 确保Pod调度到合适的节点

3. **调度策略**
   - 支持自定义调度器
   - 支持节点亲和性
   - 支持Pod亲和性和反亲和性

4. **安全配置**
   - 支持Pod安全上下文
   - 支持安全组设置
   - 支持权限控制

5. **资源管理**
   - 支持资源限制
   - 支持资源请求
   - 支持QoS策略