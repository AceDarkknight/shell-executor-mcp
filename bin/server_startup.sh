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
SERVER_BIN="$PROJECT_ROOT/server"

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
CONFIG_TEMPLATE="$SCRIPT_DIR/server-template.json"

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
    sed -i "s/NODE_NAME_PLACEHOLDER/$NODE_NAME/g" "$CONFIG_FILE"
    sed -i "s/CLUSTER_TOKEN_PLACEHOLDER/$CLUSTER_TOKEN/g" "$CONFIG_FILE"
    echo "配置文件更新完成"
fi

echo "正在启动 Shell Executor MCP Server..."

# 使用 exec 确保进程被 systemd 正确管理
exec "$SERVER_BIN" run \
    --config "$CONFIG_FILE"
