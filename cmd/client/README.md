# Client 端

## 概述

Client 端是 Shell Executor MCP 系统的客户端程序，负责接收用户输入的 Shell 命令，并通过 MCP 协议发送到 Server 端执行，然后展示执行结果。

## 文件说明

- `main.go` - 客户端程序入口
- `cmd/` - 命令行子命令目录
  - `root.go` - 根命令定义和配置初始化
  - `run.go` - run 命令实现，包含主要的客户端逻辑

## 主要功能

1. **配置管理**
   - 支持从配置文件加载服务器列表
   - 支持通过命令行参数和环境变量配置
   - 支持多服务器配置，实现故障转移

2. **连接管理**
   - 自动尝试连接服务器列表中的第一个可用服务器
   - 支持连接超时控制
   - 连接失败时自动尝试下一个服务器

3. **命令执行**
   - 交互式命令行界面
   - 通过 MCP 协议调用 Server 的 `execute_command` 工具
   - 解析并展示聚合后的执行结果

4. **结果展示**
   - 支持解析 JSON 格式的聚合结果
   - 按组展示相同结果的节点
   - 显示执行状态、输出内容和错误信息

## 使用方法

```bash
# 使用配置文件启动
./client client_config.json

# 使用命令行参数启动
./client --server http://localhost:8080

# 使用环境变量启动
export MCP_SERVER=http://localhost:8080
./client
```

## 配置文件

客户端配置文件示例 (`client_config.json`):

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

## 更新记录

- 2026-01-23: 创建 README.md 文档
