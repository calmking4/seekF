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

		zlog.Info("正在初始化RAG...")

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

		zlog.Info("RAG初始化完成")
	})
	return ragInstance
}

// GetSplitter 获取文本分块器
func (r *RAG) GetSplitter() *TextSplitter {
	return r.splitter
}

// EnsureCollection 确保向量集合存在
func (r *RAG) EnsureCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().EnsureCollection(ctx, collectionName, 2048)
}

// HybridSearch 混合检索，结合 Dense 和 Sparse 向量
func (r *RAG) HybridSearch(ctx context.Context, collectionName string, query string, topK int) ([]db.VectorSearchResult, error) {
	// 1. 生成 Dense 向量
	denseVectors, err := r.embedding.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("语义向量生成失败: %w", err)
	}
	if len(denseVectors) == 0 {
		return nil, fmt.Errorf("语义向量生成失败")
	}

	// 文档和查询使用相同预处理，由Qdrant服务端计算BM25稀疏向量。
	text, err := PrepareBM25Text(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		zlog.Warn("知识库分词失败，降级为语义检索，集合=" + collectionName + ": " + err.Error())
		text = ""
	}
	return db.GetQdrant().HybridSearch(ctx, collectionName, denseVectors[0], text, topK)
}

// UpsertChunks 生成GLM语义向量，并将预处理文本交给Qdrant计算BM25向量。
func (r *RAG) UpsertChunks(ctx context.Context, collectionName string, chunks []string, docUUID string) error {
	// 先完成全部分词再调用GLM，ES不可用时不生成不匹配的稀疏索引。
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		text, err := PrepareBM25Text(ctx, chunk)
		if err != nil {
			return fmt.Errorf("分块%d分词失败: %w", i, err)
		}
		texts[i] = text
	}
	denseVectors, err := r.embedding.EmbedTexts(ctx, chunks)
	if err != nil {
		return fmt.Errorf("语义向量生成失败: %w", err)
	}
	return db.GetQdrant().UpsertChunks(ctx, collectionName, denseVectors, texts, docUUID)
}

// DeleteChunks 删除指定文档的向量数据
func (r *RAG) DeleteChunks(ctx context.Context, collectionName string, docUUID string) error {
	return db.GetQdrant().DeleteByDocUUID(ctx, collectionName, docUUID)
}
