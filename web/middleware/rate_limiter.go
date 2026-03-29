package middleware

import (
	"fmt"
	"sync"
	"time"

	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器管理器
// 使用 Token Bucket 算法实现限流
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // 每秒允许的请求数（速率）
	burst    int        // 突发容量（桶大小）
}

// NewRateLimiter 创建限流器
// interval: 时间间隔（如1*time.Second表示每秒）
// maxRequests: 时间间隔内最大请求数
func NewRateLimiter(interval time.Duration, maxRequests int) *RateLimiter {
	// 计算每秒速率：rate.Every 返回每个请求之间的时间间隔
	r := rate.Every(interval / time.Duration(maxRequests))
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    maxRequests,
	}
}

// getLimiter 获取或创建限流器（每个 IP/用户一个独立的桶）
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// 全局限流器实例
var (
	// 全局限流：每个 IP 每秒 100 个请求（防止 DDoS）
	globalLimiter *RateLimiter

	// API 透传限流：每个用户每秒 10 个请求
	apiProxyLimiter *RateLimiter

	// 视频代理限流：每个用户每秒 5 个请求
	videoProxyLimiter *RateLimiter

	rateLimiterOnce sync.Once
)

// InitRateLimiters 初始化限流器（在应用启动时调用一次）
func InitRateLimiters() {
	rateLimiterOnce.Do(func() {
		// 全局限流：100 req/s per IP，突发 200
		globalLimiter = NewRateLimiter(time.Second, 100)

		// API 透传限流：10 req/s per user，突发 20
		apiProxyLimiter = NewRateLimiter(time.Second, 10)

		// 视频代理限流：5 req/s per user，突发 10
		videoProxyLimiter = NewRateLimiter(time.Second, 5)

		utils.Logger.Info("限流器初始化完成",
			zap.Int("global_rate", 100),
			zap.Int("api_proxy_rate", 10),
			zap.Int("video_proxy_rate", 5))
	})
}

// GlobalRateLimiter 全局限流中间件（防止 DDoS）
func GlobalRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		ip := c.ClientIP()

		if !globalLimiter.Allow(ip) {
			utils.Logger.Warn("全局请求频率超限",
				zap.String("trace_id", traceID),
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path))
			utils.AbortWithError(c, utils.ErrWebRateLimitExceeded, nil)
			return
		}

		c.Next()
	}
}

// APIProxyRateLimiter API 透传限流中间件
func APIProxyRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		// 优先使用用户 ID，否则使用 IP
		key := c.GetString("user_id")
		if key == "" {
			key = c.ClientIP()
		}

		if !apiProxyLimiter.Allow(key) {
			utils.Logger.Warn("API透传请求频率超限",
				zap.String("trace_id", traceID),
				zap.String("key", key),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithError(c, utils.ErrWebRateLimitExceeded, nil)
			return
		}

		c.Next()
	}
}

// VideoProxyRateLimiter 视频代理限流中间件
func VideoProxyRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		key := c.GetString("user_id")
		if key == "" {
			key = c.ClientIP()
		}

		if !videoProxyLimiter.Allow(key) {
			utils.Logger.Warn("视频代理请求频率超限",
				zap.String("trace_id", traceID),
				zap.String("key", key),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithError(c, utils.ErrWebRateLimitExceeded, nil)
			return
		}

		c.Next()
	}
}

// CleanupRateLimiters 定期清理不活跃的限流器（节省内存）
// 每小时清理一次，防止 limiters map 无限增长
func CleanupRateLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// 清理所有限流器的缓存
		cleanLimiter := func(rl *RateLimiter, name string) {
			rl.mu.Lock()
			oldSize := len(rl.limiters)
			rl.limiters = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
			utils.Logger.Info(fmt.Sprintf("%s限流器清理完成", name),
				zap.Int("cleaned_count", oldSize))
		}

		cleanLimiter(globalLimiter, "全局")
		cleanLimiter(apiProxyLimiter, "API透传")
		cleanLimiter(videoProxyLimiter, "视频代理")
	}
}

// GenericRateLimiter 通用限流中间件（可自定义配置）
// keyFunc: 生成限流 key 的函数（如按 IP、按用户 ID）
// limiter: 限流器实例
func GenericRateLimiter(keyFunc func(*gin.Context) string, limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		key := keyFunc(c)

		if !limiter.Allow(key) {
			utils.Logger.Warn("请求频率超限",
				zap.String("trace_id", traceID),
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path))
			utils.AbortWithError(c, utils.ErrWebRateLimitExceeded, nil)
			return
		}

		c.Next()
	}
}
