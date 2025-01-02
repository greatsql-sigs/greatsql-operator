# greatsql-operator
[![GitHub stars](https://img.shields.io/github/stars/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/issues)
[![GitHub license](https://img.shields.io/github/license/greatsql-sigs/greatsql-operator)](https://github.com/greatsql-sigs/greatsql-operator/blob/main/LICENSE)

GreatSQL Operator 为 Kubernetes 提供稳固可靠的 GreatSQL 支持。它管理部署和管理高可用 GreatSQL 集群所需的所有资源。在保持集群高可用的同时，还提供轻松的备份功能。

🍺 🍕 ☕ 如果这个 operator 帮助了您的项目，请考虑赞助以加快开发进度。本仓库的问题将尽最大努力回答。

本 operator 由 GreatSQL sigs 社区开发和维护，并且是开源的。

## 项目描述
GreatSQL Operator 是一个用于在 Kubernetes 上部署和管理 GreatSQL 数据库集群的工具。它提供以下核心功能：

- 自动化部署和管理 GreatSQL 单实例和集群（MGR）
- 自动故障转移和自我修复能力
- 备份和恢复管理
- 监控集成
- 资源使用优化
- 滚动升级支持

本项目基于 `Kubebuilder` 框架开发，允许您像管理原生 Kubernetes 资源一样管理 GreatSQL 数据库。

## 安装
有关详细的部署说明，请参阅[部署指南](docs/deployment_guide_zh.md)。

## 贡献指南
请查看我们的[贡献指南](./CONTRIBUTING_zh.md)了解如何为该项目做出贡献。

## 路线图
 - [ ] webhooks验证
 - [ ] 多主集群
 - [ ] Proxy SQL 集成
 - [ ] 逻辑备份
 - [ ] 物理备份
 - [ ] [Prometheus](https://github.com/prometheus/prometheus) 指标导出器

## 版本说明

### v1.0.1 (当前版本)
- 初始发布版本
- 核心功能:
  - 支持 GreatSQL 单实例部署
  - 支持 GreatSQL MGR 集群部署
  - 基于 Kubebuilder v4 框架开发
  - 支持自动故障转移
  - 支持滚动升级
  - 支持资源使用优化

### 已知问题
- Webhook 验证功能尚未完成
- 备份功能尚在开发中
- Prometheus 指标导出器尚未实现

### 后续计划
请参考[路线图](#路线图)章节了解未来版本规划。

## 许可证

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