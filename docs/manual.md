# Gretasql Operator 手册

## 快速开始
### 前置条件
- Kubernetes v1.22.0+ 集群
- cert-manager v1.5.0+

> **❗❗❗注意**：请确保集群已经安装了 cert-manager，否则将无法使用webhook功能，由此可能会导致在部署过程中引起其他问题，详情请参考 [cert-manager](https://cert-manager.io/docs/installation/kubernetes/)

### 安装
```sh
# 安装 Operator
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/dist/install.yaml

# 验证安装
kubectl get pods -n greatsql-system
```

### 部署示例
```sh
# 部署单实例 GreatSQL
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/example/single/singleinstance.yaml

# 部署组复制集群
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/example/cluster/groupreplicationcluster.yaml

# 验证部署
kubectl get pods -n greatsql
```

### helm快速安装
```sh
# 添加 helm 仓库
helm repo add greatsql https://greatsql-sigs.github.io/greatsql-operator
helm repo update

# 安装 GreatSQL Operator
helm install greatsql greatsql-operator/greatsql-operator

# 验证安装
kubectl get pods -n greatsql-system
```
### 卸载
```sh
# 卸载 Operator
helm uninstall greatsql -n greatsql-system
kubectl delete ns greatsql-system

# 卸载 CRD
kubectl delete crd singleinstances.greatsql.io
kubectl delete crd groupreplicationclusters.greatsql.io

# 卸载 webhook
kubectl delete mutatingwebhookconfiguration mgroupreplicationcluster.kb.io
kubectl delete validatingwebhookconfiguration vgroupreplicationcluster.kb.io
```

## 其他文档
- [描述文件安装](docs/description_install.md)
- [Helm 安装](docs/helm_install.md)
- [Operator 字段说明](docs/operator_field.md)
