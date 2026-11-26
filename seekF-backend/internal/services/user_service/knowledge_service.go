package userservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/ai/rag"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"
)

// httpClient 带超时的 HTTP 客户端
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 10,
	},
}

// maxDownloadSize 最大下载大小 10MB
const maxDownloadSize = 10 * 1024 * 1024

// SearchResult 保存命中分块正文、文档信息及检索分数。
type SearchResult struct {
	Text        string
	Score       float32
	DocUUID     string
	FileName    string
	ChunkIndex  int
	RerankScore *float64
}

type KnowledgeService interface {
	AddDocument(ctx context.Context, userId, fileName, fileURL, fileType string) (*DocInfo, error)
	ListDocuments(ctx context.Context, userId string) ([]DocInfo, error)
	RemoveDocument(ctx context.Context, userId, uuid string) error
	Search(ctx context.Context, userId, query string, topK int) ([]SearchResult, error)
	// 获取文档正文，可按指定分块生成高亮预览。
	GetDocumentContent(ctx context.Context, userId, uuid string, chunkNumbers []int) (*KnowledgeDocumentContent, error)
}

type KnowledgeServiceImpl struct {
	knowledgeDAO      userdao.KnowledgeDAO
	knowledgeChunkDAO userdao.KnowledgeChunkDAO
}

// DocInfo 文档信息
type DocInfo struct {
	Uuid      string
	FileName  string
	FileUrl   string
	FileType  string
	ChunkCnt  int
	CreatedAt string
}

func NewKnowledgeService(knowledgeDAO userdao.KnowledgeDAO, knowledgeChunkDAO userdao.KnowledgeChunkDAO) KnowledgeService {
	return &KnowledgeServiceImpl{
		knowledgeDAO:      knowledgeDAO,
		knowledgeChunkDAO: knowledgeChunkDAO,
	}
}

// collectionName 获取用户专属的向量集合名称
func (s *KnowledgeServiceImpl) collectionName(userId string) string {
	return "knowledge_" + userId
}

