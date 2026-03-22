package middleware

import (
	"sync"
	"time"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// StreamUploadLimiter Stream服务专用的上传限流器
type StreamUploadLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

var uploadLimiter *StreamUploadLimiter
var once sync.Once

// InitUploadRateLimiter 初始化上传限流器
func InitUploadRateLimiter() {
	once.Do(func() {
		// 每用户每分钟最多3次上传
		r := rate.Every(20 * time.Second) // 20秒1个
		uploadLimiter = &StreamUploadLimiter{
			limiters: make(map[string]*rate.Limiter),
			rate:     r,
			burst:    3, // 突发容量3
		}
	})
}

func (rl *StreamUploadLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

func (rl *StreamUploadLimiter) Allow(key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// UploadRateLimiter 上传频率限流中间件
func UploadRateLimiter() gin.HandlerFunc {
	InitUploadRateLimiter()

	return func(c *gin.Context) {
		traceID := GetTraceID(c)

		// 从请求头获取用户ID（由API服务的认证中间件设置）
		userID := c.GetHeader("X-User-Id")
		if userID == "" {
			// 如果没有用户ID，使用IP
			userID = c.ClientIP()
		}

		if !uploadLimiter.Allow(userID) {
			utils.Logger.Warn("视频上传频率超限",
				zap.String("trace_id", traceID),
				zap.String("user_id", userID),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, utils.ErrStreamRateLimitExceeded, "")
			return
		}

		c.Next()
	}
}
