package middleware

import (
	"strconv"
	"strings"
	"video-server/api/cache"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	HEADER_FILED_SESSION   = "X-Session-Id"
	HEADER_FILED_UNAME     = "X-User-Name"
	HEADER_FILED_UID       = "X-User-Id"
	HEADER_FIELD_NEW_TOKEN = "X-New-Access-Token" // 新的 accessToken
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
			"/auth/refresh", // 刷新token（保留作为手动刷新接口）
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

		// 解析JWT token（严格模式）
		claims, err := utils.ParseToken(sid)
		if err != nil {
			// Token 过期或无效，尝试自动刷新
			utils.Logger.Info("Token过期，尝试自动刷新",
				zap.String("trace_id", traceID),
				zap.String("path", c.Request.URL.Path))

			// 使用宽容模式解析（允许过期token）
			claims, err = utils.ParseTokenLenient(sid)
			if err != nil {
				utils.Logger.Warn("Token无效，无法刷新",
					zap.String("trace_id", traceID),
					zap.Error(err))
				utils.AbortWithError(c, utils.ErrAPITokenExpired, err)
				return
			}

			// 从 Redis 获取 refreshToken（验证是否存在）
			ctx := c.Request.Context()
			if _, err = cache.GetRefreshToken(ctx, claims.UserId); err != nil {
				utils.Logger.Warn("未找到refreshToken，需要重新登录",
					zap.String("trace_id", traceID),
					zap.Int("user_id", claims.UserId),
					zap.Error(err))
				utils.AbortWithError(c, utils.ErrAPITokenExpired, err)
				return
			}

			// 生成新的 accessToken
			newAccessToken, err := utils.GenerateAccessToken(claims.Username, claims.UserId)
			if err != nil {
				utils.Logger.Error("生成新accessToken失败",
					zap.String("trace_id", traceID),
					zap.Error(err))
				utils.AbortWithError(c, utils.ErrAPIInternal, err)
				return
			}

			utils.Logger.Info("Token自动刷新成功",
				zap.String("trace_id", traceID),
				zap.String("user_name", claims.Username),
				zap.Int("user_id", claims.UserId))

			// 将新 token 添加到响应头（前端会自动更新）
			c.Header(HEADER_FIELD_NEW_TOKEN, newAccessToken)
			// 继续使用刷新后的 claims
		}

		// 将用户信息存入context和header
		c.Set("user_name", claims.Username)
		c.Set("user_id", strconv.Itoa(claims.UserId))
		c.Request.Header.Add(HEADER_FILED_UNAME, claims.Username)
		c.Request.Header.Add(HEADER_FILED_UID, strconv.Itoa(claims.UserId))

		c.Next()
	}
}
