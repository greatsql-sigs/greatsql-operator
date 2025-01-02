# greatsql-operator

[![GitHub stars](https://img.shields.io/github/stars/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/issues)
[![GitHub license](https://img.shields.io/github/license/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/blob/main/LICENSE)

[简体中文](./README_zh.md)

GreatSQL Operator enables bulletproof GreatSQL on Kubernetes. It manages all the necessary resources for deploying and managing a highly available GreatSQL cluster. It provides effortless backups, while keeping the cluster highly available.

🍺 🍕 ☕ If the operator has helped you out with your projects, please consider sponsoring it to speed up the development. Issues are answered in this repo on a best-effort basis.

This operator is developed and maintained by the GreatSQL sigs community and is open source.

## Description
GreatSQL Operator is a tool for deploying and managing GreatSQL database clusters on Kubernetes. It provides the following core features:

- Automated deployment and management of GreatSQL single instances and clusters (MGR)
- Automatic failover and self-healing capabilities
- Backup and recovery management
- Monitoring integration
- Resource usage optimization
- Rolling upgrade support

This project is developed based on the Kubernetes Operator pattern and allows you to manage GreatSQL databases just like native Kubernetes resources.

## Installation
For detailed deployment instructions, please refer to the [Deployment Guide](docs/deployment_guide.md).

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=registry.cn-beijing.aliyuncs.com/greatsql/greatsql-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

```sh
kubectl apply -f https://raw.githubusercontent.com/greatsql-sigs/greatsql/main/dist/install.yaml
```

## Contributing
Please see our [Contributing Guidelines](./CONTRIBUTING.md) for details on how to contribute to this project.

## Roadmap
 - [ ] webhooks validation
 - [ ] Multi-master and cluster
 - [ ] Proxy SQL integration
 - [ ] Logical backup
 - [ ] Physical backups 
 - [ ] [Prometheus](https://github.com/prometheus/prometheus) metrics exporter

## Version Notes
### v1.0.1 (current version)
- Initial release version
- Core features:
- Support GreatSQL single instance deployment
- Support GreatSQL MGR cluster deployment
- Developed based on Kubebuilder v4 framework
- Support automatic failover
- Support rolling upgrade
- Support resource usage optimization

### Known issues
- Webhook verification function is not yet completed
- Backup function is still under development
- Prometheus indicator exporter is not yet implemented

### Subsequent plans
Please refer to the [Roadmap](#Roadmap) section for future version plans.

## License

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

