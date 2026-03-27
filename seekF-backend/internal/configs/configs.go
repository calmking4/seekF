package configs

import (
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type MainConfig struct {
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type MysqlConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	Password     string `toml:"password"`
	DatabaseName string `toml:"databaseName"`
}

type RedisConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Password string `toml:"password"`
	Db       int    `toml:"db"`
}

type AuthCodeConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	SignName        string `toml:"signName"`
	TemplateCode    string `toml:"templateCode"`
}

type OSSConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	Region          string `toml:"region"`
	Bucket          string `toml:"bucket"`
	BaseURL         string `toml:"baseURL"`
}

type LogConfig struct {
	LogPath string `toml:"logPath"`
}

type KafkaConfig struct {
	MessageMode string        `toml:"messageMode"`
	HostPort    string        `toml:"hostPort"`
	LoginTopic  string        `toml:"loginTopic"`
	LogoutTopic string        `toml:"logoutTopic"`
	ChatTopic   string        `toml:"chatTopic"`
	Partition   int           `toml:"partition"`
	Timeout     time.Duration `toml:"timeout"`
}

type StaticSrcConfig struct {
	StaticAvatarPath string `toml:"staticAvatarPath"`
	StaticFilePath   string `toml:"staticFilePath"`
}

type JWTConfig struct {
	Secret        string `toml:"secret"`
	ExpireMinutes int64  `toml:"expireMinutes"`
	Issuer        string `toml:"issuer"`
}

// AuthConfig 认证方案配置
// mode: "redis"（默认，不透明token+redis会话）或 "jwt"
type AuthConfig struct {
	Mode                 string `toml:"mode"`
	SessionExpireMinutes int64  `toml:"sessionExpireMinutes"`
}

type Config struct {
	MainConfig      `toml:"mainConfig"`
	MysqlConfig     `toml:"mysqlConfig"`
	RedisConfig     `toml:"redisConfig"`
	AuthCodeConfig  `toml:"authCodeConfig"`
	OSSConfig       `toml:"ossConfig"`
	LogConfig       `toml:"logConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	JWTConfig       `toml:"jwtConfig"`
	AuthConfig      `toml:"authConfig"`
}

var config *Config

// loadEnvConfig 从环境变量加载敏感配置
func loadEnvConfig(cfg *Config) {
	// MySQL Password
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		cfg.MysqlConfig.Password = v
	}

	// Redis Password
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.RedisConfig.Password = v
	}

	// JWT Secret
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTConfig.Secret = v
	}

	// SMS Access Key
	if v := os.Getenv("SMS_ACCESS_KEY_ID"); v != "" {
		cfg.AuthCodeConfig.AccessKeyID = v
	}
	if v := os.Getenv("SMS_ACCESS_KEY_SECRET"); v != "" {
		cfg.AuthCodeConfig.AccessKeySecret = v
	}

	// OSS Access Key
	if v := os.Getenv("OSS_ACCESS_KEY_ID"); v != "" {
		cfg.OSSConfig.AccessKeyID = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY_SECRET"); v != "" {
		cfg.OSSConfig.AccessKeySecret = v
	}
}

func LoadConfig() error {
	config = new(Config)

	if _, err := toml.DecodeFile("./config/config.toml", config); err != nil {
		log.Fatal(err.Error())
		return err
	}

	// 从环境变量加载敏感配置（覆盖配置文件中的值）
	loadEnvConfig(config)

	return nil
}

func GetConfig() *Config {
	if config == nil {
		_ = LoadConfig()
	}
	return config
}
