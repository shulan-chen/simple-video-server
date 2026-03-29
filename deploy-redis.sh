#!/bin/bash

# Redis容器部署脚本（阿里云服务器）
# 使用方法：./deploy-redis.sh

set -e

echo "=========================================="
echo "  Redis 容器部署脚本"
echo "=========================================="

# 配置变量
CONTAINER_NAME="video-server-redis"
REDIS_PASSWORD="123456"
REDIS_PORT="6379"

# 检查是否已有同名容器
if docker ps -a | grep -q $CONTAINER_NAME; then
    echo "⚠️  发现已存在的容器: $CONTAINER_NAME"
    read -p "是否删除旧容器并重新部署？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "停止并删除旧容器..."
        docker stop $CONTAINER_NAME || true
        docker rm $CONTAINER_NAME || true
    else
        echo "取消部署"
        exit 0
    fi
fi

# 创建数据持久化目录
echo ""
echo "创建数据持久化目录..."
mkdir -p ~/data/redis_data

# 启动Redis容器
echo ""
echo "启动Redis容器..."
docker run -d \
    --name $CONTAINER_NAME \
    --restart unless-stopped \
    -p $REDIS_PORT:6379 \
    -e REDIS_PASSWORD=$REDIS_PASSWORD \
    -v ~/data/redis_data:/data \
    redis:alpine \
    redis-server --requirepass $REDIS_PASSWORD --appendonly yes

echo ""
echo "等待Redis启动..."
sleep 5

# 检查容器状态
if docker ps | grep -q $CONTAINER_NAME; then
    echo "✅ Redis容器启动成功！"
else
    echo "❌ Redis容器启动失败！"
    echo "查看日志："
    docker logs $CONTAINER_NAME
    exit 1
fi

# 测试连接
echo ""
echo "测试Redis连接..."
docker exec $CONTAINER_NAME redis-cli -a $REDIS_PASSWORD ping 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ Redis连接测试成功！"
else
    echo "❌ Redis连接测试失败！"
    exit 1
fi

echo ""
echo "=========================================="
echo "  部署完成！"
echo "=========================================="
echo ""
echo "容器信息："
echo "  容器名称: $CONTAINER_NAME"
echo "  端口映射: 0.0.0.0:$REDIS_PORT -> 6379"
echo "  数据目录: ~/redis_data"
echo ""
echo "连接信息："
echo "  地址: localhost:$REDIS_PORT"
echo "  密码: $REDIS_PASSWORD"
echo ""
echo "下一步："
echo "  1. 查看Redis信息："
echo "     docker exec $CONTAINER_NAME redis-cli -a $REDIS_PASSWORD info"
echo ""
echo "  2. 查看容器日志："
echo "     docker logs -f $CONTAINER_NAME"
echo ""
echo "=========================================="