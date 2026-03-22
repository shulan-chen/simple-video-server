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
		// 尝试从请求头获取TraceID
		traceID := c.GetHeader(TraceIDHeader)

		// 如果没有，生成新的
		if traceID == "" {
			traceID, _ = utils.NewUUID()
		}

		// 存储到context
		c.Set(TraceIDKey, traceID)

		// 添加到响应头
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
