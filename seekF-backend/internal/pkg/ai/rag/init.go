package rag

import (
	"context"
	"fmt"
	"sync"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"
)

var (
	ragInstance *RAG
	ragOnce     sync.Once
)

type RAG struct {
	embedding *Embedding
	splitter  *TextSplitter
}

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

func (r *RAG) GetEmbedding() *Embedding {
	return r.embedding
}

func (r *RAG) GetSplitter() *TextSplitter {
	return r.splitter
}

func (r *RAG) EnsureCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().EnsureCollection(ctx, collectionName, 2048)
}

func (r *RAG) DeleteCollection(ctx context.Context, collectionName string) error {
	return db.GetQdrant().DeleteCollection(ctx, collectionName)
}

func (r *RAG) Search(ctx context.Context, collectionName string, query string, topK int) ([]string, error) {
	vectors, err := r.embedding.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding failed")
	}

	return db.GetQdrant().Search(ctx, collectionName, vectors[0], topK)
}

func (r *RAG) UpsertChunks(ctx context.Context, collectionName string, chunks []string, docUUID string) error {
	vectors, err := r.embedding.EmbedTexts(ctx, chunks)
	if err != nil {
		return err
	}

	return db.GetQdrant().UpsertChunks(ctx, collectionName, chunks, vectors, docUUID)
}

func (r *RAG) DeleteChunks(ctx context.Context, collectionName string, docUUID string) error {
	return db.GetQdrant().DeleteByDocUUID(ctx, collectionName, docUUID)
}
