package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	mykafka "seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/cloudwego/eino/schema"
	"github.com/segmentio/kafka-go"
)

// AICommentPayload AI评论回复Kafka载荷
type AICommentPayload struct {
	PostUuid       string `json:"post_uuid"`
	Content        string `json:"content"`
	ParentUuid     string `json:"parent_uuid"`
	ReplyToUserId  string `json:"reply_to_user_id"`
	ReplyToContent string `json:"reply_to_content"` // 被回复评论的内容
}

// aiCommentConsumer AI评论消费者（持有DAO依赖）
type aiCommentConsumer struct {
	discoverDAO userdao.DiscoverDAO
	userInfoDAO userdao.UserInfoDAO
}

var aiCommentCons *aiCommentConsumer

// InitAICommentConsumer 初始化AI评论消费者
func InitAICommentConsumer(discoverDAO userdao.DiscoverDAO, userInfoDAO userdao.UserInfoDAO) {
	aiCommentCons = &aiCommentConsumer{
		discoverDAO: discoverDAO,
		userInfoDAO: userInfoDAO,
	}
}

// SendAICommentTask 发送AI评论任务到Kafka
func SendAICommentTask(payload AICommentPayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		zlog.Error("marshal AI comment payload failed: " + err.Error())
		return
	}

	err = mykafka.KafkaServiceInstance.AICommentWriter.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})
	if err != nil {
		zlog.Error("send AI comment task to kafka failed: " + err.Error())
	} else {
		zlog.Info("AI comment task sent to kafka, post: " + payload.PostUuid)
	}
}

// StartAICommentConsumer 启动AI评论消费者
func StartAICommentConsumer() {
	if aiCommentCons == nil {
		zlog.Error("AI comment consumer not initialized, call InitAICommentConsumer first")
		return
	}

	go func() {
		for {
			msg, err := mykafka.KafkaServiceInstance.AICommentReader.ReadMessage(context.Background())
			if err != nil {
				zlog.Error("read AI comment task from kafka failed: " + err.Error())
				time.Sleep(time.Second)
				continue
			}

			var payload AICommentPayload
			if err := json.Unmarshal(msg.Value, &payload); err != nil {
				zlog.Error("unmarshal AI comment payload failed: " + err.Error())
				continue
			}

			aiCommentCons.processAIComment(payload)
		}
	}()
}

// processAIComment 处理AI评论回复
func (c *aiCommentConsumer) processAIComment(payload AICommentPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. 查帖子详情
	post, err := c.discoverDAO.FindPostByUuid(payload.PostUuid)
	if err != nil || post == nil {
		zlog.Error("AI comment: find post failed: " + payload.PostUuid)
		return
	}

	// 2. 获取帖子标签
	var tags []string
	if len(post.Tags) > 0 {
		json.Unmarshal(post.Tags, &tags)
	}

	// 3. 获取帖子图片URL（仅取图片类型，type=0）
	mediaList, _ := c.discoverDAO.FindMediaByPostId(post.Id)
	var imageUrls []string
	for _, m := range mediaList {
		if m.Type == 0 { // 0=图片，1=视频
			imageUrls = append(imageUrls, m.Url)
		}
	}

	// 4. 构建system prompt（文本上下文 + 被回复评论）
	systemPrompt := buildAIPostContext(post, tags, payload.ReplyToContent)

	// 5. 构建用户消息（多模态：文本 + 图片）
	var userMsg *schema.Message
	if len(imageUrls) > 0 {
		parts := []schema.MessageInputPart{
			{Type: schema.ChatMessagePartTypeText, Text: payload.Content},
		}
		for _, url := range imageUrls {
			u := url
			parts = append(parts, schema.MessageInputPart{
				Type:  schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{MessagePartCommon: schema.MessagePartCommon{URL: &u}},
			})
		}
		userMsg = &schema.Message{
			Role:                  schema.User,
			UserInputMultiContent: parts,
		}
	} else {
		userMsg = schema.UserMessage(payload.Content)
	}

	// 6. 调用glm-4v多模态模型（非流式）
	chatModel := GetModelPool().GetModel("glm-4v")
	if chatModel == nil {
		zlog.Error("AI comment: glm-4v model not available")
		return
	}

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		userMsg,
	}

	result, err := chatModel.Generate(ctx, messages)
	if err != nil {
		zlog.Error("AI comment: generate failed: " + err.Error())
		return
	}

	aiReply := strings.TrimSpace(result.Content)
	if aiReply == "" {
		zlog.Error("AI comment: empty reply")
		return
	}

	// 截断超过500字的回复（数据库字段限制）
	if len([]rune(aiReply)) > 500 {
		aiReply = string([]rune(aiReply)[:497]) + "..."
	}

	// 7. 保存AI评论
	commentUUID := "C" + util.GetNowAndLenRandomString(11)
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		PostId:        post.Id,
		UserId:        constants.AIAssistantUserId,
		ParentId:      payload.ParentUuid,
		ReplyToUserId: payload.ReplyToUserId,
		Content:       aiReply,
	}

	if err := c.discoverDAO.CreateComment(comment); err != nil {
		zlog.Error("AI comment: save comment failed: " + err.Error())
		return
	}

	c.discoverDAO.IncrementCommentCount(post.Id)
	zlog.Info(fmt.Sprintf("AI comment saved, post: %s, comment: %s", payload.PostUuid, commentUUID))
}

// buildAIPostContext 构造含帖子文本上下文的system prompt（图片通过多模态消息传递，不在此处列出URL）
func buildAIPostContext(post *models.DiscoverPost, tags []string, replyToContent string) string {
	var sb strings.Builder
	sb.WriteString("你是一个友善、专业的AI助手。当前用户在一个社交帖子下发起了\"@AI助手\"向你提问。\n\n")
	sb.WriteString("以下是帖子的上下文信息：\n")

	if post.Title != "" {
		sb.WriteString(fmt.Sprintf("- 帖子标题：%s\n", post.Title))
	}
	if post.Content != "" {
		sb.WriteString(fmt.Sprintf("- 帖子正文：%s\n", post.Content))
	}
	if len(tags) > 0 {
		sb.WriteString(fmt.Sprintf("- 帖子标签：%s\n", strings.Join(tags, ", ")))
	}
	if replyToContent != "" {
		sb.WriteString(fmt.Sprintf("- 用户正在回复的评论内容：\"%s\"\n", replyToContent))
	}

	sb.WriteString("\n请基于以上帖子上下文以及附带的图片，回答用户的问题。回答要求：\n")
	sb.WriteString("1. 简洁明了，适合评论区展示（不超过300字）\n")
	sb.WriteString("2. 与帖子内容相关时优先结合帖子上下文和图片内容回答\n")
	sb.WriteString("3. 如果问题与帖子无关，也请友好地回答\n")
	sb.WriteString("4. 不要使用markdown格式，使用纯文本\n")

	return sb.String()
}
