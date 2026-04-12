package web

import (
	"net/http/httputil"
	"net/url"
	"time"

	"video-server/api/utils"
	"video-server/web/metrics"
	"video-server/web/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// proxyToAPIHandler 代理所有 /api/* 请求到 API 服务（带熔断器保护）
func proxyToAPIHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	path := c.Param("path")
	start := time.Now()

	utils.Logger.Info("代理API请求",
		zap.String("trace_id", traceID),
		zap.String("method", c.Request.Method),
		zap.String("path", path))

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

	utils.Logger.Info("代理Stream请求",
		zap.String("trace_id", traceID),
		zap.String("method", c.Request.Method),
		zap.String("path", path))

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

// proxyToAPI 代理请求到 API 服务
func proxyToAPI(c *gin.Context, traceID string, path string) error {
	apiAddr := getAPIAddr()
	u, err := url.Parse(apiAddr)
	if err != nil {
		utils.Logger.Error("解析API服务URL失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(u)

	// 修改请求路径（去掉 /api 前缀）
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = path
	if c.Request.URL.RawPath != "" {
		c.Request.URL.RawPath = path
	}

	// 添加 TraceID 到请求头
	if traceID != "" {
		c.Request.Header.Set("X-Trace-ID", traceID)
	}

	utils.Logger.Debug("代理到API服务",
		zap.String("trace_id", traceID),
		zap.String("original_path", originalPath),
		zap.String("proxy_path", path),
		zap.String("target", apiAddr))

	// 执行代理
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}

// proxyToStream 代理请求到 Stream 服务
func proxyToStream(c *gin.Context, traceID string, path string) error {
	streamAddr := getStreamAddr()
	u, err := url.Parse(streamAddr)
	if err != nil {
		utils.Logger.Error("解析Stream服务URL失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(u)

	// 修改请求路径（去掉 /stream 前缀）
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = path
	if c.Request.URL.RawPath != "" {
		c.Request.URL.RawPath = path
	}

	// 添加 TraceID 到请求头
	if traceID != "" {
		c.Request.Header.Set("X-Trace-ID", traceID)
	}

	utils.Logger.Debug("代理到Stream服务",
		zap.String("trace_id", traceID),
		zap.String("original_path", originalPath),
		zap.String("proxy_path", path),
		zap.String("target", streamAddr))

	// 执行代理
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}

