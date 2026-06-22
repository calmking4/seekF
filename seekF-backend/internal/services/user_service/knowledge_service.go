package userservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/ai/rag"
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

type KnowledgeService interface {
	AddDocument(ctx context.Context, userId, fileName, fileURL, fileType string) (*DocInfo, error)
	ListDocuments(ctx context.Context, userId string) ([]DocInfo, error)
	RemoveDocument(ctx context.Context, userId, uuid string) error
	Search(ctx context.Context, userId, query string, topK int) ([]string, error)
	GetDocumentContent(ctx context.Context, userId, uuid string) (string, error)
}
type KnowledgeServiceImpl struct {
	knowledgeDAO userdao.KnowledgeDAO
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

func NewKnowledgeService(knowledgeDAO userdao.KnowledgeDAO) KnowledgeService {
	return &KnowledgeServiceImpl{
		knowledgeDAO: knowledgeDAO,
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
		content = removeMarkdownMarkers(content)
	}

	ragInst := rag.GetRAG()
	splitter := ragInst.GetSplitter()
	chunks := splitter.SplitText(content)

	collectionName := s.collectionName(userId)
	err = ragInst.EnsureCollection(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("创建collection失败: %v", err)
	}

	docUUID := "K" + util.GetNowAndLenRandomString(11)
	err = ragInst.UpsertChunks(ctx, collectionName, chunks, docUUID)
	if err != nil {
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

	err = s.knowledgeDAO.Create(doc)
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
	docs, err := s.knowledgeDAO.FindByUserId(userId)
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
	doc, err := s.knowledgeDAO.FindByUuid(uuid)
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
	collectionName := s.collectionName(userId)
	err = ragInst.DeleteChunks(ctx, collectionName, uuid)
	if err != nil {
		zlog.Error("从向量数据库删除数据失败: " + err.Error())
	}

	err = s.knowledgeDAO.Delete(uuid)
	if err != nil {
		return err
	}

	return nil
}

// Search 在知识库中搜索相关内容
func (s *KnowledgeServiceImpl) Search(ctx context.Context, userId, query string, topK int) ([]string, error) {
	ragInst := rag.GetRAG()
	collectionName := s.collectionName(userId)
	return ragInst.Search(ctx, collectionName, query, topK)
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

// removeMarkdownMarkers 移除Markdown标记符号(如标题(#)、代码块(```)、分隔线(---))
func removeMarkdownMarkers(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "```") ||
			strings.HasPrefix(line, "---") {
			continue
		}
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// GetDocumentContent 获取文档原始内容
func (s *KnowledgeServiceImpl) GetDocumentContent(ctx context.Context, userId, uuid string) (string, error) {
	doc, err := s.knowledgeDAO.FindByUuid(uuid)
	if err != nil {
		return "", err
	}
	if doc.UserId != userId {
		return "", fmt.Errorf("无权限访问")
	}

	content, err := s.downloadFile(ctx, doc.FileUrl)
	if err != nil {
		return "", fmt.Errorf("获取文件内容失败: %v", err)
	}

	return content, nil
}
