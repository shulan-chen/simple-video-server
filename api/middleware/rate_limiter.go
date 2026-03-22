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
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // 每秒允许的请求数
	burst    int        // 突发容量
}

// NewRateLimiter 创建限流器
// interval: 时间间隔（如20*time.Second表示每20秒）
// maxRequests: 时间间隔内最大请求数
func NewRateLimiter(interval time.Duration, maxRequests int) *RateLimiter {
	// 计算每秒速率
	r := rate.Every(interval / time.Duration(maxRequests))
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    maxRequests,
	}
}

// getLimiter 获取或创建限流器
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
	// 视频上传限流：每用户每分钟3个
	uploadLimiter *RateLimiter

	// 评论发布限流：每用户每分钟10条
	commentLimiter *RateLimiter

	// 用户注册限流：每IP每小时5个
	registerLimiter *RateLimiter

	once sync.Once
)

// InitRateLimiters 初始化限流器（在应用启动时调用一次）
func InitRateLimiters() {
	once.Do(func() {
		uploadLimiter = NewRateLimiter(20*time.Second, 3)   // 每20秒1个，突发3个 = 每分钟最多3个
		commentLimiter = NewRateLimiter(6*time.Second, 10)  // 每6秒1个，突发10个 = 每分钟最多10个
		registerLimiter = NewRateLimiter(12*time.Minute, 5) // 每12分钟1个，突发5个 = 每小时最多5个
	})
}

// UploadRateLimiter 视频上传限流中间件
func UploadRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		userID := c.GetString("user_id")

		if userID == "" {
			// 如果没有用户ID（未登录），用IP限流
			userID = c.ClientIP()
		}

		if !uploadLimiter.Allow(userID) {
			utils.Logger.Warn("视频上传频率超限",
				zap.String("trace_id", traceID),
				zap.String("user_id", userID),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, utils.ErrAPIRateLimitExceeded, "")
			return
		}

		c.Next()
	}
}

// CommentRateLimiter 评论发布限流中间件
func CommentRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		userID := c.GetString("user_id")

		if userID == "" {
			userID = c.ClientIP()
		}

		if !commentLimiter.Allow(userID) {
			utils.Logger.Warn("评论发布频率超限",
				zap.String("trace_id", traceID),
				zap.String("user_id", userID),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, utils.ErrAPIRateLimitExceeded, "")
			return
		}

		c.Next()
	}
}

// RegisterRateLimiter 用户注册限流中间件
func RegisterRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		ip := c.ClientIP()

		if !registerLimiter.Allow(ip) {
			utils.Logger.Warn("注册频率超限",
				zap.String("trace_id", traceID),
				zap.String("ip", ip))
			utils.AbortWithErrorMsg(c, utils.ErrAPIRateLimitExceeded, "")
			return
		}

		c.Next()
	}
}

// CleanupRateLimiters 定期清理不活跃的限流器（可选，节省内存）
func CleanupRateLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// 简单清理：重置所有限流器
		// 实际生产环境可以根据LRU策略清理
		uploadLimiter.mu.Lock()
		uploadLimiter.limiters = make(map[string]*rate.Limiter)
		uploadLimiter.mu.Unlock()

		commentLimiter.mu.Lock()
		commentLimiter.limiters = make(map[string]*rate.Limiter)
		commentLimiter.mu.Unlock()

		registerLimiter.mu.Lock()
		registerLimiter.limiters = make(map[string]*rate.Limiter)
		registerLimiter.mu.Unlock()

		utils.Logger.Info("限流器清理完成")
	}
}

// GenericRateLimiter 通用限流中间件（可自定义配置）
func GenericRateLimiter(keyFunc func(*gin.Context) string, limiter *RateLimiter, errCode int) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		key := keyFunc(c)

		if !limiter.Allow(key) {
			utils.Logger.Warn(fmt.Sprintf("请求频率超限 (key=%s)", key),
				zap.String("trace_id", traceID),
				zap.String("key", key),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, errCode, "")
			return
		}

		c.Next()
	}
}
