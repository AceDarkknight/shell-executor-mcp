#!/bin/bash

# =================================================================
# Shell Executor MCP Server 启动脚本
# 适配系统：CentOS, Ubuntu, RedHat
# 依赖命令：grep, cat, echo, tr, fold, head, hostname, nohup, kill, ps
# 功能特性：
#   - 自动停止旧进程（基于 PID 文件）
#   - 后台运行服务器（使用 nohup）
#   - PID 文件管理（server.pid）
#   - 日志重定向到 server.log
# 注意：如果使用 systemd 管理，systemd 会处理进程生命周期，
#       本脚本的后台运行逻辑更适合手动直接运行。
# =================================================================

# 1. 确定项目路径
# 使用最通用的方式获取脚本所在目录
SCRIPT_PATH=$(readlink -f "$0" 2>/dev/null || python -c 'import os,sys;print(os.path.realpath(sys.argv[1]))' "$0" 2>/dev/null || echo "$0")
SCRIPT_DIR=$(dirname "$SCRIPT_PATH")

# 智能判断项目根目录：如果脚本在 bin/ 目录下，则 PROJECT_ROOT 是上一级目录；否则就是当前目录
SCRIPT_DIR_NAME=$(basename "$SCRIPT_DIR")
if [ "$SCRIPT_DIR_NAME" = "bin" ]; then
    # 脚本在 bin/ 目录下运行
    PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
else
    # 脚本在项目根目录下运行
    PROJECT_ROOT=$(cd "$SCRIPT_DIR" && pwd)
fi

echo "脚本所在目录: $SCRIPT_DIR"
echo "项目根目录: $PROJECT_ROOT"

# PID 文件和日志文件路径
PID_FILE="$PROJECT_ROOT/server.pid"
LOG_FILE="$PROJECT_ROOT/server.log"

# 2. 停止旧进程（如果存在）
# 检查 PID 文件是否存在
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if [ -n "$OLD_PID" ]; then
        # 检查该 PID 对应的进程是否仍在运行
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            echo "发现正在运行的服务器进程 (PID: $OLD_PID)，正在停止..."
            # 尝试优雅停止（发送 SIGTERM）
            kill "$OLD_PID" 2>/dev/null
            # 等待最多 5 秒
            for i in {1..5}; do
                if ! ps -p "$OLD_PID" > /dev/null 2>&1; then
                    echo "服务器进程已停止"
                    break
                fi
                sleep 1
            done
            # 如果进程仍在运行，强制停止
            if ps -p "$OLD_PID" > /dev/null 2>&1; then
                echo "进程未响应，正在强制停止..."
                kill -9 "$OLD_PID" 2>/dev/null
                sleep 1
            fi
        else
            echo "PID 文件存在但进程已停止，清理 PID 文件"
        fi
    fi
    # 删除旧的 PID 文件
    rm -f "$PID_FILE"
fi

# 3. 增强二进制文件探测
# 按顺序尝试寻找以下名称的二进制文件：shell-executor-mcp-server, k8s-mcp-server, server
# 在两个位置查找：$PROJECT_ROOT/bin/ 和 $PROJECT_ROOT/
BINARY_NAMES=("shell-executor-mcp-server" "k8s-mcp-server" "server")
SEARCH_PATHS=("$PROJECT_ROOT/bin" "$PROJECT_ROOT")
SERVER_BIN=""

for binary_name in "${BINARY_NAMES[@]}"; do
    for search_path in "${SEARCH_PATHS[@]}"; do
        candidate="${search_path}/${binary_name}"
        if [ -f "$candidate" ] && [ -x "$candidate" ]; then
            SERVER_BIN="$candidate"
            echo "找到服务器二进制文件: $SERVER_BIN"
            break 2
        fi
    done
done

# 检查二进制文件是否存在
if [ -z "$SERVER_BIN" ]; then
    echo "错误: 未找到服务器二进制文件"
    echo "已尝试在以下位置查找:"
    for search_path in "${SEARCH_PATHS[@]}"; do
        for binary_name in "${BINARY_NAMES[@]}"; do
            echo "  - ${search_path}/${binary_name}"
        done
    done
    echo "请确保二进制文件已编译并放置在正确的位置"
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
CONFIG_TEMPLATE="$PROJECT_ROOT/server-template.json"

# 检查配置文件，如果不存在则尝试从模板拷贝
if [ ! -f "$CONFIG_FILE" ]; then
    echo "警告: 未找到配置文件 $CONFIG_FILE"
    if [ -f "$CONFIG_TEMPLATE" ]; then
        echo "正在从模板拷贝配置文件: $CONFIG_TEMPLATE -> $CONFIG_FILE"
        cp "$CONFIG_TEMPLATE" "$CONFIG_FILE"
        if [ $? -eq 0 ]; then
            echo "配置文件拷贝成功"
        else
            echo "错误: 配置文件拷贝失败，将使用默认配置或命令行参数"
        fi
    else
        echo "错误: 未找到配置模板 $CONFIG_TEMPLATE，将使用默认配置或命令行参数"
    fi
fi

# 5. 更新配置文件中的 node_name 和 cluster_token
if [ -f "$CONFIG_FILE" ]; then
    echo "正在更新配置文件中的节点名称和集群令牌..."
    # 使用 sed -i 替换占位符
    # 注意：不同系统的 sed -i 语法可能不同，这里使用通用的方式
    # 使用 | 作为分隔符，避免与变量中的 / 冲突
    sed -i "s|NODE_NAME_PLACEHOLDER|$NODE_NAME|g" "$CONFIG_FILE"
    sed -i "s|CLUSTER_TOKEN_PLACEHOLDER|$CLUSTER_TOKEN|g" "$CONFIG_FILE"
    echo "配置文件更新完成"
fi

echo "正在启动 Shell Executor MCP Server..."

# 使用 nohup 后台运行服务器，重定向输出到日志文件
# 注意：如果使用 systemd 管理，systemd 会处理进程生命周期和日志，
#       本脚本的后台运行逻辑更适合手动直接运行。
nohup "$SERVER_BIN" run \
    --config "$CONFIG_FILE" \
    >> "$LOG_FILE" 2>&1 &

# 获取新启动的进程 PID
NEW_PID=$!

# 将 PID 写入文件
echo "$NEW_PID" > "$PID_FILE"

echo "服务器已在后台启动"
echo "进程 PID: $NEW_PID"
echo "PID 文件: $PID_FILE"
echo "日志文件: $LOG_FILE"
echo ""
echo "查看日志: tail -f $LOG_FILE"
echo "停止服务器: kill \$(cat $PID_FILE)"
echo ""
