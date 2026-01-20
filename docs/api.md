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
