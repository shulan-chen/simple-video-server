package web

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"video-server/api/utils"
	"video-server/internal/config"
	"video-server/web/metrics"
	"video-server/web/middleware"

	"go.uber.org/zap"
)

var httpClient *http.Client

func init() {
	// 配置HTTP客户端，设置超时时间
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// apiRequestProcess 处理API透传请求（带熔断器保护）
func apiRequestProcess(apiBody *ApiBody, w http.ResponseWriter, req *http.Request) {
	traceID := req.Header.Get("X-Trace-ID")
	start := time.Now()

	// 通过熔断器执行请求
	circuitBreaker := middleware.GetAPICircuitBreaker()
	err := circuitBreaker.Call(func() error {
		return doAPIRequest(apiBody, w, req)
	})

	// 记录 Prometheus 指标
	duration := time.Since(start)
	statusCode := 200
	if err != nil {
		if err.Error() == "circuit breaker is open" {
			statusCode = 503
			utils.Logger.Error("熔断器打开，拒绝请求",
				zap.String("trace_id", traceID),
				zap.String("url", apiBody.Url))
			writeErrorResponse(w, utils.ErrWebCircuitBreakerOpen, "API服务暂时不可用", traceID)
		} else {
			statusCode = 500
		}
	}
	metrics.RecordProxyRequest("api-service", apiBody.Method, duration, statusCode)
}

// doAPIRequest 实际执行 API 请求
func doAPIRequest(apiBody *ApiBody, w http.ResponseWriter, req *http.Request) error {
	traceID := req.Header.Get("X-Trace-ID")

	// 根据方法创建请求
	var netRequest *http.Request
	var err error

	// 构建完整的 API 地址
	apiAddr := getAPIAddr()
	fullURL := apiAddr + apiBody.Url

	switch apiBody.Method {
	case http.MethodGet:
		netRequest, err = http.NewRequestWithContext(req.Context(), "GET", fullURL, nil)
	case http.MethodPost:
		netRequest, err = http.NewRequestWithContext(req.Context(), "POST", fullURL, strings.NewReader(apiBody.ReqBody))
	case http.MethodDelete:
		netRequest, err = http.NewRequestWithContext(req.Context(), "DELETE", fullURL, nil)
	default:
		utils.Logger.Warn("不支持的API方法",
			zap.String("trace_id", traceID),
			zap.String("method", apiBody.Method))
		writeErrorResponse(w, utils.ErrWebMethodNotAllowed, "不支持的请求方法", traceID)
		return fmt.Errorf("unsupported method: %s", apiBody.Method)
	}

	if err != nil {
		utils.Logger.Error("创建代理请求失败",
			zap.String("trace_id", traceID),
			zap.String("url", fullURL),
			zap.Error(err))
		writeErrorResponse(w, utils.ErrWebProxyFailed, err.Error(), traceID)
		return err
	}

	// 复制原始请求头（传递认证信息）
	netRequest.Header = req.Header.Clone()
	netRequest.Header.Del("Content-Length")
	if traceID != "" {
		netRequest.Header.Set("X-Trace-ID", traceID)
	}

	// 发起请求
	utils.Logger.Info("发起代理请求",
		zap.String("trace_id", traceID),
		zap.String("method", apiBody.Method),
		zap.String("url", fullURL))

	resp, err := httpClient.Do(netRequest)
	if err != nil {
		utils.Logger.Error("代理请求失败",
			zap.String("trace_id", traceID),
			zap.String("url", fullURL),
			zap.Error(err))
		writeErrorResponse(w, utils.ErrWebProxyFailed, err.Error(), traceID)
		return err
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}

	// 设置响应状态码并写入响应体
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		utils.Logger.Error("写入响应失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("代理请求完成",
		zap.String("trace_id", traceID),
		zap.Int("status", resp.StatusCode))
	return nil
}

// writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, code int, detail string, traceID string) {
	appErr := utils.NewAppError(code, detail, traceID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(utils.GetHTTPStatus(code))

	// 手动构建 JSON 响应
	resp := fmt.Sprintf(`{"code":%d,"message":"%s","detail":"%s","trace_id":"%s"}`,
		appErr.Code, appErr.Message, appErr.Detail, appErr.TraceID)
	io.WriteString(w, resp)
}

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
