package userservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	businessmetrics "seekF-backend/internal/pkg/observability/metrics"
	redispkg "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

// processFeedOnce 锁定事件和作品，Redis幂等分发成功后才完成任务，失败保留事件以供重试。
func (s *discoverEventConsumer) processFeedOnce(ctx context.Context, id int64, cfg configs.DiscoverFeedConfig) (int64, error) {
	var inserted int64
	err := s.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		outbox := userdao.NewDiscoverOutboxDAO(tx)
		event, err := outbox.LockEvent(ctx, id, false)
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
		post, err := outbox.CurrentPostForUpdate(ctx, event.PostUuid)
		if err != nil {
			return err
		}
		if post != nil && event.PostVersion != post.Version {
			// 旧任务不覆盖新发布的收件箱，最新公开版本有自己的分发任务。
		} else if post == nil || post.DeletedAt.Valid || post.Status != constants.DiscoverPostPublished || post.Visibility != 0 || post.CreatedAt.Before(discoverDistributionFrom(cfg)) || post.FeedMode == constants.DiscoverFeedPull {
			// 失效索引由后续新增按容量淘汰；读取时始终复核MySQL当前状态，不因缓存残留泄露内容。
		} else {
			threshold := discoverDistributionThreshold(cfg)
			fans, err := dao.ListFans(ctx, post.UserId, post.CreatedAt, int(threshold))
			if err != nil {
				return err
			}
			if int64(len(fans)) >= threshold {
				// 仅处理发布时符合时间条件的关注关系；异常达到上限时转拉，不能截断粉丝后报分发完成。
				if err := dao.SetPostFeedMode(ctx, post.Id, constants.DiscoverFeedPull); err != nil {
					return err
				}
			} else {
				users := make([]string, 0, len(fans))
				for _, fan := range fans {
					users = append(users, fan.UserId)
				}
				attempt, cancel := context.WithTimeout(ctx, constants.DiscoverInboxWriteTimeout)
				defer cancel()
				// Redis无法参与MySQL事务；部分成功后重试ZADD不会重复，事件状态提交失败也可安全重放。
				inserted, err = fanoutDiscoverInbox(attempt, users, discoverInboxMember(*post), post.CreatedAt.Unix(), uuid.NewString())
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
		return outbox.Save(ctx, event)
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
			event, lockErr := dao.LockEvent(ctx, id, false)
			if lockErr != nil {
				return lockErr
			}
			if event == nil || event.Status == constants.DiscoverEventCompleted {
				return nil
			}
			setDiscoverEventFailure(event, err)
			return dao.Save(ctx, event)
		})
		return errors.Join(err, failureErr)
	}
	return nil
}

// StartDiscoverFeedConsumer 启动关注消费者，Redis分发及事件状态提交完成后才提交Kafka位点。
func StartDiscoverFeedConsumer() {
	if discoverFeedCons == nil {
		zlog.Error("关注分发消费者未初始化")
		return
	}
	go discoverFeedCons.consume(context.Background())
}

// StartDiscoverFeedPublisher 启动关注分发事件可靠投递，Redis收件箱新增时按容量淘汰最旧索引。
func StartDiscoverFeedPublisher() {
	if discoverFeedCons == nil {
		zlog.Error("关注分发投递器未初始化")
		return
	}
	s := discoverFeedCons
	go func() {
		ctx := context.Background()
		for ctx.Err() == nil {
			if err := s.PublishDue(ctx); err != nil {
				zlog.Warn("投递关注分发失败: " + err.Error())
			}
			if !waitDiscoverRetry(ctx, time.Second) {
				return
			}
		}
	}()
}

// discoverInboxMember 将数据库ID补齐为20位，同秒成员按数值主键排序，与MySQL游标一致。
func discoverInboxMember(post models.DiscoverPost) string {
	return fmt.Sprintf("%020d", post.Id)
}

// parseDiscoverInboxMember 解析固定长度的数据库ID，格式损坏时回源而不返回不完整分页。
func parseDiscoverInboxMember(member string) (int64, error) {
	if len(member) != 20 {
		return 0, fmt.Errorf("Redis收件箱索引格式无效")
	}
	for _, digit := range member {
		if digit < '0' || digit > '9' {
			return 0, fmt.Errorf("Redis收件箱帖子ID格式无效")
		}
	}
	id, err := strconv.ParseInt(member, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("Redis收件箱帖子ID无效")
	}
	return id, nil
}

