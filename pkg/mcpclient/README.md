# mcpclient

本包提供了 MCP (Model Context Protocol) 客户端的 Go 语言 SDK，允许其他 Go 程序方便地调用 MCP 服务。

## 功能

- 封装 MCP 客户端和会话管理
- 提供简洁的高级 API
- 支持配置文件和配置参数初始化
- 支持函数式选项模式（Functional Options Pattern）进行个性化配置
- 支持自定义日志记录器
- 支持自定义 HTTP 客户端和请求头
- 自动解析和格式化命令执行结果

## 安装

```bash
go get shell-executor-mcp/pkg/mcpclient
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"

    "shell-executor-mcp/internal/logger"
    "shell-executor-mcp/pkg/configs"
    "shell-executor-mcp/pkg/mcpclient"
)

func main() {
    // 1. 加载配置
    cfg, err := configs.LoadClientConfig("client_config.json")
    if err != nil {
        panic(err)
    }

    // 2. 初始化日志
    logCfg := cfg.Log.ToLoggerConfig()
    if err := logger.InitLogger(logCfg, "client.log"); err != nil {
        panic(err)
    }
    defer logger.Sync()

    // 3. 创建客户端
    client, err := mcpclient.NewClient(cfg)
    if err != nil {
        panic(err)
    }

    // 4. 连接到服务器
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        panic(err)
    }
    defer client.Close()

    // 5. 执行命令
    result, err := client.ExecuteCommand(ctx, "ls -la")
    if err != nil {
        panic(err)
    }

    // 6. 处理结果
    fmt.Println(result.String())
}
```

### 使用配置参数

```go
// 直接使用配置参数创建客户端
cfg := &configs.ClientConfig{
    Servers: []configs.ServerConfig{
        {
            Name: "local-server",
            URL:  "http://127.0.0.1:8090",
        },
    },
    Log: configs.LogConfig{
        Level:  "info",
        LogDir: "logs",
    },
}

client, err := mcpclient.NewClient(cfg)
```

### 使用选项模式

```go
import (
    "net/http"
    "time"
)

// 创建客户端并使用选项进行个性化配置
client, err := mcpclient.NewClient(cfg,
    mcpclient.WithTimeout(60*time.Second),  // 设置超时
    mcpclient.WithHeader("X-Custom-Header", "value"),  // 添加请求头
    mcpclient.WithServerURL("http://custom-server:8090"),  // 覆盖服务器地址
)
```

### 使用自定义 HTTP 客户端

```go
// 创建自定义 HTTP 客户端
customHTTPClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

client, err := mcpclient.NewClient(cfg,
    mcpclient.WithHTTPClient(customHTTPClient),
)
```

### 使用自定义日志记录器

```go
// 实现自定义日志记录器
type MyLogger struct{}

func (l *MyLogger) Debugf(template string, args ...interface{}) {
    fmt.Printf("[DEBUG] "+template+"\n", args...)
}

func (l *MyLogger) Infof(template string, args ...interface{}) {
    fmt.Printf("[INFO] "+template+"\n", args...)
}

func (l *MyLogger) Warnf(template string, args ...interface{}) {
    fmt.Printf("[WARN] "+template+"\n", args...)
}

func (l *MyLogger) Errorf(template string, args ...interface{}) {
    fmt.Printf("[ERROR] "+template+"\n", args...)
}

// 使用自定义日志记录器
myLogger := &MyLogger{}
client, err := mcpclient.NewClient(cfg,
    mcpclient.WithLogger(myLogger),
)
```

## API 文档

### 核心类型

#### Client

客户端的主要结构体，封装了 MCP 客户端和会话。

**方法：**
- `Connect(ctx context.Context) error` - 连接到服务器
- `Close() error` - 关闭连接
- `ExecuteCommand(ctx context.Context, command string) (*Result, error)` - 执行命令
- `GetSession() *mcp.ClientSession` - 获取底层会话（高级用法）
- `GetClient() *mcp.Client` - 获取底层客户端（高级用法）
- `GetConfig() *configs.ClientConfig` - 获取配置

#### Result

命令执行结果的结构体。

**方法：**
- `GetTextContents() []string` - 获取所有文本内容
- `GetAggregatedResults() []*AggregatedResult` - 获取所有聚合结果
- `String() string` - 获取结果的字符串表示

### 初始化函数

#### NewClient

```go
func NewClient(cfg *configs.ClientConfig, opts ...Option) (*Client, error)
```

创建一个新的 MCP 客户端。

**参数：**
- `cfg`: 客户端配置（必需）
- `opts`: 可选配置参数

**返回：**
- `*Client`: 客户端实例
- `error`: 错误信息

**错误：**
- 如果 `cfg` 为 nil，返回错误
- 如果 `cfg.Servers` 为空，返回错误
- 如果服务器配置无效，返回错误

### 选项函数

- `WithLogger(l Logger) Option` - 设置自定义日志记录器
- `WithTimeout(timeout time.Duration) Option` - 设置超时时间
- `WithHTTPClient(client *http.Client) Option` - 设置自定义 HTTP 客户端
- `WithHeaders(headers map[string]string) Option` - 设置请求头
- `WithHeader(key, value string) Option` - 添加单个请求头
- `WithServerURL(url string) Option` - 覆盖服务器地址

### 配置加载

#### LoadClientConfig

```go
func LoadClientConfig(path string) (*configs.ClientConfig, error)
```

从文件加载客户端配置。

**参数：**
- `path`: 配置文件路径

**返回：**
- `*configs.ClientConfig`: 配置对象
- `error`: 错误信息

## 配置文件格式

```json
{
  "servers": [
    {
      "name": "local-node",
      "url": "http://127.0.0.1:8090"
    }
  ],
  "log": {
    "level": "debug",
    "log_dir": "logs/client",
    "max_size": 100,
    "max_backups": 3,
    "max_age": 28,
    "compress": false
  }
}
```

## 注意事项

1. **配置验证**：`NewClient` 会验证配置参数，确保服务器列表不为空且每个服务器配置有效。
2. **连接管理**：使用 `Connect()` 连接后，记得调用 `Close()` 关闭连接，或者使用 `defer` 确保资源释放。
3. **上下文使用**：建议使用 `context.Background()` 或带有超时的 `context.WithTimeout()`。
4. **错误处理**：所有方法都可能返回错误，建议进行适当的错误处理。

## 许可证

请参考项目根目录的 LICENSE 文件。
