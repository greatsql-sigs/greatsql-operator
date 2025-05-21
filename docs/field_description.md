# CR参数说明

本文档详细说明了 GreatSQL Operator 自定义资源类型的字段含义与使用方式

GreatSQL Operator 的CRD大致可以分为以下几类：
- 数据库实例（单机/集群）
- 数据库备份
- 数据库恢复
- 数据库网络
- 数据库监控

> `json字段` 为inline的，在yaml中不需要配置

## Standalone
- apiVersion: database.greatsql.cn/v1alpha1</br>
API 版本，当前版本为 `database.greatsql.cn/v1alpha1`，版本说明见[版本说明](version-notes.md)

- kind: Standalone</br>
资源类型，固定为 `Standalone`

- metadata
  - name: `string`</br>
  实例名称 [必填]
  - namespace: `string`</br>
  命名空间 [必填]
  - labels: `map[string]string`</br>
  标签 [可选]
  - finalizers: `[]string`</br>
  终结器列表 [可选]
  - annotations: `map[string]string`</br>
  注解 [可选]

- spec

| 参数 | json 字段 | 数据类型 | 是否必填 | 说明 |
|------|----------|----------|----------|------|
| size | size | int32 | 否 | 实例数量 |
| Pod | (inline) | [Pod](#Pod) | 是 | Pod 配置 |
| Upgrade | upgrade | [Upgrade](#Upgrade) | 否 | 升级配置 |
| UpdateStrategy | updateStrategy | [UpdateStrategy](#UpdateStrategy) | 否 | 更新策略 |
| Service | service | [Service](#Service) | 是 | Service 配置 |

- status

| 参数 | json 字段 | 数据类型 | 说明 |
|------|----------|----------|------|
| Phase | phase | string | 集群状态 |
| Message | message | string | 状态信息 |
| Reason | reason | string | 状态原因 |
| Age | age | string | 运行时长 |
| Ready | ready | int32 | 就绪节点数量 |

## GroupReplicationCluster

- apiVersion: database.greatsql.cn/v1alpha1</br>
API 版本，当前版本为 `database.greatsql.cn/v1alpha1`，版本说明见[版本说明](version-notes.md)

- kind: GroupReplicationCluster</br>
资源类型

- metadata
  - name: `string`</br>
  集群名称 [必填]
  - namespace: `string`</br>
  命名空间 [必填]
  - labels: `map[string]string`</br>
  标签 [可选]
  - finalizers: `[]string`</br>
  终结器列表 [可选]
  - annotations: `map[string]string`</br>
  注解 [可选]

- spec

| 参数 | json 字段 | 数据类型 | 是否必填 | 说明 |
|------|----------|----------|----------|------|
| mode | mode | string | 是 | 集群模式 (single/multiple) |
| member | member | []Member | 是 | 集群成员配置 |
| member[].role | role | string | 是 | 成员角色 (primary/secondary/arbitrator) |
| member[].size | size | int32 | 是 | 成员数量 |
| Pod | (inline) | [Pod](#Pod) | 是 | Pod 配置 |
| Upgrade | upgrade | [Upgrade](#Upgrade) | 否 | 升级配置 |
| UpdateStrategy | updateStrategy | [UpdateStrategy](#UpdateStrategy) | 否 | 更新策略 |
| Service | service | [Service](#Service) | 否 | Service 配置 |

## 公共类型字段定义

### Pod
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| version | version | string | 否 | GreatSQL 版本 |
| serviceAccountName | serviceAccountName | string | 否 | ServiceAccount 名称 |
| serviceName | serviceName | string | 是 | Service 名称 |
| containers | containers | []Container | 是 | 容器配置列表 |
| storages | storages | []Storage | 否 | 存储配置列表 |
| affinity | affinity | Affinity | 否 | Pod 亲和性配置 |
| nodeSelector | nodeSelector | map[string]string | 否 | 节点选择器 |
| tolerations | tolerations | []Toleration | 否 | 容忍配置 |
| schedulerName | schedulerName | string | 否 | 调度器名称 |
| terminationGracePeriodSeconds | terminationGracePeriodSeconds | int64 | 否 | 终止宽限期 |
| podSecurityContext | podSecurityContext | PodSecurityContext | 否 | Pod 安全上下文 |
| dnsPolicy | dnsPolicy | DNSPolicy | 否 | DNS 策略 |
| restartPolicy | restartPolicy | RestartPolicy | 否 | 重启策略 |

### Container
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| name | name | string | 是 | 容器名称 |
| image | image | string | 是 | 容器镜像 |
| imagePullPolicy | imagePullPolicy | PullPolicy | 否 | 镜像拉取策略 |
| imagePullSecrets | imagePullSecrets | []LocalObjectReference | 否 | 镜像拉取密钥 |
| resources | resources | ResourceRequirements | 否 | 资源需求 |
| startupProbe | startupProbe | Probe | 是 | 启动探针 |
| readinessProbe | readinessProbe | Probe | 是 | 就绪探针 |
| livenessProbe | livenessProbe | Probe | 是 | 存活探针 |
| securityContext | securityContext | SecurityContext | 否 | 安全上下文 |
| envs | env | []EnvVar | 否 | 环境变量 |

### Storage
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| storageClassName | storageClassName | string | 否 | 存储类名称 |
| accessModes | accessModes | []PersistentVolumeAccessMode | 是 | 访问模式 |
| volumeMode | volumeMode | PersistentVolumeMode | 否 | 卷模式 |
| size | size | string | 否 | 存储大小 (如 "10Gi") |

### Service
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| type | type | ServiceType | 是 | Service 类型 |
| externalTrafficPolicy | externalTrafficPolicy | ServiceExternalTrafficPolicyType | 否 | 外部流量策略 |
| internalTrafficPolicy | internalTrafficPolicy | ServiceInternalTrafficPolicyType | 否 | 内部流量策略 |
| ports | ports | []ServicePort | 否 | 端口列表 |
| selector | selector | map[string]string | 否 | 选择器 |
| loadBalancerClass | loadBalancerClass | string | 否 | LoadBalancer 类 |
| annotations | annotations | map[string]string | 否 | 注解 |

### Upgrade
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| versionServiceEndpoint | versionServiceEndpoint | string | 否 | 版本服务端点 |
| apply | apply | string | 否 | 应用配置 |

### UpdateStrategy
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| type | type | StatefulSetUpdateStrategyType | 否 | 更新策略类型 |
| rollingUpdate | rollingUpdate | RollingUpdate | 否 | 滚动更新配置 |

### RollingUpdate
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| partition | partition | int32 | 否 | 分区值 |
| maxUnavailable | maxUnavailable | IntOrString | 否 | 最大不可用数量 |

### Affinity
| 字段名 | json 字段 | 类型 | 是否必填 | 说明 |
|--------|----------|------|----------|------|
| topologyKey | antiAffinityTopologyKey | string | 否 | 反亲和性拓扑键 (默认 "kubernetes.io/hostname") |