// discoverInboxSnapshot 保存同一次原子读取的完整性标记、变更标识和有序帖子索引。
type discoverInboxSnapshot struct {
	Ready    bool
	Revision string
	Members  []string
	Scores   []int64
}

// discoverInboxKeys 返回同一用户的ZSet、元信息及重建锁键。
func discoverInboxKeys(userID string) []string {
	key := constants.DiscoverInboxKeyPrefix + "{" + userID + "}"
	return []string{key, key + ":meta", key + ":build"}
}

// readDiscoverInbox 原子转换旧成员并读取有界ID及时间分数，哨兵用于区分合法空收件箱与ZSet被淘汰。
func readDiscoverInbox(ctx context.Context, userID string) (discoverInboxSnapshot, error) {
	const script = `
redis.call('PERSIST', KEYS[1])
redis.call('PERSIST', KEYS[2])
local existing = redis.call('ZRANGE', KEYS[1], 0, -1, 'WITHSCORES')
-- 兼容旧组合成员；同一ID已存在时保留较新的发布时间。
for i = 1, #existing, 2 do
  local timestamp, id = string.match(existing[i], '^(%d+):(%d+):.+$')
  if timestamp and #timestamp == 20 and #id == 20 then
    local current = redis.call('ZSCORE', KEYS[1], id)
    if not current or tonumber(existing[i + 1]) > tonumber(current) then
      redis.call('ZADD', KEYS[1], existing[i + 1], id)
    end
    redis.call('ZREM', KEYS[1], existing[i])
  end
end
local excess = redis.call('ZCARD', KEYS[1]) - tonumber(ARGV[1]) - 1
if redis.call('ZSCORE', KEYS[1], '!') and excess > 0 then redis.call('ZREMRANGEBYRANK', KEYS[1], 1, excess) end
local revision = redis.call('HGET', KEYS[2], 'revision') or ''
if redis.call('HGET', KEYS[2], 'ready') ~= '1' or not redis.call('ZSCORE', KEYS[1], '!') then
  return {revision, '0'}
end
local result = {revision, '1'}
local members = redis.call('ZREVRANGE', KEYS[1], 0, tonumber(ARGV[1]) - 1, 'WITHSCORES')
for i = 1, #members, 2 do
  if members[i] ~= '!' then
    table.insert(result, members[i])
    table.insert(result, members[i + 1])
  end
end
return result`
	values, err := redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID)[:2], constants.DiscoverInboxCapacity).StringSlice()
	if err != nil {
		return discoverInboxSnapshot{}, err
	}
	if len(values) < 2 || (len(values)-2)%2 != 0 {
		return discoverInboxSnapshot{}, errors.New("Redis收件箱返回格式无效")
	}
	snapshot := discoverInboxSnapshot{Revision: values[0], Ready: values[1] == "1"}
	for i := 2; i < len(values); i += 2 {
		seconds, err := strconv.ParseInt(values[i+1], 10, 64)
		if err != nil || seconds < 0 {
			return discoverInboxSnapshot{}, errors.New("Redis收件箱时间无效")
		}
		snapshot.Members = append(snapshot.Members, values[i])
		snapshot.Scores = append(snapshot.Scores, seconds)
	}
	return snapshot, nil
}

// discoverInboxUnchanged 检查候选查询期间是否发生分发、失效或缓存淘汰，变化时交由MySQL兜底。
func discoverInboxUnchanged(ctx context.Context, userID, revision string) (bool, error) {
	const script = `
if redis.call('HGET', KEYS[2], 'revision') == ARGV[1]
and redis.call('HGET', KEYS[2], 'ready') == '1'
and redis.call('ZSCORE', KEYS[1], '!') then return 1 end
return 0`
	result, err := redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID)[:2], revision).Int64()
	return result == 1, err
}

