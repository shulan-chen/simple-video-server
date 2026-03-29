# 限流与熔断快速参考卡片

## 🎯 限流（Rate Limiting）

### 一句话总结
**控制请求速率，防止系统被打垮**

### 核心算法：Token Bucket
```
┌─────────────┐
│   🪙🪙🪙🪙   │ ← 桶（burst=5）
│             │
│    💧       │ ← 水龙头（rate=1/s）
└─────────────┘
      ↓
  请求拿令牌
```

### 关键参数
- **Rate**: 令牌生成速率（如 10/秒）
- **Burst**: 桶容量（如 20）

### 工作原理
```
请求来了 → 桶里有令牌？
             ├─ 是 → ✅ 取走令牌，通过
             └─ 否 → ❌ 拒绝，返回 429
```

### 使用场景
| 场景 | 策略 |
|------|------|
| 防 DDoS | 全局限流（100 req/s per IP） |
| 防滥用 | 用户限流（10 req/s per user） |
| 保护资源 | 视频限流（5 req/s per user） |

### 代码示例
```go
// 创建限流器
limiter := NewRateLimiter(time.Second, 100)

// 检查是否允许
if !limiter.Allow(userID) {
	return errors.New("too many requests")
}
```

---

## 🔌 熔断（Circuit Breaker）

### 一句话总结
**下游故障时快速失败，防止故障蔓延**

### 核心：三状态状态机
```
  CLOSED ──(失败5次)→ OPEN ──(30秒)→ HALF_OPEN
    ↑                              │
    └───────(成功3次)───────────────┘
               │
               └─(失败1次)→ OPEN
```

### 三种状态

| 状态 | 行为 | 何时转换 |
|------|------|----------|
| **CLOSED**<br>（关闭=正常） | 所有请求发送到后端<br>记录失败次数 | 连续失败5次 → OPEN |
| **OPEN**<br>（打开=熔断） | 所有请求立即返回错误<br>不发送到后端 | 等待30秒 → HALF_OPEN |
| **HALF_OPEN**<br>（半开=试探） | 允许少量请求<br>检测是否恢复 | 成功3次 → CLOSED<br>失败1次 → OPEN |

### 关键参数
- **maxFailures**: 触发熔断的失败次数（5）
- **timeout**: 熔断持续时间（30秒）
- **halfOpenSuccess**: 关闭熔断的成功次数（3）

### 工作原理
```
请求来了 → 熔断器状态？
             ├─ CLOSED → 发送到后端
             ├─ OPEN → ❌ 直接返回 503
             └─ HALF_OPEN → 试探（可能通过）
```

### 使用场景
| 场景 | 效果 |
|------|------|
| 后端故障 | 快速失败（0.1ms vs 30s） |
| 后端慢查询 | 避免 goroutine 堆积 |
| 依赖服务挂了 | 防止雪崩效应 |

### 代码示例
```go
// 创建熔断器
cb := NewCircuitBreaker("api-service", 5, 30*time.Second, 3)

// 通过熔断器执行请求
err := cb.Call(func() error {
	return doRequest()
})

if err != nil && err.Error() == "circuit breaker is open" {
	// 熔断器打开，使用降级策略
	return fallbackData
}
```

---

## 🆚 限流 vs 熔断

| 维度 | 限流（Rate Limiting） | 熔断（Circuit Breaker） |
|------|----------------------|------------------------|
| **目标** | 保护自己不被打垮 | 保护自己不被拖垮 |
| **触发条件** | 请求速率过高 | 下游服务故障 |
| **保护对象** | 自己的资源 | 自己不被下游拖累 |
| **返回状态码** | 429 Too Many Requests | 503 Service Unavailable |
| **类比** | 收银台排队限流 | 收银系统宕机关门 |
| **关注点** | 请求数量 | 请求质量（成功/失败） |
| **时间维度** | 每秒/每分钟 | 连续失败 |

### 配合使用

```
用户请求
   │
   ▼
┌────────┐  超过 100 req/s?
│ 限流器  │ ──────→ ❌ 429（保护自己的资源）
└───┬────┘
    │ ✅ 通过
    ▼
┌────────┐  后端服务挂了?
│ 熔断器  │ ──────→ ❌ 503（保护自己不被拖垮）
└───┬────┘
    │ ✅ 通过
    ▼
  发送到后端
```

---

## 📊 Prometheus 指标类型

### Counter（计数器）
**特点**: 只增不减
**用途**: 累计计数
```go
httpRequestsTotal.Inc()  // +1
```
**查询**:
```promql
rate(web_http_requests_total[1m])  # 每秒增长率（QPS）
```

