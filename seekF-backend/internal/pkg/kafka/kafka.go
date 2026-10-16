package kafka

import (
	"context"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"github.com/segmentio/kafka-go"
)

var Ctx = context.Background()

// KafkaService Kafka服务（全局单例）
type KafkaService struct {
	ChatWriter         *kafka.Writer
	ChatReader         *kafka.Reader
	AIChatWriter       *kafka.Writer
	AIChatReader       *kafka.Reader
	AICommentWriter    *kafka.Writer
	AICommentReader    *kafka.Reader
	DiscoverWriter     *kafka.Writer
	DiscoverReader     *kafka.Reader
	DiscoverFeedWriter *kafka.Writer
	DiscoverFeedReader *kafka.Reader
}

var KafkaServiceInstance = new(KafkaService)

// Init 初始化Kafka聊天、AI评论和发现页事件的Writer/Reader。
func (k *KafkaService) Init() {
	if err := k.CreateTopic(); err != nil {
		zlog.Warn("Kafka主题初始化失败，将后台重试: " + err.Error())
		go func() {
			for {
				time.Sleep(5 * time.Second)
				if err := k.CreateTopic(); err != nil {
					zlog.Warn("重试创建Kafka主题失败: " + err.Error())
					continue
				}
				return
			}
		}()
	}
	kafkaConfig := configs.GetConfig().KafkaConfig

	// 人聊 Writer/Reader
	k.ChatWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.ChatTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           kafkaConfig.Timeout * time.Second,
		RequiredAcks:           kafka.RequireOne,
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
		RequiredAcks:           kafka.RequireOne,
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
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: false,
	}
	k.AICommentReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.AICommentTopic,
		CommitInterval: kafkaConfig.Timeout * time.Second,
		GroupID:        "ai_comment",
		StartOffset:    kafka.LastOffset,
	})

	// 发现页事件 Writer/Reader
	// 投递等待全部副本确认，消费完成后显式提交位点。
	k.DiscoverWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.DiscoverTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           5 * time.Second,
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: false,
	}
	k.DiscoverReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.DiscoverTopic,
		CommitInterval: 0,
		GroupID:        "discover_search_sync",
		StartOffset:    kafka.FirstOffset,
	})

	// 关注分发 Writer/Reader，沿用搜索同步的全部副本确认与手动位点提交，避免批次完成前丢任务。
	k.DiscoverFeedWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),
		Topic:                  kafkaConfig.DiscoverFeedTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           5 * time.Second,
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: false,
	}
	k.DiscoverFeedReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},
		Topic:          kafkaConfig.DiscoverFeedTopic,
		CommitInterval: 0,
		GroupID:        "discover_feed_distribution",
		StartOffset:    kafka.FirstOffset,
	})

	zlog.Info("Kafka服务初始化完成，聊天主题: " + kafkaConfig.ChatTopic + "，AI聊天主题: " + kafkaConfig.AIChatTopic)
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
	if k.DiscoverWriter != nil {
		if err := k.DiscoverWriter.Close(); err != nil {
			zlog.Error("关闭发现页Kafka写入器失败: " + err.Error())
		}
	}
	if k.DiscoverReader != nil {
		if err := k.DiscoverReader.Close(); err != nil {
			zlog.Error("关闭发现页Kafka读取器失败: " + err.Error())
		}
	}
	if k.DiscoverFeedWriter != nil {
		if err := k.DiscoverFeedWriter.Close(); err != nil {
			zlog.Error("关闭发现页Kafka写入器失败: " + err.Error())
		}
	}
	if k.DiscoverFeedReader != nil {
		if err := k.DiscoverFeedReader.Close(); err != nil {
			zlog.Error("关闭发现页Kafka读取器失败: " + err.Error())
		}
	}
}

// CreateTopic 统一创建聊天、AI评论和发现页事件主题。
func (k *KafkaService) CreateTopic() error {
	kafkaConfig := configs.GetConfig().KafkaConfig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := kafka.DialContext(ctx, "tcp", kafkaConfig.HostPort)
	if err != nil {
		return fmt.Errorf("连接Kafka失败: %w", err)
	}
	defer conn.Close() // 使用后关闭连接，避免泄漏
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("设置Kafka主题操作超时失败: %w", err)
	}

	// 创建主题需要正数分区数量，旧配置中的0按单分区处理。
	partitions := kafkaConfig.Partition
	if partitions < 1 {
		partitions = 1
	}
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             kafkaConfig.ChatTopic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.AIChatTopic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.AICommentTopic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.DiscoverFeedTopic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
		{
			Topic:             kafkaConfig.DiscoverTopic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
	}

	if err = conn.CreateTopics(topicConfigs...); err != nil {
		return fmt.Errorf("创建Kafka主题失败: %w", err)
	}
	return nil
}
