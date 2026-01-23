# 使用 Cobra 和 Viper 重构 CLI

## 1. 目标
重构服务端（Server）和客户端（Client）应用程序，使用 `spf13/cobra` 进行命令行接口管理，使用 `spf13/viper` 进行配置管理。这将实现标准的 CLI 模式，支持环境变量（例如 `MCP_PORT`），并提供灵活的配置方式。

## 2. 依赖项
- `github.com/spf13/cobra`
- `github.com/spf13/viper`

## 3. 架构设计

### 3.1 Server 端 (`cmd/server`)

#### Root Command: `k8s-mcp-server`

#### Flags (对应环境变量):
- `--port` / `MCP_PORT`: 监听端口
- `--cert` / `MCP_CERT`: TLS 证书文件路径
- `--key` / `MCP_KEY`: TLS 密钥文件路径
- `--insecure` / `MCP_INSECURE`: 是否使用不安全连接 (默认 false)
- `--token` / `MCP_TOKEN`: 安全令牌
- `--log-dir` / `MCP_LOG_DIR`: 日志目录
- `--config` / `MCP_CONFIG`: 配置文件路径 (默认 `server_config.json`)
- `--node-name` / `MCP_NODE_NAME`: 节点名称 (默认读取 `os.Hostname()`)
- `--log-level` / `MCP_LOG_LEVEL`: 日志级别

#### 配置文件 (Viper)
- **优先级顺序**：命令行标志 (Flag) > 环境变量 > 配置文件 > 默认值。
- **环境变量前缀**：`MCP_`

### 3.2 Client 端 (`cmd/client`)

#### Root Command: `k8s-mcp-client`

#### Flags (对应环境变量):
- `--server` / `MCP_CLIENT_SERVER`: 服务器地址
- `--token` / `MCP_CLIENT_TOKEN`: 连接令牌
- `--insecure-skip-verify` / `MCP_CLIENT_INSECURE_SKIP_VERIFY`: 跳过 TLS 验证
- `--log-dir` / `MCP_LOG_DIR`: 日志目录
- `--config` / `MCP_CLIENT_CONFIG`: 配置文件路径 (默认 `client_config.json`)
- `--log-level` / `MCP_LOG_LEVEL`: 日志级别

#### Interactive Mode
- 保持现有的 REPL 交互模式作为默认行为。

## 4. 实施步骤

### 第一步：依赖管理
- 添加 `github.com/spf13/cobra` 和 `github.com/spf13/viper`。

### 第二步：Server 端重构
1. 创建 `cmd/server/cmd/root.go`，定义 Root Command。
2. 使用 `viper` 绑定 Flags 和环境变量。
3. 移除 `kubeconfig` 相关参数。
4. 设置 `node-name` 的默认值为 `os.Hostname()`。
5. 将原 `main` 函数逻辑迁移到 Root Command 的 `Run` 函数中。
6. 在 `cmd/server/main.go` 中调用 `cmd.Execute()`。

### 第三步：Client 端重构
1. 创建 `cmd/client/cmd/root.go`，定义 Root Command。
2. 使用 `viper` 绑定 Flags 和环境变量。
3. 将原 `main` 函数逻辑迁移到 Root Command 的 `Run` 函数中。
4. 在 `cmd/client/main.go` 中调用 `cmd.Execute()`。

## 5. 预期效果
- 命令行参数解析更加健壮。
- 支持 `k8s-mcp-server --help` 查看详细帮助。
- 支持通过环境变量配置，便于容器化部署。
- 保留所有现有功能和逻辑。

## 6. 验证计划
1. 构建 Server 和 Client。
2. 使用 `--help` 检查参数说明。
3. 测试命令行参数启动。
4. 测试环境变量启动（不带命令行参数）。
