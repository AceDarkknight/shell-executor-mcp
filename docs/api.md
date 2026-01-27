# API 接口文档：Shell Executor MCP

## 1. 概述
Server 遵循 MCP (Model Context Protocol) 规范，通过 HTTP SSE (Server-Sent Events) 暴露服务。主要的交互方式是通过 MCP 的 `CallTool` 请求。

## 2. MCP Tools

### 2.1 `execute_command`
在服务器集群上执行 Shell 命令。

- **Input Schema (JSON Schema)**:
  ```json
  {
    "type": "object",
    "properties": {
      "command": {
        "type": "string",
        "description": "需要执行的 Shell 命令。禁止包含高危操作。"
      }
    },
    "required": ["command"]
  }
  ```

- **Output**:
  返回一个 JSON 字符串，包含聚合后的执行结果。
  
  **示例 Output (Text)**:
  ```text
  [Group 1] Count: 98 | Status: Success
  Output: 
  v1.0.0
  Nodes: node-01, node-02, ... (98 nodes)
  
  [Group 2] Count: 1 | Status: Failed
  Error: connection refused
  Nodes: node-99
  
  [Group 3] Count: 1 | Status: Success
  Output:
  v1.0.1-beta
  Nodes: node-100
  ```

## 3. 配置文件

### 3.1 `client_config.json`
Client 端使用的配置文件。Client 会按顺序尝试连接服务器列表，实现简单的故障转移。

```json
{
  "servers": [
    {
      "name": "primary-01",
      "url": "http://192.168.1.10:8080/sse"
    },
    {
      "name": "backup-02",
      "url": "http://192.168.1.11:8080/sse"
    }
  ]
}
```

### 3.2 `server_config.json`
Server 端使用的配置文件。

```json
{
  "port": 8080,
  "node_name": "node-01",
  "peers": [
    "http://localhost:8081/sse",
    "http://localhost:8082/sse"
  ],
  "security": {
    "blacklist": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": [
      "rm\\s+-[a-zA-Z]*r[a-zA-Z]*\\s+/" 
    ]
  }
}
```

## 4. 错误码说明
由于 MCP 协议封装了底层错误，以下错误通常出现在 Tool 执行结果的 `content` 中或作为 MCP Protocol Error 返回。

- **SECURITY_VIOLATION**: 命令包含禁止的关键词或模式。
- **EXECUTION_ERROR**: Shell 命令执行失败（非 0 退出码）。
- **CLUSTER_PARTIAL_FAILURE**: 部分节点执行失败。
- **TIMEOUT**: 执行超时。

## 5. Go Client SDK API

为了方便外部 Go 程序集成，本项目提供了一个独立的客户端库 `pkg/mcpclient`。

### 5.1 核心类型

#### `Client`
封装了 MCP 连接管理、故障转移和命令执行逻辑。

```go
type Client struct {
    // 内部字段包含配置、连接状态等
}
```

#### `Result`
表示命令在集群上的执行结果。

```go
type Result struct {
    Summary string        // 简要汇总信息
    Groups  []ResultGroup // 分组后的详细结果
}

type ResultGroup struct {
    Count  int      // 该组包含的节点数量
    Status string   // 执行状态 (success/failed)
    Output string   // 标准输出内容
    Error  string   // 错误信息
    Nodes  []string // 属于该组的节点名称列表
}
```

### 5.2 初始化

#### `NewClient`
创建一个新的客户端实例。

```go
func NewClient(cfg *configs.ClientConfig, opts ...Option) (*Client, error)
```
- **cfg**: 客户端配置对象，包含服务器列表等信息。
- **opts**: 可选配置项（见下文）。

#### `LoadConfig`
从指定路径加载配置文件（通常由 `pkg/configs` 包提供）。

```go
// 位于 pkg/configs 包中
func LoadClientConfig(path string) (*configs.ClientConfig, error)
```

### 5.3 配置选项 (Options)

使用函数式选项模式进行配置。

- `WithServerURL(url string) Option`: 覆盖配置中的首选服务器 URL。
- `WithLogger(l *zap.Logger) Option`: 使用自定义的 Logger 实例。
- `WithTimeout(d time.Duration) Option`: 设置命令执行的超时时间。

### 5.4 客户端方法

#### `Connect`
连接到 MCP 服务器集群。会自动尝试配置列表中的可用服务器。

```go
func (c *Client) Connect(ctx context.Context) error
```

#### `Close`
关闭连接并释放资源。

```go
func (c *Client) Close() error
```

#### `ExecuteCommand`
在集群上执行 Shell 命令并返回结构化结果。

```go
func (c *Client) ExecuteCommand(ctx context.Context, cmd string) (*Result, error)
```

### 5.5 使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "shell-executor-mcp/pkg/configs"
    "shell-executor-mcp/pkg/mcpclient"
)

func main() {
    // 1. 加载配置
    cfg, err := configs.LoadClientConfig("client_config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. 创建客户端
    client, err := mcpclient.NewClient(cfg, mcpclient.WithTimeout(10*time.Second))
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // 3. 连接服务器
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }

    // 4. 执行命令
    result, err := client.ExecuteCommand(ctx, "uptime")
    if err != nil {
        log.Fatalf("Command execution failed: %v", err)
    }

    // 5. 处理结果
    fmt.Println("Summary:", result.Summary)
    for _, group := range result.Groups {
        fmt.Printf("Nodes: %v, Output: %s\n", group.Nodes, group.Output)
    }
}
```
