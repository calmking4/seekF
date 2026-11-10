package rag

import (
	"context"
	"fmt"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"
)

// ragInstance RAG单例实例
var ragInstance *RAG

// RAG RAG核心模块,封装向量化和分块功能
type RAG struct {
	embedding *Embedding
	splitter  *TextSplitter
}

// InitRAG 显式创建Embedding包装对象和分块器，不执行远程向量化请求。
func InitRAG() error {
	cfg := configs.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未初始化，无法初始化RAG")
	}
	ragInstance = &RAG{
		embedding: NewEmbedding(cfg.AIModelConfig.GlmApiKey, cfg.AIModelConfig.GlmEmbeddingModel, cfg.AIModelConfig.GlmBaseUrl),
		splitter:  NewTextSplitter(constants.KnowledgeChunkSize, constants.KnowledgeChunkOverlap),
	}
	zlog.Info("RAG初始化完成")
	return nil
}

// GetRAG 返回已初始化的RAG实例，不执行初始化。
func GetRAG() *RAG {
	return ragInstance
}

// GetSplitter 获取文本分块器
func (r *RAG) GetSplitter() *TextSplitter {
	return r.splitter
}

// EnsureCollection 确保向量集合存在
func (r *RAG) EnsureCollection(ctx context.Context, collectionName string) error {
	if db.GetQdrant() == nil {
		return fmt.Errorf("Qdrant客户端未初始化")
	}
	return db.GetQdrant().EnsureCollection(ctx, collectionName, 2048)
}

// HybridSearch 混合检索，结合 Dense 和 Sparse 向量
func (r *RAG) HybridSearch(ctx context.Context, collectionName string, query string, topK int) ([]db.VectorSearchResult, error) {
	if db.GetQdrant() == nil {
		return nil, fmt.Errorf("Qdrant客户端未初始化")
	}
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
	if db.GetQdrant() == nil {
		return fmt.Errorf("Qdrant客户端未初始化")
	}
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
	if db.GetQdrant() == nil {
		return fmt.Errorf("Qdrant客户端未初始化")
	}
	return db.GetQdrant().DeleteByDocUUID(ctx, collectionName, docUUID)
}
