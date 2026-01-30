# Server 启动脚本及 Systemd 集成计划

本计划旨在创建一个用于启动 Shell Executor MCP Server 的 Shell 脚本，并提供 Systemd 服务集成方案，以满足自动化部署和后台管理的需求。

## 1. 现有启动参数分析

通过分析 `cmd/server/cmd/root.go` 和 `cmd/server/cmd/run.go`，我们确认了以下关键启动参数：

*   **Config File (`--config`)**: 默认为 `server_config.json`。需求指定使用 `server.json`，因此启动时需显式指定。
*   **Node Name (`--node-name`, `-n`)**: 默认为 `hostname`。需求要求动态获取，虽然代码有默认行为，但在启动脚本中显式获取并传递可以确保行为的一致性和可控性。
*   **Cluster Token (`--token`)**: 用于集群通信的认证令牌。需求要求动态生成随机 Token。
*   **Port (`--port`, `-p`)**: 监听端口，默认为 8080。
*   **Log Directory (`--log-dir`)**: 日志目录。

## 2. 启动脚本设计 (`bin/server_startup.sh`)

脚本将作为服务的入口点，负责准备环境变量和参数，然后启动服务器进程。该脚本已针对 CentOS、Ubuntu 和 RedHat 系统进行适配，并使用最简化的通用命令。

### 2.1 脚本逻辑流程

1.  **获取主机名**: 优先从 `/etc/hostname` 读取，若不存在则尝试使用 `hostname` 命令。这是在 CentOS、Ubuntu 和 RedHat 中最通用的方法。
2.  **生成 Cluster Token**:
    *   使用 `cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1` 生成随机字符串。
    *   该方法避免了对 `openssl` 或 `od` 的依赖，仅使用 `cat`, `tr`, `fold`, `head` 等极其通用的基础命令。
3.  **确定项目路径**: 使用 `dirname` 和 `pwd` 获取路径，确保脚本在不同系统环境下都能正确定位二进制文件。
4.  **拼接并执行命令**:
    *   使用 `exec` 替换当前进程，确保 Systemd 能够正确捕捉信号和管理生命周期。

### 2.2 脚本草稿

```bash
#!/bin/bash

# =================================================================
# Shell Executor MCP Server 启动脚本
# 适配系统：CentOS, Ubuntu, RedHat
# 依赖命令：grep, cat, echo, tr, fold, head, hostname
# =================================================================

# 1. 确定项目路径
# 使用最通用的方式获取脚本所在目录
SCRIPT_PATH=$(readlink -f "$0" 2>/dev/null || python -c 'import os,sys;print(os.path.realpath(sys.argv[1]))' "$0" 2>/dev/null || echo "$0")
SCRIPT_DIR=$(dirname "$SCRIPT_PATH")
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
SERVER_BIN="$PROJECT_ROOT/k8s-mcp-server"

# 检查二进制文件是否存在
if [ ! -f "$SERVER_BIN" ]; then
    echo "错误: 在 $SERVER_BIN 未找到服务器二进制文件"
    exit 1
fi

# 2. 获取主机名 (Node Name)
# 优先读取 /etc/hostname，这是 Linux 系统的通用标准
if [ -f /etc/hostname ]; then
    NODE_NAME=$(cat /etc/hostname)
else
    NODE_NAME=$(hostname)
fi

# 如果还是为空，设置默认值
if [ -z "$NODE_NAME" ]; then
    NODE_NAME="linux-node-$(cat /dev/urandom | tr -dc '0-9' | head -c 4)"
fi
echo "使用节点名称: $NODE_NAME"

# 3. 生成随机 Cluster Token (32位字母数字组合)
# 使用 cat /dev/urandom 和 tr 这种最通用的组合，避开 openssl 和复杂的 sed
CLUSTER_TOKEN=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
echo "生成的集群令牌: $CLUSTER_TOKEN"

# 4. 启动 Server
CONFIG_FILE="$PROJECT_ROOT/server.json"

# 检查配置文件
if [ ! -f "$CONFIG_FILE" ]; then
    echo "警告: 未找到配置文件 $CONFIG_FILE，将使用默认配置或命令行参数"
fi

echo "正在启动 Shell Executor MCP Server..."

# 使用 exec 确保进程被 systemd 正确管理
exec "$SERVER_BIN" run \
    --config "$CONFIG_FILE" \
    --node-name "$NODE_NAME" \
    --token "$CLUSTER_TOKEN"
```

## 3. Systemd 集成方案

为了满足“脚本中声明使用 systemd 来管理这个服务”的需求，我们将创建一个 Systemd Unit 文件模板 (`bin/shell-executor-mcp.service`)，并在启动脚本的注释中说明如何安装和使用它。

### 3.1 Unit 文件内容 (`bin/shell-executor-mcp.service`)

```ini
[Unit]
Description=Shell Executor MCP Server
Documentation=https://github.com/your-repo/shell-executor-mcp
After=network.target

[Service]
# Service 类型为 simple，因为启动脚本使用 exec 启动主进程
Type=simple
# 建议修改为实际运行的用户
User=root
# 建议修改为项目实际部署路径
WorkingDirectory=/opt/shell-executor-mcp
# 指向启动脚本
ExecStart=/opt/shell-executor-mcp/bin/server_startup.sh
Restart=on-failure
RestartSec=5s

# 环境变量设置（如果需要）
# Environment=MCP_LOG_LEVEL=info

[Install]
WantedBy=multi-user.target
```

### 3.2 部署说明 (添加到启动脚本注释或单独文档)

1.  将项目代码部署到 `/opt/shell-executor-mcp` (或修改 `.service` 文件中的路径)。
2.  确保二进制文件 `k8s-mcp-server` 已编译并位于项目根目录。
3.  确保 `bin/server_startup.sh` 有执行权限 (`chmod +x bin/server_startup.sh`)。
4.  复制服务文件: `cp bin/shell-executor-mcp.service /etc/systemd/system/`。
5.  修改 `/etc/systemd/system/shell-executor-mcp.service` 中的路径和用户以匹配实际环境。
6.  重载 Systemd: `systemctl daemon-reload`。
7.  启动服务: `systemctl start shell-executor-mcp`。
8.  设置开机自启: `systemctl enable shell-executor-mcp`。

## 4. 验证计划

1.  **脚本功能验证**:
    *   在开发环境直接运行 `./bin/server_startup.sh`。
    *   检查是否成功输出 Node Name 和 Token。
    *   检查 Server 是否成功启动并加载了参数。
    *   检查是否能够通过 curl 或 MCP Client 连接（使用日志中打印的 Token）。

2.  **参数正确性验证**:
    *   确认 Server 日志中显示的 Node Name 是否与 `hostname` 一致。
    *   确认 Server 使用了 `server.json` 作为配置文件。

3.  **Systemd 模拟验证** (由于无法在当前环境真正运行 Systemd):
    *   检查生成的 `.service` 文件语法是否正确。
    *   确认脚本中的路径逻辑在假设的部署结构下是有效的。

## 5. 下一步行动

*   [ ] 创建 `bin/server_startup.sh`。
*   [ ] 创建 `bin/shell-executor-mcp.service`。
*   [ ] 赋予脚本执行权限 (虽然 Windows 下不适用，但在 Git 中标记)。
*   [ ] 更新 `README.md` 或相关文档说明部署方法。
