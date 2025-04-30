English ｜ [简体中文](./README_zh.md)
<!-- markdownlint-disable MD033 -->
# greatsql-operator

[![GitHub stars](https://img.shields.io/github/stars/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/issues)
[![GitHub license](https://img.shields.io/github/license/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/blob/main/LICENSE)

GreatSQL Operator provides robust and reliable support for GreatSQL on Kubernetes. It manages all the resources needed for deploying and managing highly available GreatSQL clusters. It also offers easy backup functionality while maintaining high availability of the cluster.

🍺 🍕 ☕ If this operator has helped your project, please consider sponsoring to accelerate development. Issues in this repository will be addressed on a best-effort basis.

This project is developed and maintained by the GreatSQL sigs community and is open source, following the [Apache 2.0](LICENSE) license.

## Project Description
GreatSQL Operator is a tool for deploying and managing GreatSQL database clusters on Kubernetes. It provides the following core features:

- Automated deployment and management of GreatSQL Standalone instances and clusters (MGR)
- Automatic failover and self-healing capabilities
- Backup and recovery management
- Monitoring integration
- Resource usage optimization
- Rolling upgrade support

This project is developed based on the `Kubebuilder` framework, allowing you to manage GreatSQL databases like native Kubernetes resources.

## Compatibility List

| Component                | Version          | Status | Notes       |
|--------------------------|------------------|--------|-------------|
| Kubernetes               | 1.22+            | ✅     | Recommended version <=1.29 |
| OpenShift                | 4.8+             | ✅     |             |
| GreatSQL                 | 8.0.25-26        | ✅     | Recommended version |
| GreatSQL                 | 8.0.25-25        | ✅     |             |

## Installation
For detailed deployment instructions, please refer to the [Project Manual](docs/manual.md).

## Contribution Guide
Please see our [Contribution Guide](./CONTRIBUTING_zh.md) for details on how to contribute to this project.

## Roadmap
 - [ ] Webhook validation
 - [ ] Multi-master cluster
 - [ ] Proxy SQL integration
 - [ ] Logical backup
 - [ ] Physical backups
 - [ ] [Prometheus](https://github.com/prometheus/prometheus) metrics exporter

## Version Notes

### v1.0.1 (Current Version)
- Initial release version
- Core features:
  - Support for GreatSQL single instance deployment
  - Support for GreatSQL MGR cluster deployment
  - Developed based on Kubebuilder v4 framework
  - Support for automatic failover
  - Support for rolling upgrades
  - Support for resource usage optimization

### Known Issues
- Webhook validation function is not yet completed
- Backup function is still under development
- Prometheus metrics exporter is not yet implemented

### Subsequent Plans
Please refer to the [Roadmap](#roadmap) section for future version plans.

## License

Copyright 2024 greatsql.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and limitations under the License.

