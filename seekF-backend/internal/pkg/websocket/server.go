package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/enum/message_enum/message_status_enum"
	"seekF-backend/internal/pkg/enum/message_enum/message_type_enum"
	"seekF-backend/internal/pkg/kafka"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	segmentioKafka "github.com/segmentio/kafka-go"

	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	userservice "seekF-backend/internal/services/user_service"
)

// ChatServer 是全局的WebSocket服务器实例
var ChatServer *Server

// OnlineCount 在锁保护下返回当前在线用户数。
func (s *Server) OnlineCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.Clients)
}

// Server 管理所有WebSocket客户端
type Server struct {
	Clients        map[string]*Client
	mutex          *sync.Mutex
	Login          chan *Client // 登录通道
	Logout         chan *Client // 退出登录通道
	sessionService userservice.SessionService
	messageDAO     userdao.MessageDAO
	sessionDAO     userdao.SessionDAO
}

// NewServer 创建新的WebSocket服务器
func NewServer(sessionService userservice.SessionService, messageDAO userdao.MessageDAO, sessionDAO userdao.SessionDAO) *Server {
	return &Server{
		Clients:        make(map[string]*Client),
		mutex:          &sync.Mutex{},
		Login:          make(chan *Client, constants.CHANNEL_SIZE),
		Logout:         make(chan *Client, constants.CHANNEL_SIZE),
		sessionService: sessionService,
		messageDAO:     messageDAO,
		sessionDAO:     sessionDAO,
	}
}

// Start 启动WebSocket服务器
func (s *Server) Start() {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error(fmt.Sprintf("server panic: %v", r))
		}
	}()

	// 启动Kafka消息读取协程
	go s.readKafkaMessages()

	// 启动心跳检测协程，清理超时连接
	go s.heartbeatCheck()

	// 处理登录和登出
	for {
		select {
		case client := <-s.Login:
			s.handleLogin(client)
		case client := <-s.Logout:
			s.handleLogout(client)
		}
	}
}

// heartbeatCheck 定期检查客户端心跳，清理超时连接
func (s *Server) heartbeatCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		for uuid, client := range s.Clients {
			if time.Since(client.LastPongTime) > 60*time.Second {
				zlog.Info(fmt.Sprintf("用户 %s 心跳超时，断开连接", uuid))
				client.Conn.Close()
				delete(s.Clients, uuid)
			}
		}
		s.mutex.Unlock()
	}
}

// readKafkaMessages 读取Kafka消息
func (s *Server) readKafkaMessages() {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error(fmt.Sprintf("读取聊天消息异常: %v", r))
		}
	}()

	for {
		kafkaMessage, err := kafka.KafkaServiceInstance.ChatReader.ReadMessage(kafka.Ctx)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		// zlog.Info(fmt.Sprintf("收到Kafka消息: topic=%s, partition=%d, offset=%d",
		// 	kafkaMessage.Topic, kafkaMessage.Partition, kafkaMessage.Offset))

		data := kafkaMessage.Value
		var chatMessageReq userreq.ChatMessageRequest
		if err := json.Unmarshal(data, &chatMessageReq); err != nil {
			zlog.Error("解析消息失败: " + err.Error())
			continue
		}
		// zlog.Info("原消息反序列化后: " + fmt.Sprintf("%+v", chatMessageReq))

		// 同时校验队列中的旧消息，群聊和通话不再持久化或投递。
		if err := s.validatePrivateMessage(&chatMessageReq); err != nil {
			s.sendErrorToClient(chatMessageReq.SendId, err.Error())
			continue
		}
		// 根据消息类型处理
		switch chatMessageReq.Type {
		case message_type_enum.Text:
			s.handleTextMessage(&chatMessageReq)
		case message_type_enum.File:
			s.handleFileMessage(&chatMessageReq)
		}
	}
}

