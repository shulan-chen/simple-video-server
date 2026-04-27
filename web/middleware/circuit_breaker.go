package middleware

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"video-server/api/utils"
	"video-server/web/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	// StateClosed 关闭状态：正常请求
	StateClosed CircuitState = iota
	// StateOpen 打开状态：快速失败，不发送请求
	StateOpen
	// StateHalfOpen 半开状态：允许部分请求尝试，检测服务是否恢复
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name         string        // 熔断器名称（如 "api-service", "stream-service"）
	state        CircuitState  // 当前状态
	failureCount int           // 连续失败次数
	successCount int           // 半开状态下的成功次数
	lastFailTime time.Time     // 最后一次失败时间
	mu           sync.RWMutex  // 读写锁

	// 配置参数
	maxFailures     int           // 触发熔断的最大失败次数
	timeout         time.Duration // 熔断超时时间（打开状态持续时间）
	halfOpenSuccess int           // 半开状态下需要的成功次数（才能关闭熔断）
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, maxFailures int, timeout time.Duration, halfOpenSuccess int) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		state:           StateClosed,
		maxFailures:     maxFailures,
		timeout:         timeout,
		halfOpenSuccess: halfOpenSuccess,
	}
}

// Call 执行请求（通过熔断器）
func (cb *CircuitBreaker) Call(fn func() error) error {
	// 1. 检查是否允许请求
	if !cb.AllowRequest() {
		return errors.New("circuit breaker is open")
	}

	// 2. 执行请求
	err := fn()

	// 3. 记录结果
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest 检查是否允许请求
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	state := cb.state
	lastFailTime := cb.lastFailTime
	cb.mu.RUnlock()

	switch state {
	case StateClosed:
		// 关闭状态：允许所有请求
		return true

	case StateOpen:
		// 打开状态：检查是否超时，如果超时则进入半开状态
		if time.Since(lastFailTime) > cb.timeout {
			cb.mu.Lock()
			if cb.state == StateOpen { // 双重检查
				utils.Logger.Info("熔断器进入半开状态",
					zap.String("circuit", cb.name))
				cb.state = StateHalfOpen
				cb.successCount = 0
				metrics.UpdateCircuitBreakerState(cb.name, int(StateHalfOpen))
			}
			cb.mu.Unlock()
			return true
		}
		return false

	case StateHalfOpen:
		// 半开状态：允许部分请求通过（用于探测服务是否恢复）
		return true

	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successCount++
		utils.Logger.Info("熔断器半开状态下请求成功",
			zap.String("circuit", cb.name),
			zap.Int("success_count", cb.successCount),
			zap.Int("required", cb.halfOpenSuccess))

		// 如果连续成功次数达到阈值，关闭熔断器
		if cb.successCount >= cb.halfOpenSuccess {
			utils.Logger.Info("熔断器关闭（服务恢复）",
				zap.String("circuit", cb.name))
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			metrics.UpdateCircuitBreakerState(cb.name, int(StateClosed))
		}
	} else if cb.state == StateClosed {
		// 关闭状态下成功，重置失败计数
		cb.failureCount = 0
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	utils.Logger.Warn("熔断器记录失败",
		zap.String("circuit", cb.name),
		zap.Int("failure_count", cb.failureCount),
		zap.Int("max_failures", cb.maxFailures),
		zap.String("state", cb.state.String()))

	// 如果失败次数达到阈值，打开熔断器
	metrics.RecordCircuitBreakerFailure(cb.name)
	if cb.state == StateClosed && cb.failureCount >= cb.maxFailures {
		utils.Logger.Error("熔断器打开（服务故障）",
			zap.String("circuit", cb.name),
			zap.Int("failure_count", cb.failureCount),
			zap.Duration("timeout", cb.timeout))
		cb.state = StateOpen
		metrics.UpdateCircuitBreakerState(cb.name, int(StateOpen))
	} else if cb.state == StateHalfOpen {
		// 半开状态下失败，立即打开熔断器
		utils.Logger.Error("熔断器重新打开（服务未恢复）",
			zap.String("circuit", cb.name))
		cb.state = StateOpen
		cb.successCount = 0
		metrics.UpdateCircuitBreakerState(cb.name, int(StateOpen))
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics 获取熔断器指标
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":          cb.name,
		"state":         cb.state.String(),
		"failure_count": cb.failureCount,
		"success_count": cb.successCount,
		"last_fail":     cb.lastFailTime,
	}
}

// 全局熔断器实例
var (
	// API 服务熔断器
	apiCircuitBreaker *CircuitBreaker

	// Stream 服务熔断器
	streamCircuitBreaker *CircuitBreaker

	circuitBreakerOnce sync.Once
)

// InitCircuitBreakers 初始化熔断器（在应用启动时调用一次）
func InitCircuitBreakers() {
	circuitBreakerOnce.Do(func() {
		// API 服务熔断器：5次失败触发，30秒超时，3次成功关闭
		apiCircuitBreaker = NewCircuitBreaker("api-service", 5, 30*time.Second, 3)

		// Stream 服务熔断器：5次失败触发，30秒超时，3次成功关闭
		streamCircuitBreaker = NewCircuitBreaker("stream-service", 5, 30*time.Second, 3)

		utils.Logger.Info("熔断器初始化完成",
			zap.Int("max_failures", 5),
			zap.Duration("timeout", 30*time.Second),
			zap.Int("half_open_success", 3))
	})
}

// GetAPICircuitBreaker 获取 API 服务熔断器
func GetAPICircuitBreaker() *CircuitBreaker {
	return apiCircuitBreaker
}

// GetStreamCircuitBreaker 获取 Stream 服务熔断器
func GetStreamCircuitBreaker() *CircuitBreaker {
	return streamCircuitBreaker
}

// CircuitBreakerMiddleware 熔断器中间件（全局检查）
func CircuitBreakerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := GetTraceID(c)
		path := c.Request.URL.Path

		// 检查 API 服务熔断器状态（仅针对 API 透传路由）
		if path == "/api" && apiCircuitBreaker.GetState() == StateOpen {
			utils.Logger.Error("API服务熔断器打开，拒绝请求",
				zap.String("trace_id", traceID),
				zap.String("path", path))
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, errors.New("API service unavailable"))
			return
		}

		// 检查 Stream 服务熔断器状态（仅针对视频路由）
		if (path == fmt.Sprintf("/videos/%s", c.Param("vid-id")) || path == fmt.Sprintf("/videos/upload/%s", c.Param("vid-id"))) &&
			streamCircuitBreaker.GetState() == StateOpen {
			utils.Logger.Error("Stream服务熔断器打开，拒绝请求",
				zap.String("trace_id", traceID),
				zap.String("path", path))
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, errors.New("Stream service unavailable"))
			return
		}

		c.Next()
	}
}
