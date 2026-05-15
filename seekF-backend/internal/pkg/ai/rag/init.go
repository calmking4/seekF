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
	embedding *Embedding
	splitter  *TextSplitter
}

// GetRAG 获取RAG单例实例(线程安全)
func GetRAG() *RAG {
	ragOnce.Do(func() {
		cfg := configs.GetConfig()

		zlog.Info("initializing RAG...")

		emb := NewEmbedding(
			cfg.AIModelConfig.GlmApiKey,
			cfg.AIModelConfig.GlmEmbeddingModel,
			cfg.AIModelConfig.GlmBaseUrl,
		)

		spl := NewTextSplitter(500, 50)

		ragInstance = &RAG{
			embedding: emb,
			splitter:  spl,
		}

		zlog.Info("RAG initialization completed")
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

// EnsureCollection 确保向量集合存在
func (r *RAG) EnsureCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().EnsureCollection(ctx, collectionName, 2048)
}

// DeleteCollection 删除向量集合
func (r *RAG) DeleteCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().DeleteCollection(ctx, collectionName)
}

// Search 语义搜索
func (r *RAG) Search(ctx context.Context, collectionName string, query string, topK int) ([]string, error) {
	vectors, err := r.embedding.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("向量化失败")
	}

	return db.GetQdrant().Search(ctx, collectionName, vectors[0], topK)
}

// UpsertChunks 批量插入向量数据
func (r *RAG) UpsertChunks(ctx context.Context, collectionName string, chunks []string, docUUID string) error {
	vectors, err := r.embedding.EmbedTexts(ctx, chunks)
	if err != nil {
		return err
	}

	return db.GetQdrant().UpsertChunks(ctx, collectionName, chunks, vectors, docUUID)
}

// DeleteChunks 删除指定文档的向量数据
func (r *RAG) DeleteChunks(ctx context.Context, collectionName string, docUUID string) error {
	return db.GetQdrant().DeleteByDocUUID(ctx, collectionName, docUUID)
}
