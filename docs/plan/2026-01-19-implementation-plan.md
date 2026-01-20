# 开发计划 (2026-01-19)

## 目标
实现一个支持集群分发和安全控制的 Shell Executor MCP 系统。

## 步骤

### 第一阶段：项目初始化与 Server 基础功能
- [ ] 初始化 Go 项目结构 (cmd, internal, pkg)。
- [ ] 定义配置结构体 (`ServerConfig`, `SecurityConfig`) 并实现加载逻辑。
- [ ] 实现 `Security Guard` 模块，编写单元测试验证黑名单拦截逻辑。
- [ ] 实现 `Command Executor` 模块，支持本地命令执行。
- [ ] 集成 `go-sdk`，实现基本的 MCP Server，注册 `execute_command` Tool。
- [ ] 验证单个 Server 的 MCP 接口（使用简单的 Client 或 Curl/Postman 模拟）。

### 第二阶段：Client 实现
- [ ] 实现 Client 端的配置加载 (`ClientConfig`)。
- [ ] 使用 `go-sdk/client` 实现与 Server 的 SSE 连接。
- [ ] 实现交互式命令行界面 (CLI)，接收用户输入并调用 Server Tool。
- [ ] 格式化展示 Server 返回的结果。

### 第三阶段：集群分发功能
- [ ] 扩展 Server 端功能，增加 `Cluster Dispatcher` 模块。
- [ ] 实现 Server 间的 Client 连接池或动态连接创建。
- [ ] 实现 Scatter-Gather 逻辑：主节点收到请求后，并发分发给 Peers。
- [ ] 聚合结果并返回。

### 第四阶段：测试与文档
- [ ] 编写集成测试：模拟 3 个节点的集群，验证分发逻辑。
- [ ] 完善 README.md，包含使用说明和配置示例。
- [ ] 进行安全测试，尝试绕过黑名单（自测）。

## 预期效果
- 启动 2-3 个 Server 实例组成集群。
- Client 连接其中一个 Server，输入 `hostname`。
- Client 收到所有 Server 的 hostname 输出。
- Client 输入 `rm -rf /`，收到安全拦截警告。
