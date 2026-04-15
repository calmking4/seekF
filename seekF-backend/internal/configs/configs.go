package configs

import (
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
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
	ChatTopic   string        `toml:"chatTopic"`
	AIChatTopic string        `toml:"aiChatTopic"`
	LogoutTopic string        `toml:"logoutTopic"`
	Partition   int           `toml:"partition"`
	Timeout     time.Duration `toml:"timeout"`
}

type StaticSrcConfig struct {
	StaticAvatarPath string `toml:"staticAvatarPath"`
	StaticFilePath   string `toml:"staticFilePath"`
}

type AIModelConfig struct {
	DeepseekApiKey    string `toml:"deepseekApiKey"`
	DeepseekModel     string `toml:"deepseekModel"`
	DeepseekUrl       string `toml:"deepseekBaseUrl"`
	QwenApiKey        string `toml:"qwenApiKey"`
	QwenModel         string `toml:"qwenModel"`
	QwenBaseUrl       string `toml:"qwenBaseUrl"`
	GlmApiKey         string `toml:"glmApiKey"`
	GlmModel          string `toml:"glmModel"`
	GlmBaseUrl        string `toml:"glmBaseUrl"`
	Glm4vModel        string `toml:"glm4vModel"`
	GlmEmbeddingModel string `toml:"glmEmbeddingModel"`
}

type QdrantConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
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
	AIModelConfig   `toml:"aiModelConfig"`
	QdrantConfig    `toml:"qdrantConfig"`
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

	// AI Model Configuration
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.AIModelConfig.DeepseekApiKey = v
	}
	if v := os.Getenv("DEEPSEEK_MODEL"); v != "" {
		cfg.AIModelConfig.DeepseekModel = v
	}
	if v := os.Getenv("DEEPSEEK_BASE_URL"); v != "" {
		cfg.AIModelConfig.DeepseekUrl = v
	}
	if v := os.Getenv("QWEN_API_KEY"); v != "" {
		cfg.AIModelConfig.QwenApiKey = v
	}
	if v := os.Getenv("QWEN_MODEL"); v != "" {
		cfg.AIModelConfig.QwenModel = v
	}
	if v := os.Getenv("QWEN_BASE_URL"); v != "" {
		cfg.AIModelConfig.QwenBaseUrl = v
	}
	if v := os.Getenv("GLM_API_KEY"); v != "" {
		cfg.AIModelConfig.GlmApiKey = v
	}
	if v := os.Getenv("GLM_MODEL"); v != "" {
		cfg.AIModelConfig.GlmModel = v
	}
	if v := os.Getenv("GLM_BASE_URL"); v != "" {
		cfg.AIModelConfig.GlmBaseUrl = v
	}
	if v := os.Getenv("GLM4V_MODEL"); v != "" {
		cfg.AIModelConfig.Glm4vModel = v
	}
	if v := os.Getenv("GLM_EMBEDDING_MODEL"); v != "" {
		cfg.AIModelConfig.GlmEmbeddingModel = v
	}

	// Qdrant Configuration
	if v := os.Getenv("QDRANT_HOST"); v != "" {
		cfg.QdrantConfig.Host = v
	}
	if v := os.Getenv("QDRANT_PORT"); v != "" {
		cfg.QdrantConfig.Port = 6334
		if v != "6334" {
			port := 0
			for _, c := range v {
				if c >= '0' && c <= '9' {
					port = port*10 + int(c-'0')
				}
			}
			cfg.QdrantConfig.Port = port
		}
	}
}

func LoadConfig() error {
	config = new(Config)

	// 加载 .env 文件
	_ = godotenv.Load()

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
