# 贡献指南

感谢您考虑为 GreatSQL Operator 项目做出贡献！本文档将指导您完成贡献过程。

## 行为准则

本项目采用[贡献者公约](https://www.contributor-covenant.org/version/2/0/code_of_conduct/)。参与本项目即表示您同意遵守其条款。

## 如何贡献

### 报告 Bug

- 在提交 bug 之前，请先搜索现有的 Issues 以避免重复
- 使用 Issue 模板提交 bug 报告
- 包含详细的问题描述和复现步骤
- 如果可能，提供最小复现示例

### 提出新功能

- 在提交功能请求之前，请先搜索现有的 Issues
- 使用功能请求模板描述新功能
- 解释为什么这个功能对项目有价值

### 提交代码

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的修改 (`git commit -m '添加某个功能'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 开发环境设置

1. 确保您已安装以下依赖：
   - Go 123.0+
   - Docker 17.03+
   - kubectl 1.11.3+
   - 可访问的 Kubernetes v1.22.0+ 集群

2. 克隆代码库：
```bash
git clone https://github.com/greatsql-sigs/greatsql-operator
cd greatsql-operator
```

3. 运行测试确保环境正常：
```bash
make test
```

## 代码规范

### Go 代码规范

- 遵循 [Go 代码规范](https://golang.org/doc/effective_go)
- 使用 `gofmt` 格式化代码
- 添加适当的注释和文档
- 确保代码通过 `golint` 和 `go vet` 检查

### 提交信息规范

提交信息应遵循以下格式：
```
<类型>: <描述>

[可选的正文]

[可选的脚注]
```

类型可以是：
- feat: 新功能
- fix: Bug 修复
- docs: 文档更新
- style: 代码格式调整
- refactor: 代码重构
- test: 测试相关
- chore: 构建过程或辅助工具的变动

### 测试要求

- 为新功能添加单元测试
- 确保所有测试通过
- 保持测试覆盖率

## Pull Request 流程

1. 更新您的分支以包含最新的主分支更改
2. 确保所有测试通过
3. 更新相关文档
4. 填写 PR 模板
5. 等待代码审查
6. 根据反馈进行修改
7. 等待合并

## 文档贡献

- 确保文档清晰易懂
- 更新 README.md 和其他相关文档
- 添加示例和使用说明
- 修正拼写和语法错误

## 获取帮助

- 查看项目 [README.md](README.md)
- 搜索现有 Issues
- 在 Issues 中提问
- 加入社区讨论

## 许可证

通过向本项目提交代码，您同意您的贡献将根据项目的 Apache 2.0 许可证进行许可。 