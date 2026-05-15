package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/zlog"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MessageBack struct {
	Message []byte
	Uuid    string
}

type Client struct {
	Conn         *websocket.Conn
	Uuid         string
	SendBack     chan *MessageBack // 给前端
	LastPongTime time.Time        // 最后一次收到 pong 的时间
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	// 检查连接的Origin头
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 读取websocket消息并发送给kafka
func (c *Client) Read() {
	zlog.Info("ws read goroutine start")
	defer func() {
		ChatServer.SendClientToLogout(c)
	}()

	// 设置读取超时为 30 秒
	c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.LastPongTime = time.Now()
		c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	for {
		// 阻塞有一定隐患，因为下面要处理缓冲的逻辑，但是可以先不做优化，问题不大
		_, jsonMessage, err := c.Conn.ReadMessage() // 阻塞状态
		if err != nil {
			zlog.Error(err.Error())
			return // 直接断开websocket
		}

		// 处理心跳 ping 消息
		var wsMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(jsonMessage, &wsMsg); err == nil && wsMsg.Type == "ping" {
			// 回复 pong
			pongMsg, _ := json.Marshal(map[string]string{"type": "pong"})
			c.SendBack <- &MessageBack{Message: pongMsg, Uuid: ""}
			c.LastPongTime = time.Now()
			c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			continue
		}

		var message = userreq.ChatMessageRequest{}
		if err := json.Unmarshal(jsonMessage, &message); err != nil {
			zlog.Error(err.Error())
		}
		// zlog.Info("接受到消息为: " + string(jsonMessage))

		// 发送消息到Kafka
		if err := SendMessageToKafka(jsonMessage); err != nil {
			zlog.Error(err.Error())
		}
		// zlog.Info("已发送消息到Kafka: " + string(jsonMessage))
	}
}

// 从send通道读取消息发送给websocket
func (c *Client) Write() {
	zlog.Info("ws write goroutine start")
	for messageBack := range c.SendBack { // 阻塞状态
		// 通过 WebSocket 发送消息
		err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
		if err != nil {
			zlog.Error(err.Error())
			return // 直接断开websocket
		}
		// 说明顺利发送，修改状态为已发送
		ChatServer.UpdateMessageStatus(messageBack.Uuid)
	}
}

// NewClientInit 当接受到前端有登录消息时，会调用该函数
func NewClientInit(c *gin.Context, clientId string) error {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	client := &Client{
		Conn:         conn,
		Uuid:         clientId,
		SendBack:     make(chan *MessageBack, constants.CHANNEL_SIZE),
		LastPongTime: time.Now(),
	}

	ChatServer.SendClientToLogin(client)

	go client.Read()
	go client.Write()
	zlog.Info("ws连接成功")
	return nil
}

// ClientLogout 当接受到前端有登出消息时，会调用该函数
func ClientLogout(clientId string) error {
	client := ChatServer.Clients[clientId]
	if client != nil {
		ChatServer.SendClientToLogout(client)
		if err := client.Conn.Close(); err != nil {
			zlog.Error(err.Error())
			return err
		}
		close(client.SendBack)
	}
	return nil
}
