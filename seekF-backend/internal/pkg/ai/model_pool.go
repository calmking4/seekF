package ai

import (
	"context"
	"os"
	"sync"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type ModelPool struct {
	DeepSeek model.BaseChatModel
	Qwen     model.BaseChatModel
	GLM      model.BaseChatModel
}

var (
	modelPool     *ModelPool
	modelPoolOnce sync.Once
)

func GetModelPool() *ModelPool {
	modelPoolOnce.Do(func() {
		cfg := configs.GetConfig().AIModelConfig
		ctx := context.Background()

		modelPool = &ModelPool{}

		if cfg.DeepseekApiKey != "" {
			ds, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.DeepseekApiKey,
				Model:   cfg.DeepseekModel,
				BaseURL: cfg.DeepseekUrl,
			})
			if err != nil {
				zlog.Error("init deepseek model failed: " + err.Error())
			} else {
				modelPool.DeepSeek = ds
				zlog.Info("deepseek model initialized successfully")
			}
		} else if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
			mdl := os.Getenv("DEEPSEEK_MODEL")
			if mdl == "" {
				mdl = "deepseek-chat"
			}
			baseURL := os.Getenv("DEEPSEEK_BASE_URL")
			if baseURL == "" {
				baseURL = "https://api.deepseek.com/v1"
			}
			ds, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  apiKey,
				Model:   mdl,
				BaseURL: baseURL,
			})
			if err != nil {
				zlog.Error("init deepseek model failed: " + err.Error())
			} else {
				modelPool.DeepSeek = ds
				zlog.Info("deepseek model initialized from env")
			}
		}

		if cfg.QwenApiKey != "" {
			qw, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.QwenApiKey,
				Model:   cfg.QwenModel,
				BaseURL: cfg.QwenBaseUrl,
			})
			if err != nil {
				zlog.Error("init qwen model failed: " + err.Error())
			} else {
				modelPool.Qwen = qw
				zlog.Info("qwen model initialized successfully")
			}
		} else if apiKey := os.Getenv("QWEN_API_KEY"); apiKey != "" {
			mdl := os.Getenv("QWEN_MODEL")
			if mdl == "" {
				mdl = "qwen-plus"
			}
			baseURL := os.Getenv("QWEN_BASE_URL")
			if baseURL == "" {
				baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
			}
			qw, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  apiKey,
				Model:   mdl,
				BaseURL: baseURL,
			})
			if err != nil {
				zlog.Error("init qwen model failed: " + err.Error())
			} else {
				modelPool.Qwen = qw
				zlog.Info("qwen model initialized from env")
			}
		}

		if cfg.GlmApiKey != "" {
			gm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.GlmApiKey,
				Model:   cfg.GlmModel,
				BaseURL: cfg.GlmBaseUrl,
			})
			if err != nil {
				zlog.Error("init glm model failed: " + err.Error())
			} else {
				modelPool.GLM = gm
				zlog.Info("glm model initialized successfully")
			}
		} else if apiKey := os.Getenv("GLM_API_KEY"); apiKey != "" {
			mdl := os.Getenv("GLM_MODEL")
			if mdl == "" {
				mdl = "glm-4"
			}
			baseURL := os.Getenv("GLM_BASE_URL")
			if baseURL == "" {
				baseURL = "https://open.bigmodel.cn/api/paas/v4/"
			}
			gm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  apiKey,
				Model:   mdl,
				BaseURL: baseURL,
			})
			if err != nil {
				zlog.Error("init glm model failed: " + err.Error())
			} else {
				modelPool.GLM = gm
				zlog.Info("glm model initialized from env")
			}
		}

		zlog.Info("model pool initialization completed")
	})
	return modelPool
}

func (p *ModelPool) GetModel(modelType string) model.BaseChatModel {
	switch modelType {
	case "deepseek":
		return p.DeepSeek
	case "qwen":
		return p.Qwen
	case "glm":
		return p.GLM
	default:
		return p.DeepSeek
	}
}
