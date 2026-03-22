package middleware

import (
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
)

const TraceIDKey = "trace_id"
const TraceIDHeader = "X-Trace-ID"

// TraceID 中间件：为每个请求生成唯一的TraceID
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 尝试从请求头获取TraceID（客户端传递）
		traceID := c.GetHeader(TraceIDHeader)

		// 2. 如果没有，生成新的TraceID
		if traceID == "" {
			traceID, _ = utils.NewUUID()
		}

		// 3. 存储到context
		c.Set(TraceIDKey, traceID)

		// 4. 添加到响应头（方便前端追踪）
		c.Writer.Header().Set(TraceIDHeader, traceID)

		c.Next()
	}
}

// GetTraceID 从context中获取TraceID
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDKey); exists {
		return traceID.(string)
	}
	return ""
}