// AddDocument 添加文档到知识库
func (s *KnowledgeServiceImpl) AddDocument(ctx context.Context, userId, fileName, fileURL, fileType string) (*DocInfo, error) {
	content, err := s.downloadFile(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}

	if fileType == "md" {
		content = rag.NormalizeMarkdown(content)
	}

	ragInst := rag.GetRAG()
	if ragInst == nil {
		return nil, fmt.Errorf("知识库服务未初始化")
	}
	splitter := ragInst.GetSplitter()
	chunks := splitter.SplitText(content)

	collectionName := s.collectionName(userId)
	err = ragInst.EnsureCollection(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("创建collection失败: %v", err)
	}

	docUUID := "K" + util.GetNowAndLenRandomString(11)

	// 1. 存储分块文本到 MySQL
	chunkModels := make([]models.KnowledgeChunk, len(chunks))
	for i, chunk := range chunks {
		chunkModels[i] = models.KnowledgeChunk{
			DocUUID:  docUUID,
			ChunkIdx: i,
			Content:  chunk,
		}
	}
	err = s.knowledgeChunkDAO.BatchCreate(ctx, chunkModels)
	if err != nil {
		return nil, fmt.Errorf("存储分块文本失败: %v", err)
	}

	// 2. 存储GLM语义向量及Qdrant服务端生成的BM25向量
	err = ragInst.UpsertChunks(ctx, collectionName, chunks, docUUID)
	if err != nil {
		// 回滚：删除已存储的分块文本
		_ = s.knowledgeChunkDAO.DeleteByDocUUID(ctx, docUUID)
		return nil, fmt.Errorf("存储向量失败: %v", err)
	}

	doc := &models.Knowledge{
		Uuid:       docUUID,
		UserId:     userId,
		FileName:   fileName,
		FileUrl:    fileURL,
		FileType:   fileType,
		ChunkCount: len(chunks),
	}

	err = s.knowledgeDAO.Create(ctx, doc)
	if err != nil {
		zlog.Error("保存知识库记录失败: " + err.Error())
		return nil, fmt.Errorf("保存记录失败")
	}

	return &DocInfo{
		Uuid:      docUUID,
		FileName:  fileName,
		FileUrl:   fileURL,
		FileType:  fileType,
		ChunkCnt:  len(chunks),
		CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ListDocuments 获取用户的文档列表
func (s *KnowledgeServiceImpl) ListDocuments(ctx context.Context, userId string) ([]DocInfo, error) {
	docs, err := s.knowledgeDAO.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	var result []DocInfo
	for _, doc := range docs {
		result = append(result, DocInfo{
			Uuid:      doc.Uuid,
			FileName:  doc.FileName,
			FileUrl:   doc.FileUrl,
			FileType:  doc.FileType,
			ChunkCnt:  doc.ChunkCount,
			CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

// RemoveDocument 从知识库删除文档
func (s *KnowledgeServiceImpl) RemoveDocument(ctx context.Context, userId, uuid string) error {
	doc, err := s.knowledgeDAO.FindByUuid(ctx, uuid)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("文档不存在")
	}
	if doc.UserId != userId {
		return fmt.Errorf("无权限删除")
	}

	ragInst := rag.GetRAG()
	if ragInst == nil {
		return fmt.Errorf("知识库服务未初始化")
	}
	collectionName := s.collectionName(userId)

	// 1. 删除向量数据
	err = ragInst.DeleteChunks(ctx, collectionName, uuid)
	if err != nil {
		zlog.Error("从向量数据库删除数据失败: " + err.Error())
	}

	// 2. 删除分块文本
	err = s.knowledgeChunkDAO.DeleteByDocUUID(ctx, uuid)
	if err != nil {
		zlog.Error("从MySQL删除分块数据失败: " + err.Error())
	}

	// 3. 删除文档记录
	err = s.knowledgeDAO.Delete(ctx, uuid)
	if err != nil {
		return err
	}

	return nil
}

// Search 在知识库中搜索相关内容，使用混合检索（Dense + Sparse 向量）
func (s *KnowledgeServiceImpl) Search(ctx context.Context, userId, query string, topK int) ([]SearchResult, error) {
	ragInst := rag.GetRAG()
	if ragInst == nil {
		return nil, fmt.Errorf("知识库服务未初始化")
	}
	collectionName := s.collectionName(userId)

	// 1. 混合检索，获取 docUUID 和 chunkIdx
	vectorResults, err := ragInst.HybridSearch(ctx, collectionName, query, topK)
	if err != nil {
		return nil, err
	}

	if len(vectorResults) == 0 {
		return nil, nil
	}

	// 2. 收集所有 docUUID，一次性批量查询
	docUUIDSet := make(map[string]struct{})
	for _, vr := range vectorResults {
		docUUIDSet[vr.DocUUID] = struct{}{}
	}
	docUUIDs := make([]string, 0, len(docUUIDSet))
	for uuid := range docUUIDSet {
		docUUIDs = append(docUUIDs, uuid)
	}

	docs, err := s.knowledgeDAO.FindByUserAndUUIDs(ctx, userId, docUUIDs)
	if err != nil {
		return nil, fmt.Errorf("查询候选文档失败: %w", err)
	}
	docNames := make(map[string]string, len(docs))
	docUUIDs = docUUIDs[:0]
	for _, doc := range docs {
		docNames[doc.Uuid] = doc.FileName
		docUUIDs = append(docUUIDs, doc.Uuid)
	}

	// 3. 批量读取归属已确认的文档分块。
	chunks, err := s.knowledgeChunkDAO.FindByDocUUIDs(ctx, docUUIDs)
	if err != nil {
		return nil, fmt.Errorf("查询分块文本失败: %w", err)
	}

	// 4. 构建索引 map: "docUUID:chunkIdx" -> content
	chunkTexts := make(map[string]string, len(chunks))
	for _, chunk := range chunks {
		key := fmt.Sprintf("%s:%d", chunk.DocUUID, chunk.ChunkIdx)
		chunkTexts[key] = chunk.Content
	}

	// 5. 组装结果
	var results []SearchResult
	for _, vr := range vectorResults {
		key := fmt.Sprintf("%s:%d", vr.DocUUID, vr.ChunkIdx)
		text, ok := chunkTexts[key]
		if !ok {
			zlog.Warn("未找到分块文本: " + key)
			continue
		}
		results = append(results, SearchResult{
			Text: text, Score: vr.Score,
			DocUUID: vr.DocUUID, FileName: docNames[vr.DocUUID], ChunkIndex: vr.ChunkIdx,
		})
	}

	return results, nil
}

// downloadFile 从URL下载文件内容（带超时和大小限制）
func (s *KnowledgeServiceImpl) downloadFile(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 限制下载大小，防止内存耗尽
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	// 检查是否超出限制
	if int64(len(body)) >= maxDownloadSize {
		return "", fmt.Errorf("文件大小超过限制（最大 10MB）")
	}

	content := string(body)
	return content, nil
}

// KnowledgeDocumentContent 返回原始文档或包含高亮信息的分段预览。
type KnowledgeDocumentContent struct {
	Content  string            `json:"content"`
	Segments []rag.TextSegment `json:"segments,omitempty"`
}

// GetDocumentContent 获取原始文档，并在指定分块时生成高亮预览。
func (s *KnowledgeServiceImpl) GetDocumentContent(ctx context.Context, userId, uuid string, chunkNumbers []int) (*KnowledgeDocumentContent, error) {
	if len(chunkNumbers) > constants.KnowledgeContextLimit {
		return nil, fmt.Errorf("最多可高亮%d个文档分块", constants.KnowledgeContextLimit)
	}
	doc, err := s.knowledgeDAO.FindByUuid(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("知识库文档不存在")
	}
	if doc.UserId != userId {
		return nil, fmt.Errorf("无权限访问")
	}

	content, err := s.downloadFile(ctx, doc.FileUrl)
	if err != nil {
		return nil, fmt.Errorf("获取文件内容失败: %v", err)
	}
	if len(chunkNumbers) == 0 {
		return &KnowledgeDocumentContent{Content: content}, nil
	}
	indices := make([]int, len(chunkNumbers))
	for i, number := range chunkNumbers {
		indices[i] = number - 1
	}
	chunks, err := s.knowledgeChunkDAO.FindByDocUUIDsAndChunkIdx(ctx, uuid, indices)
	if err != nil {
		return nil, fmt.Errorf("查询命中分块失败: %w", err)
	}
	if len(chunks) == 0 {
		return &KnowledgeDocumentContent{Content: content}, nil
	}
	matches := make(map[int]string, len(chunks))
	for _, chunk := range chunks {
		matches[chunk.ChunkIdx] = chunk.Content
	}
	previewText := content
	if doc.FileType == "md" {
		previewText = rag.NormalizeMarkdown(previewText)
	}
	splitter := rag.NewTextSplitter(constants.KnowledgeChunkSize, constants.KnowledgeChunkOverlap)
	if segments := splitter.HighlightChunks(previewText, matches); len(segments) > 0 {
		return &KnowledgeDocumentContent{Segments: segments}, nil
	}

	return &KnowledgeDocumentContent{Content: content}, nil
}
