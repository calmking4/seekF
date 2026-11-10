package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/ai/asr"
	mcppkg "seekF-backend/internal/pkg/ai/mcp"
	toolpkg "seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/ai/tts"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/ratelimit"
	redispkg "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AIChatService AI聊天服务接口
type AIChatService interface {
	// CreateSession 创建AI会话
	CreateSession(ctx context.Context, userId string, req userreq.CreateAISessionRequest) (*userresp.CreateAISessionRespond, error)
	// GetSessionList 获取用户AI会话列表
	GetSessionList(ctx context.Context, userId string, page int, pageSize int) (*userresp.GetAISessionListRespond, error)
	// GetMessageHistory 游标分页获取AI消息历史
	GetMessageHistory(ctx context.Context, userId, sessionId string, pageSize int, cursor string, direction string) ([]userresp.GetAIMessageHistoryRespond, int64, error)
	// 保存提问、初始化事件流并启动独立生成任务。
	StartMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest) (*userresp.AIStreamInfo, error)
	// 校验任务归属并查询回答是否已完成或中断。
	GetMessageStream(ctx context.Context, userId, sessionId, streamId string) (*userresp.AIStreamInfo, error)
	// 从指定事件之后读取一批事件，暂无事件时短暂阻塞。
	ReadMessageStream(ctx context.Context, streamId, after string) ([]userresp.AIStreamEvent, error)
	// DeleteSession 删除AI会话
	DeleteSession(ctx context.Context, userId, sessionId string) error
	// TextToSpeech 流式文本转语音，返回流式结果
	TextToSpeech(ctx context.Context, content string, voice string) (*tts.StreamResult, error)
	// SpeechToText 使用GLM-ASR-2512流式识别录音。
	SpeechToText(ctx context.Context, audio []byte) (*asr.StreamResult, error)
}

// SpeechToText 将录音转换为流式识别结果。
func (s *AIChatServiceImpl) SpeechToText(ctx context.Context, audio []byte) (*asr.StreamResult, error) {
	return asr.Transcribe(ctx, audio)
}

// AIChatServiceImpl AI聊天服务实现
type AIChatServiceImpl struct {
	sessionDAO       userdao.SessionDAO
	messageDAO       userdao.MessageDAO
	userInfoDAO      userdao.UserInfoDAO
	knowledgeService KnowledgeService
}

// NewAIChatService 创建AI聊天服务实例
func NewAIChatService(sessionDAO userdao.SessionDAO, messageDAO userdao.MessageDAO, userInfoDAO userdao.UserInfoDAO, knowledgeService KnowledgeService) AIChatService {
	return &AIChatServiceImpl{
		sessionDAO:       sessionDAO,
		messageDAO:       messageDAO,
		userInfoDAO:      userInfoDAO,
		knowledgeService: knowledgeService,
	}
}

// CreateSession 创建AI会话，为用户分配一个AI接收者（A前缀ID）
func (s *AIChatServiceImpl) CreateSession(ctx context.Context, userId string, req userreq.CreateAISessionRequest) (*userresp.CreateAISessionRespond, error) {
	session, err := s.sessionDAO.CreateAISession(ctx, userId, req.ModelType)
	if err != nil {
		zlog.Error("创建AI会话失败: " + err.Error())
		return nil, fmt.Errorf("创建会话失败")
	}

	return &userresp.CreateAISessionRespond{
		SessionId: session.Uuid,
		ReceiveId: session.ReceiveId,
		ModelType: req.ModelType,
	}, nil
}

