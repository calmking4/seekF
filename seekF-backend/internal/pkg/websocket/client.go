package websocket

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"seekF-backend/internal/configs"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/middlewares"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/zlog"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// MessageBack 保存等待写入网页连接的数据。
type MessageBack struct{ Message []byte }

// Client 表示一个独立网页连接，关闭时只关闭done而不关闭发送通道。
type Client struct {
	Conn      *websocket.Conn
	Uuid      string
	SendBack  chan *MessageBack
	done      chan struct{}
	closeOnce sync.Once
	lastPong  atomic.Int64
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowedOrigins := configs.GetConfig().WSConfig.AllowedOrigins
		if len(allowedOrigins) == 0 {
			return true
		}
		for _, allowed := range allowedOrigins {
			if strings.EqualFold(origin, strings.TrimSpace(allowed)) {
				return true
			}
		}
		zlog.Warn("WebSocket连接被拒绝，Origin不在白名单中: " + origin)
		return false
	},
}

// close 幂等终止读写协程，避免重复关闭与向已关闭通道写入。
func (c *Client) close() {
	c.closeOnce.Do(func() { close(c.done); c.Conn.Close() })
}

// enqueue 有界入队，慢连接关闭后由网页重连并重新查询数据库。
func (c *Client) enqueue(message *MessageBack) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case <-c.done:
		return false
	case c.SendBack <- message:
		return true
	default:
		zlog.Warn("网页连接发送队列已满，关闭慢连接，用户=" + c.Uuid)
		c.close()
		return false
	}
}

// sendJSON 将回复交给唯一写协程，不直接操作连接。
func (c *Client) sendJSON(value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		zlog.Error("序列化私聊回复失败: " + err.Error())
		return
	}
	c.enqueue(&MessageBack{Message: data})
}

// sendError 将发送失败与原始消息请求编号关联。
func (c *Client) sendError(requestId, message string) {
	c.sendJSON(map[string]string{"type": "error", "message": message, "client_msg_id": requestId})
}

// Read 读取网页命令，私信直接交给Service保存，不经过Kafka。
func (c *Client) Read() {
	defer ChatServer.SendClientToLogout(c)
	c.Conn.SetReadLimit(constants.PrivateChatReadLimit)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.lastPong.Store(time.Now().UnixNano())
		return c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		// 只解析字符串心跳，数字消息类型继续由DTO处理。
		var envelope struct {
			Type json.RawMessage `json:"type"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			c.sendError("", "私信请求格式无效")
			continue
		}
		if string(envelope.Type) == `"ping"` {
			c.lastPong.Store(time.Now().UnixNano())
			c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			c.sendJSON(map[string]string{"type": "pong"})
			continue
		}
		var request userreq.ChatMessageRequest
		if err := json.Unmarshal(data, &request); err != nil {
			c.sendError("", "私信请求格式无效")
			continue
		}
		if request.SendId != "" && request.SendId != c.Uuid {
			c.sendError(request.ClientMessageId, "不能以其他用户身份发送私信")
			continue
		}
		ChatServer.handlePrivateMessage(c, request)
	}
}

// Write 串行写入所有消息并设置超时，写入成功不代表对方已读。
func (c *Client) Write() {
	defer ChatServer.SendClientToLogout(c)
	for {
		select {
		case <-c.done:
			return
		case message := <-c.SendBack:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message.Message); err != nil {
				return
			}
		}
	}
}

// NewClientInit 注册独立网页连接并启动单读单写协程。
func NewClientInit(ctx *gin.Context, clientId string) error {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return err
	}
	middlewares.MarkWebSocketUpgrade(ctx)
	client := &Client{Conn: conn, Uuid: clientId, SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE), done: make(chan struct{})}
	client.lastPong.Store(time.Now().UnixNano())
	ChatServer.SendClientToLogin(client)
	go client.Read()
	go client.Write()
	return nil
}

// ClientLogout 主动退出登录时关闭该用户全部网页连接。
func ClientLogout(clientId string) error {
	ChatServer.RemoveClient(clientId)
	return nil
}
