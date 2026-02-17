package stream

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StreamMiddleware 统一处理 CORS 和连接限流
func StreamMiddleware(connLimitNumber int) gin.HandlerFunc {
	limiter := NewConnLimiter(connLimitNumber)

	return func(c *gin.Context) {
		// 1. 设置 CORS 头
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")

		// 2. 处理 OPTIONS 请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
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
