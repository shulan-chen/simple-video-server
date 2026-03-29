# Web 网关生产级优化详解

## 目录

1. [限流保护（Rate Limiting）](#1-限流保护rate-limiting)
2. [熔断降级（Circuit Breaker）](#2-熔断降级circuit-breaker)
3. [HTTP 缓存](#3-http-缓存)
4. [下游服务健康检查](#4-下游服务健康检查)
5. [配置完善](#5-配置完善)
6. [Prometheus 监控指标](#6-prometheus-监控指标)

---

## 1. 限流保护（Rate Limiting）

### 1.1 什么是限流？为什么需要限流？

**限流（Rate Limiting）** 是一种**控制请求速率**的技术，用于保护系统不被过多的请求压垮。

#### 为什么需要限流？

1. **防止 DDoS 攻击**：
   - 黑客可以在短时间内发送大量请求，导致服务器资源耗尽
   - 例如：1秒内发送10000个请求，服务器会崩溃

2. **防止资源耗尽**：
   - 即使没有恶意攻击，某个用户的异常行为（如死循环调用API）也可能影响其他用户
   - 例如：一个用户的前端代码有 bug，不停地请求视频上传接口

3. **保证服务质量（QoS）**：
   - 确保每个用户都能公平地使用系统资源
   - 防止某个用户占用所有资源，导致其他用户无法访问

4. **成本控制**：
   - 限制对第三方API的调用频率（如阿里云OSS），避免产生过高的费用

#### 常见场景

| 场景 | 限流策略 | 理由 |
|------|----------|------|
| 用户注册 | 每IP每小时5次 | 防止批量注册账号 |
| 视频上传 | 每用户每分钟3次 | 防止滥用存储资源 |
| 评论发布 | 每用户每分钟10条 | 防止垃圾评论 |
| 全局请求 | 每IP每秒100次 | 防止 DDoS 攻击 |

### 1.2 限流算法：Token Bucket（令牌桶）

我们使用的是 **Token Bucket（令牌桶）** 算法，这是最常用的限流算法之一。

#### 原理讲解

想象有一个桶，桶里装着令牌（Token）：

```
┌─────────────────────┐
│   Token Bucket      │
│                     │
│  🪙🪙🪙🪙🪙         │  ← 桶里有5个令牌（burst=5）
│                     │
│  ┌───────────────┐  │
│  │ 水龙头         │  │  ← 以固定速率（rate）往桶里滴令牌
│  └───────────────┘  │
└─────────────────────┘
        ↓
    请求来了
```

**工作流程**：

1. **令牌生成**：以固定速率（rate）往桶里添加令牌
   - 例如：每秒添加10个令牌（rate = 10/s）

2. **令牌容量**：桶有最大容量（burst）
   - 例如：桶最多装20个令牌（burst = 20）
   - 如果桶满了，新产生的令牌会被丢弃

3. **请求处理**：
   - 请求来了，从桶里取1个令牌
   - 有令牌 → 允许请求通过，取走令牌
   - 无令牌 → 拒绝请求，返回 429 Too Many Requests

#### 为什么叫"令牌桶"？

- **令牌（Token）**：代表一次请求的许可证
- **桶（Bucket）**：存储令牌的容器
- **速率（Rate）**：令牌生成的速度
- **突发（Burst）**：桶的容量，允许的突发流量

#### 示例：每分钟最多3个请求

```go
// 创建限流器：每20秒1个令牌，突发容量3
// 20秒 × 3 = 60秒 = 1分钟
limiter := NewRateLimiter(20*time.Second, 3)
```

**时间轴演示**：

```
时间 | 桶里的令牌 | 请求 | 结果
-----|------------|------|------
0s   | 🪙🪙🪙 (3个) | 请求1 | ✅ 通过，剩2个令牌
1s   | 🪙🪙 (2个)   | 请求2 | ✅ 通过，剩1个令牌
2s   | 🪙 (1个)     | 请求3 | ✅ 通过，剩0个令牌
3s   | 0个          | 请求4 | ❌ 拒绝（无令牌）
20s  | 🪙 (1个)     | 请求5 | ✅ 通过（生成了1个新令牌）
40s  | 🪙🪙 (2个)   | 请求6 | ✅ 通过
```

#### Token Bucket vs Leaky Bucket（漏桶）

| 算法 | 特点 | 适用场景 |
|------|------|----------|
| **Token Bucket（令牌桶）** | 允许突发流量（burst）| 大多数场景 |
| **Leaky Bucket（漏桶）** | 平滑输出，不允许突发 | 需要严格匀速的场景 |

**我们选择 Token Bucket**：
- ✅ 允许用户短时间内发多个请求（burst=3）
- ✅ 长期平均速率仍受限（rate=3/分钟）
- ✅ 用户体验更好（不会因为一次突发被拒绝）

### 1.3 实现细节

#### 代码位置

```
web/middleware/rate_limiter.go
```

#### 核心结构

```go
type RateLimiter struct {
	limiters map[string]*rate.Limiter  // 每个用户/IP一个独立的限流器
	mu       sync.RWMutex               // 读写锁（并发安全）
	rate     rate.Limit                 // 令牌生成速率
	burst    int                        // 桶容量（突发大小）
}
```

**为什么每个用户一个独立的桶？**
- 不同用户之间互不影响
- 用户A的请求不会消耗用户B的令牌
- 这叫做 **per-user rate limiting**

#### 三层限流策略

| 层级 | 限流器 | 速率 | 作用 |
|------|--------|------|------|
| **全局限流** | `globalLimiter` | 100 req/s per IP | 防止 DDoS，保护整个网关 |
| **API透传限流** | `apiProxyLimiter` | 10 req/s per user | 保护 API 服务 |
| **视频代理限流** | `videoProxyLimiter` | 5 req/s per user | 保护 Stream 服务 |

#### 为什么需要三层限流？

1. **全局限流**（第一道防线）：
   - 阻止明显的 DDoS 攻击
   - 即使攻击者不停地换用户，也会被IP限流阻止

2. **API透传限流**（第二道防线）：
   - 保护后端 API 服务不被单个用户压垮
   - 防止用户的前端代码有bug导致无限循环请求

3. **视频代理限流**（第三道防线）：
   - 保护 Stream 服务和 OSS
   - 视频服务通常更重（上传/下载耗时长），需要更严格的限流

### 1.4 使用示例

#### 中间件注册

```go
// start.go 中
router.Use(middleware.GlobalRateLimiter())        // 全局限流
router.POST("/api", middleware.APIProxyRateLimiter(), apiHandler)  // API限流
router.GET("/videos/:vid-id", middleware.VideoProxyRateLimiter(), ...)  // 视频限流
```

#### 被限流时的响应

```json
{
  "code": 400601,
  "message": "请求过于频繁，请稍后再试",
  "detail": "",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP状态码：`429 Too Many Requests`

#### 如何查看限流日志？

```bash
# 查看限流日志
tail -f logs/web-service.log | grep "频率超限"

# 输出示例：
{
  "level": "warn",
  "msg": "全局请求频率超限",
  "trace_id": "xxx",
  "ip": "192.168.1.100",
  "path": "/api"
}
```

### 1.5 限流器清理（内存优化）

#### 问题

每个用户/IP都有一个独立的限流器，这会导致：
- `limiters` map 越来越大
- 不活跃用户的限流器仍占用内存

#### 解决方案

**定期清理**（每小时清理一次）：

```go
func CleanupRateLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		// 清空所有限流器的缓存
		globalLimiter.limiters = make(map[string]*rate.Limiter)
		apiProxyLimiter.limiters = make(map[string]*rate.Limiter)
		videoProxyLimiter.limiters = make(map[string]*rate.Limiter)
	}
}
```

**为什么可以这样做？**
- 清理后，不活跃用户的令牌桶被删除
- 下次请求时会重新创建（从满桶开始）
- 这对用户体验没有影响（因为他们本来就不活跃）

### 1.6 限流的生产环境考虑

#### 1. 分布式限流

**当前问题**：
- 我们的限流是在单个 Web 服务实例内的
- 如果有多个 Web 实例，限流不会生效（每个实例独立计数）

**解决方案**：
- 使用 Redis 存储令牌桶状态
- 所有实例共享同一个 Redis
- 使用 Redis 的 INCR 和 EXPIRE 命令实现分布式限流

```go
// 伪代码：Redis 分布式限流
func (rl *RateLimiter) Allow(key string) bool {
	count := redis.Incr("rate_limit:" + key)
	if count == 1 {
		redis.Expire("rate_limit:" + key, 60) // 60秒过期
	}
	return count <= 10 // 每分钟最多10次
}
```

#### 2. 动态调整限流阈值

**需求**：
- 正常情况：10 req/s
- 高峰期：5 req/s（更严格）
- 特殊用户：100 req/s（VIP用户）

**实现**：
- 从配置中心读取限流阈值
- 支持热更新（不重启服务）
- 支持按用户级别设置不同的限流策略

#### 3. 监控和告警

**需要监控**：
- 限流拒绝次数（按时间、按用户统计）
- 限流拒绝比例（拒绝次数/总请求数）
- 如果拒绝比例 > 10%，触发告警

**Prometheus 指标**（已实现）：
```go
rateLimitRejects.WithLabelValues("global").Inc()
```

---

## 2. 熔断降级（Circuit Breaker）

### 2.1 什么是熔断？为什么需要熔断？

**熔断（Circuit Breaker）** 是一种**快速失败**机制，当下游服务故障时，自动切断请求，防止故障蔓延。

#### 类比：家里的电闸

想象你家里的电闸（断路器）：

```
正常情况：
┌─────────┐     电流     ┌─────────┐
│  电源    │ ─────────→  │  电器    │
└─────────┘              └─────────┘
         电闸关闭（通电）

电路短路：
┌─────────┐    电流过大    ┌─────────┐
│  电源    │ ─────X─────→  │  电器    │
└─────────┘                └─────────┘
         电闸自动打开（断电）
         防止电器烧坏！
```

**熔断器也是同样的原理**：

```
正常情况：
┌─────────┐    请求正常    ┌─────────┐
│ Web网关  │ ──────────→  │ API服务  │
└─────────┘              └─────────┘
      熔断器 CLOSED（关闭状态）

API服务故障：
┌─────────┐   快速失败     ┌─────────┐
│ Web网关  │ ─────X────→  │ API服务  │  ❌ 挂了
└─────────┘              └─────────┘
      熔断器 OPEN（打开状态）
      不再发送请求，直接返回错误
```

#### 为什么需要熔断？

**场景：API 服务挂了**

没有熔断器：
```
Web → API（等待30秒超时）→ 超时失败
Web → API（等待30秒超时）→ 超时失败
Web → API（等待30秒超时）→ 超时失败
...
```
- 每个请求都要等30秒才知道失败
- Web 服务的线程/goroutine 被阻塞
- 最终 Web 服务也会被拖垮（资源耗尽）

**有熔断器**：
```
Web → API（失败1次）
Web → API（失败2次）
Web → API（失败3次）
Web → API（失败4次）
Web → API（失败5次）→ 触发熔断！
Web → 直接返回错误（不再请求API）  ← 0.1ms 就返回
Web → 直接返回错误（不再请求API）
Web → 直接返回错误（不再请求API）
...（30秒后自动尝试恢复）
Web → API（尝试1次）→ 成功 → 关闭熔断器
```

**好处**：
1. ✅ 快速失败（0.1ms vs 30秒）
2. ✅ 保护 Web 服务（不会被拖垮）
3. ✅ 给 API 服务恢复的时间（不再发送请求）
4. ✅ 自动恢复（API 恢复后自动关闭熔断器）

### 2.2 熔断器的三种状态

熔断器是一个**状态机**，有三种状态：

```
                  连续失败5次
     CLOSED ─────────────────→ OPEN
       ↑                         │
       │                         │
       │                         │ 30秒超时
       │                         ↓
       └───────────────── HALF_OPEN
            连续成功3次
```

#### 1. CLOSED（关闭状态）- 正常工作

- **含义**：熔断器关闭（电路接通），请求正常发送
- **行为**：
  - 所有请求都发送到下游服务
  - 记录失败次数
  - 如果连续失败达到阈值（如5次），切换到 OPEN 状态

**示例**：
```
请求1 → API服务 → ✅ 成功（失败计数 = 0）
请求2 → API服务 → ✅ 成功（失败计数 = 0）
请求3 → API服务 → ❌ 失败（失败计数 = 1）
请求4 → API服务 → ❌ 失败（失败计数 = 2）
请求5 → API服务 → ❌ 失败（失败计数 = 3）
请求6 → API服务 → ❌ 失败（失败计数 = 4）
请求7 → API服务 → ❌ 失败（失败计数 = 5）→ 触发熔断！
```

#### 2. OPEN（打开状态）- 熔断器打开

- **含义**：熔断器打开（电路断开），拒绝所有请求
- **行为**：
  - 所有请求立即返回错误（不发送到下游）
  - 记录熔断开始时间
  - 等待超时时间（如30秒），然后切换到 HALF_OPEN 状态

**示例**：
```
请求8 → 熔断器 → ❌ 直接返回错误（不请求API）
请求9 → 熔断器 → ❌ 直接返回错误（不请求API）
...（30秒后）
请求10 → 熔断器 → 进入 HALF_OPEN 状态
```

**为什么需要 OPEN 状态？**
- 给下游服务恢复的时间（不再打扰它）
- 保护自身（不浪费资源等待超时）
- 快速失败（立即返回错误，用户体验更好）

#### 3. HALF_OPEN（半开状态）- 尝试恢复

- **含义**：熔断器半开（尝试接通电路），允许部分请求通过
- **行为**：
  - 允许少量请求发送到下游（探测是否恢复）
  - 如果连续成功（如3次），切换到 CLOSED 状态
  - 如果失败，立即切换回 OPEN 状态

**示例**：
```
请求10 → API服务 → ✅ 成功（成功计数 = 1）
请求11 → API服务 → ✅ 成功（成功计数 = 2）
请求12 → API服务 → ✅ 成功（成功计数 = 3）→ 关闭熔断器！
```

**如果又失败了？**
```
请求10 → API服务 → ✅ 成功（成功计数 = 1）
请求11 → API服务 → ❌ 失败 → 立即重新打开熔断器！
```

**为什么需要 HALF_OPEN 状态？**
- 自动检测下游服务是否恢复（不需要人工干预）
- 如果未恢复，立即打开熔断器（不会雪上加霜）
- 如果已恢复，自动关闭熔断器（恢复正常服务）

### 2.3 熔断器参数调优

#### 核心参数

| 参数 | 含义 | 建议值 | 说明 |
|------|------|--------|------|
| `maxFailures` | 触发熔断的失败次数 | 5 | 连续5次失败，打开熔断器 |
| `timeout` | 熔断器打开的时间 | 30秒 | 30秒后进入 HALF_OPEN 状态 |
| `halfOpenSuccess` | 关闭熔断的成功次数 | 3 | 半开状态下连续3次成功，关闭熔断器 |

#### 如何选择参数？

**1. maxFailures（失败阈值）**

- 太小（如2）：可能误判（偶尔失败就熔断）
- 太大（如20）：下游服务已经挂了，还在一直请求
- **建议**：5-10次

**场景分析**：
```
API服务平均响应时间：100ms
如果挂了，每次超时：30秒

maxFailures = 5：
- 5 × 30秒 = 150秒后才熔断
- 这期间会阻塞5个请求

maxFailures = 10：
- 10 × 30秒 = 300秒后才熔断
- 这期间会阻塞10个请求

权衡：5次是比较合理的阈值
```

**2. timeout（熔断时间）**

- 太短（如5秒）：下游服务还没恢复，就开始尝试请求
- 太长（如5分钟）：下游服务已经恢复了，但熔断器还开着
- **建议**：30-60秒

**场景分析**：
```
API服务重启需要：10秒
如果熔断时间设置为5秒：
- 5秒后尝试请求 → 失败（服务还在重启）
- 立即重新打开熔断器 → 又等5秒
- 不停地重试，浪费资源

如果熔断时间设置为30秒：
- 30秒后尝试请求 → 成功（服务已重启完毕）
- 关闭熔断器 → 恢复正常
```

**3. halfOpenSuccess（恢复阈值）**

- 太小（如1）：可能误判（偶尔成功就关闭熔断）
- 太大（如10）：恢复太慢
- **建议**：3-5次

### 2.4 实现细节

#### 代码位置

```
web/middleware/circuit_breaker.go
```

#### 核心结构

```go
type CircuitBreaker struct {
	name         string        // 熔断器名称（如 "api-service"）
	state        CircuitState  // 当前状态（CLOSED/OPEN/HALF_OPEN）
	failureCount int           // 连续失败次数
	successCount int           // 半开状态下的成功次数
	lastFailTime time.Time     // 最后一次失败时间
	mu           sync.RWMutex  // 读写锁（并发安全）

	maxFailures     int           // 触发熔断的最大失败次数
	timeout         time.Duration // 熔断超时时间
	halfOpenSuccess int           // 半开状态下需要的成功次数
}
```

#### 使用示例

```go
// client.go 中使用熔断器
func apiRequestProcess(apiBody *ApiBody, w http.ResponseWriter, req *http.Request) {
	// 获取 API 服务的熔断器
	circuitBreaker := middleware.GetAPICircuitBreaker()

	// 通过熔断器执行请求
	err := circuitBreaker.Call(func() error {
		return doAPIRequest(apiBody, w, req)  // 实际的请求逻辑
	})

	if err != nil {
		if err.Error() == "circuit breaker is open" {
			// 熔断器打开，直接返回错误
			writeErrorResponse(w, utils.ErrWebCircuitBreakerOpen, "API服务暂时不可用", traceID)
		}
	}
}
```

#### 熔断器工作流程

```go
func (cb *CircuitBreaker) Call(fn func() error) error {
	// 1. 检查是否允许请求
	if !cb.AllowRequest() {
		return errors.New("circuit breaker is open")
	}

	// 2. 执行请求
	err := fn()

	// 3. 记录结果
	if err != nil {
		cb.RecordFailure()  // 记录失败
	} else {
		cb.RecordSuccess()  // 记录成功
	}

	return err
}
```

### 2.5 熔断器的生产环境考虑

#### 1. 熔断器监控

**需要监控的指标**：
- 熔断器状态（CLOSED/OPEN/HALF_OPEN）
- 失败次数
- 熔断器打开/关闭的时间点

**Prometheus 指标**（已实现）：
```go
// 熔断器状态（0=CLOSED, 1=OPEN, 2=HALF_OPEN）
circuitBreakerState.WithLabelValues("api-service").Set(1)

// 熔断器失败计数
circuitBreakerFailures.WithLabelValues("api-service").Inc()
```

**告警规则**：
```yaml
# 熔断器打开超过5分钟，触发告警
alert: CircuitBreakerOpen
expr: web_circuit_breaker_state{circuit="api-service"} == 1 for: 5m
labels:
  severity: critical
annotations:
  summary: "API服务熔断器已打开超过5分钟"
```

#### 2. 降级策略

当熔断器打开时，不应该只是返回错误，应该有**降级方案**：

```go
func apiRequestProcess(...) {
	err := circuitBreaker.Call(func() error {
		return doAPIRequest(...)
	})

	if err != nil && err.Error() == "circuit breaker is open" {
		// 降级策略：返回缓存数据
		cachedData := cache.GetFallbackData()
		if cachedData != nil {
			w.Write(cachedData)
			return
		}

		// 如果没有缓存，返回友好的错误信息
		writeErrorResponse(w, ..., "服务繁忙，请稍后再试")
	}
}
```

**常见降级策略**：
- 返回缓存数据
- 返回默认值
- 返回简化版数据
- 返回友好的错误页面

#### 3. 熔断器配置管理

**当前问题**：
- 熔断器参数硬编码在代码中
- 修改参数需要重启服务

**改进方案**：
- 从配置中心读取参数
- 支持热更新（不重启服务）
- 支持按服务设置不同的熔断策略

```json
// config.json 中
{
  "circuit_breakers": {
    "api-service": {
      "max_failures": 5,
      "timeout": "30s",
      "half_open_success": 3
    },
    "stream-service": {
      "max_failures": 3,
      "timeout": "60s",
      "half_open_success": 2
    }
  }
}
```

---

## 3. HTTP 缓存

### 3.1 为什么需要 HTTP 缓存？

**HTTP 缓存** 是让**浏览器或CDN**缓存响应，减少服务器负载和网络传输。

#### 缓存的好处

1. **减少服务器负载**：
   - 静态资源（CSS、JS、图片）被浏览器缓存
   - 不需要每次都从服务器下载

2. **加快页面加载速度**：
   - 浏览器缓存：0ms（直接读取本地）
   - 服务器请求：100-500ms

3. **节省带宽**：
   - 减少网络传输
   - 对移动端用户特别重要

### 3.2 三种缓存策略

#### 策略1：强缓存（Static Assets）

**适用于**：静态资源（CSS、JS、图片）

```http
Cache-Control: public, max-age=31536000, immutable
Expires: Mon, 01 Jan 2027 00:00:00 GMT
```

**含义**：
- `public`: 可以被任何缓存（浏览器、CDN）存储
- `max-age=31536000`: 缓存1年（365天）
- `immutable`: 资源不会改变，不需要验证

**工作流程**：
```
第一次访问：
浏览器 → 服务器 → 返回 style.css + Cache-Control
浏览器缓存1年

第二次访问（1年内）：
浏览器 → 直接读取本地缓存（不发送请求）
```

**文件修改怎么办？**
- 使用**文件名哈希**：`style.abc123.css`
- 修改文件后，文件名变化：`style.def456.css`
- 浏览器认为是新文件，会重新下载

#### 策略2：协商缓存（HTML Pages）

**适用于**：HTML 页面（可能会更新）

```http
Cache-Control: no-cache, must-revalidate
ETag: "1234567890"
```

**含义**：
- `no-cache`: 必须验证后才能使用缓存
- `must-revalidate`: 过期后必须重新验证
- `ETag`: 资源的唯一标识（如内容的哈希值）

**工作流程**：
```
第一次访问：
浏览器 → 服务器 → 返回 index.html + ETag: "abc123"
浏览器缓存

第二次访问：
浏览器 → 服务器（带 If-None-Match: "abc123"）
         ↓
      文件没变？
         ↓ 是
      返回 304 Not Modified（不返回内容）
         ↓
      浏览器使用本地缓存

      文件变了？
         ↓ 是
      返回 200 OK + 新内容 + 新ETag
```

**好处**：
- 文件没变：只传输几十字节（304响应）
- 文件变了：传输新内容

#### 策略3：不缓存（API Responses）

**适用于**：API 响应（实时数据）

```http
Cache-Control: no-store, no-cache, must-revalidate
Pragma: no-cache
Expires: 0
```

**含义**：
- `no-store`: 不存储任何缓存
- `Pragma: no-cache`: HTTP/1.0 兼容
- `Expires: 0`: 立即过期

**为什么 API 不缓存？**
- 数据实时性要求高（如用户余额）
- 缓存可能导致数据不一致

### 3.3 实现细节

#### 代码位置

```
web/middleware/cache.go
```

#### 中间件逻辑

```go
func CacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. 静态资源 - 长期缓存
		if isStaticResource(path) {  // /statics/*.css, *.js, *.png
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		// 2. HTML 页面 - 协商缓存
		if isHTMLPage(path) {  // /, /userhome
			c.Writer.Header().Set("Cache-Control", "no-cache, must-revalidate")
			etag := generateETag(path)
			c.Writer.Header().Set("ETag", etag)
		}

		// 3. API 响应 - 不缓存
		if isAPIEndpoint(path) {  // /api, /videos/*
			c.Writer.Header().Set("Cache-Control", "no-store, no-cache")
		}

		c.Next()
	}
}
```

### 3.4 缓存的生产环境考虑

#### 1. CDN 集成

**当前问题**：
- 只有浏览器缓存
- 用户分布在全国各地，访问速度慢

**解决方案**：
- 使用 CDN（如阿里云CDN、Cloudflare）
- 静态资源上传到 CDN
- CDN 缓存在全球各地的边缘节点

```
用户（北京） → CDN 北京节点（2ms）
用户（上海） → CDN 上海节点（2ms）
用户（广州） → CDN 广州节点（2ms）
```

#### 2. 缓存刷新策略

**问题**：发布新版本后，用户看到的还是旧版本

**解决方案**：
```html
<!-- 方案1：文件名哈希 -->
<link rel="stylesheet" href="/statics/style.abc123.css">
<script src="/statics/app.def456.js"></script>

<!-- 方案2：版本号参数 -->
<link rel="stylesheet" href="/statics/style.css?v=1.2.3">

<!-- 方案3：时间戳 -->
<link rel="stylesheet" href="/statics/style.css?t=1678901234">
```

---

## 4. 下游服务健康检查

### 4.1 为什么需要健康检查？

**健康检查（Health Check）** 是主动检测下游服务是否正常，提前发现问题。

#### 场景

```
情况1：没有健康检查
Web 服务启动 → 看起来正常
用户发请求 → Web → API服务（挂了）→ 失败
用户体验差！

情况2：有健康检查
Web 服务启动 → 检查 API 服务 → API 挂了！
                ↓
              记录日志、发送告警
                ↓
用户发请求 → Web → 熔断器已打开 → 快速失败（返回降级数据）
用户体验好！
```

### 4.2 实现细节

#### 代码位置

```
web/health/downstream.go
```

#### 健康检查逻辑

```go
type DownstreamChecker struct {
	apiAddr    string        // API 服务地址
	streamAddr string        // Stream 服务地址
	timeout    time.Duration // 超时时间（5秒）
}

// 检查 API 服务
func (dc *DownstreamChecker) CheckAPIService(ctx context.Context) error {
	url := fmt.Sprintf("%s/health/ready", dc.apiAddr)

	// 创建带超时的请求
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
```

#### 健康检查端点

```go
// 获取完整健康状态（包括下游服务）
router.GET("/health/ready", func(c *gin.Context) {
	status := health.GetHealthStatus(c.Request.Context())

	if status["status"] == "degraded" {
		c.JSON(503, status)  // Service Unavailable
	} else {
		c.JSON(200, status)  // OK
	}
})
```

**响应示例**：
```json
{
  "status": "healthy",  // 或 "degraded"
  "downstream": {
    "api": "healthy",
    "stream": "unhealthy: connection refused"
  }
}
```

### 4.3 健康检查的使用场景

#### 1. Kubernetes 健康探针

```yaml
# deployment.yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health/ready  # 包括下游服务检查
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

**工作原理**：
- `readinessProbe` 失败 → K8s 停止向该 Pod 发送流量
- `livenessProbe` 失败 → K8s 重启该 Pod

#### 2. 负载均衡器健康检查

```nginx
# Nginx upstream 健康检查
upstream web_backend {
    server 192.168.1.10:8080 max_fails=3 fail_timeout=30s;
    server 192.168.1.11:8080 max_fails=3 fail_timeout=30s;

    # 健康检查配置（Nginx Plus）
    health_check interval=10s uri=/health/ready;
}
```

#### 3. 监控和告警

```yaml
# Prometheus 告警规则
alert: DownstreamServiceUnhealthy
expr: web_downstream_health{service="api"} != 1
for: 2m
labels:
  severity: warning
annotations:
  summary: "API服务不健康已持续2分钟"
```

---

## 5. 配置完善

### 5.1 问题

**当前代码中的硬编码**：

```go
// handlers.go 中
u, _ := url.Parse("http://localhost:9090/")  // ❌ 硬编码

// client.go 中
apiAddr := "http://localhost:8000"  // ❌ 硬编码
```

**问题**：
- 无法适配不同环境（开发/测试/生产）
- 部署时需要修改代码
- 不符合 12-Factor App 原则

### 5.2 解决方案

#### 配置结构

```json
// config/config.json
{
  "api_addr": ":8000",         // API 服务地址
  "web_addr": ":8080",         // Web 服务地址（本服务）
  "stream_addr": ":9090",      // Stream 服务地址
  "db_addr": "localhost:3306",
  "redis_addr": "localhost:6379",
  ...
}
```

#### 配置读取逻辑

```go
// web/client.go
func getAPIAddr() string {
	addr := config.AppConfig.APIAddr
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr  // ":8000" → "http://localhost:8000"
	}
	if !strings.HasPrefix(addr, "http://") {
		return "http://" + addr  // "api-service" → "http://api-service"
	}
	return addr  // "http://api.example.com"
}
```

**支持三种格式**：
1. `:8000` → `http://localhost:8000`（开发环境）
2. `api-service` → `http://api-service`（K8s 服务名）
3. `http://api.example.com`（完整URL）

### 5.3 环境变量覆盖

```bash
# 通过环境变量覆盖配置
export API_ADDR="http://api-prod.example.com"
export STREAM_ADDR="http://stream-prod.example.com"

./bin/web-service
```

**Viper 会自动读取环境变量**（已在 internal/config 中配置）。

---

## 6. Prometheus 监控指标

### 6.1 为什么需要监控？

**监控（Monitoring）** 是观察系统运行状态，发现问题、分析性能的基础。

#### 没有监控的痛苦

```
用户：网站好慢啊！
运维：哪里慢？
用户：就是慢！
运维：？？？
```

#### 有监控后

```
用户：网站好慢啊！
运维：（打开 Grafana）
      → 发现 P99 延迟从 100ms 上升到 5秒
      → 发现 API 服务 CPU 100%
      → 发现数据库慢查询增加
      → 定位到具体的慢查询 SQL
      → 优化索引，问题解决
```

### 6.2 Prometheus 简介

**Prometheus** 是一个开源的监控系统，特点：
- **Pull 模式**：Prometheus 主动拉取指标（而不是被动接收）
- **时序数据库**：存储带时间戳的指标
- **强大的查询语言**：PromQL
- **自带告警系统**：Alertmanager

**工作流程**：
```
Web 服务 /metrics 端点 ← Prometheus Server（每15秒拉取一次）
                             ↓
                         时序数据库
                             ↓
                         Grafana（可视化）
```

### 6.3 我们收集的指标

#### 1. HTTP 请求指标

```go
// 请求总数（Counter - 只增不减）
httpRequestsTotal.WithLabelValues(method, path, status).Inc()

// 请求耗时（Histogram - 分桶统计）
httpRequestDuration.WithLabelValues(method, path).Observe(duration)

// 请求大小（Histogram）
httpRequestSize.WithLabelValues(method, path).Observe(size)

// 响应大小（Histogram）
httpResponseSize.WithLabelValues(method, path, status).Observe(size)

// 并发请求数（Gauge - 可增可减）
httpInFlightRequests.Inc()
defer httpInFlightRequests.Dec()
```

**指标类型说明**：

| 类型 | 说明 | 示例 |
|------|------|------|
| **Counter** | 只增不减的计数器 | 请求总数、错误总数 |
| **Gauge** | 可增可减的仪表 | 并发请求数、内存使用量 |
| **Histogram** | 分桶统计（用于计算百分位） | 请求耗时、响应大小 |
| **Summary** | 客户端计算百分位 | 不常用 |

#### 2. 代理请求指标

```go
// 代理请求总数
proxyRequestsTotal.WithLabelValues(target, method, status).Inc()

// 代理请求耗时
proxyRequestDuration.WithLabelValues(target, method).Observe(duration)
```

**示例**：
```
proxyRequestsTotal{target="api-service",method="POST",status="200"} 1234
proxyRequestDuration{target="api-service",method="POST"} 0.123
```

#### 3. 熔断器指标

```go
// 熔断器状态（0=CLOSED, 1=OPEN, 2=HALF_OPEN）
circuitBreakerState.WithLabelValues(circuit).Set(state)

// 熔断器失败计数
circuitBreakerFailures.WithLabelValues(circuit).Inc()
```

#### 4. 限流指标

```go
// 限流拒绝计数
rateLimitRejects.WithLabelValues(limiter).Inc()
```

### 6.4 Prometheus 查询示例

#### 查询请求速率（QPS）

```promql
# 每秒请求数
rate(web_http_requests_total[1m])

# 按 path 分组
sum by (path) (rate(web_http_requests_total[1m]))
```

#### 查询错误率

```promql
# 错误率（5xx 状态码）
sum(rate(web_http_requests_total{status=~"5.."}[1m])) /
sum(rate(web_http_requests_total[1m]))
```

#### 查询 P99 延迟

```promql
# P99 延迟（99% 的请求在多少秒内完成）
histogram_quantile(0.99,
  rate(web_http_request_duration_seconds_bucket[5m])
)
```

#### 查询熔断器状态

```promql
# 熔断器是否打开（1=OPEN）
web_circuit_breaker_state{circuit="api-service"} == 1
```

### 6.5 Grafana 可视化

#### 仪表盘示例

```
┌────────────────────────────────────────┐
│  Web Gateway Dashboard                 │
├────────────────────────────────────────┤
│ QPS: 1234 req/s     错误率: 0.5%     │
│ P99延迟: 150ms      并发: 42         │
├────────────────────────────────────────┤
│ 请求速率（折线图）                     │
│     ╱╲                                │
│    ╱  ╲╱╲                            │
│───╱────────╲────────                 │
├────────────────────────────────────────┤
│ 熔断器状态                             │
│ API:    ✅ CLOSED                      │
│ Stream: ❌ OPEN (5分钟)               │
├────────────────────────────────────────┤
│ 限流统计                               │
│ 全局限流拒绝: 123 次/分钟             │
│ API限流拒绝:  45 次/分钟              │
└────────────────────────────────────────┘
```

---

## 总结

### 六大优化对比

| 优化项 | 作用 | 受益方 | 优先级 |
|--------|------|--------|--------|
| **限流保护** | 防止 DDoS，保护系统资源 | 系统、所有用户 | ⭐⭐⭐⭐⭐ |
| **熔断降级** | 快速失败，防止故障蔓延 | 系统稳定性 | ⭐⭐⭐⭐⭐ |
| **HTTP 缓存** | 减少服务器负载，加快加载速度 | 用户体验、服务器 | ⭐⭐⭐⭐ |
| **健康检查** | 主动发现问题，配合熔断器 | 运维、监控 | ⭐⭐⭐⭐ |
| **配置完善** | 适配不同环境，易于部署 | 运维 | ⭐⭐⭐ |
| **Prometheus** | 可观测性，问题诊断 | 运维、开发 | ⭐⭐⭐⭐⭐ |

### 生产环境检查清单

- [ ] 限流器已启用
- [ ] 熔断器已配置
- [ ] 静态资源缓存已生效
- [ ] 下游服务健康检查已配置
- [ ] 配置从环境变量读取
- [ ] Prometheus metrics 端点可访问
- [ ] Grafana 仪表盘已创建
- [ ] 告警规则已配置（限流、熔断、错误率）
- [ ] 日志已集成到 ELK/Loki
- [ ] TraceID 已传递到所有下游服务

### 面试回答模板

**面试官**："你们的网关是如何保护后端服务的？"

**你**："我们使用了三层防护：

1. **限流保护**：采用 Token Bucket 算法，分为全局限流（100 req/s per IP）、API限流（10 req/s per user）、视频限流（5 req/s per user），防止 DDoS 和滥用。

2. **熔断降级**：当后端服务连续失败5次时，熔断器自动打开，请求快速失败（0.1ms vs 30秒超时），30秒后自动尝试恢复。如果恢复成功则关闭熔断器，失败则继续熔断。

3. **健康检查**：定期检查下游服务（API、Stream）的健康状态，配合熔断器使用，提前发现问题并告警。

此外，我们还使用 Prometheus 收集所有指标（QPS、延迟、错误率、熔断器状态、限流统计），通过 Grafana 可视化，配置了告警规则。"

**面试官**："👍 不错！"
