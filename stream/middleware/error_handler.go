package middleware

import (
	"os"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	isDev := os.Getenv("ENV") == "development" || os.Getenv("ENV") == "dev"

	return func(c *gin.Context) {
		c.Next()

		// 只有当发生错误时才处理
		if len(c.Errors) == 0 {
			return
		}

		// 获取最后一个错误
		err := c.Errors.Last()
		traceID := GetTraceID(c)

		// 记录详细日志
		utils.Logger.Error("请求处理失败",
			zap.String("trace_id", traceID),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
			zap.Error(err.Err),
		)

		// 检查是否已经设置了响应
		if c.Writer.Written() {
			return
		}

		// 如果错误是 AppError 类型，使用它的信息
		if appErr, ok := err.Meta.(*utils.AppError); ok {
			// 开发环境返回详细错误，生产环境隐藏
			if !isDev {
				appErr.Detail = ""
			}
			c.JSON(utils.GetHTTPStatus(appErr.Code), appErr)
			return
		}

		// 否则返回通用内部错误
		appErr := utils.NewAppError(utils.ErrStreamInternal, "", traceID)
		if isDev {
			appErr.Detail = err.Error()
		}
		c.JSON(utils.GetHTTPStatus(utils.ErrStreamInternal), appErr)
	}
}