// validatePrivateMessage 检查消息类型、会话归属及私聊目标，拒绝群聊和通话。
func (s *Server) validatePrivateMessage(message *userreq.ChatMessageRequest) error {
	if !strings.HasPrefix(message.SendId, "U") || !strings.HasPrefix(message.ReceiveId, "U") || message.SendId == message.ReceiveId {
		return fmt.Errorf("只能向其他用户发送私聊消息")
	}
	if message.Type != message_type_enum.Text && message.Type != message_type_enum.File {
		return fmt.Errorf("私聊仅支持文字和文件消息，通话功能已下线")
	}
	session, err := s.sessionDAO.GetSessionByUuid(message.SessionId)
	if err != nil {
		return fmt.Errorf("查询私聊会话失败: %w", err)
	}
	if session.SendId != message.SendId || session.ReceiveId != message.ReceiveId {
		return fmt.Errorf("无权向该私聊会话发送消息")
	}
	allowed, err := s.sessionService.CheckOpenSessionAllowed(message.SendId, message.ReceiveId)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("当前无法向该用户发送私聊消息")
	}
	return nil
}

// handleTextMessage 处理文本消息
func (s *Server) handleTextMessage(chatMessageReq *userreq.ChatMessageRequest) {
	// 创建消息记录
	message := &models.Message{
		Uuid:       fmt.Sprintf("M%s", util.GetNowAndLenRandomString(11)),
		SessionId:  chatMessageReq.SessionId,
		Type:       chatMessageReq.Type,
		Content:    chatMessageReq.Content,
		Url:        "",
		SendId:     chatMessageReq.SendId,
		SendName:   chatMessageReq.SendName,
		SendAvatar: normalizePath(chatMessageReq.SendAvatar),
		ReceiveId:  chatMessageReq.ReceiveId,
		FileSize:   "0B",
		FileType:   "",
		FileName:   "",
		Status:     message_status_enum.Unsent,
		CreatedAt:  time.Now(),
	}

	// 保存消息到数据库
	if err := s.sessionService.SavePrivateMessage(context.Background(), message); err != nil {
		zlog.Error("保存消息失败: " + err.Error())
		s.sendErrorToClient(chatMessageReq.SendId, err.Error())
		return
	}

	// 异步索引消息到ES（搜索用）
	go func() {
		if db.ESClient != nil {
			esDAO := userdao.NewESMessageDAO()
			if err := esDAO.IndexMessage(message); err != nil {
				zlog.Error("索引文本消息到ES失败: " + err.Error())
			}
		}
	}()

	// 构建响应消息
	messageRsp := userresp.GetMessageListRespond{
		Uuid:       message.Uuid,
		SessionId:  message.SessionId,
		SendId:     message.SendId,
		SendName:   message.SendName,
		SendAvatar: chatMessageReq.SendAvatar,
		ReceiveId:  message.ReceiveId,
		Type:       message.Type,
		Content:    message.Content,
		Url:        message.Url,
		FileSize:   message.FileSize,
		FileName:   message.FileName,
		FileType:   message.FileType,
		CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	jsonMessage, err := json.Marshal(messageRsp)
	if err != nil {
		zlog.Error("序列化消息失败: " + err.Error())
		return
	}

	messageBack := &MessageBack{
		Message: jsonMessage,
		Uuid:    message.Uuid,
	}

	// 私聊消息向双方投递。
	s.sendToUser(message, messageBack, &messageRsp)
}

// handleFileMessage 处理文件消息
func (s *Server) handleFileMessage(chatMessageReq *userreq.ChatMessageRequest) {
	// 创建消息记录
	message := &models.Message{
		Uuid:       fmt.Sprintf("M%s", util.GetNowAndLenRandomString(11)),
		SessionId:  chatMessageReq.SessionId,
		Type:       chatMessageReq.Type,
		Content:    "",
		Url:        chatMessageReq.Url,
		SendId:     chatMessageReq.SendId,
		SendName:   chatMessageReq.SendName,
		SendAvatar: normalizePath(chatMessageReq.SendAvatar),
		ReceiveId:  chatMessageReq.ReceiveId,
		FileSize:   chatMessageReq.FileSize,
		FileType:   chatMessageReq.FileType,
		FileName:   chatMessageReq.FileName,
		Status:     message_status_enum.Unsent,
		CreatedAt:  time.Now(),
	}

	// 保存消息到数据库
	if err := s.sessionService.SavePrivateMessage(context.Background(), message); err != nil {
		zlog.Error("保存消息失败: " + err.Error())
		s.sendErrorToClient(chatMessageReq.SendId, err.Error())
		return
	}

	// 异步索引消息到ES（搜索用）
	go func() {
		if db.ESClient != nil {
			esDAO := userdao.NewESMessageDAO()
			if err := esDAO.IndexMessage(message); err != nil {
				zlog.Error("索引文件消息到ES失败: " + err.Error())
			}
		}
	}()

	// 构建响应消息
	messageRsp := userresp.GetMessageListRespond{
		Uuid:       message.Uuid,
		SessionId:  message.SessionId,
		SendId:     message.SendId,
		SendName:   message.SendName,
		SendAvatar: chatMessageReq.SendAvatar,
		ReceiveId:  message.ReceiveId,
		Type:       message.Type,
		Content:    message.Content,
		Url:        message.Url,
		FileSize:   message.FileSize,
		FileName:   message.FileName,
		FileType:   message.FileType,
		CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	jsonMessage, err := json.Marshal(messageRsp)
	if err != nil {
		zlog.Error("序列化消息失败: " + err.Error())
		return
	}

	messageBack := &MessageBack{
		Message: jsonMessage,
		Uuid:    message.Uuid,
	}

	// 私聊消息向双方投递。
	s.sendToUser(message, messageBack, &messageRsp)
}

// sendToUser 发送消息给单个用户
func (s *Server) sendToUser(message *models.Message, messageBack *MessageBack, messageRsp *userresp.GetMessageListRespond) {
	if _, err := s.sessionService.CheckOpenSessionAllowed(message.SendId, message.ReceiveId); err != nil {
		s.sendErrorToClient(message.SendId, err.Error())
		return
	}
	// 为接收者创建会话（如果不存在）
	sessionId, err := s.sessionService.OpenSession(message.ReceiveId, message.SendId)
	if err != nil {
		zlog.Error("为接收者创建会话失败: " + err.Error())
		s.sendErrorToClient(message.SendId, err.Error())
		return
	} else {
		// 更新接收者的会话最后消息
		lastMessage := message.Content
		if message.Type == message_type_enum.File {
			lastMessage = "[文件]" + message.FileName
		}
		s.updateSessionLastMessage(sessionId, lastMessage)
	}

	// 更新发送者的会话最后消息
	s.updateSessionLastMessage(message.SessionId, message.Content)

	s.mutex.Lock()
	// 发送给接收者
	if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
		receiveClient.SendBack <- messageBack
	}
	// 发送给发送者（回显）
	if sendClient, ok := s.Clients[message.SendId]; ok {
		sendClient.SendBack <- messageBack
	}
	s.mutex.Unlock()

	// 更新Redis缓存
	s.updateUserMessageListCache(message.SendId, message.ReceiveId, messageRsp)
}

// updateUserMessageListCache 更新用户消息列表缓存
func (s *Server) updateUserMessageListCache(sendId, receiveId string, messageRsp *userresp.GetMessageListRespond) {
	rspString, err := myredis.GetKeyNilIsErr("message_list_" + sendId + "_" + receiveId)
	if err == nil {
		var rsp []userresp.GetMessageListRespond
		if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
			zlog.Error("解析缓存消息失败: " + err.Error())
			return
		}
		rsp = append(rsp, *messageRsp)
		rspByte, err := json.Marshal(rsp)
		if err != nil {
			zlog.Error("序列化缓存消息失败: " + err.Error())
			return
		}
		if err := myredis.SetKeyEx("message_list_"+sendId+"_"+receiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			zlog.Error("设置缓存失败: " + err.Error())
		}
	} else {
		if !errors.Is(err, redis.Nil) {
			zlog.Error("获取缓存失败: " + err.Error())
		}
	}
}

