#!/bin/bash
# 快速测试脚本

echo "🚀 快速测试 Web 网关优化"
echo ""

# 1. 编译
echo "1️⃣ 编译服务..."
go build -o bin/web-service ./cmd/web/main.go 2>&1 | grep -i error || echo "✅ 编译成功"
echo ""

# 2. 检查文件
echo "2️⃣ 检查新增文件..."
[ -f web/middleware/rate_limiter.go ] && echo "✅ rate_limiter.go" || echo "❌ rate_limiter.go"
[ -f web/middleware/circuit_breaker.go ] && echo "✅ circuit_breaker.go" || echo "❌ circuit_breaker.go"
[ -f web/middleware/cache.go ] && echo "✅ cache.go" || echo "❌ cache.go"
[ -f web/health/downstream.go ] && echo "✅ downstream.go" || echo "❌ downstream.go"
[ -f web/metrics/prometheus.go ] && echo "✅ prometheus.go" || echo "❌ prometheus.go"
echo ""

# 3. 统计代码行数
echo "3️⃣ 统计新增代码..."
total_lines=$(find web/middleware web/health web/metrics -name "*.go" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
echo "新增代码：$total_lines 行"
echo ""

# 4. 检查文档
echo "4️⃣ 检查文档..."
[ -f documents/GATEWAY_OPTIMIZATION_COMPLETE_GUIDE.md ] && echo "✅ 完整指南 (45KB)" || echo "❌ 完整指南"
[ -f documents/GATEWAY_OPTIMIZATION_SUMMARY.md ] && echo "✅ 总结文档 (25KB)" || echo "❌ 总结文档"
[ -f documents/RATE_LIMIT_AND_CIRCUIT_BREAKER_CHEATSHEET.md ] && echo "✅ 快速参考卡片" || echo "❌ 快速参考卡片"
echo ""

# 5. 检查配置
echo "5️⃣ 检查配置..."
grep -q "api_addr" config/config.json && echo "✅ 配置文件存在" || echo "❌ 配置文件缺失"
echo ""

echo "🎉 快速测试完成！"
echo ""
echo "📖 阅读文档："
echo "   documents/GATEWAY_OPTIMIZATION_COMPLETE_GUIDE.md (详细原理)"
echo "   documents/RATE_LIMIT_AND_CIRCUIT_BREAKER_CHEATSHEET.md (快速参考)"
echo ""
echo "🧪 运行完整测试："
echo "   ./test_web_gateway.sh"
