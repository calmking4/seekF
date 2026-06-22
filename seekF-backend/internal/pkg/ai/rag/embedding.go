package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"seekF-backend/internal/pkg/zlog"
	"time"
)

// embeddingHTTPClient 带超时的 HTTP 客户端
var embeddingHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 5,
	},
}

// Embedding 向量化模块,负责将文本转换为向量
type Embedding struct {
	apiKey  string
	model   string
	baseURL string
}

// NewEmbedding 创建向量化实例
func NewEmbedding(apiKey, model, baseURL string) *Embedding {
	return &Embedding{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// EmbeddingRequest 向量化请求
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

// EmbeddingResponse 向量化响应
type EmbeddingResponse struct {
	Data   []EmbeddingData `json:"data"`
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData 向量化数据
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// EmbedTexts 批量文本向量化
func (e *Embedding) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var embeddings [][]float32

	for _, text := range texts {
		emb, err := e.embedSingle(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, emb)
	}

	return embeddings, nil
}

// embedSingle 单个文本向量化
func (e *Embedding) embedSingle(ctx context.Context, text string) ([]float32, error) {
	url := e.baseURL + "/embeddings"
	reqBody, _ := json.Marshal(EmbeddingRequest{
		Input: text,
		Model: e.model,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := embeddingHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		zlog.Error("向量化请求失败: " + string(body))
		return nil, fmt.Errorf("向量化请求失败，状态码: %d", resp.StatusCode)
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("未返回向量化结果")
	}

	return result.Data[0].Embedding, nil
}