### Gauge（仪表）
**特点**: 可增可减
**用途**: 瞬时值
```go
httpInFlightRequests.Inc()  // +1
httpInFlightRequests.Dec()  // -1
```
**查询**:
```promql
web_http_in_flight_requests  # 当前值
```

### Histogram（直方图）
**特点**: 分桶统计
**用途**: 百分位计算
```go
httpRequestDuration.Observe(0.123)  # 记录123ms
```
**查询**:
```promql
histogram_quantile(0.99, ...)  # P99 延迟
```

### Summary（摘要）
**特点**: 客户端计算
**用途**: 少用（Histogram 更好）

---

## 🔧 常用命令

### 查看限流日志
```bash
tail -f logs/web-service.log | grep "频率超限"
```

### 查看熔断日志
```bash
tail -f logs/web-service.log | grep "熔断器"
```

### 查看 Prometheus 指标
```bash
curl http://localhost:8080/metrics | grep web_http_requests_total
```

### 查看健康状态
```bash
curl http://localhost:8080/health/ready | jq
```

### 压力测试
```bash
# 使用 wrk
wrk -t12 -c400 -d30s http://localhost:8080/

# 使用 Apache Bench
ab -n 1000 -c 100 http://localhost:8080/
```

---

## 🚨 告警规则（PromQL）

### 错误率告警
```promql
sum(rate(web_http_requests_total{status=~"5.."}[5m])) /
sum(rate(web_http_requests_total[5m])) > 0.05
```

### P99 延迟告警
```promql
histogram_quantile(0.99,
  rate(web_http_request_duration_seconds_bucket[5m])
) > 1
```

### 熔断器打开告警
```promql
web_circuit_breaker_state == 1
```

### 限流拒绝率告警
```promql
rate(web_rate_limit_rejects_total[5m]) > 10
```

---

## 💡 记忆口诀

### 限流
> 令牌桶里拿令牌，<br>
> 有令牌就放行，<br>
> 没令牌就拒绝，<br>
> 防止系统被打爆。

### 熔断
> 后端挂了快失败，<br>
> 不等超时直接断，<br>
> 等待片刻再试探，<br>
> 成功恢复自动关。

### 三状态
> CLOSED 正常发请求，<br>
> OPEN 拒绝不发送，<br>
> HALF_OPEN 试探看，<br>
> 成功关闭失败开。

---

## 📞 紧急处理

### 限流误伤
```bash
# 临时提高限流阈值（需要重启）
# 修改 rate_limiter.go
globalLimiter = NewRateLimiter(time.Second, 200)  # 100 → 200

# 或：临时禁用限流
# 注释掉 router.Use(middleware.GlobalRateLimiter())
```

### 熔断器误触发
```bash
# 查看熔断器状态
curl http://localhost:8080/metrics | grep circuit_breaker

# 如果确认后端正常，可手动调整参数（需重启）
# 修改 circuit_breaker.go
maxFailures := 10  # 5 → 10（更宽容）
```

### 监控异常
```bash
# 检查 Prometheus 是否正常拉取
curl http://localhost:9090/api/v1/targets

# 检查 Grafana 数据源
curl http://localhost:3000/api/datasources
```

---

## 🎓 学习资源

### 限流
- Go rate package: https://pkg.go.dev/golang.org/x/time/rate
- Token Bucket 算法: https://en.wikipedia.org/wiki/Token_bucket
- Leaky Bucket 算法: https://en.wikipedia.org/wiki/Leaky_bucket

### 熔断
- Netflix Hystrix: https://github.com/Netflix/Hystrix/wiki
- Circuit Breaker Pattern: https://martinfowler.com/bliki/CircuitBreaker.html
- Go 熔断库: https://github.com/sony/gobreaker

### Prometheus
- 官方文档: https://prometheus.io/docs/
- PromQL 教程: https://prometheus.io/docs/prometheus/latest/querying/basics/
- Grafana 仪表盘: https://grafana.com/docs/

---

## ✅ 快速检查清单

部署前检查：
- [ ] 限流器已初始化
- [ ] 熔断器已初始化
- [ ] Prometheus 已注册
- [ ] 配置文件已准备
- [ ] 环境变量已设置
- [ ] 健康检查可访问
- [ ] /metrics 端点可访问
- [ ] 日志输出正常
- [ ] TraceID 传递正常
- [ ] 编译无错误

运行时监控：
- [ ] QPS 是否正常
- [ ] 错误率是否 < 5%
- [ ] P99 延迟是否 < 1s
- [ ] 熔断器是否频繁触发
- [ ] 限流拒绝率是否合理
- [ ] 下游服务健康状态

---

打印本卡片，贴在显示器旁边！📌
