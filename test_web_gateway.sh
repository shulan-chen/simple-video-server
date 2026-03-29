#!/bin/bash

# Web 网关优化测试脚本

echo "========================================="
echo "Web 网关生产级优化测试"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WEB_URL="http://localhost:8080"

# 测试1：限流保护
echo "📊 测试1：限流保护"
echo "-------------------"
echo "发送 150 个请求（超过全局限流 100 req/s）"

success_count=0
reject_count=0

for i in {1..150}; do
  response=$(curl -s -o /dev/null -w "%{http_code}" $WEB_URL/ 2>/dev/null)
  if [ "$response" = "200" ]; then
    success_count=$((success_count + 1))
  elif [ "$response" = "429" ]; then
    reject_count=$((reject_count + 1))
  fi
done

echo "成功: $success_count 个"
echo "限流拒绝: $reject_count 个"

if [ $reject_count -gt 30 ]; then
  echo -e "${GREEN}✅ 限流器工作正常${NC}"
else
  echo -e "${RED}❌ 限流器可能未生效${NC}"
fi
echo ""

# 测试2：熔断器状态
echo "🔌 测试2：熔断器状态"
echo "-------------------"

# 检查 API 服务熔断器
api_circuit=$(curl -s $WEB_URL/metrics 2>/dev/null | grep 'web_circuit_breaker_state{circuit="api-service"}' | awk '{print $2}')
if [ "$api_circuit" = "0" ]; then
  echo -e "API 熔断器: ${GREEN}CLOSED (正常)${NC}"
elif [ "$api_circuit" = "1" ]; then
  echo -e "API 熔断器: ${RED}OPEN (熔断中)${NC}"
elif [ "$api_circuit" = "2" ]; then
  echo -e "API 熔断器: ${YELLOW}HALF_OPEN (恢复中)${NC}"
fi

# 检查 Stream 服务熔断器
stream_circuit=$(curl -s $WEB_URL/metrics 2>/dev/null | grep 'web_circuit_breaker_state{circuit="stream-service"}' | awk '{print $2}')
if [ "$stream_circuit" = "0" ]; then
  echo -e "Stream 熔断器: ${GREEN}CLOSED (正常)${NC}"
elif [ "$stream_circuit" = "1" ]; then
  echo -e "Stream 熔断器: ${RED}OPEN (熔断中)${NC}"
elif [ "$stream_circuit" = "2" ]; then
  echo -e "Stream 熔断器: ${YELLOW}HALF_OPEN (恢复中)${NC}"
fi
echo ""

# 测试3：HTTP 缓存头
echo "💾 测试3：HTTP 缓存头"
echo "-------------------"

# 测试静态资源缓存
echo "静态资源 (/statics/test.css):"
cache_control=$(curl -s -I "$WEB_URL/statics/test.css" 2>/dev/null | grep -i "Cache-Control" | cut -d: -f2- | xargs)
if [[ "$cache_control" == *"max-age=31536000"* ]]; then
  echo -e "  ${GREEN}✅ 长期缓存 (1年)${NC}"
  echo "  Cache-Control: $cache_control"
else
  echo -e "  ${YELLOW}⚠️  缓存头：$cache_control${NC}"
fi

# 测试 HTML 缓存
echo "HTML 页面 (/):"
cache_control=$(curl -s -I "$WEB_URL/" 2>/dev/null | grep -i "Cache-Control" | cut -d: -f2- | xargs)
if [[ "$cache_control" == *"no-cache"* ]]; then
  echo -e "  ${GREEN}✅ 协商缓存${NC}"
  echo "  Cache-Control: $cache_control"
else
  echo -e "  ${YELLOW}⚠️  缓存头：$cache_control${NC}"
fi
echo ""

# 测试4：Prometheus 指标
echo "📈 测试4：Prometheus 指标"
echo "-------------------"

metrics_response=$(curl -s $WEB_URL/metrics 2>/dev/null)

# 检查指标是否存在
check_metric() {
  metric_name=$1
  if echo "$metrics_response" | grep -q "$metric_name"; then
    echo -e "  ${GREEN}✅ $metric_name${NC}"
    return 0
  else
    echo -e "  ${RED}❌ $metric_name 不存在${NC}"
    return 1
  fi
}

check_metric "web_http_requests_total"
check_metric "web_http_request_duration_seconds"
check_metric "web_http_in_flight_requests"
check_metric "web_proxy_requests_total"
check_metric "web_circuit_breaker_state"
check_metric "web_rate_limit_rejects_total"
echo ""

# 测试5：TraceID 传递
echo "🔍 测试5：TraceID 传递"
echo "-------------------"

trace_id=$(curl -s -I "$WEB_URL/" | grep -i "X-Trace-ID" | cut -d: -f2- | xargs)
if [ -n "$trace_id" ]; then
  echo -e "${GREEN}✅ TraceID 生成成功${NC}"
  echo "  TraceID: $trace_id"
else
  echo -e "${RED}❌ TraceID 未生成${NC}"
fi
echo ""

# 测试6：健康检查
echo "🏥 测试6：健康检查"
echo "-------------------"

health_status=$(curl -s "$WEB_URL/health/ready" 2>/dev/null)
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✅ 健康检查端点可访问${NC}"
  echo "$health_status" | head -5
else
  echo -e "${RED}❌ 健康检查端点不可访问${NC}"
fi
echo ""

# 总结
echo "========================================="
echo "测试总结"
echo "========================================="
echo ""

total_tests=6
passed_tests=0

[ $reject_count -gt 30 ] && passed_tests=$((passed_tests + 1))
[ -n "$api_circuit" ] && passed_tests=$((passed_tests + 1))
[[ "$cache_control" == *"max-age"* ]] && passed_tests=$((passed_tests + 1))
echo "$metrics_response" | grep -q "web_http_requests_total" && passed_tests=$((passed_tests + 1))
[ -n "$trace_id" ] && passed_tests=$((passed_tests + 1))
[ -n "$health_status" ] && passed_tests=$((passed_tests + 1))

echo "通过测试: $passed_tests / $total_tests"
echo ""

if [ $passed_tests -eq $total_tests ]; then
  echo -e "${GREEN}🎉 所有测试通过！Web 网关生产级优化成功！${NC}"
else
  echo -e "${YELLOW}⚠️  部分测试未通过，请检查服务配置${NC}"
fi
echo ""

echo "📖 查看详细文档："
echo "  - documents/GATEWAY_OPTIMIZATION_COMPLETE_GUIDE.md"
echo "  - documents/WEB_GATEWAY_PRODUCTION_OPTIMIZATION.md"
echo ""
echo "🚀 访问 Prometheus metrics: $WEB_URL/metrics"
echo "📊 访问 Grafana: http://localhost:3000 (需先启动 Grafana)"
