package aiservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/cloudwego/eino/schema"
)

// AIChatService AI聊天服务接口
type AIChatService interface {
	// CreateSession 创建AI会话
	CreateSession(userId string, req userreq.CreateAISessionRequest) (*userresp.CreateAISessionRespond, error)
	// GetSessionList 获取用户AI会话列表
	GetSessionList(userId string, page int, pageSize int) (*userresp.GetAISessionListRespond, error)
	// GetMessageHistory 分页获取AI消息历史
	GetMessageHistory(sessionId string, page int, pageSize int) ([]userresp.GetAIMessageHistoryRespond, int64, error)
	// SendMessageStream 流式发送消息，通过SSE推送实时响应
	SendMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest, onChunk func(chunk string) error, onComplete func(fullContent string) error) error
	// DeleteSession 删除AI会话
	DeleteSession(sessionId string) error
}

// AIChatServiceImpl AI聊天服务实现
type AIChatServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	messageDAO  userdao.MessageDAO
	userInfoDAO userdao.UserInfoDAO
}

// NewAIChatService 创建AI聊天服务实例
func NewAIChatService(sessionDAO userdao.SessionDAO, messageDAO userdao.MessageDAO, userInfoDAO userdao.UserInfoDAO) AIChatService {
	return &AIChatServiceImpl{
		sessionDAO:  sessionDAO,
		messageDAO:  messageDAO,
		userInfoDAO: userInfoDAO,
	}
}

// CreateSession 创建AI会话，为用户分配一个AI接收者（A前缀ID）
func (s *AIChatServiceImpl) CreateSession(userId string, req userreq.CreateAISessionRequest) (*userresp.CreateAISessionRespond, error) {
	session, err := s.sessionDAO.CreateAISession(userId, req.ModelType)
	if err != nil {
		zlog.Error("create AI session failed: " + err.Error())
		return nil, fmt.Errorf("创建会话失败")
	}

	return &userresp.CreateAISessionRespond{
		SessionId: session.Uuid,
		ReceiveId: session.ReceiveId,
		ModelType: req.ModelType,
	}, nil
}

