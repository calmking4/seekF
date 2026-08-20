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

// maxBatchSize 单次批量请求的最大文本数量
const maxBatchSize = 16

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

// EmbeddingBatchRequest 批量向量化请求
type EmbeddingBatchRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
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

// EmbedTexts 批量文本向量化，自动分批调用 API
func (e *Embedding) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// 如果文本数量较少，直接单次请求
	if len(texts) <= maxBatchSize {
		return e.embedBatch(ctx, texts)
	}

	// 分批处理
	var allEmbeddings [][]float32
	for i := 0; i < len(texts); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("批量向量化失败(批次 %d-%d): %w", i, end-1, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// embedBatch 单批次批量向量化
func (e *Embedding) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	url := e.baseURL + "/embeddings"
	reqBody, _ := json.Marshal(EmbeddingBatchRequest{
		Input: texts,
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

	// 按 Index 排序，确保返回顺序与输入一致
	embeddings := make([][]float32, len(result.Data))
	for _, data := range result.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	// 检查是否有缺失的 embedding
	for i, emb := range embeddings {
		if emb == nil {
			return nil, fmt.Errorf("第 %d 个文本的向量化结果缺失", i)
		}
	}

	zlog.Debug(fmt.Sprintf("批量向量化完成: %d 个文本", len(texts)))
	return embeddings, nil
}
