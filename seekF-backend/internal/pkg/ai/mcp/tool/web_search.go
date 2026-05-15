package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"bytes"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
)

// 复用连接，降低 Tavily 请求 TLS 握手开销
var tavilyHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
}

// SearchSource 搜索来源
type SearchSource struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score,omitempty"`
}

type WebSearchTool struct {
	apiKey string
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		apiKey: configs.GetConfig().TavilyConfig.APIKey,
	}
}

// GetWebSearchTool 获取网页搜索工具
func (t *WebSearchTool) GetWebSearchTool() mcp.Tool {
	return mcp.NewTool(
		"web_search",
		mcp.WithDescription("搜索互联网获取最新信息。当用户询问需要最新数据、实时新闻、时事热点、或你不确定的事实时使用此工具。返回搜索结果摘要和参考链接。"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索关键词，应简洁明确，如：'2026年最新AI进展'、'Go语言泛型教程'。"),
		),
	)
}

// HandleWebSearchRequest 处理网页搜索请求
// 返回 TextContent（摘要给 AI）+ EmbeddedResource（结构化来源 JSON 给前端）
func (t *WebSearchTool) HandleWebSearchRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultText("参数解析失败"), nil
	}

	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultText("请提供搜索关键词"), nil
	}

	result, err := t.searchWeb(ctx, query)
	if err != nil {
		zlog.Error("网页搜索失败: " + err.Error())
		return mcp.NewToolResultText("搜索失败: " + err.Error()), nil
	}

	return result, nil
}

// tavilyResponse Tavily API 响应结构
type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

// searchWeb 调用 Tavily Search API 搜索网页
// 返回包含 TextContent + EmbeddedResource 的 CallToolResult
func (t *WebSearchTool) searchWeb(ctx context.Context, query string) (*mcp.CallToolResult, error) {
	if t.apiKey == "" {
		return nil, fmt.Errorf("Tavily搜索API密钥未配置")
	}

	// 构建请求体
	reqBody := map[string]any{
		"api_key":        t.apiKey,
		"query":          query,
		"max_results":    5,
		"include_answer": true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// 发送 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tavilyHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tavily API请求失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return mcp.NewToolResultText("未找到相关搜索结果"), nil
	}

	// 构建文本摘要（给 AI 模型消费）
	var sb strings.Builder
	if result.Answer != "" {
		sb.WriteString("搜索结果摘要: ")
		sb.WriteString(result.Answer)
		sb.WriteString("\n\n")
	}

	sb.WriteString("参考来源:\n")
	for i, r := range result.Results {
		snippet := r.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d. %s - %s\n   %s\n", i+1, r.Title, r.URL, snippet))
	}

	// 构建结构化来源数据（给前端渲染）
	sources := make([]SearchSource, 0, len(result.Results))
	for _, r := range result.Results {
		sources = append(sources, SearchSource{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Score:   r.Score,
		})
	}

	// 返回 TextContent（AI 摘要）+ StructuredContent（前端结构化数据）
	return mcp.NewToolResultStructured(sources, sb.String()), nil
}
