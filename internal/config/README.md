# 配置管理模块 (config)

## 概述

配置管理模块负责加载、解析和管理服务器端的配置信息，包括监听端口、节点名称、集群节点列表、安全配置和日志配置等。

## 文件说明

- `config.go` - 配置结构定义和配置加载逻辑

## 数据结构

### ServerConfig

服务器配置结构，包含以下字段：

- `Port` - 监听端口
- `NodeName` - 节点名称
- `Peers` - 集群中其他节点的地址列表
- `Security` - 安全配置
- `ClusterToken` - 集群内部通信Token
- `LogConfig` - 日志配置
- `mu` - 读写锁，用于保护 Peers 的并发修改

### SecurityConfig

安全配置结构，包含以下字段：

- `BlacklistedCommands` - 黑名单命令列表
- `DangerousArgsRegex` - 危险参数正则表达式列表

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
   - 从 JSON 文件加载配置
   - 解析配置内容到结构体

2. **线程安全操作**
   - 使用读写锁保护 Peers 列表的并发访问
   - `GetPeers()` - 线程安全地获取 Peers 列表
   - `SetPeers()` - 线程安全地设置 Peers 列表
   - `AddPeer()` - 线程安全地添加一个 Peer

3. **配置持久化**
   - `Save()` - 将当前配置保存到指定路径

## 使用示例

```go
// 加载配置
cfg, err := config.LoadServerConfig("server_config.json")
if err != nil {
    log.Fatal(err)
}

// 获取 Peers 列表（线程安全）
peers := cfg.GetPeers()

// 添加新节点（线程安全）
cfg.AddPeer("http://new-node:8080")

// 保存配置
err = cfg.Save("server_config.json")
```

## 配置文件示例

```json
{
  "port": 8080,
  "node_name": "node-01",
  "peers": [
    "http://localhost:8081",
    "http://localhost:8082"
  ],
  "cluster_token": "your-cluster-token",
  "security": {
    "blacklisted_commands": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": [
      "rm\\s+-[a-zA-Z]*r[a-zA-Z]*\\s+/"
    ]
  },
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

## 更新记录

- 2026-01-23: 创建 README.md 文档
