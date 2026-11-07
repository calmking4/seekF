package redis

import (
	"context"
	"errors"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := configs.GetConfig()
	host := conf.RedisConfig.Host
	port := conf.RedisConfig.Port
	password := conf.RedisConfig.Password
	db := conf.Db
	addr := host + ":" + strconv.Itoa(port)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		// 让即时探测传入的截止时间约束网络读写，既有后台上下文调用保持原行为。
		ContextTimeoutEnabled: true,
	})
}

// Ping 使用现有连接检查Redis命令响应，不读写业务键。
func Ping(ctx context.Context) error {
	if redisClient == nil {
		return errors.New("Redis客户端未初始化")
	}
	return redisClient.Ping(ctx).Err()
}

// SetKeyEx 设置带过期时间的键值对
func SetKeyEx(key string, value string, timeout time.Duration) error {
	err := redisClient.Set(ctx, key, value, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

// GetKey 获取指定键的值
func GetKey(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info("该key不存在")
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// GetKeyNilIsErr 获取指定键的值，如果键不存在则返回错误
func GetKeyNilIsErr(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

// GetKeyWithPrefixNilIsErr 根据前缀查找键，如果找到多个或没找到则返回错误
func GetKeyWithPrefixNilIsErr(prefix string) (string, error) {
	var foundKeys []string
	var cursor uint64

	// 使用 SCAN 命令迭代匹配的键，避免阻塞 Redis
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return "", err
		}
		foundKeys = append(foundKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关前缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关前缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

// GetKeyWithSuffixNilIsErr 根据后缀查找键，如果找到多个或没找到则返回错误
func GetKeyWithSuffixNilIsErr(suffix string) (string, error) {
	var foundKeys []string
	var cursor uint64

	// 使用 SCAN 命令迭代匹配的键，避免阻塞 Redis
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return "", err
		}
		foundKeys = append(foundKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关后缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关后缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

// DelKeyIfExists 删除存在的键，如果键不存在则不做任何操作
func DelKeyIfExists(key string) error {
	exists, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 { // 键存在
		delErr := redisClient.Del(ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	// 无论键是否存在，都不返回错误
	return nil
}

// DelKeysWithPattern 删除匹配指定模式的所有键（使用 SCAN 避免阻塞 Redis）
func DelKeysWithPattern(pattern string) error {
	var cursor uint64

	for {
		// 使用 SCAN 命令迭代匹配的键
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if _, err := redisClient.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DelKeysWithPrefix 删除所有具有指定前缀的键（使用 SCAN 避免阻塞 Redis）
func DelKeysWithPrefix(prefix string) error {
	var cursor uint64

	for {
		// 使用 SCAN 命令迭代匹配的键
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if _, err := redisClient.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DelKeysWithSuffix 删除所有具有指定后缀的键（使用 SCAN 避免阻塞 Redis）
func DelKeysWithSuffix(suffix string) error {
	var cursor uint64

	for {
		// 使用 SCAN 命令迭代匹配的键
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if _, err := redisClient.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DeleteAllRedisKeys 删除 Redis 数据库中的所有键
func DeleteAllRedisKeys() error {
	var cursor uint64 = 0
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*", 0).Result()
		if err != nil {
			return err
		}
		cursor = nextCursor

		if len(keys) > 0 {
			_, err := redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

// EvalRateLimit 原子执行限流脚本，返回需等待的毫秒数，零表示允许。
func EvalRateLimit(ctx context.Context, script string, keys []string, args ...any) (int64, error) {
	return redisClient.Eval(ctx, script, keys, args...).Int64()
}

// GetClient 返回共用Redis连接，供业务模块执行公共工具未封装的存储操作。
func GetClient() *redis.Client {
	return redisClient
}

// CompareAndDeleteString 原子比较字符串值并删除，仅匹配的调用方可以消费一次性记录。
func CompareAndDeleteString(ctx context.Context, key, expected string) (bool, error) {
	const script = `
	if redis.call('GET', KEYS[1]) == ARGV[1] then
		return redis.call('DEL', KEYS[1])
	end
	return 0`
	deleted, err := redisClient.Eval(ctx, script, []string{key}, expected).Int64()
	return deleted == 1, err
}
