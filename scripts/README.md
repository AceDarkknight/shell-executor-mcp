# Scripts Directory

本目录包含用于测试和验证 Shell Executor MCP 系统功能的脚本。

## 文件说明

### `test_single_node.sh`
单节点测试脚本，用于测试单个服务器节点的基本功能。

**测试内容：**
- 基本命令执行（echo）
- 错误处理（exit 1）
- 安全检查（黑名单命令拦截）
- 系统命令（hostname）

**使用方法：**
```bash
bash scripts/test_single_node.sh
```

**前提条件：**
- Go 1.25.1 或更高版本
- 端口 8080 可用

### `test_cluster.sh`
集群测试脚本，用于测试多个服务器节点的集群功能。

**测试内容：**
- 集群命令分发（echo）
- 多节点结果聚合（hostname）
- 集群安全检查（黑名单命令拦截）
- 多节点并发执行（date）

**使用方法：**
```bash
bash scripts/test_cluster.sh
```

**前提条件：**
- Go 1.25.1 或更高版本
- 端口 8080、8081、8082 可用

## 注意事项

1. **Windows 用户**：这些脚本是为 Unix/Linux 环境设计的。在 Windows 上，您需要使用 Git Bash 或 WSL (Windows Subsystem for Linux) 来运行这些脚本。

2. **端口占用**：在运行测试脚本之前，请确保所需的端口没有被其他应用程序占用。

3. **清理**：测试脚本会自动清理生成的临时文件和日志文件。

4. **日志文件**：如果测试失败，脚本会显示相关的日志文件内容以帮助调试。

## 手动测试

如果您想手动测试系统，可以按照以下步骤操作：

### 启动单节点服务器

```bash
# 构建服务器
cd cmd/server
go build -o server.exe main.go
cd ../..

# 启动服务器
cmd/server/server.exe server_config.json
```

### 启动客户端

```bash
# 构建客户端
cd cmd/client
go build -o client.exe main.go
cd ../..

# 启动客户端
cmd/client/client.exe client_config.json
```

### 启动集群

```bash
# 构建服务器
cd cmd/server
go build -o server.exe main.go
cd ../..

# 启动三个节点（在不同的终端中）
cmd/server/server.exe node1_config.json
cmd/server/server.exe node2_config.json
cmd/server/server.exe node3_config.json

# 启动客户端连接到任意节点
cmd/client/client.exe client_config.json
```

## 故障排除

### 服务器启动失败
- 检查端口是否被占用
- 检查配置文件格式是否正确
- 查看服务器日志获取详细错误信息

### 客户端连接失败
- 确认服务器正在运行
- 检查客户端配置文件中的服务器地址是否正确
- 确认网络连接正常

### 集群测试失败
- 确认所有节点都在运行
- 检查节点之间的网络连接
- 确认所有节点使用相同的 cluster_token
