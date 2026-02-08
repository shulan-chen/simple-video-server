#!/bin/bash

# 查找名为 ./server 的进程 ID
PID=$(pgrep -f "./server")

if [ -z "$PID" ]; then
    echo "⚠️  未发现正在运行的服务。"
else
    # 强制杀掉进程
    kill -9 $PID
    echo "🛑 服务已停止 (PID: $PID)。"
fi