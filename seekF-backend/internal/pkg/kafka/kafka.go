package kafka

import (
	"context"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"github.com/segmentio/kafka-go"
)

var Ctx = context.Background()

// KafkaService Kafka服务（全局单例）
type KafkaService struct {
	ChatWriter      *kafka.Writer
	ChatReader      *kafka.Reader
	AIChatWriter    *kafka.Writer
	AIChatReader    *kafka.Reader
	AICommentWriter *kafka.Writer
	AICommentReader *kafka.Reader
	KafkaConn       *kafka.Conn
}

var KafkaServiceInstance = new(KafkaService)

// Init 初始化Kafka人聊和AI聊Writer/Reader
func (k *KafkaService) Init() {
	k.CreateTopic()
	kafkaConfig := configs.GetConfig().KafkaConfig

	// 人聊 Writer/Reader
	k.ChatWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.ChatTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           kafkaConfig.Timeout * time.Second,
		RequiredAcks:           kafka.RequireNone,
		AllowAutoTopicCreation: false,
	}
	k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.ChatTopic,
		CommitInterval: kafkaConfig.Timeout * time.Second,
		GroupID:        "chat",
		StartOffset:    kafka.LastOffset,
	})

	// AI聊 Writer/Reader
	k.AIChatWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.AIChatTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           kafkaConfig.Timeout * time.Second,
		RequiredAcks:           kafka.RequireNone,
		AllowAutoTopicCreation: false,
	}
	k.AIChatReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.AIChatTopic,
		CommitInterval: kafkaConfig.Timeout * time.Second,
		GroupID:        "ai_chat",
		StartOffset:    kafka.LastOffset,
	})

	// AI评论回复 Writer/Reader
	k.AICommentWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.AICommentTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           kafkaConfig.Timeout * time.Second,
		RequiredAcks:           kafka.RequireNone,
		AllowAutoTopicCreation: false,
	}
	k.AICommentReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.AICommentTopic,
		CommitInterval: kafkaConfig.Timeout * time.Second,
		GroupID:        "ai_comment",
		StartOffset:    kafka.LastOffset,
	})

	zlog.Info("Kafka service initialized, chat topic: " + kafkaConfig.ChatTopic + ", ai chat topic: " + kafkaConfig.AIChatTopic)
}

// Close 关闭Kafka连接
func (k *KafkaService) Close() {
	if err := k.ChatWriter.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if err := k.ChatReader.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if k.AIChatWriter != nil {
		if err := k.AIChatWriter.Close(); err != nil {
			zlog.Error(err.Error())
		}
	}
	if k.AIChatReader != nil {
		if err := k.AIChatReader.Close(); err != nil {
			zlog.Error(err.Error())
		}
	}
	if k.AICommentWriter != nil {
		if err := k.AICommentWriter.Close(); err != nil {
			zlog.Error(err.Error())
		}
	}
	if k.AICommentReader != nil {
		if err := k.AICommentReader.Close(); err != nil {
			zlog.Error(err.Error())
		}
	}
}

// CreateTopic 创建topic（人聊 + AI聊）
func (k *KafkaService) CreateTopic() {
	kafkaConfig := configs.GetConfig().KafkaConfig

	var err error
	k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             kafkaConfig.ChatTopic,
			NumPartitions:     kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.AIChatTopic,
			NumPartitions:     kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.AICommentTopic,
			NumPartitions:     kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
	}

	if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
		zlog.Error(err.Error())
	}
}
