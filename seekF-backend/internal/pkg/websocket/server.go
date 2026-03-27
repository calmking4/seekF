package websocket

import (
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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	segmentioKafka "github.com/segmentio/kafka-go"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
)

// ChatServer 是全局的WebSocket服务器实例
var ChatServer = NewServer()

// Server 管理所有WebSocket客户端
type Server struct {
	Clients map[string]*Client
	mutex   *sync.Mutex
	Login   chan *Client // 登录通道
	Logout  chan *Client // 退出登录通道
}

// NewServer 创建新的WebSocket服务器
func NewServer() *Server {
	return &Server{
		Clients: make(map[string]*Client),
		mutex:   &sync.Mutex{},
		Login:   make(chan *Client, constants.CHANNEL_SIZE),
		Logout:  make(chan *Client, constants.CHANNEL_SIZE),
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

// readKafkaMessages 读取Kafka消息
func (s *Server) readKafkaMessages() {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error(fmt.Sprintf("readKafkaMessages panic: %v", r))
		}
	}()

	for {
		kafkaMessage, err := kafka.KafkaServiceInstance.ChatReader.ReadMessage(kafka.Ctx)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		zlog.Info(fmt.Sprintf("收到Kafka消息: topic=%s, partition=%d, offset=%d",
			kafkaMessage.Topic, kafkaMessage.Partition, kafkaMessage.Offset))

		data := kafkaMessage.Value
		var chatMessageReq userreq.ChatMessageRequest
		if err := json.Unmarshal(data, &chatMessageReq); err != nil {
			zlog.Error("解析消息失败: " + err.Error())
			continue
		}
		zlog.Info("原消息反序列化后: " + fmt.Sprintf("%+v", chatMessageReq))

		// 根据消息类型处理
		switch chatMessageReq.Type {
		case message_type_enum.Text:
			s.handleTextMessage(&chatMessageReq)
		case message_type_enum.File:
			s.handleFileMessage(&chatMessageReq)
		case message_type_enum.AVCall:
			s.handleAVCallMessage(&chatMessageReq)
		}
	}
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
		AVdata:     "",
	}

	// 保存消息到数据库
	if err := db.GormDB.Create(message).Error; err != nil {
		zlog.Error("保存消息失败: " + err.Error())
		return
	}

	// 更新会话的最后消息
	s.updateSessionLastMessage(message.SessionId, message.Content)

	// 构建响应消息
	messageRsp := userresp.GetMessageListRespond{
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

	// 判断是单聊还是群聊
	if message.ReceiveId[0] == 'U' {
		// 单聊
		s.sendToUser(message, messageBack, &messageRsp)
	} else if message.ReceiveId[0] == 'G' {
		// 群聊
		s.sendToGroup(message, messageBack, &messageRsp)
	}
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
		AVdata:     "",
	}

	// 保存消息到数据库
	if err := db.GormDB.Create(message).Error; err != nil {
		zlog.Error("保存消息失败: " + err.Error())
		return
	}

	// 更新会话的最后消息
	s.updateSessionLastMessage(message.SessionId, "[文件]"+message.FileName)

	// 构建响应消息
	messageRsp := userresp.GetMessageListRespond{
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

	// 判断是单聊还是群聊
	if message.ReceiveId[0] == 'U' {
		// 单聊
		s.sendToUser(message, messageBack, &messageRsp)
	} else {
		// 群聊
		s.sendToGroup(message, messageBack, &userresp.GetMessageListRespond{
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
		})
	}
}

