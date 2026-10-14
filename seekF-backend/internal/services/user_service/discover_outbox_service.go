package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	mykafka "seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/zlog"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// ErrDiscoverEventInvalid 表示消息没有对应的持久化事件，应记录异常后跳过。
var ErrDiscoverEventInvalid = errors.New("发现页事件无效")

// DiscoverEventWriter 提供同步确认的Kafka写入能力。
type DiscoverEventWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

// DiscoverEventReader 显式读取与提交Kafka位点，禁止处理前自动提交。
type DiscoverEventReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
}

// discoverEventConsumer 发现页事件消费者，负责持久化事件的投递、消费和重试。
type discoverEventConsumer struct {
	database *gorm.DB
	writer   DiscoverEventWriter
	reader   DiscoverEventReader
	search   userdao.ESDiscoverDAO
}

var discoverCons *discoverEventConsumer

// InitDiscoverConsumer 初始化发现页消费者，Kafka和数据库沿用项目全局连接。
func InitDiscoverConsumer(searchDAO userdao.ESDiscoverDAO) {
	discoverCons = &discoverEventConsumer{
		database: db.GormDB,
		writer:   mykafka.KafkaServiceInstance.DiscoverWriter,
		reader:   mykafka.KafkaServiceInstance.DiscoverReader,
		search:   searchDAO,
	}
}

// discoverEventMessage Kafka只携带事件ID，正文与权限状态从数据库读取。
type discoverEventMessage struct {
	EventId int64 `json:"event_id"`
}

// discoverRetryDelay 返回失败退避时间，最大60秒，避免故障期间持续冲击依赖。
func discoverRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt >= 7 {
		return 60 * time.Second
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

// setDiscoverEventFailure 保存有限长度的错误和下次重试时间。
func setDiscoverEventFailure(event *models.DiscoverOutbox, err error) {
	event.RetryCount++
	detail := []rune(err.Error())
	if len(detail) > 1000 {
		detail = detail[:1000]
	}
	event.LastError = string(detail)
	event.NextAttemptAt = time.Now().Add(discoverRetryDelay(event.RetryCount))
}

// PublishDue 每轮最多处理50个事件，不同进程依靠行锁避免同时领取。
func (s *discoverEventConsumer) PublishDue(ctx context.Context) error {
	events, err := userdao.NewDiscoverOutboxDAO(s.database.WithContext(ctx)).ListDue(50)
	if err != nil {
		return fmt.Errorf("读取待投递发现页事件失败: %w", err)
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.publishEvent(ctx, event.Id); err != nil {
			zlog.Warn(fmt.Sprintf("投递发现页事件失败，eventId=%d，postUuid=%s: %s", event.Id, event.PostUuid, err.Error()))
		}
	}
	return nil
}

// publishEvent 只锁事件行并限制外部调用耗时，发送与状态提交之间中断允许重复投递。
func (s *discoverEventConsumer) publishEvent(ctx context.Context, id int64) error {
	var sendErr error
	err := s.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverOutboxDAO(tx)
		event, err := dao.LockEvent(id, true)
		if err != nil {
			return err
		}
		if event == nil || event.Status == constants.DiscoverEventCompleted || event.NextAttemptAt.After(time.Now()) {
			return nil
		}
		data, err := json.Marshal(discoverEventMessage{EventId: event.Id})
		if err != nil {
			return fmt.Errorf("编码发现页事件失败: %w", err)
		}
		attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if s.writer == nil {
			sendErr = fmt.Errorf("发现页Kafka投递器未初始化")
		} else {
			sendErr = s.writer.WriteMessages(attempt, kafka.Message{Key: []byte(event.PostUuid), Value: data})
			if sendErr != nil {
				sendErr = fmt.Errorf("发送发现页事件到Kafka失败: %w", sendErr)
			}
		}
		if sendErr != nil {
			setDiscoverEventFailure(event, sendErr)
		} else {
			now := time.Now()
			event.Status = constants.DiscoverEventPublished
			event.PublishedAt = &now
			// 已投递但五分钟仍未完成时再次投递，补偿消费中断及消息保留期等故障。
			event.NextAttemptAt = now.Add(5 * time.Minute)
		}
		return dao.Save(event)
	})
	if err != nil {
		return fmt.Errorf("保存发现页投递状态失败，eventId=%d: %w", id, err)
	}
	return sendErr
}

