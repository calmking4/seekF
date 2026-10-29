package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"
)

// ChatServer 管理当前进程的网页连接。
var ChatServer *Server

// Server 只管理网页连接与直接推送，消息事务交给Service。
type Server struct {
	Clients        map[string]map[*Client]struct{}
	mutex          sync.Mutex
	sessionService userservice.SessionService
}

// NewServer 创建不依赖Kafka和ES的私聊连接服务。
func NewServer(sessionService userservice.SessionService) *Server {
	return &Server{Clients: make(map[string]map[*Client]struct{}), sessionService: sessionService}
}

// OnlineCount 返回在线用户数，同一用户多个标签页只计一次。
func (s *Server) OnlineCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.Clients)
}

// Start 定期清理超时连接，消息直接在接收连接上处理。
func (s *Server) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mutex.Lock()
		var expired []*Client
		for _, connections := range s.Clients {
			for client := range connections {
				if time.Since(time.Unix(0, client.lastPong.Load())) > 60*time.Second {
					expired = append(expired, client)
				}
			}
		}
		s.mutex.Unlock()
		for _, client := range expired {
			s.SendClientToLogout(client)
		}
	}
}

// SendClientToLogin 同步注册连接，避免第一条请求早于注册。
func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	if s.Clients[client.Uuid] == nil {
		s.Clients[client.Uuid] = make(map[*Client]struct{})
	}
	s.Clients[client.Uuid][client] = struct{}{}
	s.mutex.Unlock()
	client.enqueue(&MessageBack{Message: []byte(`{"type":"connected"}`)})
}

// SendClientToLogout 仅移除指定连接，旧标签页退出不影响新连接。
func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	connections := s.Clients[client.Uuid]
	delete(connections, client)
	if len(connections) == 0 {
		delete(s.Clients, client.Uuid)
	}
	s.mutex.Unlock()
	client.close()
}

// clientsFor 返回连接快照，发送时不持有全局锁。
func (s *Server) clientsFor(userId string) []*Client {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	clients := make([]*Client, 0, len(s.Clients[userId]))
	for client := range s.Clients[userId] {
		clients = append(clients, client)
	}
	return clients
}

// RemoveClient 关闭指定用户的所有网页连接。
func (s *Server) RemoveClient(userId string) {
	for _, client := range s.clientsFor(userId) {
		s.SendClientToLogout(client)
	}
}

// PushNotification 向用户的在线网页推送，通道满时关闭慢连接以触发重连补拉。
func (s *Server) PushNotification(userId string, notification interface{}) {
	msg, err := json.Marshal(notification)
	if err != nil {
		zlog.Error("序列化网页通知失败: " + err.Error())
		return
	}
	for _, client := range s.clientsFor(userId) {
		client.enqueue(&MessageBack{Message: msg})
	}
}

// handlePrivateMessage 先完成MySQL事务，再向发送者确认并直接推送接收者。
func (s *Server) handlePrivateMessage(client *Client, request userreq.ChatMessageRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := s.sessionService.SendPrivateMessage(ctx, client.Uuid, request)
	if err != nil {
		zlog.Warn(fmt.Sprintf("保存私信失败，用户=%s，请求=%s: %s", client.Uuid, request.ClientMessageId, err.Error()))
		client.sendError(request.ClientMessageId, err.Error())
		return
	}
	message := result.Message
	// 重试不能重新展示发送者已删除的正文，也不重复投递给接收方。
	if message.SendDeletedAt != nil {
		client.sendJSON(map[string]interface{}{"type": "message_deleted", "uuid": message.Uuid, "user_id": client.Uuid, "client_msg_id": request.ClientMessageId})
		return
	}
	response := userresp.GetMessageListRespond{
		ClientMessageId: request.ClientMessageId, Id: message.Id, Uuid: message.Uuid,
		SessionId: message.SessionId, SendId: message.SendId, SendName: message.SendName,
		SendAvatar: message.SendAvatar, ReceiveId: message.ReceiveId, Type: message.Type,
		Content: message.Content, Url: message.Url, FileType: message.FileType,
		FileName: message.FileName, FileSize: message.FileSize, CreatedAt: message.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if message.Type == constants.MessageRecalled {
		client.sendJSON(map[string]interface{}{"type": "message_recalled", "uuid": message.Uuid, "send_id": message.SendId, "receive_id": message.ReceiveId, "client_msg_id": request.ClientMessageId})
		return
	}
	if result.Duplicate {
		client.sendJSON(response)
		return
	}
	s.PushNotification(message.SendId, response)
	response.SessionId = result.ReceiverSessionId
	response.ClientMessageId = ""
	s.PushNotification(message.ReceiveId, response)
}