// handleLogin 处理客户端登录
func (s *Server) handleLogin(client *Client) {
	s.mutex.Lock()
	s.Clients[client.Uuid] = client
	s.mutex.Unlock()
	// zlog.Info(fmt.Sprintf("用户 %s 已连接", client.Uuid))
	if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("欢迎来到seekF聊天服务器")); err != nil {
		zlog.Error(err.Error())
	}
}

// handleLogout 处理客户端登出
func (s *Server) handleLogout(client *Client) {
	s.mutex.Lock()
	delete(s.Clients, client.Uuid)
	s.mutex.Unlock()
	zlog.Info(fmt.Sprintf("用户 %s 已断开连接", client.Uuid))
}

// SendClientToLogin 发送客户端登录信号（非阻塞，避免死锁）
func (s *Server) SendClientToLogin(client *Client) {
	select {
	case s.Login <- client:
	default:
		zlog.Warn("登录通道已满，丢弃连接请求: " + client.Uuid)
	}
}

// SendClientToLogout 发送客户端登出信号（非阻塞，避免死锁）
func (s *Server) SendClientToLogout(client *Client) {
	select {
	case s.Logout <- client:
	default:
		zlog.Warn("登出通道已满，丢弃请求: " + client.Uuid)
	}
}

// RemoveClient 移除客户端
func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}

