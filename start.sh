#!/bin/bash

# 1. 编译项目 (生成名为 server 的可执行文件)
echo "正在构建项目..."
go build -o server .

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo "⚠️  构建失败，请检查代码错误。"
    exit 1
fi

# 2. 检查是否已经在运行
PID=$(pgrep -f "./server")
if [ -n "$PID" ]; then
    echo "⚠️  服务已经在运行中 (PID: $PID)。请先停止服务。"
    exit 0
fi

# 3. 后台启动服务
# nohup: 让程序在终端关闭后继续运行
# > app.log: 标准输出重定向到 app.log
# 2>&1: 错误输出重定向到标准输出（也进入 app.log）
# &: 在后台运行
nohup ./server > app.log 2>&1 &

echo "✅ 服务已启动！"
echo "📄 日志文件: app.log"