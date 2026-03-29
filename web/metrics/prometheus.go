package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus 指标定义
var (
	// HTTP 请求总数
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "web_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求耗时（直方图）
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
		[]string{"method", "path"},
	)

	// HTTP 请求大小（字节）
	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000}, // 100B, 1KB, 10KB, 100KB, 1MB
		},
		[]string{"method", "path"},
	)

	// HTTP 响应大小（字节）
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000}, // 100B, 1KB, 10KB, 100KB, 1MB
		},
		[]string{"method", "path", "status"},
	)

	// 并发请求数
	httpInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "web_http_in_flight_requests",
			Help: "Current number of HTTP requests being served",
		},
	)

	// 代理请求总数
	proxyRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "web_proxy_requests_total",
			Help: "Total number of proxy requests",
		},
		[]string{"target", "method", "status"},
	)

	// 代理请求耗时
	proxyRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_proxy_request_duration_seconds",
			Help:    "Proxy request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target", "method"},
	)

	// 熔断器状态（0=CLOSED, 1=OPEN, 2=HALF_OPEN）
	circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "web_circuit_breaker_state",
			Help: "Circuit breaker state (0=CLOSED, 1=OPEN, 2=HALF_OPEN)",
		},
		[]string{"circuit"},
	)

	// 熔断器失败计数
	circuitBreakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "web_circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		},
		[]string{"circuit"},
	)

	// 限流拒绝计数
	rateLimitRejects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "web_rate_limit_rejects_total",
			Help: "Total number of rate limit rejects",
		},
		[]string{"limiter"},
	)
)

// InitMetrics 初始化 Prometheus 指标（注册到 prometheus）
func InitMetrics() {
	// 注册所有指标
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		httpInFlightRequests,
		proxyRequestsTotal,
		proxyRequestDuration,
		circuitBreakerState,
		circuitBreakerFailures,
		rateLimitRejects,
	)
}

// PrometheusMiddleware Prometheus 指标收集中间件
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过 /metrics 端点（避免递归）
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		// 增加并发请求数
		httpInFlightRequests.Inc()
		defer httpInFlightRequests.Dec()

		// 记录请求大小
		reqSize := computeRequestSize(c.Request)
		httpRequestSize.WithLabelValues(c.Request.Method, c.Request.URL.Path).Observe(float64(reqSize))

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start).Seconds()

		// 记录指标
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path).Observe(duration)

		// 记录响应大小
		respSize := c.Writer.Size()
		if respSize > 0 {
			httpResponseSize.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Observe(float64(respSize))
		}
	}
}

// RecordProxyRequest 记录代理请求指标
func RecordProxyRequest(target, method string, duration time.Duration, statusCode int) {
	status := strconv.Itoa(statusCode)
	proxyRequestsTotal.WithLabelValues(target, method, status).Inc()
	proxyRequestDuration.WithLabelValues(target, method).Observe(duration.Seconds())
}

// UpdateCircuitBreakerState 更新熔断器状态指标
func UpdateCircuitBreakerState(circuit string, state int) {
	circuitBreakerState.WithLabelValues(circuit).Set(float64(state))
}

// RecordCircuitBreakerFailure 记录熔断器失败
func RecordCircuitBreakerFailure(circuit string) {
	circuitBreakerFailures.WithLabelValues(circuit).Inc()
}

// RecordRateLimitReject 记录限流拒绝
func RecordRateLimitReject(limiter string) {
	rateLimitRejects.WithLabelValues(limiter).Inc()
}

// PrometheusHandler 返回 Prometheus metrics handler
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// computeRequestSize 计算请求大小
func computeRequestSize(r *http.Request) int {
	size := 0
	if r.URL != nil {
		size += len(r.URL.Path)
	}

	size += len(r.Method)
	size += len(r.Proto)

	for name, values := range r.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}

	if r.ContentLength > 0 {
		size += int(r.ContentLength)
	}

	return size
}
