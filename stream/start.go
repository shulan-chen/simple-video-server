package stream

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// StreamMiddleware 统一处理 CORS 和连接限流
func StreamMiddleware(connLimitNumber int) gin.HandlerFunc {
	limiter := NewConnLimiter(connLimitNumber)

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
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Session-Id")
		}

		// 2. 处理 OPTIONS 预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 3. 限流器逻辑：获取连接
		if !limiter.GetConn() {
			c.String(http.StatusTooManyRequests, "Too many requests") // 使用 Gin 的输出方法
			c.Abort()                                                 // 拦截请求，不再往下执行
			return
		}

		// 4. 执行后续的处理函数
		c.Next()

		// 5. 请求处理完毕后：释放连接
		limiter.Release()
	}
}

func RegisterHandlers() *gin.Engine {
	r := gin.Default()

	// 注册全局中间件：限流 + CORS
	r.Use(StreamMiddleware(10))

	// 路由注册
	r.GET("/videos/:vid-id", streamOssHandler)
	r.POST("/videos/upload/:vid-id", uploadOssHandler)
	r.GET("/testVideoPage", testPageHandler)

	return r
}

func Start() {
	r := RegisterHandlers()
	// 注意：stream 服务监听 9090 端口
	r.Run(":9090")
}
