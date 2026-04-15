package user

import (
	"net/http"
	"seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type KnowledgeController struct {
	knowledgeService userservice.KnowledgeService
}

func NewKnowledgeController(knowledgeService userservice.KnowledgeService) *KnowledgeController {
	return &KnowledgeController{
		knowledgeService: knowledgeService,
	}
}

func (c *KnowledgeController) AddDocument(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.AddKnowledgeRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if req.FileName == "" || req.FileUrl == "" || req.FileType == "" {
		resp.Error(ctx, "缺少必要参数", http.StatusBadRequest)
		return
	}

	docInfo, err := c.knowledgeService.AddDocument(ctx.Request.Context(), userId, req.FileName, req.FileUrl, req.FileType)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "添加成功", userresp.AddKnowledgeRespond{
		Uuid:     docInfo.Uuid,
		ChunkCnt: docInfo.ChunkCnt,
	})
}

func (c *KnowledgeController) ListDocuments(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	list, err := c.knowledgeService.ListDocuments(ctx.Request.Context(), userId)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.KnowledgeDocItem
	for _, doc := range list {
		items = append(items, userresp.KnowledgeDocItem{
			Uuid:      doc.Uuid,
			FileName:  doc.FileName,
			FileUrl:   doc.FileUrl,
			FileType:  doc.FileType,
			ChunkCnt:  doc.ChunkCnt,
			CreatedAt: doc.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListKnowledgeRespond{
		List: items,
	})
}

func (c *KnowledgeController) RemoveDocument(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.RemoveKnowledgeRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if req.Uuid == "" {
		resp.Error(ctx, "uuid不能为空", http.StatusBadRequest)
		return
	}

	err := c.knowledgeService.RemoveDocument(ctx.Request.Context(), userId, req.Uuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除成功", nil)
}

func (c *KnowledgeController) GetDocumentContent(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.GetKnowledgeContentRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	content, err := c.knowledgeService.GetDocumentContent(ctx.Request.Context(), userId, req.Uuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", map[string]string{
		"content": content,
	})
}
