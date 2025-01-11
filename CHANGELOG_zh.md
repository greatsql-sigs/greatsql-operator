# 更新日志

本文件记录 GreatSQL Kubernetes Operator 项目的所有重要更改。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.1-alpha-1] - 2025-01-11

### Added
- 初始化项目框架
- 实现基础的 Kubernetes CRD 定义
- 添加 GreatSQL Single instance CRD 支持
  - 支持单实例的创建和管理
  - 添加 Webhook 验证逻辑（未实现）
- 添加 GreatSQL Single master GroupReplicationCluster CRD 支持
  - 支持单主 MGR 集群的创建和管理
  - 实现集群成员管理（至少3个节点）
  - 添加 Webhook 验证逻辑（未实现）
- 支持以下特性：
  - Pod 亲和性配置
  - 服务暴露配置
  - 容器资源限制
  - 存储卷配置
  - 健康检查探针
  - 滚动更新策略

### Security
- 添加基础的安全特性
  - Pod 安全上下文配置
  - 容器安全上下文配置
  - 服务账号配置

### TODO
- 计划支持以下特性：
  - 多主 MGR 集群（Multi-Master GroupReplicationCluster）
  - 主从复制集群（ReplicaofCluster）
  - MySQL Router 代理
  - 备份调度
  - 指标收集 