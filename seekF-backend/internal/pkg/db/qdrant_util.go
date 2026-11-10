package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/constants"
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

// VectorSearchResult 混合检索结果，包含文档ID、分块索引和RRF排名分数。
type VectorSearchResult struct {
	DocUUID  string
	ChunkIdx int
	Score    float32
}

// BM25Chunk 保存索引迁移需要的分块标识及已预处理的文本。
type BM25Chunk struct {
	DocUUID  string
	ChunkIdx int
	Text     string
}

// InitQdrant 初始化Qdrant客户端
func InitQdrant(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if qdrantClient != nil {
		return qdrantClient.Ping(ctx)
	}
	cfg := configs.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未初始化，无法初始化Qdrant")
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.QdrantConfig.Host,
		Port: cfg.QdrantConfig.Port,
	})
	if err != nil {
		return err
	}

	qdrantClient = &QdrantUtil{client: client}
	if err := qdrantClient.Ping(ctx); err != nil {
		return fmt.Errorf("连接Qdrant失败: %w", err)
	}
	zlog.Info(fmt.Sprintf("已连接Qdrant: %s:%d", cfg.QdrantConfig.Host, cfg.QdrantConfig.Port))
	return nil
}

// GetQdrant 获取Qdrant客户端实例
func GetQdrant() *QdrantUtil {
	return qdrantClient
}

// Ping 通过gRPC健康检查确认Qdrant响应，不访问业务集合。
func (q *QdrantUtil) Ping(ctx context.Context) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("Qdrant客户端未初始化")
	}
	_, err := q.client.HealthCheck(ctx)
	return err
}

// EnsureCollection 确保GLM语义向量和BM25稀疏向量集合存在。
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
			constants.KnowledgeDenseVector: {
				Size:     vectorSize,
				Distance: qdrant.Distance_Cosine,
			},
		}),
		SparseVectorsConfig: bm25VectorConfig(),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("已创建Qdrant集合: %s（GLM语义向量及BM25向量）", collectionName))
	return nil
}

// generateChunkID 基于 docUUID 和 chunkIndex 生成确定性 UUID
func generateChunkID(docUUID string, chunkIndex int) string {
	// 使用固定的命名空间和 docUUID+index 生成确定性 UUID
	namespace := uuid.NameSpaceURL
	name := fmt.Sprintf("%s-chunk-%d", docUUID, chunkIndex)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// UpsertChunks 写入GLM语义向量和Qdrant服务端生成的BM25向量，等待索引写入完成。
func (q *QdrantUtil) UpsertChunks(ctx context.Context, collectionName string, denseVectors [][]float32, texts []string, docUUID string) error {
	if len(denseVectors) != len(texts) {
		return fmt.Errorf("语义向量与分块数量不一致")
	}
	points := make([]*qdrant.PointStruct, len(texts))
	for i, text := range texts {
		vectors := map[string]*qdrant.Vector{constants.KnowledgeDenseVector: qdrant.NewVector(denseVectors[i]...)}
		if text != "" {
			vectors[constants.KnowledgeBM25Vector] = qdrant.NewVectorDocument(newBM25Document(text))
		}
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(generateChunkID(docUUID, i)),
			Vectors: qdrant.NewVectorsMap(vectors),
			Payload: qdrant.NewValueMap(map[string]any{
				"doc_uuid": docUUID, "chunk_idx": int64(i),
			}),
		}
	}
	if len(points) == 0 {
		return nil
	}
	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{CollectionName: collectionName, Wait: qdrant.PtrOf(true), Points: points})
	if err != nil {
		return fmt.Errorf("写入知识库向量失败: %w", err)
	}
	zlog.Info(fmt.Sprintf("已插入%d个分块到集合%s（GLM语义向量及BM25向量）", len(points), collectionName))
	return nil
}

