package web

import (
	"strings"

	"video-server/internal/config"
)

// 注意：原有的 apiRequestProcess 和 doAPIRequest 函数已删除
// 现在使用反向代理方式直接转发请求到后端服务，不再需要手动构建 HTTP 请求

// getAPIAddr 获取API服务地址（从配置读取）
func getAPIAddr() string {
	if config.AppConfig.APIAddr != "" {
		// 移除端口号前的冒号，构建完整 HTTP URL
		addr := config.AppConfig.APIAddr
		if strings.HasPrefix(addr, ":") {
			return "http://localhost" + addr
		}
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			return "http://" + addr
		}
		return addr
	}
	return "http://localhost:8000"
}

// getStreamAddr 获取Stream服务地址（从配置读取）
func getStreamAddr() string {
	if config.AppConfig.StreamAddr != "" {
		addr := config.AppConfig.StreamAddr
		if strings.HasPrefix(addr, ":") {
			return "http://localhost" + addr
		}
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			return "http://" + addr
		}
		return addr
	}
	return "http://localhost:9090"
}