// normalizePath 去除路径中/static之前的所有内容
func normalizePath(path string) string {
	// 简单实现：找到/static的位置并返回从那里开始的路径
	for i := 0; i < len(path)-7; i++ {
		if path[i:i+7] == "/static" {
			return path[i:]
		}
	}
	return path
}

// SendMessageToKafka 发送消息到Kafka
func SendMessageToKafka(jsonMessage []byte) error {
	kafkaConfig := configs.GetConfig().KafkaConfig
	return kafka.KafkaServiceInstance.ChatWriter.WriteMessages(kafka.Ctx, segmentioKafka.Message{
		Key:   []byte(strconv.Itoa(kafkaConfig.Partition)),
		Value: jsonMessage,
	})
}

// UpdateMessageStatus 更新消息状态为已发送
func (s *Server) UpdateMessageStatus(uuid string) {
	if err := s.messageDAO.UpdateMessageStatus(uuid, message_status_enum.Sent); err != nil {
		zlog.Error("更新消息状态失败: " + err.Error())
	}
}

// updateSessionLastMessage 更新会话的最后消息
func (s *Server) updateSessionLastMessage(sessionId string, lastMessage string) {
	now := time.Now()

	// 更新会话最后消息
	if err := s.sessionDAO.UpdateSessionLastMessage(sessionId, lastMessage, now); err != nil {
		zlog.Error("更新会话最后消息失败: " + err.Error())
		return
	}

	// 清除会话列表缓存
	session, err := s.sessionDAO.GetSessionByUuid(sessionId)
	if err == nil {
		myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + session.SendId)
	}
}

// PushNotification 推送通知给指定用户
func (s *Server) PushNotification(userId string, notification interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if client, ok := s.Clients[userId]; ok {
		msg, err := json.Marshal(notification)
		if err != nil {
			zlog.Error("序列化通知失败: " + err.Error())
			return
		}

		notificationMsg := &MessageBack{
			Message: msg,
			Uuid:    "",
		}

		select {
		case client.SendBack <- notificationMsg:
			// 推送成功
		default:
			zlog.Warn("用户 " + userId + " 的通知通道已满，丢弃通知")
		}
	}
}

// sendErrorToClient 发送错误消息给客户端
func (s *Server) sendErrorToClient(clientId string, errMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if client, ok := s.Clients[clientId]; ok {
		errorResp := map[string]interface{}{
			"type":    "error",
			"message": errMsg,
		}
		msg, err := json.Marshal(errorResp)
		if err != nil {
			zlog.Error("序列化错误消息失败: " + err.Error())
			return
		}

		errorMsg := &MessageBack{
			Message: msg,
			Uuid:    "",
		}

		select {
		case client.SendBack <- errorMsg:
			// 发送成功
		default:
			zlog.Warn("用户 " + clientId + " 的错误消息通道已满，丢弃消息")
		}
	}
}
