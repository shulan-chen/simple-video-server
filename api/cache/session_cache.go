package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	api "video-server/api/defs"

	"github.com/redis/go-redis/v9"
)

// 缓存键前缀（Session相关）
const (
	prefixSession      = "session:"      // 单个session: session:<sid>
	prefixSessionList  = "session:list:" // session列表: session:list:<username>
	prefixRefreshToken = "refresh:"      // Refresh Token: refresh:<userId>
)

// Session缓存过期时间（与JWT token过期时间一致）
const (
	sessionTTL      = 15 * time.Minute      // Session：15分钟（与access token一致）
	refreshTokenTTL = 7 * 24 * time.Hour    // Refresh Token：7天
)

// ============= Session 缓存操作 =============

// AddSession 添加Session到Redis
func AddSession(ctx context.Context, sid string, session *api.SimpleSession) error {
	key := prefixSession + sid
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, sessionTTL).Err()
}

// GetSession 从Redis获取Session
func GetSession(ctx context.Context, sid string) (*api.SimpleSession, error) {
	key := prefixSession + sid
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	session := &api.SimpleSession{}
	if err := json.Unmarshal(data, session); err != nil {
		return nil, err
	}
	return session, nil
}

// DeleteSession 从Redis删除Session
func DeleteSession(ctx context.Context, sid string) error {
	key := prefixSession + sid
	return redisClient.Del(ctx, key).Err()
}

// UpdateSessionList 更新用户的Session列表
func UpdateSessionList(ctx context.Context, username string, sessionIds []string) error {
	key := prefixSessionList + username
	data, err := json.Marshal(sessionIds)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, sessionTTL).Err()
}

// LoadSessionList 加载用户的Session列表
func LoadSessionList(ctx context.Context, username string) ([]api.SimpleSession, error) {
	key := prefixSessionList + username
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return []api.SimpleSession{}, nil
		}
		return nil, err
	}

	var sessionIds []string
	if err := json.Unmarshal([]byte(data), &sessionIds); err != nil {
		return nil, err
	}

	sessions := make([]api.SimpleSession, 0, len(sessionIds))
	for _, sid := range sessionIds {
		session, err := GetSession(ctx, sid)
		if err != nil {
			continue // 跳过已过期或损坏的session
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// ============= Refresh Token 操作 =============

// SaveRefreshToken 保存 Refresh Token 到 Redis
// Key: refresh:<userId>, Value: refreshToken, TTL: 7天
// 设计说明：key 是 userId 而不是 token，这样后端可以直接通过 userId 查找 token
func SaveRefreshToken(ctx context.Context, userId int, refreshToken string) error {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Set(ctx, key, refreshToken, refreshTokenTTL).Err()
}

// GetRefreshToken 根据 userId 获取 Refresh Token
func GetRefreshToken(ctx context.Context, userId int) (string, error) {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Get(ctx, key).Result()
}

// ValidateRefreshToken 验证 Refresh Token 并返回 user_id
// 通过遍历所有 refresh token 找到匹配的（兼容旧接口）
// 注意：这个函数主要用于兼容前端传 refreshToken 的场景
func ValidateRefreshToken(ctx context.Context, refreshToken string) (int, error) {
	// 方案：遍历所有 refresh:* key 查找匹配的 token
	pattern := prefixRefreshToken + "*"
	iter := redisClient.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		storedToken, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		if storedToken == refreshToken {
			// 找到匹配的 token，从 key 中提取 userId
			// key 格式: refresh:<userId>
			userIdStr := key[len(prefixRefreshToken):]
			userId, err := strconv.Atoi(userIdStr)
			if err != nil {
				return 0, fmt.Errorf("解析用户ID失败")
			}
			return userId, nil
		}
	}

	if err := iter.Err(); err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("refresh token无效或已过期")
}

// DeleteRefreshToken 删除 Refresh Token（根据 userId）
func DeleteRefreshToken(ctx context.Context, userId int) error {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Del(ctx, key).Err()
}

// RotateRefreshToken 轮换 Refresh Token（更新同一用户的 token）
func RotateRefreshToken(ctx context.Context, userId int, newToken string) error {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Set(ctx, key, newToken, refreshTokenTTL).Err()
}

// DeleteUserAllRefreshTokens 删除用户的 Refresh Token（用于强制登出）
func DeleteUserAllRefreshTokens(ctx context.Context, userId int) error {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Del(ctx, key).Err()
}

// RefreshTokenExists 检查 Refresh Token 是否存在
func RefreshTokenExists(ctx context.Context, userId int) bool {
	key := prefixRefreshToken + strconv.Itoa(userId)
	result, err := redisClient.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// ExtendRefreshToken 延长 Refresh Token 有效期（滑动过期）
func ExtendRefreshToken(ctx context.Context, userId int) error {
	key := prefixRefreshToken + strconv.Itoa(userId)
	return redisClient.Expire(ctx, key, refreshTokenTTL).Err()
}

// GetRefreshTokenLastUsed 获取 Refresh Token 最后使用时间（通过TTL反推）
func GetRefreshTokenLastUsed(ctx context.Context, userId int) (time.Time, error) {
	key := prefixRefreshToken + strconv.Itoa(userId)
	ttl, err := redisClient.TTL(ctx, key).Result()
	if err != nil {
		return time.Time{}, err
	}

	lastUsed := time.Now().Add(ttl - refreshTokenTTL)
	return lastUsed, nil
}

// GetRefreshTokenTTL 获取 Refresh Token 的TTL（供外部使用）
func GetRefreshTokenTTL() time.Duration {
	return refreshTokenTTL
}
