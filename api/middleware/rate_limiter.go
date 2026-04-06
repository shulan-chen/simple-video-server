package middleware

import (
	"fmt"
	"sync"
	"time"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang/groupcache/lru"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器管理器
// 使用 Token Bucket 算法实现限流，基于 LRU 缓存自动淘汰不活跃的限流器
type RateLimiter struct {
	cache *lru.Cache // LRU 缓存，自动淘汰最少使用的限流器
	mu    sync.Mutex
	rate  rate.Limit // 每秒允许的请求数
	burst int        // 突发容量
}

// NewRateLimiter 创建限流器
// interval: 时间间隔（如20*time.Second表示每20秒）
// maxRequests: 时间间隔内最大请求数
// cacheSize: LRU 缓存大小（限制最多保留多少个限流器）
func NewRateLimiter(interval time.Duration, maxRequests int, cacheSize int) *RateLimiter {
	// 计算每秒速率
	r := rate.Every(interval / time.Duration(maxRequests))
	return &RateLimiter{
		cache: lru.New(cacheSize),
		rate:  r,
		burst: maxRequests,
	}
}

// getLimiter 获取或创建限流器
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 从 LRU 缓存中获取
	if v, ok := rl.cache.Get(key); ok {
		return v.(*rate.Limiter)
	}

	// 不存在则创建新的限流器
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.cache.Add(key, limiter)

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
		// 视频上传限流：每20秒1个，突发3个，最多缓存 1000 个用户
		uploadLimiter = NewRateLimiter(20*time.Second, 3, 1000)

		// 评论发布限流：每6秒1个，突发10个，最多缓存 3000 个用户
		commentLimiter = NewRateLimiter(6*time.Second, 10, 3000)

		// 用户注册限流：每12分钟1个，突发5个，最多缓存 5000 个 IP
		registerLimiter = NewRateLimiter(12*time.Minute, 5, 5000)

		utils.Logger.Info("限流器初始化完成（基于 LRU 自动淘汰）",
			zap.Int("upload_cache_size", 1000),
			zap.Int("comment_cache_size", 3000),
			zap.Int("register_cache_size", 5000))
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
