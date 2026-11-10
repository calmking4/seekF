package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/elastic/go-elasticsearch/v8"
)

// ESClient Elasticsearch客户端单例
var ESClient *elasticsearch.Client

// InitES 显式创建ES客户端，限时探测连接并准备帖子索引。
func InitES(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cfg := configs.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未初始化，无法初始化ES")
	}
	if ESClient == nil {
		esCfg := elasticsearch.Config{Addresses: strings.Split(cfg.ESConfig.Addresses, ",")}
		if cfg.ESConfig.Username != "" {
			esCfg.Username = cfg.ESConfig.Username
			esCfg.Password = cfg.ESConfig.Password
		}
		client, err := elasticsearch.NewClient(esCfg)
		if err != nil {
			return fmt.Errorf("创建ES客户端失败: %w", err)
		}
		ESClient = client
	}
	res, err := ESClient.Info(ESClient.Info.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("连接ES失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES连接返回错误，状态码=%d", res.StatusCode)
	}
	if err := EnsureSearchIndex(ctx); err != nil {
		return fmt.Errorf("准备ES搜索索引失败: %w", err)
	}
	zlog.Info("ES初始化完成")
	return nil
}

// AnalyzeText 使用指定ES分析器返回词项，保留重复词项以反映原文词频。
func AnalyzeText(ctx context.Context, analyzer, text string) ([]string, error) {
	if ESClient == nil {
		return nil, fmt.Errorf("ES客户端未初始化")
	}
	body, err := json.Marshal(map[string]string{"analyzer": analyzer, "text": text})
	if err != nil {
		return nil, fmt.Errorf("编码ES文本分析请求失败: %w", err)
	}
	res, err := ESClient.Indices.Analyze(
		ESClient.Indices.Analyze.WithContext(ctx),
		ESClient.Indices.Analyze.WithBody(bytes.NewReader(body)),
		// 显式使用普通JSON，兼容项目现有ES 7.14服务端。
		ESClient.Indices.Analyze.WithHeader(map[string]string{"Content-Type": "application/json", "Accept": "application/json"}),
	)
	if err != nil {
		return nil, fmt.Errorf("调用ES文本分析失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("ES文本分析返回错误（状态码%d）", res.StatusCode)
	}
	var result struct {
		Tokens []struct {
			Token string `json:"token"`
		} `json:"tokens"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4*1024*1024)).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析ES文本分析结果失败: %w", err)
	}
	if result.Tokens == nil {
		return nil, fmt.Errorf("ES文本分析结果缺少词项列表")
	}
	tokens := make([]string, 0, len(result.Tokens))
	for _, item := range result.Tokens {
		if token := strings.TrimSpace(item.Token); token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}
