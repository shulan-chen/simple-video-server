package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ttl = time.Duration(30 * time.Minute)
var jwtSecret = []byte("your_secret_key")

type Claims struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 Token
func GenerateToken(username string, userId int) (string, error) {
	expirationTime := time.Now().Add(ttl) // 有效期 30 分钟
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
	tokenString, err := token.SignedString(jwtSecret)
	return tokenString, err
}

// ParseToken 解析 Token
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
