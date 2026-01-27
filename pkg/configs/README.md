# configs

本包提供客户端配置相关的结构体和加载函数。

## 功能

- 定义客户端配置结构体
- 定义服务器配置结构体
- 定义日志配置结构体
- 提供从文件加载配置的方法

## 使用示例

```go
import "shell-executor-mcp/pkg/configs"

// 从文件加载配置
cfg, err := configs.LoadClientConfig("client_config.json")
if err != nil {
    log.Fatal(err)
}

// 访问配置
for _, server := range cfg.Servers {
    fmt.Printf("Server: %s, URL: %s\n", server.Name, server.URL)
}

// 转换日志配置
logCfg := cfg.Log.ToLoggerConfig()
```

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
