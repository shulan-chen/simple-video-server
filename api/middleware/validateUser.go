package middleware

import (
	"strconv"
	"strings"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	HEADER_FILED_SESSION = "X-Session-Id"
	HEADER_FILED_UNAME   = "X-User-Name"
	HEADER_FILED_UID     = "X-User-Id"
)

func ValidateUserMiddleware() gin.HandlerFunc {
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
