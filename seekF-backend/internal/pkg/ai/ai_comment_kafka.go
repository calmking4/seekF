package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	mykafka "seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/ratelimit"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"

	"github.com/cloudwego/eino/schema"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// AICommentPayload AI评论回复Kafka载荷
type AICommentPayload struct {
	RequestUserID  string `json:"request_user_id,omitempty"`
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
var aiCommentStart sync.Once

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
		zlog.Error("序列化AI评论负载失败: " + err.Error())
		return
	}

	err = mykafka.KafkaServiceInstance.AICommentWriter.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})
	if err != nil {
		zlog.Error("发送AI评论任务到Kafka失败: " + err.Error())
	} else {
		zlog.Info("AI评论任务已发送到Kafka，帖子: " + payload.PostUuid)
	}
}

// StartAICommentConsumer 启动AI评论消费者
func StartAICommentConsumer() {
	if aiCommentCons == nil {
		zlog.Error("AI评论消费者未初始化，请先调用InitAICommentConsumer")
		return
	}

	aiCommentStart.Do(func() { go aiCommentCons.runCommentPool() })
}

// runCommentPool 启动AI评论专用协程池，进程内只启动一次。
func (c *aiCommentConsumer) runCommentPool() {
	consumeAICommentBatches(context.Background(), mykafka.KafkaServiceInstance.AICommentReader, c.processCommentMessage)
}

// processCommentMessage 失败重试，并以Kafka位置生成稳定评论ID支持重放去重。
func (c *aiCommentConsumer) processCommentMessage(message kafka.Message) {
	var payload AICommentPayload
	if err := json.Unmarshal(message.Value, &payload); err != nil {
		zlog.Error("反序列化AI评论负载失败: " + err.Error())
		return
	}
	if payload.PostUuid == "" || strings.TrimSpace(payload.Content) == "" {
		zlog.Error("AI评论任务缺少帖子标识或提问内容")
		return
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", message.Topic, message.Partition, message.Offset)))
	commentUUID := fmt.Sprintf("C%x", sum[:])[:20]
	delay := time.Second
	for {
		if err := c.processAIComment(payload, commentUUID); err == nil {
			return
		} else {
			zlog.Error(fmt.Sprintf("处理AI评论任务失败，帖子=%s，评论=%s: %s", payload.PostUuid, commentUUID, err.Error()))
			time.Sleep(delay)
			if delay < 30*time.Second {
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
		}
	}
}

// processAIComment 处理AI评论回复
func (c *aiCommentConsumer) processAIComment(payload AICommentPayload, commentUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	exists, err := c.discoverDAO.HasCommentIncludingDeleted(commentUUID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 1. 查帖子详情
	post, err := c.discoverDAO.FindPostByUuid(payload.PostUuid)
	if err != nil {
		return err
	}
	if post == nil {
		return nil
	}

	// 2. 获取帖子标签
	var tags []string
	if len(post.Tags) > 0 {
		json.Unmarshal(post.Tags, &tags)
	}

	// 3. 获取帖子图片URL（仅取图片类型，type=0）
	mediaList, err := c.discoverDAO.FindMediaByPostId(post.Id)
	if err != nil {
		return fmt.Errorf("查询AI评论图片失败，帖子=%s: %w", payload.PostUuid, err)
	}
	var imageUrls []string
	for _, m := range mediaList {
		if m.Type == 0 { // 0=图片，1=视频
			imageURL := m.Url
			key := oss.ObjectKeyFromURL(imageURL)
			// 本站私有图片使用短时链接；历史外部图片仍沿用原地址。
			if strings.HasPrefix(key, "discover/images/") && imageURL == oss.FileURL(key) {
				imageURL, err = oss.SignResourceURL(ctx, key)
				if err != nil {
					zlog.Error("AI评论: 准备图片失败，帖子=" + payload.PostUuid + ": " + err.Error())
					return err
				}
			}
			imageUrls = append(imageUrls, imageURL)
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
		zlog.Error("AI评论: glm-4v模型不可用")
		return fmt.Errorf("AI评论处理失败，帖子=%s", payload.PostUuid)
	}

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		userMsg,
	}

	// 入队只检查请求频率，真正调用模型时才与HTTP请求共用并发名额。
	userID := payload.RequestUserID
	if userID == "" {
		userID = constants.LegacyAICommentUser
	}
	release, err := ratelimit.AcquireAI(ctx, userID)
	if err != nil {
		return err
	}
	defer release()
	result, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("生成AI评论失败，帖子=%s: %w", payload.PostUuid, err)
	}
	if result == nil {
		zlog.Error("AI评论: 未返回生成结果，帖子=" + payload.PostUuid)
		return fmt.Errorf("AI评论处理失败，帖子=%s", payload.PostUuid)
	}

	aiReply := strings.TrimSpace(result.Content)
	if aiReply == "" {
		zlog.Error("AI评论: 空回复")
		return fmt.Errorf("AI评论处理失败，帖子=%s", payload.PostUuid)
	}

	// 截断超过500字的回复（数据库字段限制）
	if len([]rune(aiReply)) > 500 {
		aiReply = string([]rune(aiReply)[:497]) + "..."
	}

	// 7. 保存AI评论
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		PostId:        post.Id,
		UserId:        constants.AIAssistantUserId,
		ParentId:      payload.ParentUuid,
		ReplyToUserId: payload.ReplyToUserId,
		Content:       aiReply,
	}

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		// 锁定帖子后复核评论ID，评论写入与计数更新使用同一事务。
		current, err := txDiscoverDAO.FindPostByUuidForUpdate(payload.PostUuid)
		if err != nil {
			return err
		}
		if current == nil {
			return nil
		}
		exists, err := txDiscoverDAO.HasCommentIncludingDeleted(commentUUID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		if err := txDiscoverDAO.CreateComment(comment); err != nil {
			zlog.Error("AI评论: 保存评论失败: " + err.Error())
			return err
		}

		return txDiscoverDAO.IncrementCommentCount(post.Id)
	})
	if err != nil {
		return fmt.Errorf("保存AI评论失败，帖子=%s: %w", payload.PostUuid, err)
	}
	if comment.Id != 0 {
		zlog.Info(fmt.Sprintf("AI评论已保存，帖子: %s, 评论: %s", payload.PostUuid, commentUUID))
	}
	return nil
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
