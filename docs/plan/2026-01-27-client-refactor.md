# 客户端重构计划

**日期:** 2026-01-27
**状态:** 草稿

## 1. 目标

重构 MCP 客户端实现，将核心逻辑从 CLI 命令 (`cmd/client`) 中解耦。目标是创建一个可复用的独立库 (`pkg/mcpclient`)，以便其他 Go 程序可以导入并使用它来与 Shell Executor MCP 集群进行交互。

## 2. 当前状态分析

- **耦合性:** 客户端逻辑目前完全驻留在 `cmd/client/cmd/run.go` 中。它将 MCP 连接建立、故障转移逻辑和交互式 Shell 循环与 Cobra 命令结构紧密绑定。
- **配置:** 配置加载依赖于 Viper (全局状态) 和 CLI 命令中的直接文件读取的混合方式。
- **复用性:** 外部程序无法在不导入 `cmd` 包或复制代码（这是一种反模式）的情况下复用客户端逻辑（例如：故障转移、结果解析）。
- **API:** 没有用于以编程方式实例化客户端、设置选项或执行命令的公共 API。

## 3. 建议架构

### 3.1 目录结构

我们将引入一个新的公共包 `pkg/mcpclient` 来存放可复用的逻辑。

```text
d:/code/shell-executor-mcp/
├── cmd/
│   └── client/
│       └── cmd/
│           └── run.go       # 重构为 pkg/mcpclient 的瘦包装器
├── pkg/
│   ├── configs/             # 现有的配置定义
│   │   └── client_config.go
│   └── mcpclient/           # 新增: 公共客户端库
│       ├── client.go        # 核心 Client 结构体和方法
│       ├── options.go       # 函数式选项模式
│       └── result.go        # 结果解析和显示辅助函数
```

### 3.2 API 设计

#### `pkg/mcpclient`

**类型:**

```go
// Client 封装了 MCP 连接和逻辑
type Client struct {
    config     *configs.ClientConfig
    mcpClient  *mcp.Client
    session    *mcp.ClientSession
    // ... 内部字段
}

// Option 定义函数式选项模式
type Option func(*Client)

// Result 代表来自集群的聚合输出
type Result struct {
    Summary string
    Groups  []ResultGroup
}

type ResultGroup struct {
    Count  int
    Status string
    Output string
    Error  string
    Nodes  []string
}
```

**函数:**

```go
// NewClient 使用配置和选项创建一个新的客户端实例
// 必须提供 config 参数。如果 config 为 nil 或包含无效字段（例如 Servers 列表为空），将返回错误。
func NewClient(cfg *configs.ClientConfig, opts ...Option) (*Client, error)

// Options (选项)
func WithServerURL(url string) Option
func WithLogger(l *zap.Logger) Option
func WithTimeout(d time.Duration) Option

// Methods (方法)
func (c *Client) Connect(ctx context.Context) error
func (c *Client) Close() error
func (c *Client) ExecuteCommand(ctx context.Context, cmd string) (*Result, error)
```

## 4. 实施步骤

### 步骤 1: 创建 `pkg/mcpclient` 包
- 创建目录 `pkg/mcpclient`。
- 在 `client.go` 和 `options.go` 中定义 `Client` 结构体和 `Option` 类型。
- 将 `mcp.Client` 初始化逻辑从 `cmd/client` 移动到 `NewClient`。
- 在 `NewClient` 中添加配置验证逻辑（例如：检查 Servers 列表是否非空）。

### 步骤 2: 实现连接和故障转移逻辑
- 将 `connectToAvailableServer` 逻辑从 `cmd/client/cmd/run.go` 移动到 `pkg/mcpclient/client.go` 作为 `Connect` 方法。
- 调整代码以使用 `Client` 结构体的内部状态。

### 步骤 3: 实现命令执行
- 将 `CallTool` ("execute_command") 逻辑封装到 `ExecuteCommand` 中。
- 实现响应解析逻辑（目前在 `displayResult` 中），作为 `pkg/mcpclient` 中的方法或辅助函数，并返回结构化的 `Result` 对象。

### 步骤 4: 重构 CLI (`cmd/client`)
- 修改 `cmd/client/cmd/run.go` 以导入 `pkg/mcpclient`。
- 将内联连接和执行逻辑替换为调用 `mcpclient.NewClient`、`client.Connect` 和 `client.ExecuteCommand`。
- CLI 仍然负责：
  - 解析标志/配置（使用 `viper` 和 `pkg/configs`）。
  - 交互式循环 (`reader.ReadString`)。
  - 格式化 `Result` 对象以便在终端显示。

### 步骤 5: 集成与验证
- 验证 CLI 是否仍像以前一样工作（向后兼容性）。
- 确保通过文件和标志加载配置的功能仍然有效。

## 5. 验证计划
- **构建检查:** 确保 `go build ./cmd/client` 成功。
- **手动测试:** 针对运行中的服务器集群运行重构后的客户端。
- **单元测试:** 为 `pkg/mcpclient` 添加基本的单元测试（如果可能，模拟 MCP 服务器，或测试不需要实时连接的逻辑）。
