package ai

import (
	"context"
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
	GLM4V    model.BaseChatModel
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
		}

		if cfg.GlmApiKey != "" && cfg.Glm4vModel != "" {
			g4v, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.GlmApiKey,
				Model:   cfg.Glm4vModel,
				BaseURL: cfg.GlmBaseUrl,
			})
			if err != nil {
				zlog.Error("init glm-4.6v model failed: " + err.Error())
			} else {
				modelPool.GLM4V = g4v
				zlog.Info("glm-4.6v model initialized successfully")
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
	case "glm-4v":
		return p.GLM4V
	default:
		return p.DeepSeek
	}
}
