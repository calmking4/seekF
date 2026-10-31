package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	businessmetrics "seekF-backend/internal/pkg/observability/metrics"
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/gorm"
)

// discoverDistributionFrom 返回收件箱保留窗口，窗口外继续读时召回。
func discoverDistributionFrom(cfg configs.DiscoverFeedConfig) time.Time {
	days := cfg.RetentionDays
	if days < 1 {
		days = constants.DiscoverDistributionDays
	}
	return time.Now().AddDate(0, 0, -days).Truncate(time.Second)
}

// discoverDistributionThreshold 返回单篇作品推送人数的上限，默认500人。
func discoverDistributionThreshold(cfg configs.DiscoverFeedConfig) int64 {
	threshold := cfg.FollowerThreshold
	if threshold < 1 {
		threshold = constants.DiscoverDistributionThreshold
	}
	return threshold
}

// processFeedOnce 在一个事务内写完普通作者全部粉丝候选并完成任务，失败整笔回滚。
func (s *discoverEventConsumer) processFeedOnce(ctx context.Context, id int64, cfg configs.DiscoverFeedConfig) (int64, error) {
	var inserted int64
	err := s.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		outbox := userdao.NewDiscoverOutboxDAO(tx)
		event, err := outbox.LockEvent(id, false)
		if err != nil {
			return err
		}
		if event == nil || event.Target != constants.DiscoverTargetFeed {
			return fmt.Errorf("关注分发事件不存在或目标错误: %w", ErrDiscoverEventInvalid)
		}
		if event.Status == constants.DiscoverEventCompleted {
			return nil
		}
		dao := userdao.NewDiscoverDistributionDAO(tx)
		post, err := outbox.CurrentPostForUpdate(event.PostUuid)
		if err != nil {
			return err
		}
		if post != nil && event.PostVersion != post.Version {
			// 旧任务不覆盖新发布的收件箱，最新公开版本有自己的分发任务。
		} else if post == nil || post.DeletedAt.Valid || post.Status != constants.DiscoverPostPublished || post.Visibility != 0 || post.CreatedAt.Before(discoverDistributionFrom(cfg)) || post.FeedMode == constants.DiscoverFeedPull {
			if err := dao.DeletePostInbox(event.PostUuid); err != nil {
				return err
			}
		} else {
			threshold := discoverDistributionThreshold(cfg)
			fans, err := dao.ListFans(post.UserId, post.CreatedAt, int(threshold))
			if err != nil {
				return err
			}
			if int64(len(fans)) >= threshold {
				// 仅处理发布时符合时间条件的关注关系；异常达到上限时转拉，不能截断粉丝后报分发完成。
				if err := dao.SetPostFeedMode(post.Id, constants.DiscoverFeedPull); err != nil {
					return err
				}
				if err := dao.DeletePostInbox(post.Uuid); err != nil {
					return err
				}
			} else {
				rows := make([]models.DiscoverFeedInbox, 0, len(fans))
				for _, fan := range fans {
					rows = append(rows, models.DiscoverFeedInbox{UserId: fan.UserId, PostUuid: post.Uuid, PostCreatedAt: post.CreatedAt})
				}
				inserted, err = dao.InsertInbox(rows)
				if err != nil {
					return err
				}
			}
		}
		now := time.Now()
		event.Status = constants.DiscoverEventCompleted
		event.ProcessedAt = &now
		event.LastError = ""
		event.UpdatedAt = time.Now()
		return outbox.Save(event)
	})
	return inserted, err
}

// processFeedEvent 一次处理新帖分发，失败持久化重试时间，消费者在成功后提交位点。
func (s *discoverEventConsumer) processFeedEvent(ctx context.Context, id int64) error {
	cfg := configs.GetConfig().DiscoverFeedConfig
	started := time.Now()
	inserted, err := s.processFeedOnce(ctx, id, cfg)
	businessmetrics.ObserveDiscoverDistributionEvent(time.Since(started), inserted, err)
	if err != nil {
		if errors.Is(err, ErrDiscoverEventInvalid) {
			return err
		}
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
		return errors.Join(err, failureErr)
	}
	return nil
}

// MaintainDistribution 仅清理过期及当前不可见候选，不再回填任何历史帖子。
func (s *discoverEventConsumer) MaintainDistribution(ctx context.Context) error {
	cfg := configs.GetConfig().DiscoverFeedConfig
	return s.posts.CleanupFeedInbox(ctx, discoverDistributionFrom(cfg).Add(-constants.DiscoverInboxCleanupGrace), constants.DiscoverInboxCleanupLimit)
}

// StartDiscoverFeedConsumer 启动关注消费者，完整分发事务提交后显式提交Kafka位点。
func StartDiscoverFeedConsumer() {
	if discoverFeedCons == nil {
		zlog.Error("关注分发消费者未初始化")
		return
	}
	go discoverFeedCons.consume(context.Background())
}

// StartDiscoverFeedPublisher 启动候选清理与可靠投递，不进行新关注或存量关系历史回填。
func StartDiscoverFeedPublisher() {
	if discoverFeedCons == nil {
		zlog.Error("关注分发投递器未初始化")
		return
	}
	s := discoverFeedCons
	go func() {
		ctx := context.Background()
		for ctx.Err() == nil {
			if err := s.MaintainDistribution(ctx); err != nil {
				zlog.Warn("维护关注分发失败: " + err.Error())
			}
			if err := s.PublishDue(ctx); err != nil {
				zlog.Warn("投递关注分发失败: " + err.Error())
			}
			if !waitDiscoverRetry(ctx, time.Second) {
				return
			}
		}
	}()
}
