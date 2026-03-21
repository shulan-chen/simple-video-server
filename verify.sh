#!/bin/bash

# 微服务架构验证脚本
# 用途：快速验证所有服务是否正常工作

set -e

echo "========================================"
echo "  微服务架构验证"
echo "========================================"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_service() {
    local service_name=$1
    local port=$2
    local health_url="http://localhost:${port}/health/ready"

    echo -n "检查 ${service_name} (端口 ${port})... "

    if curl -s -f ${health_url} > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 正常${NC}"
        return 0
    else
        echo -e "${RED}✗ 失败${NC}"
        return 1
    fi
}

# 步骤1：检查构建产物
echo -e "\n${YELLOW}[步骤1] 检查构建产物${NC}"
if [ ! -d "bin" ]; then
    echo -e "${RED}错误: bin目录不存在，请先运行 'make build'${NC}"
    exit 1
fi

for binary in api-service web-service stream-service scheduler-service; do
    if [ -f "bin/${binary}" ]; then
        echo -e "  ${GREEN}✓${NC} bin/${binary} 存在"
    else
        echo -e "  ${RED}✗${NC} bin/${binary} 不存在"
        echo "请运行: make build-${binary%-service}"
        exit 1
    fi
done

# 步骤2：检查配置文件
echo -e "\n${YELLOW}[步骤2] 检查配置文件${NC}"
if [ ! -f "config/config.json" ]; then
    echo -e "${RED}错误: config/config.json 不存在${NC}"
    echo "请复制: cp config/config.example.json config/config.json"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} config/config.json 存在"

# 步骤3：检查服务是否运行
echo -e "\n${YELLOW}[步骤3] 检查服务健康状态${NC}"

SERVICES=(
    "API服务:8000"
    "Web服务:8080"
    "Stream服务:9090"
    "Scheduler服务:8001"
)

failed_services=()

for service in "${SERVICES[@]}"; do
    IFS=':' read -r name port <<< "$service"
    if ! check_service "$name" "$port"; then
        failed_services+=("$name")
    fi
done

# 步骤4：显示结果
echo -e "\n========================================"
if [ ${#failed_services[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ 所有服务运行正常！${NC}"
    echo ""
    echo "服务地址："
    echo "  - API服务:       http://localhost:8000"
    echo "  - Web服务:       http://localhost:8080"
    echo "  - Stream服务:    http://localhost:9090"
    echo "  - Scheduler服务: http://localhost:8001"
    echo ""
    echo "健康检查："
    echo "  curl http://localhost:8000/health/ready"
    echo ""
    echo "查看日志："
    echo "  tail -f logs/api.log"
    echo "========================================"
    exit 0
else
    echo -e "${RED}❌ 以下服务未运行：${NC}"
    for service in "${failed_services[@]}"; do
        echo "  - $service"
    done
    echo ""
    echo "启动服务："
    echo "  make start        # 后台启动所有服务"
    echo "  make run-api      # 前台运行API服务"
    echo ""
    echo "或使用Docker Compose："
    echo "  docker-compose up -d"
    echo "========================================"
    exit 1
fi
