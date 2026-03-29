package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 中间件：处理跨域请求
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Session-Id, X-Trace-ID")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
