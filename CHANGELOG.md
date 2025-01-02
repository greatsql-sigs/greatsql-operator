# Changelog
[简体中文](CHANGELOG_zh.md)

This file documents all notable changes to the GreatSQL Kubernetes Operator project.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v1.0.1-alpha-1] - 2024-03-21

### Added
- Initialize project framework
- Implement basic Kubernetes CRD definitions
- Add GreatSQL Single instance CRD support
  - Support single instance creation and management
  - Add Webhook validation logic (not implemented)
- Add GreatSQL Single master GroupReplicationCluster CRD support
  - Support single-master MGR cluster creation and management
  - Implement cluster member management (minimum 3 nodes)
  - Add Webhook validation logic (not implemented)
- Support the following features:
  - Pod affinity configuration
  - Service exposure configuration
  - Container resource limits
  - Storage volume configuration
  - Health check probes
  - Rolling update strategy

### Security
- Add basic security features
  - Pod security context configuration
  - Container security context configuration
  - Service account configuration

### TODO
- Plan to support the following features:
  - Multi-Master GroupReplicationCluster
  - ReplicaofCluster
  - MySQL Router proxy
  - Backup scheduling
  - Metrics collection