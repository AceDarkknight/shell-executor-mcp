# Shell Executor MCP System

基于 MCP (Model Context Protocol) 的分布式 Shell 命令执行系统，支持集群分发和安全控制。

## 项目概述

本项目实现了一个支持集群分发和安全控制的 Shell 命令执行系统。Client 端可以连接到任意 Server 节点，该节点将作为 Coordinator 自动将命令分发给集群中的所有节点，并聚合返回结果。

### 主要特性

- **MCP 协议支持**：基于 `github.com/modelcontextprotocol/go-sdk` 实现 MCP Server 标准接口
- **集群分发**：支持多节点并发执行命令，自动聚合结果
- **安全控制**：内置黑名单机制，拦截高危命令
- **故障转移**：Client 端支持多服务器配置，自动故障转移
- **结果聚合**：相同结果的节点自动合并，减少网络传输
- **结构化日志**：使用 `zap` 进行结构化日志记录，支持日志轮转和级别控制

## 项目结构

```
shell-executor-mcp/
├── cmd/                    # 可执行程序
│   ├── client/             # MCP 客户端
│   │   └── main.go
│   └── server/             # MCP 服务器
│       └── main.go
├── internal/               # 内部模块
│   ├── config/            # 配置管理
│   │   └── config.go
│   ├── dispatch/          # 集群分发器
│   │   └── dispatcher.go
│   ├── executor/          # 命令执行器
│   │   └── executor.go
│   ├── logger/            # 日志管理
│   │   └── logger.go
│   └── security/          # 安全卫士
│       └── guard.go
├── pkg/                   # 公共包
│   └── configs/           # 客户端配置
│       └── client_config.go
├── docs/                  # 文档
│   ├── requirements.md     # 需求文档
│   ├── architecture.md     # 架构文档
│   ├── api.md            # API 接口文档
│   └── plan/            # 开发计划
└── scripts/               # 测试脚本
    ├── test_single_node.sh
    ├── test_cluster.sh
    └── README.md
```

## 快速开始

### 前置条件

- Go 1.25.1 或更高版本
- 端口 8080（单节点）或 8080-8082（集群）可用

### 构建项目

```bash
# 构建服务器
cd cmd/server
go build -o server.exe main.go
cd ../..

# 构建客户端
cd cmd/client
go build -o client.exe main.go
cd ../..
```

### 配置文件

#### 服务器配置 (`server_config.json`)

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
    "log_dir": ".",
    "max_size": 100,
    "max_backups": 3,
    "max_age": 28,
    "compress": false
  }
}
```

**日志配置说明**：
- `level`: 日志级别，可选值为 `debug`, `info`, `warn`, `error`，默认为 `info`
- `log_dir`: 日志文件目录，默认为当前目录
- `max_size`: 单个日志文件最大大小（MB），默认为 100MB
- `max_backups`: 保留的旧日志文件最大数量，默认为 3 个
- `max_age`: 保留旧日志文件的最大天数，默认为 28 天
- `compress`: 是否压缩旧日志文件，默认为 `false`

#### 客户端配置 (`client_config.json`)

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
    "log_dir": ".",
    "max_size": 100,
    "max_backups": 3,
    "max_age": 28,
    "compress": false
  }
}
```

**日志配置说明**：
- `level`: 日志级别，可选值为 `debug`, `info`, `warn`, `error`，默认为 `info`
- `log_dir`: 日志文件目录，默认为当前目录
- `max_size`: 单个日志文件最大大小（MB），默认为 100MB
- `max_backups`: 保留的旧日志文件最大数量，默认为 3 个
- `max_age`: 保留旧日志文件的最大天数，默认为 28 天
- `compress`: 是否压缩旧日志文件，默认为 `false`

### 启动服务器

```bash
# 单节点模式
cmd/server/server.exe server_config.json

# 集群模式（在不同终端中启动多个节点）
cmd/server/server.exe node1_config.json
cmd/server/server.exe node2_config.json
cmd/server/server.exe node3_config.json
```

### 启动客户端

```bash
cmd/client/client.exe client_config.json
```

### 使用示例

```
Shell Executor MCP Client
Type 'exit' or 'quit' to exit
----------------------------------------
> echo Hello World
Executing: echo Hello World
Server response:
Summary: Executed on 1 nodes, 1 groups found

[Group 1] Count: 1 | Status: success
Output:
Hello World

Nodes: node-01

----------------------------------------
> hostname
Executing: hostname
Server response:
Summary: Executed on 3 nodes, 1 groups found

[Group 1] Count: 3 | Status: success
Output:
my-hostname

Nodes: node-01, node-02, node-03

----------------------------------------
> rm -rf /
Executing: rm -rf /
Server returned an error:
  security violation: command 'rm' is blacklisted

----------------------------------------
```

## 测试

项目提供了自动化测试脚本：

### 单节点测试

```bash
bash scripts/test_single_node.sh
```

### 集群测试

```bash
bash scripts/test_cluster.sh
```

详细说明请参考 [`scripts/README.md`](scripts/README.md)。

## 架构设计

系统采用去中心化架构，任何接收到 Client 请求的 Server 节点自动承担 Coordinator 角色。

### 通信协议

- **Client → Server**：MCP over HTTP (SSE)
- **Server → Server**：Internal HTTP JSON API

### 核心流程

1. Client 连接到任意 Server 节点
2. 该节点成为 Coordinator
3. Coordinator 并发执行本地命令和分发到 Peer 节点
4. Coordinator 聚合所有结果
5. Coordinator 返回聚合结果给 Client

详细的架构设计请参考 [`docs/architecture.md`](docs/architecture.md)。

## 安全特性

- **黑名单机制**：拦截黑名单中的命令
- **正则匹配**：支持正则表达式匹配危险参数
- **Token 鉴权**：集群内部通信使用 Token 鉴权

## API 文档

详细的 API 文档请参考 [`docs/api.md`](docs/api.md)。

## 开发计划

开发计划和进度请参考 [`docs/plan/2026-01-19-implementation-plan.md`](docs/plan/2026-01-19-implementation-plan.md)。

## 技术栈

- **语言**：Go 1.25.1
- **MCP SDK**：`github.com/modelcontextprotocol/go-sdk v1.2.0`
- **日志库**：`go.uber.org/zap v1.27.1`（结构化日志）
- **日志轮转**：`gopkg.in/natefinch/lumberjack.v2 v2.2.1`（日志文件轮转）
- **传输协议**：HTTP SSE (Server-Sent Events)

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request。

## 联系方式

如有问题或建议，请通过 GitHub Issues 联系。
