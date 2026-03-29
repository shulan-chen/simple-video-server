package web

import (
	"video-server/web/middleware"
)

// CleanupRateLimiters 启动限流器清理任务（导出供 main.go 调用）
func CleanupRateLimiters() {
	middleware.CleanupRateLimiters()
}
