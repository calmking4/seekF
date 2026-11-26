package rag

import (
	"context"
	"fmt"
	"strings"

	"seekF-backend/internal/pkg/constants"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// newQueryRewriter 创建用于把追问改写为独立查询的对话客户端。
func newQueryRewriter(apiKey, baseURL, model string) (*openai.ChatModel, error) {
	model = strings.TrimSpace(model)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if model == "" || apiKey == "" || baseURL == "" {
		return nil, nil
	}
	maxTokens := constants.KnowledgeRewriteOutputTokens
	return openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      model,
		MaxTokens:  &maxTokens,
		HTTPClient: retrievalHTTPClient,
		ExtraFields: map[string]any{
			"thinking": map[string]string{"type": "disabled"},
		},
	})
}

// RewriteQuery 根据近期对话将当前问题补全为独立的检索语句。
func (r *RAG) RewriteQuery(ctx context.Context, query string, history []*schema.Message) (string, error) {
	if r.queryRewriter == nil || len(history) == 0 || strings.TrimSpace(query) == "" {
		return query, nil
	}

	messages := make([]*schema.Message, 0, len(history)+2)
	messages = append(messages, schema.SystemMessage("你负责把最后一条用户消息改写成一条可以独立检索知识库的问题。"+
		"历史对话仅用于消解指代和补全省略内容。保留原意、名称、编号和时间等约束，不猜测缺失信息。"+
		"如果原问题已经独立清楚，原样返回。只输出一行查询文本，不回答问题，不加解释或格式标记。"))
	messages = append(messages, history...)
	messages = append(messages, schema.UserMessage(query))

	rewriteCtx, cancel := context.WithTimeout(ctx, constants.KnowledgeRewriteTimeout)
	defer cancel()
	response, err := r.queryRewriter.Generate(rewriteCtx, messages)
	if err != nil {
		return query, fmt.Errorf("调用查询改写模型失败: %w", err)
	}
	if response == nil || (response.ResponseMeta != nil && response.ResponseMeta.FinishReason != "" && response.ResponseMeta.FinishReason != "stop") {
		return query, fmt.Errorf("查询改写未正常完成")
	}
	rewritten := strings.TrimSpace(response.Content)
	if rewritten == "" || strings.ContainsAny(rewritten, "\r\n") {
		return query, fmt.Errorf("查询改写结果必须是一行非空文本")
	}
	return rewritten, nil
}
