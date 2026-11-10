package ai

import (
	"context"
	"errors"
	"fmt"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ModelPool 保存启动时创建的AI模型客户端。
type ModelPool struct {
	DeepSeek  model.ToolCallingChatModel
	Qwen      model.ToolCallingChatModel
	GLM       model.ToolCallingChatModel
	GLM4V     model.ToolCallingChatModel
	QwenLocal model.ToolCallingChatModel
}

var modelPool *ModelPool

// InitModelPool 显式创建已配置的模型客户端，保留成功模型并汇总失败原因。
func InitModelPool(ctx context.Context) error {
	conf := configs.GetConfig()
	if conf == nil {
		return fmt.Errorf("配置未初始化，无法初始化AI模型池")
	}
	cfg := conf.AIModelConfig

	pool := &ModelPool{}
	var initErrors []error

	if cfg.DeepseekApiKey != "" {
		ds, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  cfg.DeepseekApiKey,
			Model:   cfg.DeepseekModel,
			BaseURL: cfg.DeepseekUrl,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("初始化deepseek模型失败: %w", err))
		} else {
			pool.DeepSeek = ds
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
			initErrors = append(initErrors, fmt.Errorf("初始化qwen模型失败: %w", err))
		} else {
			pool.Qwen = qw
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
			initErrors = append(initErrors, fmt.Errorf("初始化glm模型失败: %w", err))
		} else {
			pool.GLM = gm
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
			initErrors = append(initErrors, fmt.Errorf("初始化glm-4.6v模型失败: %w", err))
		} else {
			pool.GLM4V = g4v
			zlog.Info("glm-4.6v模型初始化成功")
		}
	}

	if cfg.QwenLocalBaseUrl != "" {
		ql, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  cfg.QwenLocalApiKey,
			Model:   cfg.QwenLocalModel,
			BaseURL: cfg.QwenLocalBaseUrl,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("初始化qwen-local模型失败: %w", err))
		} else {
			pool.QwenLocal = ql
			zlog.Info("qwen-local模型初始化成功")
		}
	}

	modelPool = pool
	zlog.Info("模型池初始化完成")
	return errors.Join(initErrors...)
}

// GetModelPool 返回已初始化的模型池，不创建模型客户端。
func GetModelPool() *ModelPool {
	return modelPool
}

// GetModel 根据模型类型返回可用客户端。
func (p *ModelPool) GetModel(modelType string) model.ToolCallingChatModel {
	if p == nil {
		return nil
	}
	switch modelType {
	case "deepseek":
		return p.DeepSeek
	case "qwen":
		return p.Qwen
	case "glm":
		return p.GLM
	case "glm-4v":
		return p.GLM4V
	case "qwen-local":
		return p.QwenLocal
	default:
		return p.DeepSeek
	}
}