// GetSessionList 获取用户的AI会话列表（receive_id以'A'开头）
func (s *AIChatServiceImpl) GetSessionList(ctx context.Context, userId string, page int, pageSize int) (*userresp.GetAISessionListRespond, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	sessions, err := s.sessionDAO.GetAISessionList(ctx, userId)
	if err != nil {
		zlog.Error("获取AI会话列表失败: " + err.Error())
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

// GetMessageHistory 游标分页获取指定AI会话的消息历史
func (s *AIChatServiceImpl) GetMessageHistory(ctx context.Context, userId, sessionId string, pageSize int, cursor string, direction string) ([]userresp.GetAIMessageHistoryRespond, int64, error) {
	session, err := s.sessionDAO.GetAISessionByUuid(ctx, sessionId)
	if err != nil || session == nil || session.SendId != userId {
		return nil, 0, fmt.Errorf("会话不存在或无权访问")
	}
	if pageSize < 1 {
		pageSize = 20
	}

	total, err := s.messageDAO.CountMessagesBySessionId(ctx, sessionId)
	if err != nil {
		zlog.Error("统计AI消息失败: " + err.Error())
		return nil, 0, fmt.Errorf("获取消息历史失败")
	}

	messages, err := s.messageDAO.GetMessagesBySessionIdWithCursor(ctx, sessionId, cursor, pageSize, direction)
	if err != nil {
		zlog.Error("获取AI消息失败: " + err.Error())
		return nil, 0, fmt.Errorf("获取消息历史失败")
	}

	var rspList []userresp.GetAIMessageHistoryRespond
	var referencedPosts []toolpkg.DiscoverPostItem
	postCounts := make(map[int]int)
	for _, msg := range messages {
		if msg.Posts != "" {
			var posts []toolpkg.DiscoverPostItem
			if err := json.Unmarshal([]byte(msg.Posts), &posts); err != nil {
				return nil, 0, fmt.Errorf("解析AI参考帖子失败: %w", err)
			}
			postCounts[len(rspList)] = len(posts)
			referencedPosts = append(referencedPosts, posts...)
		}
		rspList = append(rspList, userresp.GetAIMessageHistoryRespond{
			Uuid:      msg.Uuid,
			SessionId: msg.SessionId,
			SendId:    msg.SendId,
			SendName:  msg.SendName,
			Content:   msg.Content,
			Type:      msg.Type,
			Url:       msg.Url,
			Sources:   msg.Sources,
			Posts:     msg.Posts,
			CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	if len(referencedPosts) > 0 {
		session, err := s.sessionDAO.GetAISessionByUuid(ctx, sessionId)
		if err != nil || session == nil {
			return nil, 0, fmt.Errorf("获取AI会话失败")
		}
		if err := refreshAIPostSupport(ctx, session.SendId, referencedPosts); err != nil {
			return nil, 0, err
		}
		offset := 0
		for i := range rspList {
			if count := postCounts[i]; count > 0 {
				data, err := json.Marshal(referencedPosts[offset : offset+count])
				if err != nil {
					return nil, 0, fmt.Errorf("序列化AI参考帖子失败: %w", err)
				}
				rspList[i].Posts = string(data)
				offset += count
			}
		}
	}
	return rspList, total, nil
}

// refreshAIPostSupport 批量刷新参考帖子的助力属性及当前权益，兼容缺少属性的旧聊天记录。
func refreshAIPostSupport(ctx context.Context, userId string, items []toolpkg.DiscoverPostItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	posts, err := userdao.NewDiscoverDAO(db.GormDB).FindPublicPostsByUuids(ctx, ids, userId)
	if err != nil {
		return fmt.Errorf("查询AI参考帖子失败: %w", err)
	}
	expiries, err := discoverSupportExpiries(ctx, userId, posts)
	if err != nil {
		return err
	}
	byID := make(map[string]models.DiscoverPost, len(posts))
	for _, post := range posts {
		byID[post.Uuid] = post
	}
	now := time.Now()
	for i := range items {
		item := &items[i]
		item.CanPlay, item.SupportExpiresAt = false, ""
		if post, ok := byID[item.ID]; ok && !item.IsBlockedBy {
			item.SupportOnly = post.SupportOnly
			item.CanPlay, _, item.SupportExpiresAt = discoverPlaybackAccess(post, userId, expiries[post.UserId], now)
		}
	}
	return nil
}

// aiStreamStart 保存在事件流首条记录中，绑定归属、消息编号和生成截止时间。
type aiStreamStart struct {
	UserId    string                `json:"user_id"`
	StartedAt int64                 `json:"started_at"`
	Info      userresp.AIStreamInfo `json:"info"`
}

// aiStreamWriter 串行写入同一回答，使用请求标识防止Redis客户端重试重复追加片段。
type aiStreamWriter struct {
	streamId string
	mu       sync.Mutex
}

// ErrAIStreamMissing 表示回放记录已经过期或不存在。
var ErrAIStreamMissing = errors.New("回答缓存已过期，请重新加载会话历史")

// ErrAIStreamForbidden 表示当前用户无权读取该回答。
var ErrAIStreamForbidden = errors.New("无权访问该回答")

var aiStreamCursorPattern = regexp.MustCompile(`^[0-9]{1,20}-[0-9]{1,20}$`)

// appendAIStreamEvent 原子追加事件并续期，只使用一个Stream键，不裁剪运行中的片段。
func appendAIStreamEvent(ctx context.Context, writer *aiStreamWriter, kind string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码AI回答事件失败: %w", err)
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	attempt, cancel := context.WithTimeout(ctx, constants.AIStreamRedisTimeout)
	defer cancel()
	// 重试期间持有同一任务写锁，因此相同nonce只能出现在末条记录，不需要额外去重键。
	const script = `
	local last = redis.call('XREVRANGE', KEYS[1], '+', '-', 'COUNT', 1)
	if #last > 0 then
		local fields = last[1][2]
		for i = 1, #fields, 2 do
			if fields[i] == 'nonce' and fields[i + 1] == ARGV[4] then return last[1][1] end
		end
	end
	if ARGV[1] ~= 'start' and #last == 0 then return redis.error_reply('回答缓存已过期') end
	local id = redis.call('XADD', KEYS[1], '*', 'type', ARGV[1], 'data', ARGV[2], 'nonce', ARGV[4])
	redis.call('EXPIRE', KEYS[1], ARGV[3])
	return id`
	if err := redispkg.GetClient().Eval(attempt, script, []string{constants.AIStreamKeyPrefix + writer.streamId}, kind, string(data), int64(constants.AIStreamTTL/time.Second), uuid.NewString()).Err(); err != nil {
		return fmt.Errorf("保存AI回答事件失败: %w", err)
	}
	return nil
}

// StartMessageStream 初始化回放日志并保存提问，后台任务接管并发名额后立即返回任务编号。
func (s *AIChatServiceImpl) StartMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest) (*userresp.AIStreamInfo, error) {
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" && req.ImageURL == "" {
		return nil, fmt.Errorf("请输入消息或上传图片")
	}
	if aipkg.GetModelPool().GetModel(req.ModelType) == nil {
		return nil, fmt.Errorf("模型不可用")
	}
	session, err := s.sessionDAO.GetAISessionByUuid(ctx, req.SessionId)
	if err != nil || session == nil || session.SendId != userId {
		return nil, fmt.Errorf("会话不存在或无权访问")
	}
	release, err := ratelimit.AcquireAI(ctx, userId)
	if err != nil {
		return nil, err
	}
	transferred := false
	defer func() {
		if !transferred {
			release()
		}
	}()
	info := userresp.AIStreamInfo{
		StreamId: uuid.NewString(), SessionId: req.SessionId, ModelType: req.ModelType,
		MessageId: "M" + util.GetNowAndLenRandomString(11), UserMessageId: "M" + util.GetNowAndLenRandomString(11), Status: "running",
	}
	start := aiStreamStart{UserId: userId, StartedAt: time.Now().UnixMilli(), Info: info}
	writer := &aiStreamWriter{streamId: info.StreamId}
	if err := appendAIStreamEvent(ctx, writer, "start", start); err != nil {
		return nil, err
	}
	// 初始化成功后不再随浏览器取消，确保提问保存与任务启动之间不会因断线被打断。
	prepareCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	message, err := s.prepareAIMessage(prepareCtx, userId, req, info.UserMessageId)
	if err != nil {
		if cleanupErr := redispkg.GetClient().Del(prepareCtx, constants.AIStreamKeyPrefix+info.StreamId).Err(); cleanupErr != nil {
			zlog.Warn("清理未启动AI回答缓存失败，streamId=" + info.StreamId + ": " + cleanupErr.Error())
		}
		return nil, err
	}
	genCtx, genCancel := context.WithTimeout(context.WithoutCancel(ctx), constants.AIStreamTimeout)
	transferred = true
	go s.runMessageStream(genCtx, genCancel, release, userId, req, message, info, writer)
	return &info, nil
}

// runMessageStream 将模型事件写入Redis，心跳持续续期，失败终止生成并独立记录终态。
func (s *AIChatServiceImpl) runMessageStream(ctx context.Context, cancel context.CancelFunc, release func(), userId string, req userreq.SendAIMessageRequest, message *models.Message, info userresp.AIStreamInfo, writer *aiStreamWriter) {
	defer cancel()
	defer release()
	var resultErr error
	stopHeartbeat, heartbeatDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(constants.AIStreamHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := appendAIStreamEvent(ctx, writer, "heartbeat", struct{}{}); err != nil {
					zlog.Error("AI回答心跳写入失败，streamId=" + info.StreamId + ": " + err.Error())
					cancel()
					return
				}
			}
		}
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = fmt.Errorf("AI回答生成发生异常: %v", recovered)
		}
		close(stopHeartbeat)
		<-heartbeatDone
		// 即使生成已超时，也留出独立时间写终态；完成事件必须晚于数据库提交。
		finishCtx, finishCancel := context.WithTimeout(context.Background(), constants.AIStreamRedisTimeout)
		defer finishCancel()
		kind, payload := "done", map[string]string{"message_id": info.MessageId}
		if resultErr != nil {
			kind, payload = "error", map[string]string{"error": resultErr.Error()}
			zlog.Error("AI回答生成失败，streamId=" + info.StreamId + ": " + resultErr.Error())
		}
		if err := appendAIStreamEvent(finishCtx, writer, kind, payload); err != nil {
			zlog.Error("保存AI回答终态失败，streamId=" + info.StreamId + ": " + err.Error())
		}
	}()
	resultErr = s.generateAIMessage(ctx, userId, req, message, info.MessageId,
		func(chunk string) error {
			return appendAIStreamEvent(ctx, writer, "delta", map[string]string{"content": chunk})
		},
		func(sources []toolpkg.SearchSource) error {
			return appendAIStreamEvent(ctx, writer, "sources", map[string]any{"sources": sources})
		},
		func(posts []toolpkg.DiscoverPostItem) error {
			return appendAIStreamEvent(ctx, writer, "posts", map[string]any{"posts": posts})
		})
}

