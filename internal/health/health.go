package health

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// 为什么需要健康检查？
// 1. Kubernetes需要通过健康检查判断Pod是否ready
// 2. 负载均衡器需要知道哪些实例可以接收流量
// 3. 监控系统需要知道服务状态
// 4. 区分"服务启动中"和"服务已就绪"

// HealthChecker 健康检查器
type HealthChecker struct {
	serviceName string
	startTime   time.Time
	isReady     atomic.Bool // 使用atomic保证并发安全
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(serviceName string) *HealthChecker {
	return &HealthChecker{
		serviceName: serviceName,
		startTime:   time.Now(),
	}
}

// SetReady 设置服务为就绪状态
// 什么时候调用？
// - 数据库连接成功后
// - Redis连接成功后
// - 所有初始化完成后
func (h *HealthChecker) SetReady() {
	h.isReady.Store(true)
}

// SetNotReady 设置服务为未就绪状态
// 什么时候调用？
// - 数据库连接断开
// - 依赖服务不可用
// - 准备关闭服务时
func (h *HealthChecker) SetNotReady() {
	h.isReady.Store(false)
}

// RegisterRoutes 注册健康检查路由
func (h *HealthChecker) RegisterRoutes(router *gin.Engine) {
	// Liveness Probe: 服务是否存活（用于K8s重启决策）
	// 只要进程在运行就返回200
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": h.serviceName,
			"uptime":  time.Since(h.startTime).String(),
		})
	})

	// Readiness Probe: 服务是否就绪（用于流量转发决策）
	// 初始化完成后才返回200
	router.GET("/health/ready", func(c *gin.Context) {
		if h.isReady.Load() {
			c.JSON(http.StatusOK, gin.H{
				"status":  "READY",
				"service": h.serviceName,
				"uptime":  time.Since(h.startTime).String(),
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "NOT_READY",
				"service": h.serviceName,
			})
		}
	})

	// Startup Probe: 服务启动检查（用于慢启动服务）
	// 对于我们的服务，与readiness相同即可
	router.GET("/health/startup", func(c *gin.Context) {
		if h.isReady.Load() {
			c.JSON(http.StatusOK, gin.H{"status": "STARTED"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "STARTING"})
		}
	})
}
