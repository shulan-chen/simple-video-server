package stream

import (
	"net/http"
	"os"
	"video-server/api/utils"
	"video-server/stream/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		traceID := middleware.GetTraceID(c)

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

		// 3. 限流器逻辑：获取连接
		if !limiter.GetConn() {
			utils.Logger.Warn("超过并发连接限制",
				zap.String("trace_id", traceID),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, utils.ErrStreamConcurrentLimit, "")
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

	// 中间件执行顺序
	r.Use(middleware.TraceID())        // 1. TraceID（最先）
	r.Use(StreamMiddleware(10))        // 2. CORS + 并发限流
	r.Use(middleware.ErrorHandler())   // 3. 错误处理（最后）

	// 路由注册
	r.GET("/videos/:vid-id", streamOssHandler)
	r.POST("/videos/upload/:vid-id", middleware.UploadRateLimiter(), uploadOssHandler)  // 添加频率限流
	r.GET("/testVideoPage", testPageHandler)

	return r
}

func Start() {
	r := RegisterHandlers()
	// 注意：stream 服务监听 9090 端口
	r.Run(":9090")
}
