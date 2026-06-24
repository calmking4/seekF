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

// EmailConfig 邮件SMTP配置
type EmailConfig struct {
	SmtpHost    string `toml:"smtpHost"`
	SmtpPort    int    `toml:"smtpPort"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	FromAddress string `toml:"fromAddress"`
	FromName    string `toml:"fromName"`
}

type OSSConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	Region          string `toml:"region"`
	Bucket          string `toml:"bucket"`
	BaseURL         string `toml:"baseURL"`
}

type LogConfig struct {
	LogDir       string `toml:"logDir"`       // 日志目录，如 "./logs"
	MaxAge       int    `toml:"maxAge"`       // 保留旧日志的最大天数，默认 30
	RotationTime int    `toml:"rotationTime"` // 轮转间隔（小时），默认 24
}

type KafkaConfig struct {
	MessageMode    string        `toml:"messageMode"`
	HostPort       string        `toml:"hostPort"`
	LoginTopic     string        `toml:"loginTopic"`
	ChatTopic      string        `toml:"chatTopic"`
	AIChatTopic    string        `toml:"aiChatTopic"`
	AICommentTopic string        `toml:"aiCommentTopic"`
	LogoutTopic    string        `toml:"logoutTopic"`
	Partition      int           `toml:"partition"`
	Timeout        time.Duration `toml:"timeout"`
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
	TTSModel          string `toml:"ttsModel"`
	TTSVoice          string `toml:"ttsVoice"`
	QwenLocalApiKey   string `toml:"qwenLocalApiKey"`
	QwenLocalModel    string `toml:"qwenLocalModel"`
	QwenLocalBaseUrl  string `toml:"qwenLocalBaseUrl"`
	// 各模型的上下文窗口大小配置
	DeepseekMaxTokens  int      `toml:"deepseekMaxTokens"`
	QwenMaxTokens      int      `toml:"qwenMaxTokens"`
	GlmMaxTokens       int      `toml:"glmMaxTokens"`
	Glm4vMaxTokens     int      `toml:"glm4vMaxTokens"`
	QwenLocalMaxTokens int      `toml:"qwenLocalMaxTokens"`
	MultimodalModels   []string `toml:"multimodalModels"`
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
	MainConfig         `toml:"mainConfig"`
	MysqlConfig        `toml:"mysqlConfig"`
	RedisConfig        `toml:"redisConfig"`
	AuthCodeConfig     `toml:"authCodeConfig"`
	EmailConfig        `toml:"emailConfig"`
	OSSConfig          `toml:"ossConfig"`
	LogConfig          `toml:"logConfig"`
	KafkaConfig        `toml:"kafkaConfig"`
	StaticSrcConfig    `toml:"staticSrcConfig"`
	JWTConfig          `toml:"jwtConfig"`
	AuthConfig         `toml:"authConfig"`
	AIModelConfig      `toml:"aiModelConfig"`
	QdrantConfig       `toml:"qdrantConfig"`
	SeniverseConfig    `toml:"seniverseConfig"`
	ExchangeRateConfig `toml:"exchangeRateConfig"`
	TavilyConfig       `toml:"tavilyConfig"`
	ESConfig           `toml:"esConfig"`
	WSConfig           `toml:"wsConfig"`
	GithubOAuthConfig  `toml:"githubOAuthConfig"`
	GiteeOAuthConfig   `toml:"giteeOAuthConfig"`
}

type SeniverseConfig struct {
	APIKey string `toml:"apiKey"`
}

type ExchangeRateConfig struct {
	APIKey string `toml:"apiKey"`
}

type TavilyConfig struct {
	APIKey string `toml:"apiKey"`
}

// ESConfig Elasticsearch配置
type ESConfig struct {
	Addresses string `toml:"addresses"`
	Username  string `toml:"username"`
	Password  string `toml:"password"`
}

// WSConfig WebSocket配置
type WSConfig struct {
	AllowedOrigins []string `toml:"allowedOrigins"` // 允许的Origin白名单，为空时允许所有来源
}

// GithubOAuthConfig GitHub OAuth配置
type GithubOAuthConfig struct {
	ClientID            string `toml:"clientID"`
	ClientSecret        string `toml:"clientSecret"`
	RedirectURL         string `toml:"redirectURL"`
	FrontendRedirectURL string `toml:"frontendRedirectURL"`
	ProxyURL            string `toml:"proxyURL"` // HTTP 代理地址，如 http://127.0.0.1:7890
}

// GiteeOAuthConfig Gitee OAuth配置
type GiteeOAuthConfig struct {
	ClientID            string `toml:"clientID"`
	ClientSecret        string `toml:"clientSecret"`
	RedirectURL         string `toml:"redirectURL"`
	FrontendRedirectURL string `toml:"frontendRedirectURL"`
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

	// 邮箱配置
	if v := os.Getenv("EMAIL_USERNAME"); v != "" {
		cfg.EmailConfig.Username = v
	}
	if v := os.Getenv("EMAIL_PASSWORD"); v != "" {
		cfg.EmailConfig.Password = v
	}
	if v := os.Getenv("EMAIL_FROM_ADDRESS"); v != "" {
		cfg.EmailConfig.FromAddress = v
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
	if v := os.Getenv("TTS_MODEL"); v != "" {
		cfg.AIModelConfig.TTSModel = v
	}
	if v := os.Getenv("TTS_VOICE"); v != "" {
		cfg.AIModelConfig.TTSVoice = v
	}
	if v := os.Getenv("QWEN_LOCAL_API_KEY"); v != "" {
		cfg.AIModelConfig.QwenLocalApiKey = v
	}
	if v := os.Getenv("QWEN_LOCAL_MODEL"); v != "" {
		cfg.AIModelConfig.QwenLocalModel = v
	}
	if v := os.Getenv("QWEN_LOCAL_BASE_URL"); v != "" {
		cfg.AIModelConfig.QwenLocalBaseUrl = v
	}

	// Seniverse Configuration
	if v := os.Getenv("SENIVERSE_API_KEY"); v != "" {
		cfg.SeniverseConfig.APIKey = v
	}

	// ExchangeRate Configuration
	if v := os.Getenv("EXCHANGE_RATE_API_KEY"); v != "" {
		cfg.ExchangeRateConfig.APIKey = v
	}

	// Tavily Configuration
	if v := os.Getenv("TAVILY_API_KEY"); v != "" {
		cfg.TavilyConfig.APIKey = v
	}

	// Elasticsearch Configuration
	if v := os.Getenv("ES_USERNAME"); v != "" {
		cfg.ESConfig.Username = v
	}
	if v := os.Getenv("ES_PASSWORD"); v != "" {
		cfg.ESConfig.Password = v
	}

	// GitHub OAuth Configuration
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_ID"); v != "" {
		cfg.GithubOAuthConfig.ClientID = v
	}
	if v := os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"); v != "" {
		cfg.GithubOAuthConfig.ClientSecret = v
	}

	// Gitee OAuth Configuration
	if v := os.Getenv("GITEE_OAUTH_CLIENT_ID"); v != "" {
		cfg.GiteeOAuthConfig.ClientID = v
	}
	if v := os.Getenv("GITEE_OAUTH_CLIENT_SECRET"); v != "" {
		cfg.GiteeOAuthConfig.ClientSecret = v
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
