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

// SparseVector 稀疏向量结构
type SparseVector struct {
	Indices []uint32
	Values  []float32
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
	zlog.Info(fmt.Sprintf("已连接Qdrant: %s:%d", cfg.QdrantConfig.Host, cfg.QdrantConfig.Port))
	return nil
}

// GetQdrant 获取Qdrant客户端实例
func GetQdrant() *QdrantUtil {
	return qdrantClient
}

// EnsureCollection 确保向量集合存在,不存在则创建（支持Dense和Sparse向量）
func (q *QdrantUtil) EnsureCollection(ctx context.Context, collectionName string, vectorSize uint64) error {
	exists, err := q.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 创建集合，同时配置 Dense 和 Sparse 向量
	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{
			"dense": {
				Size:     vectorSize,
				Distance: qdrant.Distance_Cosine,
			},
		}),
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
			"sparse": {},
		}),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("已创建Qdrant集合: %s（支持Dense和Sparse向量）", collectionName))
	return nil
}

// DeleteCollection 删除向量集合
func (q *QdrantUtil) DeleteCollection(ctx context.Context, collectionName string) error {
	err := q.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("已删除Qdrant集合: %s", collectionName))
	return nil
}

// generateChunkID 基于 docUUID 和 chunkIndex 生成确定性 UUID
func generateChunkID(docUUID string, chunkIndex int) string {
	// 使用固定的命名空间和 docUUID+index 生成确定性 UUID
	namespace := uuid.NameSpaceURL
	name := fmt.Sprintf("%s-chunk-%d", docUUID, chunkIndex)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// UpsertChunks 批量插入或更新向量数据，同时存储 Dense 和 Sparse 向量
func (q *QdrantUtil) UpsertChunks(ctx context.Context, collectionName string, chunkCount int, denseVectors [][]float32, sparseVectors []SparseVector, docUUID string) error {
	points := make([]*qdrant.PointStruct, chunkCount)
	for i := 0; i < chunkCount; i++ {
		// 构建命名向量
		namedVectors := map[string]*qdrant.Vector{
			"dense": qdrant.NewVector(denseVectors[i]...),
		}

		// 添加 Sparse 向量（如果有）
		if i < len(sparseVectors) && len(sparseVectors[i].Indices) > 0 {
			namedVectors["sparse"] = qdrant.NewVectorSparse(
				sparseVectors[i].Indices,
				sparseVectors[i].Values,
			)
		}

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(generateChunkID(docUUID, i)),
			Vectors: qdrant.NewVectorsMap(namedVectors),
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

	zlog.Info(fmt.Sprintf("已插入 %d 个分块到集合 %s（包含Dense和Sparse向量）", chunkCount, collectionName))
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

	zlog.Info(fmt.Sprintf("已删除文档 %s 的向量数据，集合: %s", docUUID, collectionName))
	return nil
}

// Search 向量相似性搜索（仅Dense向量，向后兼容）
func (q *QdrantUtil) Search(ctx context.Context, collectionName string, queryVector []float32, topK int) ([]VectorSearchResult, error) {
	limit := uint64(topK)
	dense := "dense"
	result, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Using:          &dense,
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	return parseSearchResults(result)
}

// HybridSearch 混合检索，结合 Dense 和 Sparse 向量（RRF 融合）
func (q *QdrantUtil) HybridSearch(ctx context.Context, collectionName string, denseVector []float32, sparseVector SparseVector, topK int) ([]VectorSearchResult, error) {
	limit := uint64(topK)
	dense := "dense"
	sparse := "sparse"

	// 构建 Prefetch 查询：同时使用 Dense 和 Sparse 向量
	prefetch := []*qdrant.PrefetchQuery{
		{
			Query: qdrant.NewQuery(denseVector...),
			Using: &dense,
			Limit: &limit,
		},
		{
			Query: qdrant.NewQuerySparse(sparseVector.Indices, sparseVector.Values),
			Using: &sparse,
			Limit: &limit,
		},
	}

	// 使用 RRF（Reciprocal Rank Fusion）融合结果
	result, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Prefetch:       prefetch,
		Query:          qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	return parseSearchResults(result)
}

// parseSearchResults 解析搜索结果
func parseSearchResults(result []*qdrant.ScoredPoint) ([]VectorSearchResult, error) {
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
