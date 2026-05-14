package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	mcppkg "seekF-backend/internal/pkg/ai/mcp"
	toolpkg "seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/ai/rag"
	"seekF-backend/internal/pkg/ai/tts"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
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
	SendMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest, onChunk func(chunk string) error, onSources func(sources []toolpkg.SearchSource) error, onPosts func(posts []toolpkg.DiscoverPostItem) error, onComplete func(fullContent string) error) error
	// DeleteSession 删除AI会话
	DeleteSession(sessionId string) error
	// TextToSpeech 文本转语音，返回 WAV 音频数据
	TextToSpeech(ctx context.Context, content string, voice string) ([]byte, error)
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
			Sources:   msg.Sources,
			Posts:     msg.Posts,
			CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return rspList, total, nil
}

// SendMessageStream 流式发送消息：
// 1. 校验会话 → 2. 保存用户消息到DB → 3. 从DB读取历史构建上下文
// 4. 调用模型单例获取流式响应 → 5. 通过onChunk回调推送SSE
// 6. 完整响应保存到DB（失败则走Kafka异步持久化）
func (s *AIChatServiceImpl) SendMessageStream(ctx context.Context, userId string, req userreq.SendAIMessageRequest, onChunk func(chunk string) error, onSources func(sources []toolpkg.SearchSource) error, onPosts func(posts []toolpkg.DiscoverPostItem) error, onComplete func(fullContent string) error) error {
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
	if req.UseKnowledge {
		ragInst := rag.GetRAG()
		collectionName := "knowledge_" + userId
		knowledgeResults, err := ragInst.Search(ctx, collectionName, content, 3)
		if err == nil && len(knowledgeResults) > 0 {
			knowledgeContext := "以下是你应该了解的知识库内容：\n"
			for i, result := range knowledgeResults {
				knowledgeContext += fmt.Sprintf("%d. %s\n", i+1, result)
			}
			knowledgeContext += "请根据以上知识库内容回答用户的问题。如果知识库没有相关信息，请忽略并按你原来的知识回答。"
			systemPrompt = knowledgeContext + "\n\n" + systemPrompt
			zlog.Info("knowledge search found " + fmt.Sprint(len(knowledgeResults)) + " results")
		}
	}

	chatMessages = append(chatMessages, schema.SystemMessage(systemPrompt))
	for _, msg := range messages {
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
	chatModel := pool.GetModel(req.ModelType)
	if chatModel == nil {
		zlog.Error("model not available: " + req.ModelType)
		return fmt.Errorf("模型不可用")
	}

	// MCP Agent：默认先走第 1 次 Generate（工具决策，非流式）→ 执行 MCP → 第 2 次 Stream（无工具绑定）
	// 若 MCP 工具不可用/初始化失败，则回退为普通单轮流式。
	finalContent, handled, sources, posts, err := runMCPAgentFlow(ctx, chatModel, chatMessages, onChunk, req.UseWebSearch)
	if err != nil {
		zlog.Error("MCP agent flow failed: " + err.Error())
		return fmt.Errorf("AI响应失败")
	}
	if handled {
		// 如果有搜索来源数据，通过 onSources 回调推送给前端
		if len(sources) > 0 && onSources != nil {
			if err := onSources(sources); err != nil {
				zlog.Error("send sources to client failed: " + err.Error())
			}
		}
		// 如果有帖子数据，通过 onPosts 回调推送给前端
		if len(posts) > 0 && onPosts != nil {
			if err := onPosts(posts); err != nil {
				zlog.Error("send posts to client failed: " + err.Error())
			}
		}
		return s.persistAndCompleteAIMessage(req, userId, userMessage, finalContent, sources, posts, onChunk, onComplete)
	}

	finalContent, err = streamChatModelToSSE(ctx, chatModel, chatMessages, onChunk)
	if err != nil {
		zlog.Error("call AI model stream failed: " + err.Error())
		return fmt.Errorf("AI响应失败")
	}

	return s.persistAndCompleteAIMessage(req, userId, userMessage, finalContent, nil, nil, onChunk, onComplete)
}

// persistAndCompleteAIMessage 负责将最终回复兜底、持久化，并触发完成回调。
func (s *AIChatServiceImpl) persistAndCompleteAIMessage(req userreq.SendAIMessageRequest, userId string, userMessage *models.Message, finalContent string,
	sources []toolpkg.SearchSource, posts []toolpkg.DiscoverPostItem, onChunk func(chunk string) error, onComplete func(fullContent string) error) error {
	if finalContent == "" {
		finalContent = "抱歉，我暂时无法回答这个问题。"
		if err := onChunk(finalContent); err != nil {
			zlog.Error("send final chunk failed: " + err.Error())
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

	aiSendId := "A" + util.GetNowAndLenRandomString(11)
	aipkg.SendAIMessage(aipkg.AIMessagePayload{
		SessionId: req.SessionId,
		SendId:    aiSendId,
		SendName:  "AI助手",
		ReceiveId: userId,
		Content:   finalContent,
		ModelType: req.ModelType,
		Sources:   sourcesJSON,
		Posts:     postsJSON,
	})

	s.sessionDAO.UpdateSessionLastMessage(req.SessionId, finalContent, userMessage.CreatedAt)

	if onComplete != nil {
		onComplete(finalContent)
	}

	return nil
}

// streamChatModelToSSE 将模型的流式输出逐块透传给 SSE，并返回完整聚合后的文本。
func streamChatModelToSSE(ctx context.Context, m einomodel.ToolCallingChatModel, messages []*schema.Message, onChunk func(chunk string) error) (string, error) {
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

	return fullContent.String(), nil
}

// runMCPAgentFlow 第 1 次请求：Generate + WithTools（非流式，工具决策）。
// 若返回 ToolCalls：执行 MCP（InvokableRun），再第 2 次请求：无工具 Stream 总结。
// 若无 ToolCalls：返回 handled=false，让外层走普通 Stream，保证非工具场景也保持流式体验。
// 返回 handled=false 时回退为普通单轮流式（无 MCP 工具绑定）。
// enableWebSearch 控制是否启用 web_search 工具。
func runMCPAgentFlow(ctx context.Context, chatModel einomodel.ToolCallingChatModel, chatMessages []*schema.Message,
	onChunk func(chunk string) error, enableWebSearch bool) (finalContent string, handled bool, sources []toolpkg.SearchSource, posts []toolpkg.DiscoverPostItem, err error) {
	toolInfos, toolByName, err := mcppkg.FilteredMCPToolBinding(ctx, enableWebSearch)
	if err != nil {
		zlog.Error("get MCP tools failed: " + err.Error())
		return "", false, nil, nil, nil
	}
	if len(toolInfos) == 0 {
		return "", false, nil, nil, nil
	}

	modelWithTools, err := chatModel.WithTools(toolInfos)
	if err != nil {
		zlog.Error("WithTools failed: " + err.Error())
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
		zlog.Error("MCP agent Generate (tool decision) failed: " + err.Error())
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
			runOut, runErr := inv.InvokableRun(ctx, tc.Function.Arguments)
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

	finalContent, err = streamChatModelToSSE(ctx, chatModel, msgs2, onChunk)
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

// TextToSpeech 文本转语音
func (s *AIChatServiceImpl) TextToSpeech(ctx context.Context, content string, voice string) ([]byte, error) {
	return tts.Synthesize(ctx, content, voice)
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
