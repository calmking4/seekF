package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"seekF-backend/internal/pkg/constants"
)

// rerankClient 保存文本重排序接口的连接信息。
type rerankClient struct {
	apiKey  string
	model   string
	baseURL string
}

// RerankResult 保存候选原始下标及模型相关性分数。
type RerankResult struct {
	Index int
	Score float64
}

// Rerank 批量评估候选正文，返回按相关性排序的结果。
func (r *RAG) Rerank(ctx context.Context, query string, documents []string, limit int) ([]RerankResult, error) {
	if r.reranker.model == "" || r.reranker.apiKey == "" || r.reranker.baseURL == "" || len(documents) == 0 || limit <= 0 {
		return nil, nil
	}
	if strings.TrimSpace(query) == "" || utf8.RuneCountInString(query) > constants.KnowledgeRerankTextLimit {
		return nil, fmt.Errorf("精排查询为空或超过%d个字符", constants.KnowledgeRerankTextLimit)
	}
	for _, text := range documents {
		if strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > constants.KnowledgeRerankTextLimit {
			return nil, fmt.Errorf("精排候选正文为空或超过%d个字符", constants.KnowledgeRerankTextLimit)
		}
	}

	topN := min(limit, len(documents))
	type rerankInput struct {
		Query     string   `json:"query"`
		Documents []string `json:"documents"`
	}
	type rerankParameters struct {
		TopN int `json:"top_n"`
	}
	payload, err := json.Marshal(struct {
		Model      string           `json:"model"`
		Input      rerankInput      `json:"input"`
		Parameters rerankParameters `json:"parameters"`
	}{Model: r.reranker.model, Input: rerankInput{Query: query, Documents: documents}, Parameters: rerankParameters{TopN: topN}})
	if err != nil {
		return nil, fmt.Errorf("编码精排请求失败: %w", err)
	}
	baseURL, err := url.Parse(r.reranker.baseURL)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "https" && baseURL.Scheme != "http") {
		return nil, fmt.Errorf("Qwen接口地址无效")
	}
	// 百炼原生精排接口与对话模型使用同一服务域名及不同路径。
	endpoint := baseURL.ResolveReference(&url.URL{Path: "/api/v1/services/rerank/text-rerank/text-rerank"})

	rerankCtx, cancel := context.WithTimeout(ctx, constants.KnowledgeRerankTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rerankCtx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("构建精排请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.reranker.apiKey)
	resp, err := retrievalHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用文本重排序接口失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("文本重排序接口返回状态码%d", resp.StatusCode)
	}

	var result struct {
		Output struct {
			Results []struct {
				Index *int     `json:"index"`
				Score *float64 `json:"relevance_score"`
			} `json:"results"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析精排结果失败: %w", err)
	}
	results := result.Output.Results
	if len(results) != topN {
		return nil, fmt.Errorf("精排返回%d条结果，预期%d条", len(results), topN)
	}
	seen := make([]bool, len(documents))
	for _, item := range results {
		if item.Index == nil || item.Score == nil || *item.Index < 0 || *item.Index >= len(documents) || seen[*item.Index] {
			return nil, fmt.Errorf("精排结果缺少分数或候选下标无效")
		}
		seen[*item.Index] = true
	}
	// 分数相同时沿用候选原有的RRF相对顺序。
	sort.Slice(results, func(i, j int) bool {
		left, right := results[i], results[j]
		if *left.Score == *right.Score {
			return *left.Index < *right.Index
		}
		return *left.Score > *right.Score
	})
	ranked := make([]RerankResult, len(results))
	for i, item := range results {
		ranked[i] = RerankResult{Index: *item.Index, Score: *item.Score}
	}
	return ranked, nil
}
