# 描述文件安装

## 描述文件安装
描述文件安装是指通过Kubernetes的描述文件来安装和配置应用程序。描述文件通常是YAML格式的文件，包含了应用程序的所有配置和资源定义

## 安装步骤
**安装描述文件**：使用kubectl命令将描述文件应用到Kubernetes集群中
```sh
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/dist/install.yaml

# or
git clone https://github.com/greatsql-sigs/greatsql-operator.git
cd greatsql-operator
kubectl apply -f dist/install.yaml
```

**验证安装**：检查GreatSQL Operator的Pod是否正常
```sh
kubectl get pods -n greatsql-system
```

**创建CRD**：根据需要创建Custom Resource Definitions（CRD），例如Standalone或GroupReplicationCluster
```sh
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/example/single/Standalone.yaml
    
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql-operator/main/example/cluster/groupreplicationcluster.yaml
```

**验证CRD**：检查创建的CRD是否正常
```sh
kubectl get pods -n greatsql
```

**卸载**：如果需要卸载GreatSQL Operator，可以使用以下命令
```sh
kubectl delete -f install.yaml
kubectl delete ns greatsql-system
kubectl delete ns greatsql
kubectl delete crd Standalones.greatsql.io
kubectl delete crd groupreplicationclusters.greatsql.io
kubectl delete mutatingwebhookconfiguration mgroupreplicationcluster.kb.io
kubectl delete validatingwebhookconfiguration vgroupreplicationcluster.kb.io
```

**注意事项**：在卸载时，请确保删除所有相关的资源和命名空间