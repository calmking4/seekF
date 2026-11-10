package rag

import (
	"context"
	"fmt"
	"strings"

	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
)

// PrepareBM25Text 复用ES的IK分词并保留词频，将词项连接后交给Qdrant计算BM25。
func PrepareBM25Text(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", nil
	}
	analyzeCtx, cancel := context.WithTimeout(ctx, constants.KnowledgeBM25AnalyzeTimeout)
	defer cancel()
	tokens, err := db.AnalyzeText(analyzeCtx, constants.KnowledgeBM25Analyzer, text)
	if err != nil {
		return "", fmt.Errorf("知识库中文分词失败: %w", err)
	}
	for i := range tokens {
		tokens[i] = strings.ToLower(tokens[i])
	}
	return strings.Join(tokens, " "), nil
}
