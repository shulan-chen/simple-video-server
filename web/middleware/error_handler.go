package middleware

import (
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) == 0 {
			return
		}

		// 获取最后一个错误
		err := c.Errors.Last()
		traceID := GetTraceID(c)

		// 记录错误日志
		utils.Logger.Error("请求处理失败",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.Error(err.Err))

		// 如果是 AppError，直接返回
		if appErr, ok := err.Meta.(*utils.AppError); ok {
			appErr.TraceID = traceID
			c.JSON(utils.GetHTTPStatus(appErr.Code), appErr)
			return
		}

		// 否则返回通用错误
		appErr := &utils.AppError{
			Code:    utils.ErrWebInternalFault,
			Message: "内部服务错误",
			Detail:  err.Error(),
			TraceID: traceID,
		}
		c.JSON(utils.GetHTTPStatus(appErr.Code), appErr)
	}
}