// GetMessageStream 从首条事件校验归属，以数据库结果和末条事件判断终态及生成中断。
func (s *AIChatServiceImpl) GetMessageStream(ctx context.Context, userId, sessionId, streamId string) (*userresp.AIStreamInfo, error) {
	if _, err := uuid.Parse(streamId); err != nil {
		return nil, ErrAIStreamMissing
	}
	attempt, cancel := context.WithTimeout(ctx, constants.AIStreamRedisTimeout)
	defer cancel()
	client, key := redispkg.GetClient(), constants.AIStreamKeyPrefix+streamId
	first, err := client.XRangeN(attempt, key, "-", "+", 1).Result()
	if err != nil {
		return nil, fmt.Errorf("查询AI回答缓存失败: %w", err)
	}
	if len(first) == 0 {
		return nil, ErrAIStreamMissing
	}
	var start aiStreamStart
	data, ok := first[0].Values["data"].(string)
	if !ok || first[0].Values["type"] != "start" || json.Unmarshal([]byte(data), &start) != nil || start.Info.StreamId != streamId {
		return nil, fmt.Errorf("AI回答缓存格式无效")
	}
	if start.UserId != userId || start.Info.SessionId != sessionId {
		return nil, ErrAIStreamForbidden
	}
	session, err := s.sessionDAO.GetAISessionByUuid(attempt, sessionId)
	if err != nil {
		return nil, fmt.Errorf("查询AI会话失败: %w", err)
	}
	if session == nil || session.SendId != userId {
		return nil, ErrAIStreamForbidden
	}
	info := start.Info
	message, err := s.messageDAO.GetMessageByUuid(attempt, info.MessageId)
	if err == nil && message.SessionId == sessionId && message.ReceiveId == userId {
		info.Status = "finished"
		return &info, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询AI回答保存状态失败: %w", err)
	}
	last, err := client.XRevRangeN(attempt, key, "+", "-", 1).Result()
	if err != nil {
		return nil, fmt.Errorf("读取AI回答状态失败: %w", err)
	}
	if len(last) == 0 {
		return nil, ErrAIStreamMissing
	}
	if last[0].Values["type"] == "error" {
		info.Status = "error"
		var failure struct {
			Error string `json:"error"`
		}
		if payload, ok := last[0].Values["data"].(string); ok {
			if err := json.Unmarshal([]byte(payload), &failure); err != nil {
				return nil, fmt.Errorf("解析AI回答错误状态失败: %w", err)
			}
		}
		info.Error = failure.Error
	} else {
		lastMillis, err := strconv.ParseInt(strings.SplitN(last[0].ID, "-", 2)[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("AI回答事件时间无效")
		}
		if time.Since(time.UnixMilli(lastMillis)) > constants.AIStreamStaleTimeout || time.Since(time.UnixMilli(start.StartedAt)) > constants.AIStreamTimeout+constants.AIStreamStaleTimeout {
			info.Status, info.Error = "interrupted", "回答生成已中断，请重新发送消息"
		}
	}
	return &info, nil
}

// ReadMessageStream 验证游标并使用普通XREAD回放，每个订阅者独立消费且不删除记录。
func (s *AIChatServiceImpl) ReadMessageStream(ctx context.Context, streamId, after string) ([]userresp.AIStreamEvent, error) {
	if !aiStreamCursorPattern.MatchString(after) {
		return nil, fmt.Errorf("回答事件游标无效")
	}
	for _, part := range strings.Split(after, "-") {
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return nil, fmt.Errorf("回答事件游标超出范围")
		}
	}
	attempt, cancel := context.WithTimeout(ctx, constants.AIStreamRedisTimeout)
	defer cancel()
	streams, err := redispkg.GetClient().XRead(attempt, &redis.XReadArgs{Streams: []string{constants.AIStreamKeyPrefix + streamId, after}, Count: constants.AIStreamReadCount, Block: time.Second}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取AI回答事件失败: %w", err)
	}
	var events []userresp.AIStreamEvent
	for _, stream := range streams {
		for _, message := range stream.Messages {
			kind, kindOK := message.Values["type"].(string)
			data, dataOK := message.Values["data"].(string)
			if !kindOK || !dataOK || !json.Valid([]byte(data)) {
				return nil, fmt.Errorf("AI回答事件格式无效")
			}
			events = append(events, userresp.AIStreamEvent{Id: message.ID, Type: kind, Data: data})
		}
	}
	return events, nil
}

