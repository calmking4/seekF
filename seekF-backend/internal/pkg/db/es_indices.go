package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"seekF-backend/internal/pkg/zlog"
)

// ESIndexDiscoverPosts 发现页帖子索引名。
const ESIndexDiscoverPosts = "discover_posts"

// discoverPostsMapping 发现页帖子索引映射（IK中文分词）
const discoverPostsMapping = `{
	"mappings": {
		"properties": {
			"id":            { "type": "long" },
			"title":         { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"content":       { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"tags":          { "type": "keyword" },
			"status":        { "type": "integer" },
			"created_at":    { "type": "date", "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd'T'HH:mm:ssZ||epoch_millis" }
		}
	}
}`

// ensureIndex 确保单个索引存在
func ensureIndex(ctx context.Context, indexName, mapping string) error {
	if ESClient == nil {
		return fmt.Errorf("ES客户端未初始化")
	}

	// 检查索引是否存在
	res, err := ESClient.Indices.Exists([]string{indexName}, ESClient.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("检查索引 %s 失败: %w", indexName, err)
	}
	defer res.Body.Close()

	// 索引已存在，跳过
	if res.StatusCode == 200 {
		return nil
	}

	// 创建索引
	res, err = ESClient.Indices.Create(
		indexName,
		ESClient.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))),
		ESClient.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("创建索引 %s 失败: %w", indexName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&errResp)
		if detail, ok := errResp["error"].(map[string]interface{}); ok && detail["type"] == "resource_already_exists_exception" {
			return nil
		}
		return fmt.Errorf("创建索引 %s 返回错误: %v", indexName, errResp)
	}

	zlog.Info("已创建ES索引: " + indexName)
	return nil
}

// EnsureSearchIndex 确保帖子搜索索引结构存在，历史内容由持久化重建任务补齐。
func EnsureSearchIndex(ctx context.Context) error {
	return ensureIndex(ctx, ESIndexDiscoverPosts, discoverPostsMapping)
}
