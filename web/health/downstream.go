package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"video-server/api/utils"
	"video-server/internal/config"

	"go.uber.org/zap"
)

// DownstreamChecker 下游服务健康检查器
type DownstreamChecker struct {
	apiAddr    string
	streamAddr string
	timeout    time.Duration
}

// NewDownstreamChecker 创建下游服务检查器
func NewDownstreamChecker() *DownstreamChecker {
	return &DownstreamChecker{
		apiAddr:    getAPIAddr(),
		streamAddr: getStreamAddr(),
		timeout:    5 * time.Second,
	}
}

// CheckAPIService 检查 API 服务健康状态
func (dc *DownstreamChecker) CheckAPIService(ctx context.Context) error {
	url := fmt.Sprintf("%s/health/ready", dc.apiAddr)
	return dc.checkService(ctx, url, "api-service")
}

// CheckStreamService 检查 Stream 服务健康状态
func (dc *DownstreamChecker) CheckStreamService(ctx context.Context) error {
	url := fmt.Sprintf("%s/health/ready", dc.streamAddr)
	return dc.checkService(ctx, url, "stream-service")
}

// CheckAllDownstream 检查所有下游服务
func (dc *DownstreamChecker) CheckAllDownstream(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// 并发检查所有下游服务
	apiChan := make(chan error, 1)
	streamChan := make(chan error, 1)

	go func() {
		apiChan <- dc.CheckAPIService(ctx)
	}()

	go func() {
		streamChan <- dc.CheckStreamService(ctx)
	}()

	// 等待结果
	results["api"] = <-apiChan
	results["stream"] = <-streamChan

	return results
}

// checkService 通用服务检查方法
func (dc *DownstreamChecker) checkService(ctx context.Context, url, serviceName string) error {
	// 创建带超时的 context
	reqCtx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		utils.Logger.Error("创建健康检查请求失败",
			zap.String("service", serviceName),
			zap.String("url", url),
			zap.Error(err))
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	client := &http.Client{
		Timeout: dc.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		utils.Logger.Error("下游服务健康检查失败",
			zap.String("service", serviceName),
			zap.String("url", url),
			zap.Error(err))
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		utils.Logger.Warn("下游服务不健康",
			zap.String("service", serviceName),
			zap.String("url", url),
			zap.Int("status", resp.StatusCode))
		return fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	utils.Logger.Debug("下游服务健康",
		zap.String("service", serviceName),
		zap.String("url", url))
	return nil
}

// getAPIAddr 获取 API 服务地址
func getAPIAddr() string {
	if config.AppConfig.APIAddr != "" {
		return config.AppConfig.APIAddr
	}
	return "http://localhost:8000"
}

// getStreamAddr 获取 Stream 服务地址
func getStreamAddr() string {
	if config.AppConfig.StreamAddr != "" {
		return config.AppConfig.StreamAddr
	}
	return "http://localhost:9090"
}

// GetHealthStatus 获取完整的健康状态（包括下游服务）
func GetHealthStatus(ctx context.Context) map[string]interface{} {
	checker := NewDownstreamChecker()
	downstreamStatus := checker.CheckAllDownstream(ctx)

	status := map[string]interface{}{
		"status": "healthy",
		"downstream": map[string]string{
			"api":    formatHealthStatus(downstreamStatus["api"]),
			"stream": formatHealthStatus(downstreamStatus["stream"]),
		},
	}

	// 如果任何下游服务不健康，整体状态为 degraded
	if downstreamStatus["api"] != nil || downstreamStatus["stream"] != nil {
		status["status"] = "degraded"
	}

	return status
}

// formatHealthStatus 格式化健康状态
func formatHealthStatus(err error) string {
	if err == nil {
		return "healthy"
	}
	return fmt.Sprintf("unhealthy: %s", err.Error())
}
