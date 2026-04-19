package web

import (
	"net/http/httputil"
	"net/url"
	"strings"

	"video-server/api/utils"
	"video-server/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 注意：原有的 apiRequestProcess 和 doAPIRequest 函数已删除
// 现在使用反向代理方式直接转发请求到后端服务，不再需要手动构建 HTTP 请求

// getAPIAddr 获取API服务地址（从配置读取）
func getAPIAddr() string {
	if config.AppConfig.APIAddr != "" {
		// 移除端口号前的冒号，构建完整 HTTP URL
		addr := config.AppConfig.APIAddr
		if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
			return addr
		}
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
		if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
			return addr
		}
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

// proxyToAPI 代理请求到 API 服务
func proxyToAPI(c *gin.Context, traceID string, path string) error {
	apiAddr := getAPIAddr()
	u, err := url.Parse(apiAddr)
	if err != nil {
		utils.Logger.Error("解析API服务URL失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(u)

	// 修改请求路径（去掉 /api 前缀，确保有前导斜杠）
	originalPath := c.Request.URL.Path
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	c.Request.URL.Path = path
	if c.Request.URL.RawPath != "" {
		c.Request.URL.RawPath = path
	}

	// 添加 TraceID 到请求头
	if traceID != "" {
		c.Request.Header.Set("X-Trace-ID", traceID)
	}

	utils.Logger.Debug("代理到API服务",
		zap.String("trace_id", traceID),
		zap.String("original_path", originalPath),
		zap.String("proxy_path", path),
		zap.String("target", apiAddr))

	// 执行代理
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}

// proxyToStream 代理请求到 Stream 服务
func proxyToStream(c *gin.Context, traceID string, path string) error {
	streamAddr := getStreamAddr()
	u, err := url.Parse(streamAddr)
	if err != nil {
		utils.Logger.Error("解析Stream服务URL失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(u)

	// 修改请求路径（去掉 /stream 前缀，确保有前导斜杠）
	originalPath := c.Request.URL.Path
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	c.Request.URL.Path = path
	if c.Request.URL.RawPath != "" {
		c.Request.URL.RawPath = path
	}

	// 添加 TraceID 到请求头
	if traceID != "" {
		c.Request.Header.Set("X-Trace-ID", traceID)
	}

	utils.Logger.Debug("代理到Stream服务",
		zap.String("trace_id", traceID),
		zap.String("original_path", originalPath),
		zap.String("proxy_path", path),
		zap.String("target", streamAddr))

	// 执行代理
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}
