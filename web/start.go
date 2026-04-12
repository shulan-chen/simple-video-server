package web

import (
	"video-server/web/metrics"
	"video-server/web/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterHandlers 注册路由和中间件
func RegisterHandlers() *gin.Engine {
	router := gin.Default()

	// ========== 初始化组件 ==========
	// 初始化限流器
	middleware.InitRateLimiters()
	// 初始化熔断器
	middleware.InitCircuitBreakers()
	// 初始化 Prometheus 指标
	metrics.InitMetrics()

	// ========== 中间件注册（顺序很重要）==========
	// 1. TraceID 中间件（最先执行，为所有请求生成追踪ID）
	router.Use(middleware.TraceID())

	// 2. CORS 中间件（处理跨域请求）
	router.Use(middleware.CORS())

	// 3. 全局限流中间件（防止 DDoS）
	router.Use(middleware.GlobalRateLimiter())

	// 4. HTTP 缓存控制中间件
	router.Use(middleware.CacheControl())

	// 5. Prometheus 指标收集中间件
	router.Use(metrics.PrometheusMiddleware())

	// 6. 熔断器中间件（检查下游服务状态）
	router.Use(middleware.CircuitBreakerMiddleware())

	// 7. ErrorHandler 中间件（最后执行，捕获所有错误）
	router.Use(middleware.ErrorHandler())

	// ========== 路由注册（网关模式 - 前后端分离）==========
	// Prometheus metrics 端点
	router.GET("/metrics", metrics.PrometheusHandler())

	// API 代理路由组（代理到 API 服务）
	apiGroup := router.Group("/api", middleware.APIProxyRateLimiter())
	{
		// 匹配所有 /api/* 的请求并代理到 API 服务
		apiGroup.Any("/*path", proxyToAPIHandler)
	}

	// Stream 代理路由组（代理到 Stream 服务）
	streamGroup := router.Group("/stream", middleware.VideoProxyRateLimiter())
	{
		// 匹配所有 /stream/* 的请求并代理到 Stream 服务
		streamGroup.Any("/*path", proxyToStreamHandler)
	}

	return router
}

// Start 启动Web服务（已废弃，使用 cmd/web/main.go 代替）
// 保留此函数以保持向后兼容
func Start() {
	r := RegisterHandlers()
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
