package db

import (
	"bytes"
	"encoding/json"
	"fmt"

	"seekF-backend/internal/pkg/zlog"
)

const (
	// ESIndexChatMessages 聊天消息索引名
	ESIndexChatMessages = "chat_messages"
	// ESIndexDiscoverPosts 发现页帖子索引名
	ESIndexDiscoverPosts = "discover_posts"
)

// chatMessagesMapping 聊天消息索引映射（IK中文分词）
const chatMessagesMapping = `{
	"mappings": {
		"properties": {
			"message_id":   { "type": "keyword" },
			"session_id":   { "type": "keyword" },
			"send_id":      { "type": "keyword" },
			"send_name":    { "type": "keyword" },
			"receive_id":   { "type": "keyword" },
			"content":      { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"message_type": { "type": "integer" },
			"status":       { "type": "integer" },
			"created_at":   { "type": "date", "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd'T'HH:mm:ssZ||epoch_millis" }
		}
	}
}`

// discoverPostsMapping 发现页帖子索引映射（IK中文分词）
const discoverPostsMapping = `{
	"mappings": {
		"properties": {
			"post_id":       { "type": "keyword" },
			"user_id":       { "type": "keyword" },
			"title":         { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"content":       { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"tags":          { "type": "keyword" },
			"like_count":    { "type": "integer" },
			"comment_count": { "type": "integer" },
			"collect_count": { "type": "integer" },
			"status":        { "type": "integer" },
			"created_at":    { "type": "date", "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd'T'HH:mm:ssZ||epoch_millis" }
		}
	}
}`

// EnsureESIndices 确保所有ES索引存在，不存在则创建
func EnsureESIndices() error {
	indices := map[string]string{
		ESIndexChatMessages:  chatMessagesMapping,
		ESIndexDiscoverPosts: discoverPostsMapping,
	}

	for indexName, mapping := range indices {
		if err := ensureIndex(indexName, mapping); err != nil {
			return fmt.Errorf("创建索引 %s 失败: %w", indexName, err)
		}
	}
	return nil
}

// ensureIndex 确保单个索引存在
func ensureIndex(indexName, mapping string) error {
	if ESClient == nil {
		return fmt.Errorf("ES客户端未初始化")
	}

	// 检查索引是否存在
	res, err := ESClient.Indices.Exists([]string{indexName})
	if err != nil {
		return fmt.Errorf("检查索引 %s 失败: %w", indexName, err)
	}
	defer res.Body.Close()

	// 索引已存在，跳过
	if res.StatusCode == 200 {
		zlog.Info("ES索引已存在: " + indexName)
		return nil
	}

	// 创建索引
	res, err = ESClient.Indices.Create(
		indexName,
		ESClient.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))),
	)
	if err != nil {
		return fmt.Errorf("创建索引 %s 失败: %w", indexName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&errResp)
		return fmt.Errorf("创建索引 %s 返回错误: %v", indexName, errResp)
	}

	zlog.Info("已创建ES索引: " + indexName)
	return nil
}
