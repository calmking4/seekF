package db

import (
	"context"
	"fmt"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// qdrantClient Qdrant客户端单例
var qdrantClient *QdrantUtil

// QdrantUtil Qdrant向量数据库工具
type QdrantUtil struct {
	client *qdrant.Client
}

// VectorSearchResult 向量搜索结果，包含文档ID和分块索引
type VectorSearchResult struct {
	DocUUID  string
	ChunkIdx int
	Score    float32
}

// InitQdrant 初始化Qdrant客户端
func InitQdrant() error {
	cfg := configs.GetConfig()

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.QdrantConfig.Host,
		Port: cfg.QdrantConfig.Port,
	})
	if err != nil {
		return err
	}

	qdrantClient = &QdrantUtil{client: client}
	zlog.Info(fmt.Sprintf("connected to qdrant at %s:%d", cfg.QdrantConfig.Host, cfg.QdrantConfig.Port))
	return nil
}

// GetQdrant 获取Qdrant客户端实例
func GetQdrant() *QdrantUtil {
	return qdrantClient
}

// EnsureCollection 确保向量集合存在,不存在则创建
func (q *QdrantUtil) EnsureCollection(ctx context.Context, collectionName string, vectorSize uint64) error {
	exists, err := q.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig:  qdrant.NewVectorsConfig(&qdrant.VectorParams{Size: vectorSize, Distance: qdrant.Distance_Cosine}),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("created qdrant collection: %s", collectionName))
	return nil
}

// DeleteCollection 删除向量集合
func (q *QdrantUtil) DeleteCollection(ctx context.Context, collectionName string) error {
	err := q.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("deleted qdrant collection: %s", collectionName))
	return nil
}

// generateChunkID 基于 docUUID 和 chunkIndex 生成确定性 UUID
func generateChunkID(docUUID string, chunkIndex int) string {
	// 使用固定的命名空间和 docUUID+index 生成确定性 UUID
	namespace := uuid.NameSpaceURL
	name := fmt.Sprintf("%s-chunk-%d", docUUID, chunkIndex)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// UpsertChunks 批量插入或更新向量数据，只存储关联信息，不存储文本
func (q *QdrantUtil) UpsertChunks(ctx context.Context, collectionName string, chunkCount int, vectors [][]float32, docUUID string) error {
	points := make([]*qdrant.PointStruct, chunkCount)
	for i := 0; i < chunkCount; i++ {
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(generateChunkID(docUUID, i)),
			Vectors: qdrant.NewVectors(vectors[i]...),
			Payload: qdrant.NewValueMap(map[string]any{
				"doc_uuid":  docUUID,
				"chunk_idx": int64(i),
			}),
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("upserted %d chunks to collection %s", chunkCount, collectionName))
	return nil
}

// DeleteByDocUUID 根据文档UUID删除对应的向量数据
func (q *QdrantUtil) DeleteByDocUUID(ctx context.Context, collectionName string, docUUID string) error {
	filter := &qdrant.Filter{
		Should: []*qdrant.Condition{
			qdrant.NewMatchKeyword("doc_uuid", docUUID),
		},
	}

	wait := true
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Wait:           &wait,
		Points:         qdrant.NewPointsSelectorFilter(filter),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("deleted chunks for doc_uuid %s from collection %s", docUUID, collectionName))
	return nil
}

// Search 向量相似性搜索，返回文档UUID、分块索引和相似度分数
func (q *QdrantUtil) Search(ctx context.Context, collectionName string, queryVector []float32, topK int) ([]VectorSearchResult, error) {
	limit := uint64(topK)
	result, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	var results []VectorSearchResult
	for _, point := range result {
		if point.Payload != nil {
			docUUID := point.Payload["doc_uuid"].GetStringValue()
			chunkIdx := int(point.Payload["chunk_idx"].GetIntegerValue())

			if docUUID != "" {
				results = append(results, VectorSearchResult{
					DocUUID:  docUUID,
					ChunkIdx: chunkIdx,
					Score:    point.Score,
				})
			}
		}
	}

	return results, nil
}

// Close 关闭Qdrant客户端连接
func (q *QdrantUtil) Close() error {
	return q.client.Close()
}