// prepareAIMessage 校验会话归属并在事务内保存提问和会话摘要。
func (s *AIChatServiceImpl) prepareAIMessage(ctx context.Context, userId string, req userreq.SendAIMessageRequest, userMsgId string) (*models.Message, error) {
	// 校验AI会话是否存在
	session, err := s.sessionDAO.GetAISessionByUuid(ctx, req.SessionId)
	if err != nil {
		zlog.Error("获取AI会话失败: " + err.Error())
		return nil, fmt.Errorf("会话不存在")
	}
	if session == nil || session.SendId != userId {
		return nil, fmt.Errorf("会话不存在或无权访问")
	}

	// 获取用户信息
	user, err := s.userInfoDAO.FindUserByUuid(ctx, userId)
	if err != nil {
		zlog.Error("获取用户信息失败: " + err.Error())
		return nil, fmt.Errorf("系统错误")
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

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txMessageDAO := userdao.NewMessageDAO(tx)
		txSessionDAO := userdao.NewSessionDAO(tx)
		current, err := txSessionDAO.GetOwnedSessionForUpdate(ctx, userId, req.SessionId)
		if err != nil {
			return fmt.Errorf("AI会话已失效: %w", err)
		}

		if err := txMessageDAO.CreateMessage(ctx, userMessage); err != nil {
			zlog.Error("保存用户消息失败: " + err.Error())
			return fmt.Errorf("发送消息失败")
		}

		// 更新会话最后一条消息
		if err := txSessionDAO.UpdateSessionLastMessage(ctx, req.SessionId, content, userMessage.CreatedAt); err != nil {
			zlog.Error("更新会话最后消息失败: " + err.Error())
			return fmt.Errorf("更新会话失败")
		}

		// 如果是第一消息，更新会话第一条消息
		if current.FirstMessage == "" {
			if err := txSessionDAO.UpdateSessionFirstMessage(ctx, req.SessionId, content); err != nil {
				zlog.Error("更新会话首条消息失败: " + err.Error())
				return fmt.Errorf("更新会话失败")
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return userMessage, nil
}

// generateAIMessage 复用模型和MCP流程，将输出交给事件存储后保存完整回答。
func (s *AIChatServiceImpl) generateAIMessage(ctx context.Context, userId string, req userreq.SendAIMessageRequest, userMessage *models.Message, aiMessageID string, onChunk func(chunk string) error, onSources func(sources []toolpkg.SearchSource) error, onPosts func(posts []toolpkg.DiscoverPostItem) error) error {
	content, userMsgId := userMessage.Content, userMessage.Uuid
	// 从DB读取历史消息构建上下文（最近100条）
	// 使用DESC排序获取最近的消息，然后反转为正序
	messages, err := s.messageDAO.GetMessagesBySessionIdDesc(ctx, req.SessionId, 100, 0)
	if err != nil {
		zlog.Error("获取消息历史上下文失败: " + err.Error())
		messages = []models.Message{}
	} else {
		// 反转为时间正序（从旧到新）
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	// 判断是否为多模态模型（从配置读取）
	isMultiModalModel := isMultiModalModel(req.ModelType)

	// 将DB消息转换为eino消息格式，添加系统提示，构建上下文
	var chatMessages []*schema.Message
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	systemPrompt := "你是一个专业的AI助手，当前使用的模型是" + req.ModelType + "。当前时间是" + currentTime + "。请根据这个身份回答用户的问题。"
	systemPrompt += "\n\n你还拥有查询平台帖子的能力（get_discover_posts 工具）。当用户询问「有什么新鲜帖子」、「推荐帖子」、「看看大家发了什么」、特定话题/标签的帖子（如「旅行」、「美食」、「编程」）、社区动态等，请使用此工具查询。搜索会自动匹配帖子的标题、正文内容和标签。"
	if req.UseWebSearch {
		systemPrompt += "\n\n你拥有联网搜索能力（web_search 工具）。当用户询问以下类型的问题时，你必须优先使用 web_search 工具搜索最新信息，而不是凭训练数据回答：\n" +
			"1. 今天的新闻、最近的事件、时事热点\n" +
			"2. 需要最新数据的问题（如股价、比分、天气）\n" +
			"3. 你训练数据中可能没有的最新信息\n" +
			"4. 用户明确要求搜索或联网查询的问题\n" +
			"重要：不要猜测或编造信息，如果问题涉及实时信息，请务必先搜索再回答。"
	}

	// 如果启用知识库，搜索相关知识
	if req.UseKnowledge && s.knowledgeService != nil {
		// 召回候选后按RRF融合排名选取上下文，不把融合分数当作相似度。
		knowledgeResults, err := s.knowledgeService.Search(ctx, userId, content, 10)
		if err != nil {
			zlog.Warn("知识库检索失败，用户ID=" + userId + ": " + err.Error())
		}
		if err == nil && len(knowledgeResults) > 0 {
			// RRF分数只用于排序，最多保留5个分块。
			selectedResults := knowledgeResults
			if len(selectedResults) > constants.KnowledgeContextLimit {
				selectedResults = selectedResults[:constants.KnowledgeContextLimit]
			}
			if len(selectedResults) > 0 {
				knowledgeContext := "以下是你应该了解的知识库内容：\n"
				for i, result := range selectedResults {
					knowledgeContext += fmt.Sprintf("%d. %s\n", i+1, result.Text)
				}
				knowledgeContext += "请根据以上知识库内容回答用户的问题。如果知识库没有相关信息，请忽略并按你原来的知识回答。"
				systemPrompt = knowledgeContext + "\n\n" + systemPrompt
				zlog.Info(fmt.Sprintf("知识库搜索找到 %d 条结果（按融合排名选取）", len(selectedResults)))
			}
		}
	}

	chatMessages = append(chatMessages, schema.SystemMessage(systemPrompt))
	for _, msg := range messages {
		// 跳过当前刚写入的消息，避免重复（多模态场景下会在后面统一处理）
		if msg.Uuid == userMsgId {
			continue
		}
		if msg.SendId == userId { //处理用户消息
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
		} else { //处理AI消息
			chatMessages = append(chatMessages, schema.AssistantMessage(msg.Content, nil))
		}
	}

	// 追加当前用户消息（多模态模型处理图片，非多模态模型处理文本）
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
	} else {
		// 非多模态模型，追加文本格式的当前消息
		chatMessages = append(chatMessages, schema.UserMessage(content))
	}

	// 获取对应模型的单例
	pool := aipkg.GetModelPool()
	chatModel := pool.GetModel(req.ModelType)
	if chatModel == nil {
		zlog.Error(req.ModelType + " 模型不可用")
		return fmt.Errorf("模型不可用")
	}

	// MCP Agent：默认先走第 1 次 Generate（工具决策，非流式）→ 执行 MCP → 第 2 次 Stream（无工具绑定）
	// 若 MCP 工具不可用/初始化失败，则回退为普通单轮流式。
	finalContent, handled, sources, posts, err := runMCPAgentFlow(ctx, userId, chatModel, chatMessages, onChunk, req.UseWebSearch)
	if err != nil {
		zlog.Error("MCP代理流程失败: " + err.Error())
		return fmt.Errorf("AI响应失败")
	}
	if handled {
		if err := refreshAIPostSupport(ctx, userId, posts); err != nil {
			return err
		}
		// 如果有搜索来源数据，通过 onSources 回调推送给前端
		if len(sources) > 0 && onSources != nil {
			if err := onSources(sources); err != nil {
				return err
			}
		}
		// 如果有帖子数据，通过 onPosts 回调推送给前端
		if len(posts) > 0 && onPosts != nil {
			if err := onPosts(posts); err != nil {
				return err
			}
		}
		return s.persistAndCompleteAIMessage(ctx, req, userId, userMessage, aiMessageID, finalContent, sources, posts, onChunk)
	}

	finalContent, err = streamChatModel(ctx, chatModel, chatMessages, onChunk)
	if err != nil {
		zlog.Error("调用AI模型流失败: " + err.Error())
		return fmt.Errorf("AI响应失败")
	}

	return s.persistAndCompleteAIMessage(ctx, req, userId, userMessage, aiMessageID, finalContent, nil, nil, onChunk)
}

// persistAndCompleteAIMessage 将完整回答和会话摘要一起提交，终态由后台任务统一写入。
func (s *AIChatServiceImpl) persistAndCompleteAIMessage(ctx context.Context, req userreq.SendAIMessageRequest, userId string, userMessage *models.Message, aiMessageID string, finalContent string,
	sources []toolpkg.SearchSource, posts []toolpkg.DiscoverPostItem, onChunk func(chunk string) error) error {
	if finalContent == "" {
		finalContent = "抱歉，我暂时无法回答这个问题。"
		if err := onChunk(finalContent); err != nil {
			return err
		}
	}

	// 序列化搜索来源数据（只保留标题和URL）
	var sourcesJSON string
	if len(sources) > 0 {
		type sourceLite struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		}
		lite := make([]sourceLite, 0, len(sources))
		for _, s := range sources {
			lite = append(lite, sourceLite{Title: s.Title, URL: s.URL})
		}
		b, err := json.Marshal(lite)
		if err == nil {
			sourcesJSON = string(b)
		}
	}

	// 序列化帖子数据
	var postsJSON string
	if len(posts) > 0 {
		b, err := json.Marshal(posts)
		if err == nil {
			postsJSON = string(b)
		}
	}

	aiMessage := &models.Message{
		Uuid:       aiMessageID,
		SessionId:  req.SessionId,
		SendId:     "A" + util.GetNowAndLenRandomString(11),
		SendName:   "AI助手",
		SendAvatar: constants.AIAssistantAvatar,
		ReceiveId:  userId,
		Content:    finalContent,
		Sources:    sourcesJSON,
		Posts:      postsJSON,
		Status:     1,
	}
	// 回答及会话摘要一起提交，写入失败时不发送完成信号。
	if err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 与会话删除使用同一行锁，避免删除期间生成出孤立回答。
		if _, err := userdao.NewSessionDAO(tx).GetOwnedSessionForUpdate(ctx, userId, req.SessionId); err != nil {
			return fmt.Errorf("AI会话已失效: %w", err)
		}
		if err := userdao.NewMessageDAO(tx).CreateMessage(ctx, aiMessage); err != nil {
			return fmt.Errorf("保存AI回答失败: %w", err)
		}
		if err := userdao.NewSessionDAO(tx).UpdateSessionLastMessage(ctx, req.SessionId, finalContent, userMessage.CreatedAt); err != nil {
			return fmt.Errorf("更新AI会话摘要失败: %w", err)
		}
		return nil
	}); err != nil {
		zlog.Error("持久化AI回答失败，sessionId=" + req.SessionId + ": " + err.Error())
		return err
	}

	return nil
}

