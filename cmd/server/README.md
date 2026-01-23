# Server 端

## 概述

Server 端是 Shell Executor MCP 系统的服务器程序，负责接收 Client 的命令请求，执行本地命令，并将命令分发给集群中的其他节点，最后聚合所有节点的执行结果返回给 Client。

## 文件说明

- `main.go` - 服务器程序入口
- `cmd/` - 命令行子命令目录
  - `root.go` - 根命令定义和配置初始化
  - `run.go` - run 命令实现，包含服务器启动和HTTP处理逻辑
  - `tools.go` - MCP工具注册和处理逻辑

## 主要功能

1. **MCP 协议支持**
   - 基于 `github.com/modelcontextprotocol/go-sdk` 实现 MCP Server 标准接口
   - 通过 HTTP SSE (Server-Sent Events) 暴露服务
   - 注册 `execute_command` 工具供 Client 调用

2. **命令执行**
   - 在本地 Shell 环境中执行接收到的命令
   - 支持超时控制，防止长时间阻塞
   - 捕获标准输出和标准错误

3. **集群分发**
   - 作为 Coordinator 将命令分发给集群中的其他节点
   - 并发执行本地命令和分发到 Peer 节点
   - 聚合所有节点的执行结果
   - 按输出内容分组，减少网络传输量

4. **安全控制**
   - 执行前对命令进行安全扫描
   - 拦截黑名单中的高危命令
   - 支持正则表达式匹配危险参数

5. **内部 API**
   - `POST /internal/exec` - 接收其他节点的执行请求
   - `POST /internal/join` - 处理新节点加入集群的请求
   - `POST /internal/sync` - 处理节点列表同步请求

6. **集群管理**
   - 支持节点动态加入
   - 支持节点列表同步
   - 支持配置持久化

## 使用方法

```bash
# 使用配置文件启动
./server server_config.json

# 使用命令行参数启动
./server --port 8080 --node-name node-01

# 使用环境变量启动
export MCP_PORT=8080
export MCP_NODE_NAME=node-01
./server
```

## 配置文件

服务器配置文件示例 (`server_config.json`):

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

## 集群部署

在集群模式下，每个节点都需要配置其他节点的地址：

```bash
# 节点1
./server node1_config.json

# 节点2
./server node2_config.json

# 节点3
./server node3_config.json
```

## 更新记录

- 2026-01-23: 创建 README.md 文档