// handleAVCallMessage 处理音视频通话消息
func (s *Server) handleAVCallMessage(chatMessageReq *userreq.ChatMessageRequest) {
	var avData userreq.AVData
	if err := json.Unmarshal([]byte(chatMessageReq.AVdata), &avData); err != nil {
		zlog.Error("解析音视频通话数据失败: " + err.Error())
		return
	}

	message := &models.Message{
		Uuid:       fmt.Sprintf("M%s", util.GetNowAndLenRandomString(11)),
		SessionId:  chatMessageReq.SessionId,
		Type:       chatMessageReq.Type,
		Content:    "",
		Url:        "",
		SendId:     chatMessageReq.SendId,
		SendName:   chatMessageReq.SendName,
		SendAvatar: normalizePath(chatMessageReq.SendAvatar),
		ReceiveId:  chatMessageReq.ReceiveId,
		FileSize:   "",
		FileType:   "",
		FileName:   "",
		Status:     message_status_enum.Unsent,
		CreatedAt:  time.Now(),
		AVdata:     chatMessageReq.AVdata,
	}

	// 特定类型的音视频通话消息需要存储
	if avData.MessageId == "PROXY" && (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
		if err := db.GormDB.Create(message).Error; err != nil {
			zlog.Error("保存音视频通话消息失败: " + err.Error())
			return
		}

		// 更新会话的最后消息
		s.updateSessionLastMessage(message.SessionId, "[音视频通话]")
	}

	// 只处理单聊音视频通话
	if chatMessageReq.ReceiveId[0] == 'U' {
		messageRsp := userresp.AVMessageRespond{
			SessionId:  message.SessionId,
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Type:       message.Type,
			Content:    message.Content,
			Url:        message.Url,
			FileSize:   message.FileSize,
			FileName:   message.FileName,
			FileType:   message.FileType,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
			AVdata:     message.AVdata,
		}

		jsonMessage, err := json.Marshal(messageRsp)
		if err != nil {
			zlog.Error("序列化音视频通话消息失败: " + err.Error())
			return
		}

		messageBack := &MessageBack{
			Message: jsonMessage,
			Uuid:    message.Uuid,
		}

		s.mutex.Lock()
		// 发送给接收者
		if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
			receiveClient.SendBack <- messageBack
		}
		// 音视频通话不回显给发送者
		s.mutex.Unlock()
	}
}

// sendToUser 发送消息给单个用户
func (s *Server) sendToUser(message *models.Message, messageBack *MessageBack, messageRsp *userresp.GetMessageListRespond) {
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

// sendToGroup 发送消息给群组
func (s *Server) sendToGroup(message *models.Message, messageBack *MessageBack, messageRsp *userresp.GetMessageListRespond) {
	// 获取群组信息
	var group models.GroupInfo
	if err := db.GormDB.Where("uuid = ?", message.ReceiveId).First(&group).Error; err != nil {
		zlog.Error("获取群组信息失败: " + err.Error())
		return
	}

	// 解析群组成员
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群成员失败: " + err.Error())
		return
	}

	s.mutex.Lock()
	for _, member := range members {
		if client, ok := s.Clients[member]; ok {
			client.SendBack <- messageBack
		}
	}
	s.mutex.Unlock()

	// 更新Redis缓存
	s.updateGroupMessageListCache(message.ReceiveId, messageRsp)
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

// updateGroupMessageListCache 更新群组消息列表缓存
func (s *Server) updateGroupMessageListCache(groupId string, messageRsp *userresp.GetMessageListRespond) {
	rspString, err := myredis.GetKeyNilIsErr("group_messagelist_" + groupId)
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
		if err := myredis.SetKeyEx("group_messagelist_"+groupId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
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
	zlog.Info(fmt.Sprintf("用户 %s 已连接", client.Uuid))
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
	if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("已退出登录")); err != nil {
		zlog.Error(err.Error())
	}
}

// SendClientToLogin 发送客户端登录信号
func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

// SendClientToLogout 发送客户端登出信号
func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	s.Logout <- client
	s.mutex.Unlock()
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
func UpdateMessageStatus(uuid string) {
	if err := db.GormDB.Model(&models.Message{}).Where("uuid = ?", uuid).Update("status", message_status_enum.Sent).Error; err != nil {
		zlog.Error("更新消息状态失败: " + err.Error())
	}
}

// updateSessionLastMessage 更新会话的最后消息（双向更新）
func (s *Server) updateSessionLastMessage(sessionId string, lastMessage string) {
	now := time.Now()

	// 获取该会话信息
	var session models.Session
	if err := db.GormDB.Where("uuid = ?", sessionId).First(&session).Error; err != nil {
		zlog.Error("获取会话信息失败: " + err.Error())
		return
	}

	// 更新发送者的session
	if err := db.GormDB.Model(&models.Session{}).Where("uuid = ?", sessionId).Updates(map[string]interface{}{
		"last_message":    lastMessage,
		"last_message_at": now,
	}).Error; err != nil {
		zlog.Error("更新发送者会话最后消息失败: " + err.Error())
	}

	// 查找接收者的session并更新
	var receiverSession models.Session
	if err := db.GormDB.Where("send_id = ? AND receive_id = ?", session.ReceiveId, session.SendId).First(&receiverSession).Error; err == nil {
		if err := db.GormDB.Model(&models.Session{}).Where("uuid = ?", receiverSession.Uuid).Updates(map[string]interface{}{
			"last_message":    lastMessage,
			"last_message_at": now,
		}).Error; err != nil {
			zlog.Error("更新接收者会话最后消息失败: " + err.Error())
		}
	}

	// 清除两人的会话列表缓存
	myredis.DelKeysWithPattern("session_list_" + session.SendId)
	myredis.DelKeysWithPattern("session_list_" + session.ReceiveId)
}
