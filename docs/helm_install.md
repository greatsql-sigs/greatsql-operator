# Helm 安装

## GreatSQL Operator Helm Chart

本 Chart 用于在 Kubernetes 集群中部署 **GreatSQL Operator**（控制器）。安装完成后，可通过创建 Standalone 或 GroupReplicationCluster 等 CR 来部署和管理 GreatSQL/MySQL 实例。

## 先决条件

- Kubernetes 1.22+
- Helm 3.0+
- cert-manager v1.5.0+（仅当启用 webhook 或 metrics TLS 时需要）

## 安装

### 添加 Helm 存储库

```sh
helm repo add greatsql-operator https://greatsql-sigs.github.io/greatsql-operator
helm repo update
```

### 安装 Chart

```sh
helm install greatsql greatsql-operator/greatsql-operator --create-namespace --namespace greatsql-system
```

使用默认值安装。可通过 `--set` 或自定义 values 文件覆盖配置（见下方「配置」）。

## 卸载

```sh
helm uninstall greatsql -n greatsql-system
```

默认会删除与 release 关联的资源。若安装时启用了 `crd.keep`，CRD 会保留，以避免误删自定义资源。

## 配置

Chart 的默认值与可配置项见 [dist/chart/values.yaml](../dist/chart/values.yaml)。常用选项如下：

| 参数 | 描述 | 默认 |
|------|------|------|
| `controllerManager.replicas` | Operator 控制器副本数（建议 1，多副本时需 leader-elect） | `3` |
| `controllerManager.container.image.repository` | Operator 镜像仓库 | `registry.cn-beijing.aliyuncs.com/greatsql/greatsql-operator` |
| `controllerManager.container.image.tag` | 镜像标签 | `latest` |
| `controllerManager.container.env` | 控制器环境变量（如协调间隔等） | 见 values.yaml |
| `controllerManager.container.resources` | 资源 requests/limits | 见 values.yaml |
| `rbac.enable` | 是否创建 RBAC（ServiceAccount、Role、RoleBinding 等） | `true` |
| `crd.enable` | 是否安装 CRD | `true` |
| `crd.keep` | 卸载时是否保留 CRD（`helm.sh/resource-policy: keep`） | `true` |
| `metrics.enable` | 是否创建 metrics Service（:8443） | `true` |
| `prometheus.enable` | 是否创建 ServiceMonitor（需 Prometheus Operator） | `false` |
| `certmanager.enable` | 是否通过 cert-manager 为 metrics 签发 TLS 证书 | `false` |
| `networkPolicy.enable` | 是否创建 NetworkPolicy | `false` |

### 示例：自定义镜像与副本数

```sh
helm install greatsql greatsql-operator/greatsql-operator \
  --create-namespace --namespace greatsql-system \
  --set controllerManager.replicas=1 \
  --set controllerManager.container.image.repository=myreg/greatsql-operator \
  --set controllerManager.container.image.tag=v1.0.0
```

### 示例：启用 Prometheus 监控

```sh
helm install greatsql greatsql-operator/greatsql-operator \
  --create-namespace --namespace greatsql-system \
  --set prometheus.enable=true
```

若需为 metrics 端点配置 TLS，可同时启用 cert-manager：

```sh
--set certmanager.enable=true
```

### 升级

```sh
helm upgrade greatsql greatsql-operator/greatsql-operator -n greatsql-system --set <参数>
```

## 监控

Operator 提供健康检查（/healthz、/readyz）和 metrics（:8443/metrics）。启用 `prometheus.enable` 后可被 Prometheus Operator 采集。

## 支持

问题与建议请提交至 [greatsql-operator 仓库](https://github.com/greatsql-sigs/greatsql-operator)。
