package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

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
		// 跳过认证的路径
		skipPaths := []string{
			"/health/live",
			"/health/ready",
			"/health/startup",
			"/user",            // 注册
			"/auth/refresh",    // 刷新token
		}

		// 检查是否是需要跳过的路径
		for _, path := range skipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// 检查是否是登录路径（/user/:username）
		if c.Request.Method == "POST" && len(c.Request.URL.Path) > 6 && c.Request.URL.Path[:6] == "/user/" {
			c.Next()
			return
		}

		// 其他路径需要认证
		if !validateUserSession(c.Writer, c.Request) {
			c.Abort()
			return
		}
		c.Next()
	}
}

func RegisterHandlers() *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())         // CORS中间件（第一个执行）
	router.Use(validateUserMiddleware()) // 认证中间件

	// 认证相关（不需要token）
	router.POST("/user", CreateUser)
	router.POST("/user/:user_name", Login)
	router.POST("/auth/refresh", RefreshToken) // 刷新token（不需要access token）

	// 用户相关（需要token）
	router.GET("/user/:user_name", GetUserInfo)
	router.POST("/user/:user_name/logout", Logout)

	router.POST("/user/:user_name/videos", AddNewVideo)
	router.GET("/user/:user_name/videos", ListUserAllVideos)
	router.DELETE("/user/:user_name/videos/:vid", DeleteVideoInfo)

	router.POST("/videos/:vid/comments", PostComments)
	router.GET("/videos/:vid/comments", ListComments)
	router.GET("/videos", ListAllVideos)
	return router
}

func Start() {
	r := RegisterHandlers()
	r.Run(":8000")
}
