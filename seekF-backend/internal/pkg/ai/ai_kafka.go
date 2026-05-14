package ai

import (
	"context"
	"encoding/json"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	mykafka "seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"github.com/segmentio/kafka-go"
)

// AIMessagePayload AI消息Kafka载荷
type AIMessagePayload struct {
	SessionId string `json:"session_id"`
	SendId    string `json:"send_id"`
	SendName  string `json:"send_name"`
	ReceiveId string `json:"receive_id"`
	Content   string `json:"content"`
	ModelType string `json:"model_type"`
	Sources   string `json:"sources,omitempty"`
	Posts     string `json:"posts,omitempty"`
}

// SendAIMessage 发送AI消息到Kafka（用于DB保存失败时的降级持久化）
func SendAIMessage(payload AIMessagePayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		zlog.Error("marshal AI message payload failed: " + err.Error())
		return
	}

	err = mykafka.KafkaServiceInstance.AIChatWriter.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})
	if err != nil {
		zlog.Error("send AI message to kafka failed: " + err.Error())
	}
}

// StartAIConsumer 启动AI消息消费者（从Kafka读取并持久化到MySQL）
func StartAIConsumer() {
	go func() {
		for {
			msg, err := mykafka.KafkaServiceInstance.AIChatReader.ReadMessage(context.Background())
			if err != nil {
				zlog.Error("read AI message from kafka failed: " + err.Error())
				time.Sleep(time.Second)
				continue
			}

			var payload AIMessagePayload
			if err := json.Unmarshal(msg.Value, &payload); err != nil {
				zlog.Error("unmarshal AI message payload failed: " + err.Error())
				continue
			}

			aiMsgId := "M" + util.GetNowAndLenRandomString(11)
			aiMessage := &models.Message{
				Uuid:       aiMsgId,
				SessionId:  payload.SessionId,
				Type:       0,
				Content:    payload.Content,
				Sources:    payload.Sources,
				Posts:      payload.Posts,
				SendId:     payload.SendId,
				SendName:   payload.SendName,
				SendAvatar: "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png",
				ReceiveId:  payload.ReceiveId,
				Status:     1,
			}

			if err := db.GormDB.Create(aiMessage).Error; err != nil {
				zlog.Error("save AI message to DB failed: " + err.Error())
			} else {
				zlog.Info("AI message saved to DB from kafka, uuid: " + aiMsgId)
			}
		}
	}()
}
