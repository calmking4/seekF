package user

import (
	"net/http"

	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/websocket"
	"seekF-backend/internal/pkg/zlog"

	"github.com/gin-gonic/gin"
)

// WsController WebSocket控制器
type WsController struct {
}

// NewWsController 创建WebSocket控制器实例
func NewWsController() *WsController {
	return &WsController{}
}

// WsLogin WebSocket登录
func (c *WsController) WsLogin(ctx *gin.Context) {
	clientId := ctx.Query("client_id")
	if clientId == "" {
		zlog.Error("clientId获取失败")
		resp.Error(ctx, "clientId获取失败", http.StatusBadRequest)
		return
	}
	// 连接身份必须与登录态一致，不能通过查询参数冒充其他私聊用户。
	if clientId != ctx.GetString("Uuid") {
		resp.Error(ctx, "不能使用其他用户身份连接私聊", http.StatusForbidden)
		return
	}
	if err := websocket.NewClientInit(ctx, clientId); err != nil {
		zlog.Error("WebSocket连接失败: " + err.Error())
		resp.Error(ctx, "WebSocket连接失败", http.StatusInternalServerError)
		return
	}
}

// WsLogout WebSocket登出
func (c *WsController) WsLogout(ctx *gin.Context) {
	// 从上下文中获取用户uuid
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "未登录或 token 无效", http.StatusUnauthorized)
		return
	}

	if err := websocket.ClientLogout(userUuid.(string)); err != nil {
		zlog.Error("WebSocket登出失败: " + err.Error())
		resp.Error(ctx, "WebSocket登出失败", http.StatusInternalServerError)
		return
	}
	resp.Success(ctx, "退出成功", nil)
}
