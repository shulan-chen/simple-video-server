package web

import (
	"time"

	"video-server/api/utils"
	"video-server/web/metrics"
	"video-server/web/middleware"

	"github.com/gin-gonic/gin"
)

// proxyToAPIHandler 代理所有 /api/* 请求到 API 服务（带熔断器保护）
func proxyToAPIHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	path := c.Param("path")
	start := time.Now()

	// 通过熔断器执行代理
	circuitBreaker := middleware.GetAPICircuitBreaker()
	err := circuitBreaker.Call(func() error {
		return proxyToAPI(c, traceID, path)
	})

	// 记录 Prometheus 指标
	duration := time.Since(start)
	statusCode := c.Writer.Status()
	if err != nil {
		if err.Error() == "circuit breaker is open" {
			statusCode = 503
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, err)
		}
	}
	metrics.RecordProxyRequest("api-service", c.Request.Method, duration, statusCode)
}

// proxyToStreamHandler 代理所有 /stream/* 请求到 Stream 服务（带熔断器保护）
func proxyToStreamHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	path := c.Param("path")
	start := time.Now()

	// 通过熔断器执行代理
	circuitBreaker := middleware.GetStreamCircuitBreaker()
	err := circuitBreaker.Call(func() error {
		return proxyToStream(c, traceID, path)
	})

	// 记录 Prometheus 指标
	duration := time.Since(start)
	statusCode := c.Writer.Status()
	if err != nil {
		if err.Error() == "circuit breaker is open" {
			statusCode = 503
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, err)
		}
	}
	metrics.RecordProxyRequest("stream-service", c.Request.Method, duration, statusCode)
}
