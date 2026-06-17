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

// ESDiscoverDAO ES帖子搜索DAO接口
type ESDiscoverDAO interface {
	// IndexPost 索引一篇帖子到ES
	IndexPost(post *models.DiscoverPost) error
	// SearchPosts 按关键词搜索帖子（分页 + 高亮）
	SearchPosts(keyword string, page, pageSize int) ([]models.DiscoverPost, int64, error)
	// UpdatePostStatus 更新帖子状态
	UpdatePostStatus(postUuid string, status int8) error
}

// ESDiscoverDAOImpl ES帖子搜索DAO实现
type ESDiscoverDAOImpl struct {
	client *elasticsearch.Client
}

// NewESDiscoverDAO 创建ES帖子搜索DAO实例
func NewESDiscoverDAO() ESDiscoverDAO {
	return &ESDiscoverDAOImpl{
		client: db.ESClient,
	}
}

// esPostDoc ES帖子文档结构
type esPostDoc struct {
	PostId       string   `json:"post_id"`
	UserId       string   `json:"user_id"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	CollectCount int      `json:"collect_count"`
	Status       int8     `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

// IndexPost 索引一篇帖子到ES
func (d *ESDiscoverDAOImpl) IndexPost(post *models.DiscoverPost) error {
	var tags []string
	if len(post.Tags) > 0 {
		json.Unmarshal(post.Tags, &tags)
	}

	doc := esPostDoc{
		PostId:       post.Uuid,
		UserId:       post.UserId,
		Title:        post.Title,
		Content:      post.Content,
		Tags:         tags,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		CollectCount: post.CollectCount,
		Status:       post.Status,
		CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("序列化帖子文档失败: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      db.ESIndexDiscoverPosts,
		DocumentID: post.Uuid,
		Body:       bytes.NewReader(data),
		Refresh:    "false",
	}

	res, err := req.Do(context.Background(), d.client)
	if err != nil {
		zlog.Error("索引帖子到ES失败: " + err.Error())
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		zlog.Error("ES索引帖子返回错误: " + res.String())
		return fmt.Errorf("ES索引帖子返回错误: %s", res.Status())
	}

	return nil
}

// SearchPosts 按关键词搜索帖子（分页 + 高亮）
func (d *ESDiscoverDAOImpl) SearchPosts(keyword string, page, pageSize int) ([]models.DiscoverPost, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	from := (page - 1) * pageSize

	// 构建查询：status过滤 + title/content/tags 多字段搜索
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"status": 0,
						},
					},
					{
						"multi_match": map[string]interface{}{
							"query":  keyword,
							"fields": []string{"title^2", "content", "tags"},
							"type":   "best_fields",
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title": map[string]interface{}{
					"pre_tags":  []string{"<em>"},
					"post_tags": []string{"</em>"},
				},
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
		"from": from,
		"size": pageSize,
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化查询失败: %w", err)
	}

	res, err := d.client.Search(
		d.client.Search.WithIndex(db.ESIndexDiscoverPosts),
		d.client.Search.WithBody(bytes.NewReader(data)),
		d.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ES搜索帖子失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("ES搜索帖子返回错误: %s", res.String())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    esPostDoc            `json:"_source"`
				Highlight map[string][]string   `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("解析ES搜索结果失败: %w", err)
	}

	var posts []models.DiscoverPost
	for _, hit := range result.Hits.Hits {
		tagsJSON, _ := json.Marshal(hit.Source.Tags)
		post := models.DiscoverPost{
			Uuid:         hit.Source.PostId,
			UserId:       hit.Source.UserId,
			Title:        hit.Source.Title,
			Content:      hit.Source.Content,
			Tags:         tagsJSON,
			LikeCount:    hit.Source.LikeCount,
			CommentCount: hit.Source.CommentCount,
			CollectCount: hit.Source.CollectCount,
			Status:       hit.Source.Status,
		}
		if t, err := time.Parse("2006-01-02 15:04:05", hit.Source.CreatedAt); err == nil {
			post.CreatedAt = t
		}
		// 如果有高亮内容，替换title/content
		if highlights, ok := hit.Highlight["title"]; ok && len(highlights) > 0 {
			post.Title = highlights[0]
		}
		if highlights, ok := hit.Highlight["content"]; ok && len(highlights) > 0 {
			post.Content = highlights[0]
		}
		posts = append(posts, post)
	}

	return posts, result.Hits.Total.Value, nil
}

// UpdatePostStatus 更新帖子状态
func (d *ESDiscoverDAOImpl) UpdatePostStatus(postUuid string, status int8) error {
	query := map[string]interface{}{
		"doc": map[string]interface{}{
			"status": status,
		},
	}

	data, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("序列化更新请求失败: %w", err)
	}

	req := esapi.UpdateRequest{
		Index:      db.ESIndexDiscoverPosts,
		DocumentID: postUuid,
		Body:       bytes.NewReader(data),
	}

	res, err := req.Do(context.Background(), d.client)
	if err != nil {
		zlog.Error("ES更新帖子状态失败: " + err.Error())
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		zlog.Error("ES更新帖子状态返回错误: " + res.String())
		return fmt.Errorf("ES更新帖子状态返回错误: %s", res.Status())
	}

	return nil
}
