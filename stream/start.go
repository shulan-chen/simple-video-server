package stream

import (
	"net/http"
	"os"
	"video-server/stream/middleware"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理 CORS（跨域资源共享）
func CORSMiddleware() gin.HandlerFunc {
	// 从环境变量读取允许的源，默认为本地开发环境
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8080" // 默认允许Web服务
	}

	return func(c *gin.Context) {
		// 1. 设置 CORS 头（只允许指定的源）
		origin := c.Request.Header.Get("Origin")
		if origin == allowedOrigin {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Session-Id, X-Trace-ID")
		}

		// 2. 处理 OPTIONS 预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func RegisterHandlers() *gin.Engine {
	r := gin.Default()

	// 中间件执行顺序
	// 注意：Web 网关已做限流（100 req/s per IP, 5 req/s per user），此处无需重复
	r.Use(middleware.TraceID())      // 1. TraceID（最先）
	r.Use(CORSMiddleware())          // 2. CORS 跨域处理
	r.Use(middleware.ErrorHandler()) // 3. 错误处理（最后）

	// 路由注册
	r.GET("/videos/:vid-id", streamOssHandler)
	r.POST("/videos/upload/:vid-id", uploadOssHandler)
	r.GET("/testVideoPage", testPageHandler)

	return r
}

// 已废弃，保留以供测试使用
func Start() {
	r := RegisterHandlers()
	// 注意：stream 服务监听 9090 端口
	r.Run(":9090")
}
