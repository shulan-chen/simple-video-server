package api

import (
	"net/http"
	"os"
	_ "video-server/api/docs" // 导入生成的Swagger文档
	"video-server/api/middleware"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
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

func RegisterHandlers() *gin.Engine {
	router := gin.Default()

	// 初始化限流器
	middleware.InitRateLimiters()

	// 中间件执行顺序（从上到下）
	// 注意：ErrorHandler 必须在 ValidateUserMiddleware 之前注册。
	// Gin 中间件类似嵌套调用：ErrorHandler 调用 c.Next() 进入 ValidateUser，
	// ValidateUser 调用 c.Abort() 后控制权回到 ErrorHandler，
	// ErrorHandler 才有机会读取 c.Errors 并写入 401 响应。
	// 若顺序反过来，ValidateUser 的 c.Abort() 会阻止 ErrorHandler 执行，
	// 导致 c.Errors 有内容但无响应写入，Gin 默认返回 HTTP 200 空体。
	router.Use(middleware.TraceID())                // 1. TraceID（最先执行，为每个请求生成ID）
	router.Use(corsMiddleware())                    // 2. CORS
	router.Use(middleware.ErrorHandler())           // 3. 错误处理（包裹认证，才能捕获认证错误）
	router.Use(middleware.ValidateUserMiddleware()) // 4. 认证

	// 认证相关（不需要token）
	router.POST("/user", middleware.RegisterRateLimiter(), CreateUser) // 注册限流
	router.POST("/user/:user_name", Login)
	router.POST("/auth/refresh", RefreshToken)

	// 用户相关（需要token）
	router.GET("/user/:user_name", GetUserInfo)
	router.POST("/user/:user_name/logout", Logout)

	// 视频相关
	router.POST("/user/:user_name/videos", middleware.UploadRateLimiter(), AddNewVideo) // 上传限流
	router.GET("/user/:user_name/videos", ListUserAllVideos)
	router.DELETE("/user/:user_name/videos/:vid", DeleteVideoInfo)

	// 评论相关
	router.POST("/videos/:vid/comments", middleware.CommentRateLimiter(), PostComments) // 评论限流
	router.GET("/videos/:vid/comments", ListComments)
	router.GET("/videos", ListAllVideos)

	// Swagger API文档
	utils.RegisterSwagger(router)

	return router
}

func Start() {
	r := RegisterHandlers()
	r.Run(":8000")
}