// streamChatModel 逐块交付模型输出，读取或保存失败立即结束，避免截断回答被标记为完成。
func streamChatModel(ctx context.Context, m einomodel.ToolCallingChatModel, messages []*schema.Message, onChunk func(chunk string) error) (string, error) {
	stream, err := m.Stream(ctx, messages)
	if err != nil {
		return "", err
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
			zlog.Error("读取流数据块失败: " + err.Error())
			return fullContent.String(), fmt.Errorf("读取模型输出失败: %w", err)
		}

		if chunk != nil && len(chunk.Content) > 0 {
			fullContent.WriteString(chunk.Content)
			if err := onChunk(chunk.Content); err != nil {
				return fullContent.String(), err
			}
		}
	}

	return fullContent.String(), nil
}

// runMCPAgentFlow 第 1 次请求：Generate + WithTools（非流式，工具决策）。
// 若返回 ToolCalls：执行 MCP（InvokableRun），再第 2 次请求：无工具 Stream 总结。
// 若无 ToolCalls：返回 handled=false，让外层走普通 Stream，保证非工具场景也保持流式体验。
// 返回 handled=false 时回退为普通单轮流式（无 MCP 工具绑定）。
// enableWebSearch 控制是否启用 web_search 工具。
func runMCPAgentFlow(ctx context.Context, userId string, chatModel einomodel.ToolCallingChatModel, chatMessages []*schema.Message,
	onChunk func(chunk string) error, enableWebSearch bool) (finalContent string, handled bool, sources []toolpkg.SearchSource, posts []toolpkg.DiscoverPostItem, err error) {
	toolInfos, toolByName, err := mcppkg.FilteredMCPToolBinding(ctx, enableWebSearch)
	if err != nil {
		zlog.Error("获取MCP工具失败: " + err.Error())
		return "", false, nil, nil, nil
	}
	if len(toolInfos) == 0 {
		return "", false, nil, nil, nil
	}

	modelWithTools, err := chatModel.WithTools(toolInfos)
	if err != nil {
		zlog.Error("绑定工具失败: " + err.Error())
		return "", false, nil, nil, nil
	}

	// 第1次调用：非流式 Generate，AI 看完用户问题和工具列表后，决定是否需要调用工具。
	// first 包含两个关键字段：
	//   first.Content   — AI 的文本回复（可能为空，也可能有内容，如"我来帮你查一下"）
	//   first.ToolCalls — AI 决定调用的工具列表，如 [{Name:"get_weather", Args:{"location":"北京"}}]
	// 	 first.ToolCalls == [
	//       {ID: "call_001", Function: {Name: "get_weather", Arguments: `{"location":"北京"}`}},
	//       {ID: "call_002", Function: {Name: "web_search",  Arguments: `{"query":"北京今日天气"}`}},
	//   ]
	// 如果 AI 觉得不需要工具（比如问"1+1=?"），ToolCalls 为空，回退到普通流式回答。
	first, err := modelWithTools.Generate(ctx, chatMessages)
	if err != nil {
		zlog.Error("MCP代理生成(工具决策)失败: " + err.Error())
		return "", false, nil, nil, nil
	}

	if len(first.ToolCalls) == 0 {
		// 不需要工具时回退到普通流式，避免只返回一次性整段文本。
		return "", false, nil, nil, nil
	}

	// msgs2 是第2次调用（流式总结）的消息列表。
	// 结构：[原始对话历史..., AI的工具决策回复, 工具执行结果1, 工具执行结果2, ...]
	msgs2 := make([]*schema.Message, 0, len(chatMessages)+1+len(first.ToolCalls))
	msgs2 = append(msgs2, chatMessages...)                                         // 原始对话历史（system + user/assistant）
	msgs2 = append(msgs2, schema.AssistantMessage(first.Content, first.ToolCalls)) // AI 的工具决策（"我要调用 get_weather"）

	for _, tc := range first.ToolCalls {
		name := tc.Function.Name
		inv := toolByName[name]
		var out string
		var rawResult string
		if inv == nil {
			out = fmt.Sprintf("错误：未找到可执行工具 %q", name)
		} else {
			arguments := tc.Function.Arguments
			if name == "get_discover_posts" {
				var params map[string]any
				if err := json.Unmarshal([]byte(arguments), &params); err != nil || params == nil {
					params = map[string]any{}
				}
				params["_viewer_id"] = userId
				encoded, err := json.Marshal(params)
				if err != nil {
					return "", true, nil, nil, fmt.Errorf("编码帖子工具参数失败: %w", err)
				}
				arguments = string(encoded)
			}
			runOut, runErr := inv.InvokableRun(ctx, arguments)
			if runErr != nil {
				out = "工具执行失败: " + runErr.Error()
			} else {
				rawResult = runOut
				out = extractTextContent(runOut)
			}
		}

		// 从工具的 返回结果 中提取结构化数据（通过 MCP 多内容项协议）
		if rawResult != "" {
			if name == "web_search" {
				if extracted := extractStructuredData[toolpkg.SearchSource](rawResult); len(extracted) > 0 {
					sources = append(sources, extracted...)
				}
			}
			if name == "get_discover_posts" {
				if extracted := extractStructuredData[toolpkg.DiscoverPostItem](rawResult); len(extracted) > 0 {
					posts = append(posts, extracted...)
				}
			}
		}

		// 把工具执行结果封装为 ToolMessage 追加到 msgs2，AI 第2次调用时会看到这些结果
		// 例如 ToolMessage("北京 晴天 25°C 湿度40%...", toolID="call_001", toolName="get_weather")
		msgs2 = append(msgs2, schema.ToolMessage(out, tc.ID, schema.WithToolName(name)))
	}

	finalContent, err = streamChatModel(ctx, chatModel, msgs2, onChunk)
	if err != nil {
		return "", false, nil, nil, err
	}
	return finalContent, true, sources, posts, nil
}

