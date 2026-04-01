package kafka

import (
	"context"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"github.com/segmentio/kafka-go"
)

var Ctx = context.Background()

type KafkaService struct {
	ChatWriter *kafka.Writer
	ChatReader *kafka.Reader
	KafkaConn  *kafka.Conn
}

var KafkaServiceInstance = new(KafkaService)

// Init 初始化kafka
func (k *KafkaService) Init() {
	//k.CreateTopic()  只运行一次
	kafkaConfig := configs.GetConfig().KafkaConfig
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
}

// Close 关闭kafka连接
func (k *KafkaService) Close() {
	if err := k.ChatWriter.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if err := k.ChatReader.Close(); err != nil {
		zlog.Error(err.Error())
	}
}

// CreateTopic 创建topic
func (k *KafkaService) CreateTopic() {
	kafkaConfig := configs.GetConfig().KafkaConfig
	chatTopic := kafkaConfig.ChatTopic

	// 连接至任意kafka节点
	var err error
	k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             chatTopic,
			NumPartitions:     kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
	}

	// 创建topic
	if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
		zlog.Error(err.Error())
	}
}
