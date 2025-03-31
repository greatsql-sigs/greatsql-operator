# Helm 安装

## GreatSQL Operator Helm Chart
GreatSQL Operator Helm Chart 旨在部署 GreatSQL Operator。它简化了在 Kubernetes 集群中部署和管理基于 MySQL 的数据库的过程，支持单实例和组复制（集群）模式。

## 先决条件
Kubernetes 1.22+</br>
Helm 3.0+</br>
cert-manager v1.5.0+（用于 webhook 功能）</br>
与 PersistentVolumes 兼容的 StorageClass（持久性所需）</br>

## 安装
### 添加 Helm 存储库
在安装图表之前，请添加官方存储库：

```sh
helm repo add greatsql-operator https://greatsql-sigs.github.io/greatsql-operator-helm
helm repo update
```

### 安装图表
要安装图表，请使用以下命令

```sh
helm install greatsql greatsql-operator/greatsql-operator
```

这将使用默认值安装图表。您可以通过指定参数来自定义安装（请参阅下面的“配置”部分）。

## 卸载
要卸载已部署的图表，请使用以下命令

```sh
helm delete greatsql
```
此命令将删除与图表关联的所有资源，但默认保留 PersistentVolumeClaims（PVC），以避免意外数据丢失。

## 配置
### 默认值
您可以通过修改默认值.yaml 文件或通过 --set 标志传递自定义值来自定义安装。

#### 关键配置选项：
| 参数 | 描述 | 默认 |
|----------------------------|---------------------------------------------------------------|---------------------------------------------------------------|
| `replicaCount` | 单实例部署的副本数 | `1` |
| `type` | 部署类型：`SingleInstance` 或 `GroupReplicationCluster` | `SingleInstance` |
| `image.repository` | 映像存储库 | `registry.cn-chengdu.aliyuncs.com/greatsql/greatsql-operator` |
| `image.tag` | 图片标签 | `latest` |
| `service.type` | Kubernetes 服务类型 | `ClusterIP` |
| `service.port` | MySQL 服务端口 | `3306` |
| `storage.enabled` | 启用持久化存储 | `true` |
| `storage.size` | PersistentVolumeClaim 请求的存储大小 | `10Gi` |
| `configFile.enabled` | 启用自定义配置文件挂载 | `false` |
| `configFile.configMapName` | 用于配置的 ConfigMap 的名称 | `greatsql-config` |

## 示例配置
### 1. 部署 SingleInstance
```sh
helm install greatsql greatsql-operator/greatsql-operator \
--set type=SingleInstance \
--set replicaCount=1 \
--set storage.size=20Gi
```

### 2. 部署 GroupReplicationCluster
```sh
helm install greatsql-cluster greatsql-operator/greatsql-operator \
--set type=GroupReplicationCluster \
--set cluster.replicas=3 \
--set storage.size=50Gi \
--set configFile.enabled=true \
```

### 3. 启用自定义 ConfigMap
[!注意]：GreatSQL Operator 中，greatsql的配置文件是由operator本身来维护，在每次发布版本的时候会将配置文件打包到二进制文件中，每个版本发布都会跟随greatsql官方的版本发布而更新配置文件，所以在一般情况下不推荐使用自定义配置文件，除非有特殊需求。

如果要挂载自定义 my.cnf 配置，请创建一个ConfigMap：
```sh
kubectl create configmap greatsql-config --from-literal=my.cnf="[mysqld]\nmax_connections=200"
```

然后使用以下命令安装图表：
```sh
helm install greatsql greatsql-operator/greatsql-operator \
--set configFile.enabled=true \
--set configFile.configMapName=greatsql-config
```
## 持久性
默认情况下，持久存储处于启用状态。要禁用它，请将 storage.enabled 设置为 false：
```sh
helm install greatsql greatsql-operator/greatsql-operator \
--set storage.enabled=false
```

启用后，将创建 PersistentVolumeClaims 来存储 MySQL 数据，即使删除或重新启动 Pod，也能确保数据持久性。

### 升级
要将图表升级到较新的版本或更新值，请使用以下命令：
```sh
helm upgrade greatsql greatsql-operator/greatsql-operator --set <parameters>
```

## 监控
GreatSQL 公开基本的活跃度和就绪性探测以确保正常运行。您还可以集成 Prometheus 和 Grafana 等监控工具以获取高级指标。

## 支持
如需GreatSQL Operator Helm支持，请访问 [GreatSQL SIGs GitHub 存储库或在存储库中提出问题](https://github.com/greatsql-sigs/greatsql-operator-helm)。

通过使用此图表，您可以轻松地在 Kubernetes 集群中部署和管理 MySQL 实例，无论是用于开发、测试还是生产环境。