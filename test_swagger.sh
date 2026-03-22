#!/bin/bash

# Swagger功能快速测试脚本

echo "=========================================="
echo "   Swagger API文档测试"
echo "=========================================="
echo ""

# 检查服务是否运行
if ! curl -s http://localhost:8000/health/ready > /dev/null 2>&1; then
    echo "❌ API服务未启动，请先运行："
    echo "   make run-api"
    echo "   或"
    echo "   bin/api-service"
    exit 1
fi

echo "✅ API服务运行中"
echo ""

# 测试Swagger JSON
echo "1️⃣  测试Swagger JSON"
echo "----------------------------------------"
TITLE=$(curl -s http://localhost:8000/swagger/doc.json | grep -o '"title":"[^"]*"' | head -1)
VERSION=$(curl -s http://localhost:8000/swagger/doc.json | grep -o '"version":"[^"]*"' | head -1)
echo "$TITLE"
echo "$VERSION"
echo "✅ Swagger JSON可访问"
echo ""

# 测试Swagger UI
echo "2️⃣  测试Swagger UI"
echo "----------------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/swagger/index.html)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Swagger UI可访问 (HTTP $HTTP_CODE)"
else
    echo "❌ Swagger UI不可访问 (HTTP $HTTP_CODE)"
fi
echo ""

# 统计接口数量
echo "3️⃣  接口统计"
echo "----------------------------------------"
API_COUNT=$(curl -s http://localhost:8000/swagger/doc.json | grep -o '"summary"' | wc -l)
echo "已文档化的接口: $API_COUNT 个"
echo ""

# 显示访问地址
echo "4️⃣  访问地址"
echo "----------------------------------------"
echo "📖 Swagger UI:   http://localhost:8000/swagger/index.html"
echo "📄 Swagger JSON: http://localhost:8000/swagger/doc.json"
echo "❤️  健康检查:     http://localhost:8000/health/ready"
echo ""

# 显示接口分组
echo "5️⃣  接口分组"
echo "----------------------------------------"
curl -s http://localhost:8000/swagger/doc.json | grep -o '"tags":\["[^"]*"\]' | sort -u | sed 's/"tags":\["/  - /g' | sed 's/"\]//g'
echo ""

echo "=========================================="
echo "   测试完成！"
echo "=========================================="
echo ""
echo "💡 提示：在浏览器中打开 Swagger UI 可以交互式测试所有接口"
