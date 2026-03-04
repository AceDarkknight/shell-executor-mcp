# 计划：修复 Client 连接超时/断连问题及并发安全性改进

## 1. 问题描述
根据之前的分析报告，Client 在长时间空闲后会出现连接超时或断连的问题。
- **根本原因**：TCP 长连接在空闲时被中间网络设备（如防火墙、负载均衡器）或 Server 端断开。
- **现状**：
    - Client 缺乏心跳保活机制，无法维持连接活跃。
    - Client 在检测到连接断开时（如执行命令失败）缺乏自动重连逻辑，导致用户必须重启 Client。
    - 当前的重连和心跳实现在高并发场景下存在竞态风险。

## 2. 实现方案
遵循“代码改动最小”原则，在 `pkg/mcpclient` 中增加心跳、自动重连机制，并强化并发安全性。

### 2.1 心跳机制 (Heartbeat)
- 在 `mcpclient.Client` 中增加一个后台协程（goroutine）。
- 定期（如每 30 秒）向 Server 发送一个轻量级请求（如 `ListTools`）。
- 通过维持数据传输来防止 TCP 连接因空闲被断开。

### 2.2 自动重连逻辑 (Auto-reconnect)
- 在 `ExecuteCommand` 方法中增加错误捕获。
- 如果捕获到连接相关的错误（如 `io.EOF`、`broken pipe` 或 session 为空），自动尝试调用 `Connect()` 重新建立连接。
- 重连成功后，自动重试当前的命令请求。
- 增加最大重试次数限制（如 3 次），避免无限死循环。

## 3. 并发安全性改进
针对可能出现的竞态条件（Race Condition）和重连风暴（Reconnection Storm），进行以下改进：

- **扩大 Mutex 保护范围**：使用 `mu sync.Mutex` 保护 `Connect`、`Close` 方法，以及对 `client`、`session`、`cancelHeartbeat` 字段的所有读写操作。
- **引入重连状态标记**：在 `Client` 结构体中增加 `isConnecting bool` 字段，配合锁机制确保同一时间只有一个重连操作在进行，防止多个请求同时触发重连导致“重连风暴”。
- **优化心跳生命周期管理**：
    - 确保 `startHeartbeat` 只有在成功建立连接且持有锁的情况下被调用。
    - 在启动新心跳前，必须先停止并清理旧的心跳协程及其 context。
    - 心跳 context 的更新必须在锁保护下进行。

## 4. 预期效果
- **稳定性**：Client 可以在长时间空闲后依然保持连接可用。
- **鲁棒性**：即使网络出现瞬时抖动导致断连，Client 也能在用户感知不到的情况下自动恢复连接。
- **并发安全**：在多协程并发调用 `ExecuteCommand` 时，能够安全地处理重连，不会出现资源泄露或状态冲突。

## 5. 修改步骤

### 5.1 `pkg/mcpclient/client.go`
1.  **结构体定义更新**：
    - 增加 `isConnecting bool` 用于标记重连状态。
    - 确保 `mu sync.Mutex` 保护范围覆盖所有状态字段。
2.  **重构 `Connect` 方法**：
    - 进入方法后首先加锁。
    - 检查 `isConnecting`，如果正在连接则直接返回或等待（此处建议直接返回，由外部重试逻辑处理）。
    - 设置 `isConnecting = true`，并在退出时（defer）重置。
    - 连接成功后，更新 `client`、`session` 并调用 `startHeartbeat`（在锁保护内）。
3.  **重构 `Close` 方法**：
    - 加锁保护。
    - 停止心跳（`cancelHeartbeat()`）并置为空。
    - 关闭 `session` 并置为空。
4.  **优化 `startHeartbeat` 方法**：
    - 确保该方法在持有锁的情况下被调用，或在内部正确处理锁。
    - 改进退出机制，确保协程能可靠停止。
5.  **修改 `ExecuteCommand` 方法**：
    - 保持现有的重试逻辑，但在调用 `Connect` 时确保并发安全。
    - 检查 session 状态时始终使用锁。

### 5.2 `pkg/configs/client_config.go` (可选)
- 增加 `HeartbeatInterval` 配置项（默认 30s）。
- 增加 `MaxRetries` 配置项（默认 3）。

## 6. 验证计划
1.  **长连接测试**：启动 Client 和 Server，空闲 10 分钟后再次执行命令，确认是否正常。
2.  **模拟断连测试**：在 Client 运行期间手动重启 Server，然后执行命令，确认 Client 是否能自动重连并执行成功。
3.  **并发重连测试**：使用多个协程并发调用 `ExecuteCommand`，同时手动断开 Server 连接，观察是否只触发了一次有效的重连，且所有协程最终都能恢复执行。
