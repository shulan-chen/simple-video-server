package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheControl HTTP 缓存控制中间件
// 为静态资源和不经常变化的内容设置缓存头
func CacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. 静态资源（CSS, JS, 图片等）- 长期缓存
		if isStaticResource(path) {
			// Cache-Control: public（可被任何缓存存储）, max-age=31536000（1年）, immutable（不会改变）
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			// Expires: 1年后过期（兼容旧浏览器）
			c.Writer.Header().Set("Expires", time.Now().Add(365*24*time.Hour).Format(time.RFC1123))
		}

		// 2. HTML 页面 - 协商缓存（验证后使用）
		if isHTMLPage(path) {
			// Cache-Control: no-cache（需要验证），must-revalidate（过期必须重新验证）
			c.Writer.Header().Set("Cache-Control", "no-cache, must-revalidate")
			// 设置 ETag（基于内容的哈希，用于验证）
			// 注意：真实环境应该使用内容的实际哈希值
			etag := fmt.Sprintf(`"%s-%d"`, path, time.Now().Unix()/3600) // 每小时变化
			c.Writer.Header().Set("ETag", etag)
		}

		// 3. API 响应 - 不缓存
		if isAPIEndpoint(path) {
			// Cache-Control: no-store（不存储任何缓存）
			c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			// Pragma: no-cache（HTTP/1.0 兼容）
			c.Writer.Header().Set("Pragma", "no-cache")
			// Expires: 0（立即过期）
			c.Writer.Header().Set("Expires", "0")
		}

		c.Next()
	}
}

// isStaticResource 判断是否为静态资源
func isStaticResource(path string) bool {
	staticPrefixes := []string{
		"/statics/",
		"/static/",
		"/assets/",
	}

	staticExtensions := []string{
		".css", ".js", ".jpg", ".jpeg", ".png", ".gif", ".svg",
		".woff", ".woff2", ".ttf", ".eot", ".ico",
	}

	// 检查前缀
	for _, prefix := range staticPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	// 检查扩展名
	for _, ext := range staticExtensions {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}

	return false
}

// isHTMLPage 判断是否为 HTML 页面
func isHTMLPage(path string) bool {
	htmlPages := []string{
		"/",
		"/userhome",
	}

	for _, page := range htmlPages {
		if path == page {
			return true
		}
	}

	return false
}

// isAPIEndpoint 判断是否为 API 端点
func isAPIEndpoint(path string) bool {
	apiPrefixes := []string{
		"/api",
		"/videos/",
	}

	for _, prefix := range apiPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// NoCacheMiddleware 禁用缓存中间件（用于需要实时数据的接口）
func NoCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Next()
	}
}

// ConditionalGetMiddleware 条件 GET 中间件（支持 ETag 验证）
// 如果客户端发送的 If-None-Match 与当前 ETag 匹配，返回 304 Not Modified
func ConditionalGetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只处理 GET 请求
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// 检查 If-None-Match 头（客户端缓存的 ETag）
		ifNoneMatch := c.GetHeader("If-None-Match")
		if ifNoneMatch == "" {
			c.Next()
			return
		}

		// 生成当前资源的 ETag
		path := c.Request.URL.Path
		currentETag := fmt.Sprintf(`"%s-%d"`, path, time.Now().Unix()/3600)

		// 如果 ETag 匹配，返回 304 Not Modified
		if ifNoneMatch == currentETag {
			c.Writer.Header().Set("ETag", currentETag)
			c.AbortWithStatus(304) // 304 Not Modified
			return
		}

		c.Next()
	}
}