// ListKnowledgeCollections 列出既有用户知识库集合，供离线迁移逐集合处理。
func (q *QdrantUtil) ListKnowledgeCollections(ctx context.Context) ([]string, error) {
	names, err := q.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询知识库集合失败: %w", err)
	}
	var result []string
	for _, name := range names {
		if strings.HasPrefix(name, "knowledge_") {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result, nil
}

// MigrateKnowledgeBM25 根据MySQL分块补建BM25向量，保留已有语义向量；迁移期间应暂停知识库读写。
func (q *QdrantUtil) MigrateKnowledgeBM25(ctx context.Context, collectionName string, chunks []BM25Chunk) error {
	total, err := q.client.Count(ctx, &qdrant.CountPoints{CollectionName: collectionName, Exact: qdrant.PtrOf(true)})
	if err != nil {
		return fmt.Errorf("统计迁移前分块失败: %w", err)
	}
	if total != uint64(len(chunks)) {
		return fmt.Errorf("集合%s的向量数%d与MySQL分块数%d不一致，请先修复数据", collectionName, total, len(chunks))
	}
	if err := q.client.UpdateCollection(ctx, &qdrant.UpdateCollection{
		CollectionName: collectionName, SparseVectorsConfig: bm25VectorConfig(),
	}); err != nil {
		return fmt.Errorf("更新BM25向量配置失败: %w", err)
	}
	for start := 0; start < len(chunks); start += constants.KnowledgeBM25BatchSize {
		end := min(start+constants.KnowledgeBM25BatchSize, len(chunks))
		ids := make([]*qdrant.PointId, 0, end-start)
		points := make([]*qdrant.PointVectors, 0, end-start)
		for _, chunk := range chunks[start:end] {
			id := qdrant.NewIDUUID(generateChunkID(chunk.DocUUID, chunk.ChunkIdx))
			ids = append(ids, id)
			vector := qdrant.NewVectorSparse([]uint32{}, []float32{})
			if chunk.Text != "" {
				vector = qdrant.NewVectorDocument(newBM25Document(chunk.Text))
			}
			points = append(points, &qdrant.PointVectors{Id: id, Vectors: qdrant.NewVectorsMap(map[string]*qdrant.Vector{
				constants.KnowledgeBM25Vector: vector,
			})})
		}
		existing, err := q.client.Get(ctx, &qdrant.GetPoints{CollectionName: collectionName, Ids: ids})
		if err != nil {
			return fmt.Errorf("校验待迁移分块失败: %w", err)
		}
		if len(existing) != len(ids) {
			return fmt.Errorf("集合%s存在缺失的分块向量，请先修复数据", collectionName)
		}
		if len(points) > 0 {
			if _, err := q.client.UpdateVectors(ctx, &qdrant.UpdatePointVectors{
				CollectionName: collectionName, Wait: qdrant.PtrOf(true), Points: points,
			}); err != nil {
				return fmt.Errorf("补建BM25向量失败: %w", err)
			}
		}

	}
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

// HybridSearch 由Qdrant融合GLM语义检索和BM25关键词检索。
func (q *QdrantUtil) HybridSearch(ctx context.Context, collectionName string, denseVector []float32, text string, topK int) ([]VectorSearchResult, error) {
	if topK <= 0 {
		return nil, fmt.Errorf("检索数量必须大于零")
	}
	limit := uint64(topK)
	dense := constants.KnowledgeDenseVector
	sparse := constants.KnowledgeBM25Vector
	prefetch := []*qdrant.PrefetchQuery{{
		Query: qdrant.NewQuery(denseVector...), Using: &dense, Limit: &limit,
	}}
	// ES分词失败或没有有效词项时，仅以语义候选参与排名。
	if text != "" {
		prefetch = append(prefetch, &qdrant.PrefetchQuery{
			Query: qdrant.NewQueryDocument(newBM25Document(text)), Using: &sparse, Limit: &limit,
		})
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

// newBM25Document 构造文档与查询共用的BM25推理参数，禁用默认英文词干和停用词处理。
func newBM25Document(text string) *qdrant.Document {
	return &qdrant.Document{
		Model: constants.KnowledgeBM25Model,
		Text:  text,
		Options: qdrant.NewValueMap(map[string]any{
			"language": "none",
			"k":        constants.KnowledgeBM25K,
			"b":        constants.KnowledgeBM25B,
			"avg_len":  constants.KnowledgeBM25AvgLen,
		}),
	}
}

// bm25VectorConfig 创建启用集合级IDF统计的BM25命名向量配置。
func bm25VectorConfig() *qdrant.SparseVectorConfig {
	modifier := qdrant.Modifier_Idf
	return qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
		constants.KnowledgeBM25Vector: {Modifier: &modifier},
	})
}
