package admin

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/pkg/resp"
	adminservice "seekF-backend/internal/services/admin_service"
	"time"
)

// ListDiscoverReview 返回当前筛选条件下的帖子和状态计数。
func (a *AdminController) ListDiscoverReview(c *gin.Context) {
	var req struct {
		Keyword string `json:"keyword"`
		Status  *int8  `json:"status"`
		Page    int    `json:"page"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "帖子筛选参数无效", 400)
		return
	}
	identity := c.MustGet("admin_identity").(adminresp.Identity)
	result, err := a.service.ListDiscoverReview(c.Request.Context(), identity.UUID, req.Keyword, req.Status, req.Page)
	if err != nil {
		resp.Error(c, err.Error(), 400)
		return
	}
	resp.Success(c, "查询帖子成功", result)
}

// DiscoverReviewDetail 返回管理员可预览的完整帖子内容。
func (a *AdminController) DiscoverReviewDetail(c *gin.Context) {
	var req struct {
		PostUuid string `json:"post_uuid"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "帖子参数无效", 400)
		return
	}
	identity := c.MustGet("admin_identity").(adminresp.Identity)
	result, err := a.service.DiscoverReviewDetail(c.Request.Context(), identity.UUID, req.PostUuid)
	if err != nil {
		resp.Error(c, err.Error(), 400)
		return
	}
	resp.Success(c, "读取帖子成功", result)
}

// ReviewDiscoverPost 按管理员看到的版本提交审核，旧版本返回冲突。
func (a *AdminController) ReviewDiscoverPost(c *gin.Context) {
	var req adminresp.DiscoverReviewRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "审核参数无效", 400)
		return
	}
	identity := c.MustGet("admin_identity").(adminresp.Identity)
	if err := a.service.ReviewDiscoverPost(c.Request.Context(), identity.UUID, req); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, adminservice.ErrDiscoverReviewConflict) {
			status = http.StatusConflict
		}
		resp.Error(c, err.Error(), status)
		return
	}
	resp.Success(c, "审核操作成功", nil)
}

// DiscoverReviewMedia 在管理员鉴权后读取媒体并沿用视频Range响应。
func (a *AdminController) DiscoverReviewMedia(c *gin.Context) {
	identity := c.MustGet("admin_identity").(adminresp.Identity)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	stream, err := a.service.OpenDiscoverReviewMedia(ctx, identity.UUID, c.Query("post_uuid"), c.Query("url"), c.GetHeader("Range"))
	if err != nil {
		resp.Error(c, "媒体不存在或无权查看", http.StatusForbidden)
		return
	}
	defer stream.Body.Close()
	status := http.StatusOK
	headers := map[string]string{"Cache-Control": "no-store", "Accept-Ranges": "bytes", "X-Content-Type-Options": "nosniff"}
	if stream.ContentRange != "" {
		status = http.StatusPartialContent
		headers["Content-Range"] = stream.ContentRange
	}
	c.DataFromReader(status, stream.Length, stream.ContentType, stream.Body, headers)
}