// extractTextContent 从 CallToolResult 中提取 TextContent 纯文本。
func extractTextContent(raw string) string {
	var result mcpgo.CallToolResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return raw
	}
	txt := mcpgo.GetTextFromContent(result.Content)
	if txt == "" {
		return raw
	}
	return txt
}

// extractStructuredData 从 CallToolResult 的 StructuredContent 字段提取结构化数据。
func extractStructuredData[T any](raw string) []T {
	var result mcpgo.CallToolResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	if result.StructuredContent == nil {
		return nil
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return nil
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	return items
}

// TextToSpeech 文本转流式语音
func (s *AIChatServiceImpl) TextToSpeech(ctx context.Context, content string, voice string) (*tts.StreamResult, error) {
	return tts.Synthesize(ctx, content, voice)
}

// DeleteSession 删除AI会话及其所有消息
func (s *AIChatServiceImpl) DeleteSession(ctx context.Context, userId, sessionId string) error {
	session, err := s.sessionDAO.GetAISessionByUuid(ctx, sessionId)
	if err != nil || session == nil || session.SendId != userId {
		return fmt.Errorf("会话不存在或无权访问")
	}
	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txMessageDAO := userdao.NewMessageDAO(tx)
		txSessionDAO := userdao.NewSessionDAO(tx)
		if _, err := txSessionDAO.GetOwnedSessionForUpdate(ctx, userId, sessionId); err != nil {
			return fmt.Errorf("锁定AI会话失败: %w", err)
		}

		if err := txMessageDAO.DeleteMessagesBySessionId(ctx, sessionId); err != nil {
			zlog.Error("删除AI会话消息失败: " + err.Error())
			return fmt.Errorf("删除会话消息失败")
		}

		if err := txSessionDAO.DeleteAISession(ctx, sessionId); err != nil {
			zlog.Error("删除AI会话失败: " + err.Error())
			return fmt.Errorf("删除会话失败")
		}

		return nil
	})
	if err != nil {
		return err
	}

	zlog.Info("删除AI会话成功: " + sessionId)
	return nil
}

// isMultiModalModel 判断是否为多模态模型（从配置读取）
func isMultiModalModel(modelType string) bool {
	cfg := configs.GetConfig().AIModelConfig
	for _, m := range cfg.MultimodalModels {
		if m == modelType {
			return true
		}
	}
	return false
}
