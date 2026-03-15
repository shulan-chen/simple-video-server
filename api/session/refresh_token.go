package session

import (
	"fmt"
	"strconv"
	"time"
	"video-server/api/utils"
)

// SaveRefreshToken 保存 Refresh Token 到 Redis
// Key: refresh:<token>, Value: user_id, TTL: 7天
func SaveRefreshToken(refreshToken string, userId int) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	ttl := utils.GetRefreshTokenTTL()
	return rconn.Set(ctx, key, userId, ttl).Err()
}

// ValidateRefreshToken 验证 Refresh Token 并返回 user_id
func ValidateRefreshToken(refreshToken string) (int, error) {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	userIdStr, err := rconn.Get(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("refresh token无效或已过期")
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, fmt.Errorf("refresh token数据损坏")
	}

	return userId, nil
}

// DeleteRefreshToken 删除 Refresh Token（用于登出）
func DeleteRefreshToken(refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	return rconn.Del(ctx, key).Err()
}

// DeleteUserAllRefreshTokens 删除用户的所有 Refresh Token（用于强制登出所有设备）
func DeleteUserAllRefreshTokens(userId int) error {
	// 使用 SCAN 命令查找所有 refresh:* 的 key
	pattern := "refresh:*"
	iter := rconn.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// 检查这个 token 是否属于该用户
		storedUserId, err := rconn.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		if storedUserId == strconv.Itoa(userId) {
			rconn.Del(ctx, key)
		}
	}

	return iter.Err()
}

// RefreshTokenExists 检查 Refresh Token 是否存在
func RefreshTokenExists(refreshToken string) bool {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	result, err := rconn.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// ExtendRefreshToken 延长 Refresh Token 有效期（滑动过期）
func ExtendRefreshToken(refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	ttl := utils.GetRefreshTokenTTL()
	return rconn.Expire(ctx, key, ttl).Err()
}

// RotateRefreshToken 轮换 Refresh Token（删除旧的，保存新的）
func RotateRefreshToken(oldToken, newToken string, userId int) error {
	// 删除旧token
	if err := DeleteRefreshToken(oldToken); err != nil {
		return err
	}

	// 保存新token
	return SaveRefreshToken(newToken, userId)
}

// GetRefreshTokenLastUsed 获取 Refresh Token 最后使用时间（通过TTL反推）
func GetRefreshTokenLastUsed(refreshToken string) (time.Time, error) {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	ttl, err := rconn.TTL(ctx, key).Result()
	if err != nil {
		return time.Time{}, err
	}

	maxTTL := utils.GetRefreshTokenTTL()
	lastUsed := time.Now().Add(ttl - maxTTL)
	return lastUsed, nil
}
