# 集群分发器模块 (dispatch)

## 概述

集群分发器模块负责将命令分发给集群中的所有节点，并聚合所有节点的执行结果。该模块实现了去中心化的协调机制，任何接收到 Client 请求的 Server 节点都可以作为 Coordinator 进行命令分发。

## 文件说明

- `dispatcher.go` - 分发器实现，包含分发和聚合逻辑

## 数据结构

### Dispatcher

分发器结构，包含以下字段：

- `peers` - 集群中其他节点的地址列表
- `token` - 集群内部通信Token
- `httpClient` - HTTP客户端，用于向其他节点发送请求

### NodeResult

单个节点的执行结果，包含以下字段：

- `NodeName` - 节点名称或地址
- `Status` - 执行状态: success, failed, timeout
- `Output` - 标准输出
- `Error` - 错误信息

### AggregatedGroup

聚合后的结果组，包含以下字段：

- `Output` - 输出内容
- `Error` - 错误信息
- `Status` - 执行状态
- `Nodes` - 属于该组的节点名称列表
- `Count` - 节点数量

### DispatchRequest

分发请求的 Body 结构，包含以下字段：

- `Cmd` - 要执行的命令

### DispatchResponse

分发响应的 Body 结构，包含以下字段：

- `ExitCode` - 退出码
- `Output` - 标准输出
- `Error` - 错误信息

## 主要功能

1. **命令分发**
   - 并发执行本地命令
   - 并发向所有 Peer 节点分发命令
   - 设置请求超时（默认5秒）

2. **结果聚合**
   - 收集所有节点的执行结果
   - 按输出内容进行分组（使用 SHA256 计算指纹）
   - 将相同输出的节点合并，减少网络传输量

3. **内部通信**
   - 通过 HTTP JSON API 与其他节点通信
   - 支持 Token 鉴权（通过 `X-Cluster-Token` Header）

## 算法说明

### 分发算法 (Scatter)

1. 创建 WaitGroup 和结果通道
2. 启动一个 goroutine 执行本地命令
3. 为每个 Peer 节点启动一个 goroutine 发送请求
4. 等待所有 goroutine 完成

### 聚合算法 (Gather & Compress)

1. 遍历所有节点结果
2. 对每个结果计算指纹（SHA256(Output + Error + Status)）
3. 使用 Map 按指纹分组
4. 将 Map 转换为 Slice 返回

## 使用示例

```go
// 创建分发器
dispatcher := dispatch.NewDispatcher(peers, "cluster-token")

// 分发命令并获取聚合结果
groups, summary := dispatcher.Dispatch(executor, "node-01", "echo Hello World")

// 遍历结果组
for _, group := range groups {
    fmt.Printf("Group: %d nodes, Status: %s\n", group.Count, group.Status)
    if group.Output != "" {
        fmt.Printf("Output: %s\n", group.Output)
    }
}
```

## 性能考虑

- 使用 goroutine 并发执行，提高效率
- 设置超时防止长尾节点阻塞
- 结果聚合使用哈希分组，减少网络传输
- 对于大规模集群（如100+节点），建议限制并发数

## 更新记录

- 2026-01-23: 创建 README.md 文档