// ProcessEvent 按事件ID幂等处理，读取最新作品并使用ES外部版本防止并发旧写覆盖。
func (s *discoverEventConsumer) ProcessEvent(ctx context.Context, id int64) error {
	err := s.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverOutboxDAO(tx)
		event, err := dao.LockEvent(id, false)
		if err != nil {
			return err
		}
		if event == nil {
			return fmt.Errorf("事件不存在，eventId=%d: %w", id, ErrDiscoverEventInvalid)
		}
		if event.Status == constants.DiscoverEventCompleted {
			return nil
		}
		post, err := dao.CurrentPost(event.PostUuid)
		if err != nil {
			return fmt.Errorf("读取当前作品失败: %w", err)
		}
		if post == nil {
			return fmt.Errorf("事件对应作品记录缺失，postUuid=%s", event.PostUuid)
		}
		if s.search == nil {
			return fmt.Errorf("发现页搜索同步器未初始化")
		}
		attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := s.search.SyncPost(attempt, post); err != nil {
			return err
		}
		now := time.Now()
		event.Status = constants.DiscoverEventCompleted
		event.ProcessedAt = &now
		event.LastError = ""
		return dao.Save(event)
	})
	if err != nil && !errors.Is(err, ErrDiscoverEventInvalid) && ctx.Err() == nil {
		failureErr := s.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			dao := userdao.NewDiscoverOutboxDAO(tx)
			event, lockErr := dao.LockEvent(id, false)
			if lockErr != nil {
				return lockErr
			}
			if event == nil || event.Status == constants.DiscoverEventCompleted {
				return nil
			}
			setDiscoverEventFailure(event, err)
			return dao.Save(event)
		})
		if failureErr != nil {
			return fmt.Errorf("处理事件失败且记录重试失败，eventId=%d: %w", id, errors.Join(err, failureErr))
		}
	}
	return err
}

// waitDiscoverRetry 使用可取消的定时器等待，停机时不阻塞后台任务退出。
func waitDiscoverRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// consume 处理成功后才提交位点，失败持续重试当前消息，不越过未处理的消息。
func (s *discoverEventConsumer) consume(ctx context.Context) {
	if s.reader == nil {
		zlog.Error("发现页Kafka消费者未初始化")
		return
	}
	for ctx.Err() == nil {
		message, err := s.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			zlog.Warn("读取发现页Kafka事件失败: " + err.Error())
			if !waitDiscoverRetry(ctx, time.Second) {
				return
			}
			continue
		}
		var payload discoverEventMessage
		invalid := json.Unmarshal(message.Value, &payload) != nil || payload.EventId <= 0
		if invalid {
			zlog.Error(fmt.Sprintf("发现页Kafka消息格式错误，partition=%d，offset=%d", message.Partition, message.Offset))
		}
		for attempt := 1; !invalid && ctx.Err() == nil; attempt++ {
			err = s.ProcessEvent(ctx, payload.EventId)
			if err == nil {
				break
			}
			if errors.Is(err, ErrDiscoverEventInvalid) {
				zlog.Error(err.Error())
				break
			}
			zlog.Warn("消费发现页事件失败，eventId=" + strconv.FormatInt(payload.EventId, 10) + ": " + err.Error())
			if !waitDiscoverRetry(ctx, discoverRetryDelay(attempt)) {
				return
			}
		}
		if ctx.Err() != nil {
			return
		}
		for ctx.Err() == nil {
			if err := s.reader.CommitMessages(ctx, message); err == nil {
				break
			} else {
				zlog.Warn("提交发现页Kafka位点失败: " + err.Error())
			}
			if !waitDiscoverRetry(ctx, time.Second) {
				return
			}
		}
	}
}

// StartDiscoverPublisher 启动发现页Outbox后台投递器。
func StartDiscoverPublisher() {
	if discoverCons == nil {
		zlog.Error("发现页事件消费者未初始化，请先调用InitDiscoverConsumer")
		return
	}
	s := discoverCons
	ctx := context.Background()
	go func() {
		for ctx.Err() == nil {
			if err := s.PublishDue(ctx); err != nil {
				zlog.Warn(err.Error())
			}
			if !waitDiscoverRetry(ctx, time.Second) {
				return
			}
		}
	}()
}

// StartDiscoverConsumer 启动发现页Kafka消费者，处理完成后提交位点。
func StartDiscoverConsumer() {
	if discoverCons == nil {
		zlog.Error("发现页事件消费者未初始化，请先调用InitDiscoverConsumer")
		return
	}
	go discoverCons.consume(context.Background())
}
