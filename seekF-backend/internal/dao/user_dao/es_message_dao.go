package userdao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"
)

// ESMessageDAO ES消息搜索DAO接口
type ESMessageDAO interface {
	// IndexMessage 索引一条消息到ES
	IndexMessage(message *models.Message) error
	// SearchMessages 按sessionId + 关键词搜索消息（分页）
	SearchMessages(sessionId string, keyword string, page, pageSize int) ([]models.Message, int64, error)
	// SearchMessagesBySessionIds 按会话列表 + 关键词搜索消息（跨会话联想）
	SearchMessagesBySessionIds(sessionIds []string, keyword string, limit int) ([]models.Message, int64, error)
	// DeleteMessagesBySessionId 删除指定会话的所有消息
	DeleteMessagesBySessionId(sessionId string) error
}

// ESMessageDAOImpl ES消息搜索DAO实现
type ESMessageDAOImpl struct {
	client *elasticsearch.Client
}

// NewESMessageDAO 创建ES消息搜索DAO实例
func NewESMessageDAO() ESMessageDAO {
	return &ESMessageDAOImpl{
		client: db.ESClient,
	}
}

// esMessageDoc ES消息文档结构
type esMessageDoc struct {
	MessageId   string `json:"message_id"`
	SessionId   string `json:"session_id"`
	SendId      string `json:"send_id"`
	SendName    string `json:"send_name"`
	ReceiveId   string `json:"receive_id"`
	Content     string `json:"content"`
	MessageType int8   `json:"message_type"`
	Status      int8   `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// IndexMessage 索引一条消息到ES
func (d *ESMessageDAOImpl) IndexMessage(message *models.Message) error {
	doc := esMessageDoc{
		MessageId:   message.Uuid,
		SessionId:   message.SessionId,
		SendId:      message.SendId,
		SendName:    message.SendName,
		ReceiveId:   message.ReceiveId,
		Content:     message.Content,
		MessageType: message.Type,
		Status:      message.Status,
		CreatedAt:   message.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("序列化消息文档失败: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      db.ESIndexChatMessages,
		DocumentID: message.Uuid,
		Body:       bytes.NewReader(data),
		Refresh:    "false",
	}

	res, err := req.Do(context.Background(), d.client)
	if err != nil {
		zlog.Error("索引消息到ES失败: " + err.Error())
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		zlog.Error("ES索引消息返回错误: " + res.String())
		return fmt.Errorf("ES索引消息返回错误: %s", res.Status())
	}

	return nil
}

// SearchMessages 按sessionId + 关键词搜索消息（分页）
func (d *ESMessageDAOImpl) SearchMessages(sessionId string, keyword string, page, pageSize int) ([]models.Message, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	from := (page - 1) * pageSize

	// 构建查询：sessionId 过滤 + content 全文搜索
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"session_id": sessionId,
						},
					},
					{
						"match": map[string]interface{}{
							"content": keyword,
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"content": map[string]interface{}{
					"pre_tags":  []string{"<em>"},
					"post_tags": []string{"</em>"},
				},
			},
		},
		"sort": []map[string]interface{}{
			{"created_at": "desc"},
		},
		"from": from,
		"size": pageSize,
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化查询失败: %w", err)
	}

	res, err := d.client.Search(
		d.client.Search.WithIndex(db.ESIndexChatMessages),
		d.client.Search.WithBody(bytes.NewReader(data)),
		d.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ES搜索消息失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("ES搜索消息返回错误: %s", res.String())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    esMessageDoc       `json:"_source"`
				Highlight map[string][]string `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("解析ES搜索结果失败: %w", err)
	}

	var messages []models.Message
	for _, hit := range result.Hits.Hits {
		msg := models.Message{
			Uuid:      hit.Source.MessageId,
			SessionId: hit.Source.SessionId,
			SendId:    hit.Source.SendId,
			SendName:  hit.Source.SendName,
			ReceiveId: hit.Source.ReceiveId,
			Content:   hit.Source.Content,
			Type:      hit.Source.MessageType,
			Status:    hit.Source.Status,
		}
		// 解析时间
		if t, err := time.Parse("2006-01-02 15:04:05", hit.Source.CreatedAt); err == nil {
			msg.CreatedAt = t
		}
		// 如果有高亮内容，替换content
		if highlights, ok := hit.Highlight["content"]; ok && len(highlights) > 0 {
			msg.Content = highlights[0]
		}
		messages = append(messages, msg)
	}

	return messages, result.Hits.Total.Value, nil
}

// SearchMessagesBySessionIds 按会话列表 + 关键词搜索消息（跨会话联想）
func (d *ESMessageDAOImpl) SearchMessagesBySessionIds(sessionIds []string, keyword string, limit int) ([]models.Message, int64, error) {
	if len(sessionIds) == 0 || keyword == "" {
		return nil, 0, nil
	}
	if limit < 1 {
		limit = 5
	}

	// 构建查询：terms 过滤多个 sessionId + match 搜索 content
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"session_id": sessionIds,
						},
					},
					{
						"match": map[string]interface{}{
							"content": keyword,
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"content": map[string]interface{}{
					"pre_tags":  []string{"<em>"},
					"post_tags": []string{"</em>"},
				},
			},
		},
		"sort": []map[string]interface{}{
			{"_score": "desc"},
			{"created_at": "desc"},
		},
		"size": limit,
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化查询失败: %w", err)
	}

	res, err := d.client.Search(
		d.client.Search.WithIndex(db.ESIndexChatMessages),
		d.client.Search.WithBody(bytes.NewReader(data)),
		d.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ES搜索消息失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("ES搜索消息返回错误: %s", res.String())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    esMessageDoc       `json:"_source"`
				Highlight map[string][]string `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("解析ES搜索结果失败: %w", err)
	}

	var messages []models.Message
	for _, hit := range result.Hits.Hits {
		msg := models.Message{
			Uuid:      hit.Source.MessageId,
			SessionId: hit.Source.SessionId,
			SendId:    hit.Source.SendId,
			SendName:  hit.Source.SendName,
			ReceiveId: hit.Source.ReceiveId,
			Content:   hit.Source.Content,
			Type:      hit.Source.MessageType,
			Status:    hit.Source.Status,
		}
		// 解析时间
		if t, err := time.Parse("2006-01-02 15:04:05", hit.Source.CreatedAt); err == nil {
			msg.CreatedAt = t
		}
		// 如果有高亮内容，替换content
		if highlights, ok := hit.Highlight["content"]; ok && len(highlights) > 0 {
			msg.Content = highlights[0]
		}
		messages = append(messages, msg)
	}

	return messages, result.Hits.Total.Value, nil
}

// DeleteMessagesBySessionId 删除指定会话的所有消息
func (d *ESMessageDAOImpl) DeleteMessagesBySessionId(sessionId string) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"session_id": sessionId,
			},
		},
	}

	data, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("序列化删除查询失败: %w", err)
	}

	res, err := d.client.DeleteByQuery(
		[]string{db.ESIndexChatMessages},
		bytes.NewReader(data),
	)
	if err != nil {
		zlog.Error("ES删除会话消息失败: " + err.Error())
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		zlog.Error("ES删除会话消息返回错误: " + res.String())
		return fmt.Errorf("ES删除会话消息返回错误: %s", res.Status())
	}

	zlog.Info("已从ES删除会话消息: " + sessionId)
	return nil
}
