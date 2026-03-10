package jwt

import (
	"errors"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/redis"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义载荷，看你系统需要存什么
type CustomClaims struct {
	Id       uint64 `json:"id"`
	UUID     string `json:"uuid"`
	Phone    string `json:"phone,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	jwt.RegisteredClaims
}

const TokenPrefix = "auth:token:"

// GenerateToken 生成 JWT
func GenerateToken(id uint64, uuid, phone, nickname string) (string, error) {
	// 获取jwt配置实例
	cfg := configs.GetConfig()

	now := time.Now()
	claims := CustomClaims{
		Id:       id,
		UUID:     uuid,
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
	return token.SignedString([]byte(cfg.JWTConfig.Secret)) // 将字符串转换为[]byte
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
		return []byte(cfg.JWTConfig.Secret), nil // 将字符串转换为[]byte
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// CheckTokenExistsInRedis 检查 token 是否存在于 Redis 中
func CheckTokenExistsInRedis(tokenString string) (bool, error) {
	tokenKey := TokenPrefix + tokenString

	// 尝试从 Redis 获取 token
	value, err := redis.GetKey(tokenKey)
	if err != nil {
		return false, err
	}

	if value == "" {
		return false, nil
	}

	// 键存在
	return true, nil
}

// SetToken 生成 JWT 并将其存储到 Redis 中
func SetToken(id uint64, uuid, phone, nickname string) (string, error) {
	// 生成 JWT token
	tokenString, err := GenerateToken(id, uuid, phone, nickname)
	if err != nil {
		return "", fmt.Errorf("generate token failed: %v", err)
	}

	// 将 token 存储到 Redis 中
	tokenKey := TokenPrefix + tokenString
	uuidStr := uuid // UUID 字符串

	// 获取配置的过期时间
	cfg := configs.GetConfig()
	expireTime := time.Duration(cfg.JWTConfig.ExpireMinutes) * time.Minute

	err = redis.SetKeyEx(tokenKey, uuidStr, expireTime)
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
