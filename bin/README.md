# bin 目录

本目录包含项目的可执行文件、启动脚本和配置模板。

## 文件说明

- `server_startup.sh` - Shell Executor MCP Server 的启动脚本，用于通过 Systemd 管理服务。
- `server-template.json` - 服务器配置文件模板，首次启动时会自动拷贝到项目根目录并填充配置。
- `shell-executor-mcp.service` - Systemd 服务单元文件模板，用于配置系统服务。
- `k8s-mcp-server` / `server` - 服务器端可执行文件。
- `k8s-mcp-client` / `client` - 客户端可执行文件。

## 使用 server_startup.sh

`server_startup.sh` 脚本设计用于在 Linux 系统（CentOS, Ubuntu, RedHat）上启动服务器，支持手动直接运行和通过 Systemd 进行服务管理。

### 功能特性

- 自动获取主机名作为 `node_name`
- 动态生成随机的 `cluster_token`
- 使用 `server.json` 作为配置文件
- **自动配置**: 如果项目根目录不存在 `server.json`，会自动从 `bin/server-template.json` 模板拷贝
- **自动填充**: 自动将 `node_name` 和 `cluster_token` 写入配置文件，替换模板中的占位符
- **自动停止旧进程**: 启动新进程前，会自动检查并停止正在运行的同名服务器进程（基于 PID 文件）
- **后台运行**: 使用 `nohup` 在后台启动服务器，脚本执行完成后可以正常退出
- **PID 管理**: 将启动后的进程 PID 记录到 `server.pid` 文件中，便于后续精准停止
- **日志管理**: 自动将日志输出重定向到 `server.log` 文件

### 前提条件

1. 确保服务器二进制文件 `server` 位于项目根目录。
2. 确保 `bin/server-template.json` 配置模板存在于 `bin/` 目录（首次启动时会自动拷贝到根目录）。
3. 脚本需要执行权限：`chmod +x bin/server_startup.sh`。

### 直接运行（推荐用于手动部署）

```bash
./bin/server_startup.sh
```

**运行后输出示例：**
```
脚本所在目录: /opt/shell-executor-mcp/bin
项目根目录: /opt/shell-executor-mcp
找到服务器二进制文件: /opt/shell-executor-mcp/server
使用节点名称: node-01
生成的集群令牌: abc123def456...
正在启动 Shell Executor MCP Server...
服务器已在后台启动
进程 PID: 12345
PID 文件: /opt/shell-executor-mcp/server.pid
日志文件: /opt/shell-executor-mcp/server.log

查看日志: tail -f /opt/shell-executor-mcp/server.log
停止服务器: kill $(cat /opt/shell-executor-mcp/server.pid)
```

**常用操作：**
- 查看日志：`tail -f server.log`
- 停止服务器：`kill $(cat server.pid)`
- 重启服务器：再次运行 `./bin/server_startup.sh`（会自动停止旧进程）
- 查看进程状态：`ps -p $(cat server.pid)`

### 通过 Systemd 运行（推荐用于生产环境）

**注意：** 如果使用 systemd 管理，systemd 会自动处理进程生命周期（启动、停止、重启）和日志管理（通过 journalctl）。脚本中的后台运行和 PID 管理逻辑主要用于手动直接运行场景。

1. 复制服务单元文件到 Systemd 目录：

```bash
cp bin/shell-executor-mcp.service /etc/systemd/system/
```

2. 根据实际部署环境修改服务文件（例如修改 `WorkingDirectory` 和 `ExecStart` 路径）：

```bash
# 编辑服务文件
vi /etc/systemd/system/shell-executor-mcp.service
```

3. 将项目部署到指定目录（例如 `/opt/shell-executor-mcp`）。
4. 重载 Systemd：`systemctl daemon-reload`。
5. 启动服务：`systemctl start shell-executor-mcp`。
6. 设置开机自启：`systemctl enable shell-executor-mcp`。
7. 查看服务状态：`systemctl status shell-executor-mcp`。
8. 查看服务日志：`journalctl -u shell-executor-mcp -f`。

**Systemd 与手动运行的区别：**

| 特性 | 手动运行 | Systemd 运行 |
|------|---------|--------------|
| 进程管理 | 脚本自动停止旧进程 | systemd 自动管理 |
| 日志位置 | `server.log` | `journalctl -u shell-executor-mcp` |
| 自动重启 | 需手动重启脚本 | `Restart=on-failure` 自动重启 |
| 开机自启 | 需手动配置 | `systemctl enable` 自动配置 |
| 适用场景 | 测试、开发环境 | 生产环境 |

### 配置说明

- **server-template.json 模板**: 位于 `bin/server-template.json`，包含默认的服务器配置和占位符。
- **自动拷贝**: 脚本启动时会自动检测项目根目录是否存在 `server.json`，如果不存在则从 `bin/server-template.json` 拷贝。
- **自动填充**: 脚本会自动将获取的 `node_name` 和生成的 `cluster_token` 写入配置文件，替换模板中的占位符。
- **自定义配置**: 用户可以在根目录的 `server.json` 中修改配置，模板文件不会被覆盖。

## 更新记录

- 2026-01-30: 添加后台运行、自动停止旧进程、PID 文件管理和日志重定向功能，更新文档说明手动运行和 Systemd 运行的区别
- 2026-01-30: 重命名 server.json 为 server-template.json，更新启动脚本支持自动填充配置文件
- 2026-01-29: 添加 server.json 配置模板和 shell-executor-mcp.service 服务文件，更新 server_startup.sh 支持自动配置
