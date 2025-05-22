# Kind说明

## Standalone
standalone为单机实例，用用于管理greatsql的单机实例，其yaml定义如下：

```yaml
apiVersion: database.greatsql.cn/v1alpha1
kind: Standalone
metadata:
  name: greatsql-standalone
  namespace: greatsql
spec:
  size: 1
```

## GroupReplicationCluster
groupreplicationcluster为集群实例，对应的是GreatSQL的Group Replication集群，主要用于管理greatsql的集群实例，其yaml定义如下：

```yaml
apiVersion: database.greatsql.cn/v1alpha1
kind: GroupReplicationCluster
metadata:
  name: greatsql-mgr
  namespace: greatsql
spec:
  mode: single
```
- GroupReplicationCluster包含两种模式，single和multiple，single模式下，只有一个主节点（single-primary），multiple模式下，有多个主节点（multi-primary），
- single模式下，member中有且只能有一个primary节点，可以有多个secondary节点;
- multiple模式下，member中可以有多个primary节点，可以有多个secondary节点;

## Proxy
// TODO: Proxy is not implemented yet

## SchedulerBuckup
// TODO: SchedulerBuckup is not implemented yet

## MetricsCollection
// TODO: MetricsCollection is not implemented yet