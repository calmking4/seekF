package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"seekF-backend/internal/pkg/zlog"
)

type Embedding struct {
	apiKey  string
	model   string
	baseURL string
}

func NewEmbedding(apiKey, model, baseURL string) *Embedding {
	return &Embedding{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type EmbeddingResponse struct {
	Data   []EmbeddingData `json:"data"`
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		zlog.Error("embedding request failed: " + string(body))
		return nil, fmt.Errorf("embedding request failed with status: %d", resp.StatusCode)
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Data[0].Embedding, nil
}
