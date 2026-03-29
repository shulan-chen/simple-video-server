package middleware

import (
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
)

// TraceID 中间件：为每个请求生成唯一的追踪ID
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用请求头中的 TraceID（用于链路追踪）
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			// 如果没有，生成新的 TraceID
			traceID, _ = utils.NewUUID()
		}

		// 将 TraceID 存储到上下文中
		c.Set("trace_id", traceID)

		// 将 TraceID 添加到响应头
		c.Writer.Header().Set("X-Trace-ID", traceID)

		c.Next()
	}
}

// GetTraceID 从 gin.Context 中获取 TraceID
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		if tid, ok := traceID.(string); ok {
			return tid
		}
	}
	return ""
}