// GetSessionList 获取用户的AI会话列表（receive_id以'A'开头）
func (s *AIChatServiceImpl) GetSessionList(userId string, page int, pageSize int) (*userresp.GetAISessionListRespond, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	sessions, err := s.sessionDAO.GetAISessionList(userId)
	if err != nil {
		zlog.Error("get AI session list failed: " + err.Error())
		return nil, fmt.Errorf("获取会话列表失败")
	}

	total := int64(len(sessions))
	offset := (page - 1) * pageSize
	if offset >= len(sessions) {
		return &userresp.GetAISessionListRespond{
			List:  []userresp.AISessionItem{},
			Total: total,
		}, nil
	}

	end := offset + pageSize
	if end > len(sessions) {
		end = len(sessions)
	}

	var items []userresp.AISessionItem
	for _, session := range sessions[offset:end] {
		items = append(items, userresp.AISessionItem{
			SessionId:    session.Uuid,
			FirstMessage: session.FirstMessage,
			CreatedAt:    session.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &userresp.GetAISessionListRespond{
		List:  items,
		Total: total,
	}, nil
}

// GetMessageHistory 分页获取指定AI会话的消息历史（按时间正序）
func (s *AIChatServiceImpl) GetMessageHistory(sessionId string, page int, pageSize int) ([]userresp.GetAIMessageHistoryRespond, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	total, err := s.messageDAO.CountMessagesBySessionId(sessionId)
	if err != nil {
		zlog.Error("count AI messages failed: " + err.Error())
		return nil, 0, fmt.Errorf("获取消息历史失败")
	}

	messages, err := s.messageDAO.GetMessagesBySessionIdDesc(sessionId, pageSize, offset)
	if err != nil {
		zlog.Error("get AI messages failed: " + err.Error())
		return nil, 0, fmt.Errorf("获取消息历史失败")
	}

	var rspList []userresp.GetAIMessageHistoryRespond
	for _, msg := range messages {
		rspList = append(rspList, userresp.GetAIMessageHistoryRespond{
			SessionId: msg.SessionId,
			SendId:    msg.SendId,
			SendName:  msg.SendName,
			Content:   msg.Content,
			Type:      msg.Type,
			Url:       msg.Url,
			CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return rspList, total, nil
}

// SendMessageStream 流式发送消息：
// 1. 校验会话 → 2. 保存用户消息到DB → 3. 从DB读取历史构建上下文
// 4. 调用模型单例获取流式响应 → 5. 通过onChunk回调推送SSE
// 6. 完整响应保存到DB（失败则走Kafka异步持久化）
func (s *AIChatServiceImpl) SendMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest, onChunk func(chunk string) error, onComplete func(fullContent string) error) error {
	// 校验AI会话是否存在
	session, err := s.sessionDAO.GetAISessionByUuid(req.SessionId)
	if err != nil {
		zlog.Error("get AI session failed: " + err.Error())
		return fmt.Errorf("会话不存在")
	}

	// 获取用户信息
	user, err := s.userInfoDAO.FindUserByUuid(userId)
	if err != nil {
		zlog.Error("get user info failed: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	var userName, userAvatar string
	if user != nil {
		userName = user.Nickname
		userAvatar = user.Avatar
	}

	// 处理content：如果有图片但没有文本，content设为"图片"
	content := req.Content
	if req.ImageURL != "" && content == "" {
		content = "图片"
	}

	// 判断消息类型：有图片时为文件类型(Type=2)，否则为文本(Type=0)
	msgType := int8(0)
	if req.ImageURL != "" {
		msgType = 2
	}

	// 同步保存用户消息到DB
	userMsgId := "M" + util.GetNowAndLenRandomString(11)
	userMessage := &models.Message{
		Uuid:       userMsgId,
		SessionId:  req.SessionId,
		Type:       msgType,
		Content:    content,
		Url:        req.ImageURL,
		SendId:     userId,
		SendName:   userName,
		SendAvatar: userAvatar,
		ReceiveId:  session.ReceiveId,
		Status:     1,
	}

	if err := s.messageDAO.CreateMessage(userMessage); err != nil {
		zlog.Error("save user message failed: " + err.Error())
		return fmt.Errorf("发送消息失败")
	}

	// 更新会话最后一条消息
	s.sessionDAO.UpdateSessionLastMessage(req.SessionId, content, userMessage.CreatedAt)
	// 如果是第一消息，更新会话第一条消息
	if session.FirstMessage == "" {
		s.sessionDAO.UpdateSessionFirstMessage(req.SessionId, content)
	}

	// 从DB读取历史消息构建上下文（最近100条）
	messages, err := s.messageDAO.GetMessagesBySessionId(req.SessionId, 100, 0)
	if err != nil {
		zlog.Error("get message history for context failed: " + err.Error())
		messages = []models.Message{}
	}

	// 判断是否为多模态模型
	isMultiModalModel := req.ModelType == "glm-4v"

	// 将DB消息转换为eino消息格式，添加系统提示
	var chatMessages []*schema.Message
	chatMessages = append(chatMessages, schema.SystemMessage("你是一个专业的AI助手，当前使用的模型是"+req.ModelType+"。请根据这个身份回答用户的问题。"))
	for _, msg := range messages {
		if msg.SendId == userId {
			// 只有多模态模型才处理图片
			if isMultiModalModel && msg.Url != "" {
				imageURL := msg.Url
				multiMsg := &schema.Message{
					Role: schema.User,
					UserInputMultiContent: []schema.MessageInputPart{
						{Type: schema.ChatMessagePartTypeText, Text: msg.Content},
						{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
							MessagePartCommon: schema.MessagePartCommon{URL: &imageURL},
						}},
					},
				}
				chatMessages = append(chatMessages, multiMsg)
			} else {
				// 非多模态模型或无图片，只发送文本
				chatMessages = append(chatMessages, schema.UserMessage(msg.Content))
			}
		} else {
			chatMessages = append(chatMessages, schema.AssistantMessage(msg.Content, nil))
		}
	}

	// 如果有图片，且是多模态模型，追加当前用户消息为多模态
	if isMultiModalModel && req.ImageURL != "" {
		imageURL := req.ImageURL
		var currentMsgContent []schema.MessageInputPart
		if content != "" && content != "图片" {
			currentMsgContent = []schema.MessageInputPart{
				{Type: schema.ChatMessagePartTypeText, Text: content},
				{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{URL: &imageURL},
				}},
			}
		} else {
			currentMsgContent = []schema.MessageInputPart{
				{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{URL: &imageURL},
				}},
			}
		}
		multiMsg := &schema.Message{
			Role:                  schema.User,
			UserInputMultiContent: currentMsgContent,
		}
		chatMessages = append(chatMessages, multiMsg)
	}

	// 获取对应模型的单例
	pool := aipkg.GetModelPool()
	model := pool.GetModel(req.ModelType)
	if model == nil {
		zlog.Error("model not available: " + req.ModelType)
		return fmt.Errorf("模型不可用")
	}

	// 调用模型流式推理
	stream, err := model.Stream(ctx, chatMessages)
	if err != nil {
		zlog.Error("call AI model stream failed: " + err.Error())
		return fmt.Errorf("AI响应失败")
	}
	defer stream.Close()

	// 聚合流式响应，逐块推送
	var fullContent strings.Builder
	for {
		// 从AI模型流中接收下一个数据块，此操作会阻塞直到模型生成新内容
		chunk, err := stream.Recv()
		// 检查是否已到达流的末尾，如果是则退出循环
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			zlog.Error("read stream chunk failed: " + err.Error())
			break
		}

		if chunk != nil && len(chunk.Content) > 0 {
			fullContent.WriteString(chunk.Content)
			if err := onChunk(chunk.Content); err != nil {
				zlog.Error("send chunk to client failed: " + err.Error())
				break
			}
		}
	}

	finalContent := fullContent.String()
	if finalContent == "" {
		finalContent = "抱歉，我暂时无法回答这个问题。"
		if err := onChunk(finalContent); err != nil {
			zlog.Error("send final chunk failed: " + err.Error())
		}
	}

	// 发送AI响应到Kafka异步持久化
	aiSendId := "A" + util.GetNowAndLenRandomString(11)
	aipkg.SendAIMessage(aipkg.AIMessagePayload{
		SessionId: req.SessionId,
		SendId:    aiSendId,
		SendName:  "AI助手",
		ReceiveId: userId,
		Content:   finalContent,
		ModelType: req.ModelType,
	})

	// 更新会话最后一条消息
	s.sessionDAO.UpdateSessionLastMessage(req.SessionId, finalContent, userMessage.CreatedAt)

	if onComplete != nil {
		onComplete(finalContent)
	}

	return nil
}

// DeleteSession 删除AI会话及其所有消息
func (s *AIChatServiceImpl) DeleteSession(sessionId string) error {
	if err := s.messageDAO.DeleteMessagesBySessionId(sessionId); err != nil {
		zlog.Error("delete AI session messages failed: " + err.Error())
		return fmt.Errorf("删除会话消息失败")
	}

	if err := s.sessionDAO.DeleteAISession(sessionId); err != nil {
		zlog.Error("delete AI session failed: " + err.Error())
		return fmt.Errorf("删除会话失败")
	}

	zlog.Info("delete AI session success: " + sessionId)
	return nil
}
