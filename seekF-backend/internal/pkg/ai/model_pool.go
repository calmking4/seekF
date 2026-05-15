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
	DeepSeek model.ToolCallingChatModel
	Qwen     model.ToolCallingChatModel
	GLM      model.ToolCallingChatModel
	GLM4V    model.ToolCallingChatModel
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
				zlog.Error("初始化deepseek模型失败: " + err.Error())
			} else {
				modelPool.DeepSeek = ds
				zlog.Info("deepseek模型初始化成功")
			}
		}

		if cfg.QwenApiKey != "" {
			qw, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.QwenApiKey,
				Model:   cfg.QwenModel,
				BaseURL: cfg.QwenBaseUrl,
			})
			if err != nil {
				zlog.Error("初始化qwen模型失败: " + err.Error())
			} else {
				modelPool.Qwen = qw
				zlog.Info("qwen模型初始化成功")
			}
		}

		if cfg.GlmApiKey != "" {
			gm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.GlmApiKey,
				Model:   cfg.GlmModel,
				BaseURL: cfg.GlmBaseUrl,
			})
			if err != nil {
				zlog.Error("初始化glm模型失败: " + err.Error())
			} else {
				modelPool.GLM = gm
				zlog.Info("glm模型初始化成功")
			}
		}

		if cfg.GlmApiKey != "" && cfg.Glm4vModel != "" {
			g4v, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  cfg.GlmApiKey,
				Model:   cfg.Glm4vModel,
				BaseURL: cfg.GlmBaseUrl,
			})
			if err != nil {
				zlog.Error("初始化glm-4.6v模型失败: " + err.Error())
			} else {
				modelPool.GLM4V = g4v
				zlog.Info("glm-4.6v模型初始化成功")
			}
		}

		zlog.Info("模型池初始化完成")
	})
	return modelPool
}

func (p *ModelPool) GetModel(modelType string) model.ToolCallingChatModel {
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
