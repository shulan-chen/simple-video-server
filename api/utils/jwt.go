package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// Access Token 有效期：15分钟
	accessTokenTTL = 15 * time.Minute
	// Refresh Token 有效期：7天
	refreshTokenTTL = 7 * 24 * time.Hour
	// JWT签名密钥（延迟初始化）
	jwtSecret []byte
)

func getJWTSecret() []byte {
	// 延迟初始化（第一次调用时才初始化）
	if jwtSecret == nil {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			// 开发环境默认值（只在Logger初始化后才记录日志）
			if Logger != nil {
				Logger.Warn("JWT_SECRET未设置，使用默认值（生产环境必须设置）")
			}
			secret = "dev-secret-key-change-in-production"
		}
		jwtSecret = []byte(secret)
	}
	return jwtSecret
}

type Claims struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成 Access Token（短期，15分钟）
func GenerateAccessToken(username string, userId int) (string, error) {
	expirationTime := time.Now().Add(accessTokenTTL)
	claims := &Claims{
		Username: username,
		UserId:   userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "video-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateRefreshToken 生成 Refresh Token（随机字符串，需存储到Redis）
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 32字节 = 64字符十六进制
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetRefreshTokenTTL 获取 Refresh Token 有效期
func GetRefreshTokenTTL() time.Duration {
	return refreshTokenTTL
}

// GetAccessTokenTTL 获取 Access Token 有效期（秒）
func GetAccessTokenTTL() int {
	return int(accessTokenTTL.Seconds())
}

// ParseToken 解析 Access Token
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GenerateToken 兼容旧代码（废弃，使用 GenerateAccessToken）
func GenerateToken(username string, userId int) (string, error) {
	return GenerateAccessToken(username, userId)
}
