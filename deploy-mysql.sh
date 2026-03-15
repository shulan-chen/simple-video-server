#!/bin/bash

# MySQL容器部署脚本（阿里云服务器）
# 使用方法：./deploy-mysql.sh

set -e

echo "=========================================="
echo "  MySQL 容器部署脚本"
echo "=========================================="

# 配置变量
CONTAINER_NAME="video-server-mysql"
MYSQL_ROOT_PASSWORD="123456"
MYSQL_DATABASE="video_server"
MYSQL_USER="yanghao"
MYSQL_PASSWORD="123456"
MYSQL_PORT="3306"

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
mkdir -p ~/data/mysql_data

# 启动MySQL容器
echo ""
echo "启动MySQL容器..."
docker run -d \
    --name $CONTAINER_NAME \
    --restart unless-stopped \
    -p $MYSQL_PORT:3306 \
    -e MYSQL_ROOT_PASSWORD=$MYSQL_ROOT_PASSWORD \
    -e MYSQL_DATABASE=$MYSQL_DATABASE \
    -e MYSQL_USER=$MYSQL_USER \
    -e MYSQL_PASSWORD=$MYSQL_PASSWORD \
    -v ~/data/mysql_data:/var/lib/mysql \
    mysql:8.0 \
    --character-set-server=utf8mb4 \
    --collation-server=utf8mb4_unicode_ci

echo ""
echo "等待MySQL启动..."
sleep 10

# 检查容器状态
if docker ps | grep -q $CONTAINER_NAME; then
    echo "✅ MySQL容器启动成功！"
else
    echo "❌ MySQL容器启动失败！"
    echo "查看日志："
    docker logs $CONTAINER_NAME
    exit 1
fi

# 测试连接
echo ""
echo "测试MySQL连接..."
docker exec $CONTAINER_NAME mysql -uroot -p$MYSQL_ROOT_PASSWORD -e "SELECT VERSION();" 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ MySQL连接测试成功！"
else
    echo "❌ MySQL连接测试失败！"
    exit 1
fi

echo ""
echo "=========================================="
echo "  部署完成！"
echo "=========================================="
echo ""
echo "容器信息："
echo "  容器名称: $CONTAINER_NAME"
echo "  端口映射: 0.0.0.0:$MYSQL_PORT -> 3306"
echo "  数据目录: ~/data/mysql_data"
echo ""
echo "连接信息："
echo "  地址: localhost:$MYSQL_PORT"
echo "  用户: $MYSQL_USER"
echo "  密码: $MYSQL_PASSWORD"
echo "  数据库: $MYSQL_DATABASE"
echo ""
echo "下一步："
echo "  1. 导入SQL文件："
echo "     docker exec -i $CONTAINER_NAME mysql -u$MYSQL_USER -p$MYSQL_PASSWORD $MYSQL_DATABASE < sql/video_server.sql"
echo ""
echo "  2. 验证表是否导入成功："
echo "     docker exec $CONTAINER_NAME mysql -u$MYSQL_USER -p$MYSQL_PASSWORD -e 'SHOW TABLES;' $MYSQL_DATABASE"
echo ""
echo "  3. 查看容器日志："
echo "     docker logs -f $CONTAINER_NAME"
echo ""
echo "=========================================="
