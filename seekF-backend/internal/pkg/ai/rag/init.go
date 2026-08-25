package rag

import (
	"context"
	"fmt"
	"sync"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"
)

// ragInstance RAG单例实例
var (
	ragInstance *RAG
	ragOnce     sync.Once
)

// RAG RAG核心模块,封装向量化和分块功能
type RAG struct {
	embedding      *Embedding
	splitter       *TextSplitter
	sparseVector   *SparseVectorizer
}

// GetRAG 获取RAG单例实例(线程安全)
func GetRAG() *RAG {
	ragOnce.Do(func() {
		cfg := configs.GetConfig()

		zlog.Info("正在初始化RAG...")

		emb := NewEmbedding(
			cfg.AIModelConfig.GlmApiKey,
			cfg.AIModelConfig.GlmEmbeddingModel,
			cfg.AIModelConfig.GlmBaseUrl,
		)

		spl := NewTextSplitter(500, 50)
		sparse := NewSparseVectorizer()

		ragInstance = &RAG{
			embedding:    emb,
			splitter:     spl,
			sparseVector: sparse,
		}

		zlog.Info("RAG初始化完成")
	})
	return ragInstance
}

// GetEmbedding 获取向量化模块
func (r *RAG) GetEmbedding() *Embedding {
	return r.embedding
}

// GetSplitter 获取文本分块器
func (r *RAG) GetSplitter() *TextSplitter {
	return r.splitter
}

// GetSparseVectorizer 获取稀疏向量化器
func (r *RAG) GetSparseVectorizer() *SparseVectorizer {
	return r.sparseVector
}

// EnsureCollection 确保向量集合存在
func (r *RAG) EnsureCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().EnsureCollection(ctx, collectionName, 2048)
}

// DeleteCollection 删除向量集合
func (r *RAG) DeleteCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().DeleteCollection(ctx, collectionName)
}

// Search 语义搜索（仅Dense向量，向后兼容）
func (r *RAG) Search(ctx context.Context, collectionName string, query string, topK int) ([]db.VectorSearchResult, error) {
	vectors, err := r.embedding.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("向量化失败")
	}

	return db.GetQdrant().Search(ctx, collectionName, vectors[0], topK)
}

// HybridSearch 混合检索，结合 Dense 和 Sparse 向量
func (r *RAG) HybridSearch(ctx context.Context, collectionName string, query string, topK int) ([]db.VectorSearchResult, error) {
	// 1. 生成 Dense 向量
	denseVectors, err := r.embedding.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("Dense向量化失败: %w", err)
	}
	if len(denseVectors) == 0 {
		return nil, fmt.Errorf("Dense向量化失败")
	}

	// 2. 生成 Sparse 向量
	sparseVector := r.sparseVector.Vectorize(query)

	// 3. 执行混合检索
	return db.GetQdrant().HybridSearch(ctx, collectionName, denseVectors[0], sparseVector, topK)
}

// BuildSparseVocabulary 从文档集合构建稀疏向量词汇表
func (r *RAG) BuildSparseVocabulary(documents []string) {
	r.sparseVector.BuildVocabulary(documents)
	zlog.Info(fmt.Sprintf("已构建稀疏向量词汇表，词汇量: %d", r.sparseVector.GetVocabularySize()))
}

// UpsertChunks 批量插入向量数据，同时生成 Dense 和 Sparse 向量
func (r *RAG) UpsertChunks(ctx context.Context, collectionName string, chunks []string, docUUID string) error {
	// 1. 生成 Dense 向量
	denseVectors, err := r.embedding.EmbedTexts(ctx, chunks)
	if err != nil {
		return fmt.Errorf("Dense向量化失败: %w", err)
	}

	// 2. 生成 Sparse 向量
	sparseVectors := make([]db.SparseVector, len(chunks))
	for i, chunk := range chunks {
		sv := r.sparseVector.Vectorize(chunk)
		sparseVectors[i] = db.SparseVector{
			Indices: sv.Indices,
			Values:  sv.Values,
		}
	}

	// 3. 存储向量到 Qdrant
	return db.GetQdrant().UpsertChunks(ctx, collectionName, len(chunks), denseVectors, sparseVectors, docUUID)
}

// DeleteChunks 删除指定文档的向量数据
func (r *RAG) DeleteChunks(ctx context.Context, collectionName string, docUUID string) error {
	return db.GetQdrant().DeleteByDocUUID(ctx, collectionName, docUUID)
}
