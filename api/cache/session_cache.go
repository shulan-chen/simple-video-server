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
	prefixSession      = "session:"        // 单个session: session:<sid>
	prefixSessionList  = "session:list:"   // session列表: session:list:<username>
	prefixRefreshToken = "session:refresh:" // Refresh Token: session:refresh:<token>
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
// Key: session:refresh:<token>, Value: user_id, TTL: 7天
func SaveRefreshToken(ctx context.Context, refreshToken string, userId int) error {
	key := prefixRefreshToken + refreshToken
	return redisClient.Set(ctx, key, userId, refreshTokenTTL).Err()
}

// ValidateRefreshToken 验证 Refresh Token 并返回 user_id
func ValidateRefreshToken(ctx context.Context, refreshToken string) (int, error) {
	key := prefixRefreshToken + refreshToken
	userIdStr, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("refresh token无效或已过期")
		}
		return 0, err
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, fmt.Errorf("refresh token数据损坏")
	}

	return userId, nil
}

// DeleteRefreshToken 删除 Refresh Token（用于登出）
func DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	key := prefixRefreshToken + refreshToken
	return redisClient.Del(ctx, key).Err()
}

// RotateRefreshToken 轮换 Refresh Token（删除旧的，保存新的）
func RotateRefreshToken(ctx context.Context, oldToken, newToken string, userId int) error {
	// 使用pipeline确保原子性
	pipe := redisClient.Pipeline()

	// 删除旧token
	oldKey := prefixRefreshToken + oldToken
	pipe.Del(ctx, oldKey)

	// 保存新token
	newKey := prefixRefreshToken + newToken
	pipe.Set(ctx, newKey, userId, refreshTokenTTL)

	// 执行pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteUserAllRefreshTokens 删除用户的所有 Refresh Token（用于强制登出所有设备）
func DeleteUserAllRefreshTokens(ctx context.Context, userId int) error {
	pattern := prefixRefreshToken + "*"
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	userIdStr := strconv.Itoa(userId)
	keysToDelete := make([]string, 0)

	// 收集需要删除的key
	for iter.Next(ctx) {
		key := iter.Val()
		storedUserId, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		if storedUserId == userIdStr {
			keysToDelete = append(keysToDelete, key)
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	// 批量删除
	if len(keysToDelete) > 0 {
		return redisClient.Del(ctx, keysToDelete...).Err()
	}

	return nil
}

// RefreshTokenExists 检查 Refresh Token 是否存在
func RefreshTokenExists(ctx context.Context, refreshToken string) bool {
	key := prefixRefreshToken + refreshToken
	result, err := redisClient.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// ExtendRefreshToken 延长 Refresh Token 有效期（滑动过期）
func ExtendRefreshToken(ctx context.Context, refreshToken string) error {
	key := prefixRefreshToken + refreshToken
	return redisClient.Expire(ctx, key, refreshTokenTTL).Err()
}

// GetRefreshTokenLastUsed 获取 Refresh Token 最后使用时间（通过TTL反推）
func GetRefreshTokenLastUsed(ctx context.Context, refreshToken string) (time.Time, error) {
	key := prefixRefreshToken + refreshToken
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
