package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"video-server/api/middleware"
	"video-server/api/utils"
	_ "video-server/api/docs" // 导入生成的Swagger文档

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// @title           Video Server API
// @version         1.0
// @description     视频服务器API文档 - 微服务架构
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@videoserver.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8000
// @BasePath  /

// @securityDefinitions.apikey  Bearer
// @in                          header
// @name                        X-Session-Id
// @description                 JWT Access Token

// @schemes http https

// corsMiddleware 处理跨域请求
func corsMiddleware() gin.HandlerFunc {
	// 从环境变量读取允许的源，默认为本地开发环境
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8080" // 默认允许Web服务
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == allowedOrigin {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Session-Id")
		}

		// 处理OPTIONS预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func validateUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetString("trace_id")

		// 跳过认证的路径
		skipPaths := []string{
			"/health/live",
			"/health/ready",
			"/health/startup",
			"/user",         // 注册
			"/auth/refresh", // 刷新token
		}

		// 跳过认证的路径前缀
		skipPrefixes := []string{
			"/swagger/", // Swagger文档
		}

		// 检查是否是需要跳过的路径
		for _, path := range skipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// 检查路径前缀
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				c.Next()
				return
			}
		}

		// 检查是否是登录路径（/user/:username）
		if c.Request.Method == "POST" {
			path := c.Request.URL.Path
			if len(path) > 6 && path[:6] == "/user/" {
				rest := path[6:]
				if !strings.Contains(rest, "/") {
					c.Next()
					return
				}
			}
		}

		// 其他路径需要认证：检查JWT token
		sid := c.GetHeader(HEADER_FILED_SESSION)
		if sid == "" {
			utils.Logger.Warn("缺少认证token",
				zap.String("trace_id", traceID),
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()))
			utils.AbortWithErrorMsg(c, utils.ErrAPIUnauthorized, "")
			return
		}

		// 解析JWT token
		claims, err := utils.ParseToken(sid)
		if err != nil {
			utils.Logger.Warn("Token无效或已过期",
				zap.String("trace_id", traceID),
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
				zap.Error(err))
			utils.AbortWithError(c, utils.ErrAPITokenExpired, err)
			return
		}

		// 将用户信息存入context和header
		c.Set("user_name", claims.Username)
		c.Set("user_id", strconv.Itoa(claims.UserId))
		c.Request.Header.Add(HEADER_FILED_UNAME, claims.Username)
		c.Request.Header.Add(HEADER_FILED_UID, strconv.Itoa(claims.UserId))

		c.Next()
	}
}

func RegisterHandlers() *gin.Engine {
	router := gin.Default()

	// 初始化限流器
	middleware.InitRateLimiters()

	// 中间件执行顺序（从上到下）
	router.Use(middleware.TraceID())         // 1. TraceID（最先执行，为每个请求生成ID）
	router.Use(corsMiddleware())             // 2. CORS
	router.Use(validateUserMiddleware())     // 3. 认证
	router.Use(middleware.ErrorHandler())    // 4. 错误处理（最后执行，捕获所有错误）

	// 认证相关（不需要token）
	router.POST("/user", middleware.RegisterRateLimiter(), CreateUser)  // 注册限流
	router.POST("/user/:user_name", Login)
	router.POST("/auth/refresh", RefreshToken)

	// 用户相关（需要token）
	router.GET("/user/:user_name", GetUserInfo)
	router.POST("/user/:user_name/logout", Logout)

	// 视频相关
	router.POST("/user/:user_name/videos", middleware.UploadRateLimiter(), AddNewVideo)  // 上传限流
	router.GET("/user/:user_name/videos", ListUserAllVideos)
	router.DELETE("/user/:user_name/videos/:vid", DeleteVideoInfo)

	// 评论相关
	router.POST("/videos/:vid/comments", middleware.CommentRateLimiter(), PostComments)  // 评论限流
	router.GET("/videos/:vid/comments", ListComments)
	router.GET("/videos", ListAllVideos)

	// Swagger API文档
	RegisterSwagger(router)

	return router
}

func Start() {
	r := RegisterHandlers()
	r.Run(":8000")
}
