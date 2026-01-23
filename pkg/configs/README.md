# 公共配置包 (configs)

## 概述

公共配置包提供客户端和服务端共享的配置数据结构和加载逻辑。该包位于 `pkg` 目录下，表示可以被其他项目引用的公共包。

## 文件说明

- `client_config.go` - 客户端配置结构和加载逻辑

## 数据结构

### ClientConfig

客户端配置结构，包含以下字段：

- `Servers` - 服务器列表配置
- `Log` - 日志配置

### ServerConfig

服务器配置项，包含以下字段：

- `Name` - 服务器名称
- `URL` - 服务器地址

### LogConfig

日志配置结构，包含以下字段：

- `Level` - 日志级别: debug, info, warn, error
- `LogDir` - 日志文件目录
- `MaxSize` - 单个日志文件最大大小（MB）
- `MaxBackups` - 保留的旧日志文件最大数量
- `MaxAge` - 保留旧日志文件的最大天数
- `Compress` - 是否压缩旧日志文件

## 主要功能

1. **配置加载**
   - `LoadClientConfig()` - 从 JSON 文件加载客户端配置
   - 解析配置内容到结构体

## 使用示例

```go
// 加载客户端配置
cfg, err := configs.LoadClientConfig("client_config.json")
if err != nil {
    log.Fatal(err)
}

// 初始化 logger
err = logger.InitLogger(cfg.Log, "client.log")
if err != nil {
    log.Fatal(err)
}

// 遍历服务器列表
for _, server := range cfg.Servers {
    fmt.Printf("Server: %s (%s)\n", server.Name, server.URL)
}
```

## 配置文件示例

### 客户端配置文件 (`client_config.json`)

```json
{
  "servers": [
    {
      "name": "primary-01",
      "url": "http://localhost:8080"
    },
    {
      "name": "backup-02",
      "url": "http://localhost:8081"
    }
  ],
  "log": {
    "level": "info",
    "log_dir": "logs",
    "max_size": 100,
    "max_backups": 3,
    "max_age": 28,
    "compress": false
  }
}
```

## 设计考虑

### 为什么放在 pkg 目录？

- `pkg` 目录中的包可以被其他项目引用
- 配置结构可能在多个项目中复用
- 遵循 Go 项目的标准布局规范

### 与 internal/config 的区别

- `pkg/configs` - 客户端配置，可被外部引用
- `internal/config` - 服务器配置，仅限内部使用

## 更新记录

- 2026-01-23: 创建 README.md 文档