// fanoutDiscoverInbox 批量幂等写入粉丝收件箱，失败允许重试，增量写入不能把不完整缓存标为完整。
func fanoutDiscoverInbox(ctx context.Context, users []string, member string, score int64, revision string) (int64, error) {
	const script = `
local existing = redis.call('ZRANGE', KEYS[1], 0, -1, 'WITHSCORES')
-- 转换后再追加及裁剪，避免新旧格式重复占用容量。
for i = 1, #existing, 2 do
  local timestamp, id = string.match(existing[i], '^(%d+):(%d+):.+$')
  if timestamp and #timestamp == 20 and #id == 20 then
    local current = redis.call('ZSCORE', KEYS[1], id)
    if not current or tonumber(existing[i + 1]) > tonumber(current) then
      redis.call('ZADD', KEYS[1], existing[i + 1], id)
    end
    redis.call('ZREM', KEYS[1], existing[i])
  end
end
if not redis.call('ZSCORE', KEYS[1], '!') then redis.call('HSET', KEYS[2], 'ready', '0') end
redis.call('ZADD', KEYS[1], -1, '!')
local inserted = redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
local excess = redis.call('ZCARD', KEYS[1]) - tonumber(ARGV[4]) - 1
if excess > 0 then redis.call('ZREMRANGEBYRANK', KEYS[1], 1, excess) end
redis.call('HSET', KEYS[2], 'revision', ARGV[3])
redis.call('PERSIST', KEYS[1])
redis.call('PERSIST', KEYS[2])
return inserted`
	if len(users) == 0 {
		return 0, nil
	}
	pipe := redispkg.GetClient().Pipeline()
	commands := make([]*redis.Cmd, 0, len(users))
	for _, userID := range users {
		commands = append(commands, pipe.Eval(ctx, script, discoverInboxKeys(userID)[:2], score, member, revision, constants.DiscoverInboxCapacity))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("写入Redis关注收件箱失败: %w", err)
	}
	var inserted int64
	for _, command := range commands {
		count, err := command.Int64()
		if err != nil {
			return 0, fmt.Errorf("读取Redis收件箱分发结果失败: %w", err)
		}
		inserted += count
	}
	return inserted, nil
}

// beginDiscoverInboxBuild 领取短期重建锁并记录变更标识，其他请求不等待重建而直接回源。
func beginDiscoverInboxBuild(ctx context.Context, userID, token string) (string, bool, error) {
	const script = `
if not redis.call('SET', KEYS[3], ARGV[1], 'NX', 'PX', ARGV[2]) then return false end
redis.call('PERSIST', KEYS[1])
redis.call('PERSIST', KEYS[2])
local revision = redis.call('HGET', KEYS[2], 'revision')
if not revision then
  revision = ARGV[1]
  redis.call('HSET', KEYS[2], 'revision', revision, 'ready', '0')
end
return revision`
	revision, err := redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID), token, constants.DiscoverInboxBuildTimeout.Milliseconds()).Text()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return revision, err == nil, err
}

// publishDiscoverInboxBuild 只在重建期间没有新分发或失效时原子发布，避免旧快照覆盖新消息。
func publishDiscoverInboxBuild(ctx context.Context, userID, token, revision string, members []string, scores []int64) (bool, error) {
	if len(members) != len(scores) || len(members) > constants.DiscoverInboxCapacity {
		return false, errors.New("Redis收件箱重建参数无效")
	}
	const script = `
if redis.call('GET', KEYS[3]) ~= ARGV[1] then return 0 end
if redis.call('HGET', KEYS[2], 'revision') ~= ARGV[2] then
  redis.call('DEL', KEYS[3])
  return 0
end
redis.call('DEL', KEYS[1])
redis.call('ZADD', KEYS[1], -1, '!')
for i = 3, #ARGV, 2 do redis.call('ZADD', KEYS[1], ARGV[i], ARGV[i + 1]) end
redis.call('HSET', KEYS[2], 'revision', ARGV[1], 'ready', '1')
redis.call('PERSIST', KEYS[1])
redis.call('PERSIST', KEYS[2])
redis.call('DEL', KEYS[3])
return 1`
	args := []any{token, revision}
	for i, member := range members {
		args = append(args, scores[i], member)
	}
	result, err := redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID), args...).Int64()
	return result == 1, err
}

// releaseDiscoverInboxBuild 只释放本次持有的重建锁，避免误删其他请求的新锁。
func releaseDiscoverInboxBuild(ctx context.Context, userID, token string) error {
	const script = `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end return 0`
	return redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID)[2:], token).Err()
}

// invalidateDiscoverInbox 在关注关系变更后标记重建，并使在途旧快照无法发布。
func invalidateDiscoverInbox(ctx context.Context, userID, revision string) error {
	const script = `
redis.call('DEL', KEYS[1])
redis.call('HSET', KEYS[2], 'revision', ARGV[1], 'ready', '0')
redis.call('PERSIST', KEYS[2])
return 1`
	return redispkg.GetClient().Eval(ctx, script, discoverInboxKeys(userID)[:2], revision).Err()
}
