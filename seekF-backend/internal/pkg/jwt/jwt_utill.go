package jwt

import (
	"errors"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/redis"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义载荷，看你系统需要存什么
type CustomClaims struct {
	UserID   uint64 `json:"user_id"`
	Phone    string `json:"phone,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	jwt.RegisteredClaims
}

const TokenPrefix = "auth:token:"

// GenerateToken 生成 JWT
func GenerateToken(userID uint64, phone, nickname string) (string, error) {
	// 获取jwt配置实例
	cfg := configs.GetConfig()

	now := time.Now()
	claims := CustomClaims{
		UserID:   userID,
		Phone:    phone,
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWTConfig.ExpireMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    cfg.JWTConfig.Issuer,
			Subject:   "user-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.JWTConfig.Secret)
}

// ParseToken 解析并校验 Token
func ParseToken(tokenString string) (*CustomClaims, error) {
	// 获取jwt配置实例
	cfg := configs.GetConfig()

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return cfg.JWTConfig.Secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// SetToken 生成 JWT 并将其存储到 Redis 中
func SetToken(userID uint64, phone, nickname string) (string, error) {
	// 生成 JWT token
	tokenString, err := GenerateToken(userID, phone, nickname)
	if err != nil {
		return "", fmt.Errorf("generate token failed: %v", err)
	}

	// 将 token 存储到 Redis 中
	tokenKey := TokenPrefix + tokenString
	userIDStr := strconv.FormatUint(userID, 10) // 将 uint64 转换为10进制字符串

	// 获取配置的过期时间
	cfg := configs.GetConfig()
	expireTime := time.Duration(cfg.JWTConfig.ExpireMinutes) * time.Minute

	err = redis.SetKeyEx(tokenKey, userIDStr, expireTime)
	if err != nil {
		return "", fmt.Errorf("store token to redis failed: %v", err)
	}

	return tokenString, nil
}

// DelToken 从 Redis 中删除指定的 token
func DelToken(tokenString string) error {
	tokenKey := TokenPrefix + tokenString
	err := redis.DelKeyIfExists(tokenKey)
	if err != nil {
		return fmt.Errorf("delete token from redis failed: %v", err)
	}
	return nil
}
