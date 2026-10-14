package userdao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ESDiscoverDAO 同步当前作品状态并返回搜索候选ID，业务内容与权限必须回数据库核对。
type ESDiscoverDAO interface {
	SyncPost(context.Context, *models.DiscoverPost) error
	SearchPostIDs(context.Context, string) ([]string, error)
}

// ESDiscoverDAOImpl 发现页搜索索引的数据访问实现。
type ESDiscoverDAOImpl struct {
	client *elasticsearch.Client
}

// NewESDiscoverDAO 复用已初始化的ES连接，失败时交由持久化事件重试。
func NewESDiscoverDAO() ESDiscoverDAO {
	return &ESDiscoverDAOImpl{client: db.ESClient}
}

// esPostDoc 只保存检索字段，互动数量与媒体等展示信息从当前数据库装配。
type esPostDoc struct {
	Id        int64    `json:"id"`
	Status    int8     `json:"status"`
	Title     string   `json:"title,omitempty"`
	Content   string   `json:"content,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// SyncPost 使用作品版本写入索引，隐藏或删除作品保留无正文的墓碑以阻止旧消息复活。
func (d *ESDiscoverDAOImpl) SyncPost(ctx context.Context, post *models.DiscoverPost) error {
	if d.client == nil {
		return fmt.Errorf("ES客户端未初始化")
	}
	if post.Version == 0 || post.Version > math.MaxInt64 {
		return fmt.Errorf("作品搜索版本无效，postUuid=%s", post.Uuid)
	}
	if err := db.EnsureDiscoverSearchIndex(ctx); err != nil {
		return err
	}
	doc := esPostDoc{Id: post.Id, Status: 1}
	if !post.DeletedAt.Valid && post.Visibility == 0 && post.Status == 0 {
		doc.Status = 0
		doc.Title = post.Title
		doc.Content = post.Content
		doc.CreatedAt = post.CreatedAt.Format("2006-01-02 15:04:05")
		if len(post.Tags) > 0 {
			if err := json.Unmarshal(post.Tags, &doc.Tags); err != nil {
				return fmt.Errorf("解析作品标签失败: %w", err)
			}
		}
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("编码搜索文档失败: %w", err)
	}
	version := int(post.Version)
	request := esapi.IndexRequest{
		Index:       db.ESIndexDiscoverPosts,
		DocumentID:  post.Uuid,
		Body:        bytes.NewReader(data),
		Version:     &version,
		VersionType: "external",
		Refresh:     "wait_for",
	}
	response, err := request.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("同步作品搜索索引失败: %w", err)
	}
	defer response.Body.Close()
	// 同版本重复写入或旧版本写入均不能覆盖索引中的较新状态。
	if response.StatusCode == http.StatusConflict {
		return nil
	}
	if response.IsError() {
		return fmt.Errorf("同步作品搜索索引返回错误: %s", response.String())
	}
	return nil
}

// SearchPostIDs 获取有界的全文候选集合，超过10000条或不完整响应时回退MySQL。
func (d *ESDiscoverDAOImpl) SearchPostIDs(ctx context.Context, keyword string) ([]string, error) {
	if d.client == nil {
		return nil, fmt.Errorf("ES客户端未初始化")
	}
	query := map[string]interface{}{
		"_source":          false,
		"size":             10000,
		"track_total_hits": true,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{"status": 0},
					},
				},
				"must": []interface{}{
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":  keyword,
							"fields": []string{"title^2", "content", "tags"},
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]string{"_score": "desc"},
			map[string]string{"created_at": "desc"},
			map[string]string{"id": "desc"},
		},
	}
	raw, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("编码搜索请求失败: %w", err)
	}
	response, err := d.client.Search(
		d.client.Search.WithContext(ctx),
		d.client.Search.WithIndex(db.ESIndexDiscoverPosts),
		d.client.Search.WithBody(bytes.NewReader(raw)),
	)
	if err != nil {
		return nil, fmt.Errorf("查询作品搜索索引失败: %w", err)
	}
	defer response.Body.Close()
	if response.IsError() {
		return nil, fmt.Errorf("查询作品搜索索引返回错误: %s", response.String())
	}
	var result struct {
		TimedOut bool `json:"timed_out"`
		Shards   struct {
			Failed int `json:"failed"`
		} `json:"_shards"`
		Hits struct {
			Total struct {
				Value    int64  `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			Hits []struct {
				Id string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析作品搜索结果失败: %w", err)
	}
	if result.TimedOut || result.Shards.Failed > 0 || result.Hits.Total.Relation != "eq" || result.Hits.Total.Value > 10000 || int64(len(result.Hits.Hits)) != result.Hits.Total.Value {
		return nil, fmt.Errorf("ES候选过多或结果不完整，改用数据库搜索")
	}
	ids := make([]string, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		if hit.Id == "" {
			return nil, fmt.Errorf("ES候选缺少作品ID")
		}
		ids = append(ids, hit.Id)
	}
	return ids, nil
}
