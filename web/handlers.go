package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"video-server/api/utils"
	"video-server/web/metrics"
	"video-server/web/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HomePage 首页数据
type HomePage struct {
	Name string
}

// UserHomePage 用户主页数据
type UserHomePage struct {
	Name string
}

// homeHandler 处理首页请求
func homeHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	cname, err1 := c.Cookie("username")
	sid, err2 := c.Cookie("sessionid")

	// 如果已登录，重定向到用户主页
	if err1 == nil && err2 == nil && cname != "" && sid != "" {
		utils.Logger.Debug("用户已登录，重定向到主页",
			zap.String("trace_id", traceID),
			zap.String("username", cname))
		c.Redirect(http.StatusFound, "/userhome")
		return
	}

	utils.Logger.Debug("渲染首页",
		zap.String("trace_id", traceID))
	c.HTML(http.StatusOK, "home.html", HomePage{Name: "unknown"})
}

// userHomeHandler 处理用户主页请求
func userHomeHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	cname, err1 := c.Cookie("username")
	sid, err2 := c.Cookie("sessionid")

	// 检查是否已登录
	if err1 != nil || err2 != nil || cname == "" || sid == "" {
		utils.Logger.Debug("用户未登录，重定向到首页",
			zap.String("trace_id", traceID))
		c.Redirect(http.StatusFound, "/")
		return
	}

	// 优先使用 Cookie 中的用户名
	var p *UserHomePage
	formUsername := c.PostForm("username")
	if cname != "" {
		p = &UserHomePage{Name: cname}
	} else if formUsername != "" {
		p = &UserHomePage{Name: formUsername}
	}

	utils.Logger.Debug("渲染用户主页",
		zap.String("trace_id", traceID),
		zap.String("username", p.Name))
	c.HTML(http.StatusOK, "userhome.html", p)
}

// apiHandler 处理API透传请求
func apiHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	if c.Request.Method != http.MethodPost {
		utils.Logger.Warn("不支持的请求方法",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method))
		utils.AbortWithError(c, utils.ErrWebMethodNotAllowed, nil)
		return
	}

	apiBody := &ApiBody{}
	if err := c.BindJSON(apiBody); err != nil {
		utils.Logger.Error("解析API请求体失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrWebRequestBodyInvalid, err)
		return
	}

	utils.Logger.Debug("处理API透传请求",
		zap.String("trace_id", traceID),
		zap.String("url", apiBody.Url),
		zap.String("method", apiBody.Method))

	apiRequestProcess(apiBody, c.Writer, c.Request)
}

// proxyUploadHandler 代理视频上传请求（带熔断器保护）
func proxyUploadHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	vid := c.Param("vid-id")
	start := time.Now()

	utils.Logger.Info("代理上传请求",
		zap.String("trace_id", traceID),
		zap.String("vid", vid))

	// 通过熔断器执行代理
	circuitBreaker := middleware.GetStreamCircuitBreaker()
	err := circuitBreaker.Call(func() error {
		return proxyToStream(c, traceID)
	})

	// 记录 Prometheus 指标
	duration := time.Since(start)
	statusCode := c.Writer.Status()
	if err != nil {
		if err.Error() == "circuit breaker is open" {
			statusCode = 503
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, err)
		}
	}
	metrics.RecordProxyRequest("stream-service", "POST", duration, statusCode)
}

// proxyVideoViewHandler 代理视频查看请求（带熔断器保护）
func proxyVideoViewHandler(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	vid := c.Param("vid-id")
	start := time.Now()

	utils.Logger.Info("代理视频查看请求",
		zap.String("trace_id", traceID),
		zap.String("vid", vid))

	// 通过熔断器执行代理
	circuitBreaker := middleware.GetStreamCircuitBreaker()
	err := circuitBreaker.Call(func() error {
		return proxyToStream(c, traceID)
	})

	// 记录 Prometheus 指标
	duration := time.Since(start)
	statusCode := c.Writer.Status()
	if err != nil {
		if err.Error() == "circuit breaker is open" {
			statusCode = 503
			utils.AbortWithError(c, utils.ErrWebCircuitBreakerOpen, err)
		}
	}
	metrics.RecordProxyRequest("stream-service", "GET", duration, statusCode)
}

// proxyToStream 代理请求到 Stream 服务
func proxyToStream(c *gin.Context, traceID string) error {
	streamAddr := getStreamAddr()
	u, err := url.Parse(streamAddr)
	if err != nil {
		utils.Logger.Error("解析代理URL失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}
