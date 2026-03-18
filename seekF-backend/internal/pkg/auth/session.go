package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/redis"
	"time"
)

const SessionPrefix = "auth:session:"

type Session struct {
	Id       uint64 `json:"id"`
	UUID     string `json:"uuid"`
	Phone    string `json:"phone,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

func sessionTTL() time.Duration {
	cfg := configs.GetConfig()
	mins := cfg.AuthConfig.SessionExpireMinutes
	return time.Duration(mins) * time.Minute
}

func GenerateToken() (string, error) {
	// 32 bytes -> 43 chars base64url (no padding)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetSession(token string, sess Session) error {
	payload, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return redis.SetKeyEx(SessionPrefix+token, string(payload), sessionTTL())
}

func GetSession(token string) (*Session, error) {
	raw, err := redis.GetKey(SessionPrefix + token)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var sess Session
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		return nil, fmt.Errorf("invalid session payload: %w", err)
	}
	return &sess, nil
}

func DelSession(token string) error {
	return redis.DelKeyIfExists(SessionPrefix + token)
}
