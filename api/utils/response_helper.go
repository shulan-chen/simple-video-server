package utils

import "github.com/gin-gonic/gin"

// AbortWithError 记录错误并终止请求处理
// 会被 ErrorHandler 中间件捕获并统一处理
func AbortWithError(c *gin.Context, code int, originalErr error) {
	traceID := c.GetString("trace_id")
	appErr := NewAppError(code, originalErr.Error(), traceID)
	c.Error(originalErr).SetMeta(appErr)
	c.Abort()
}

// AbortWithErrorMsg 记录错误（带自定义详细信息）并终止请求处理
func AbortWithErrorMsg(c *gin.Context, code int, detail string) {
	traceID := c.GetString("trace_id")
	appErr := NewAppError(code, detail, traceID)
	// 创建一个包装的error用于日志记录
	c.Error(&AppErrorWrapper{AppError: appErr}).SetMeta(appErr)
	c.Abort()
}

// AppErrorWrapper 包装 AppError 使其实现 error 接口
type AppErrorWrapper struct {
	*AppError
}

func (e *AppErrorWrapper) Error() string {
	return e.Message
}
