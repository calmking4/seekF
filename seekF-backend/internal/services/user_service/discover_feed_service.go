package userservice

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	businessmetrics "seekF-backend/internal/pkg/observability/metrics"
	redispkg "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DiscoverFeedService 信息流与关注更新业务接口。
type DiscoverFeedService interface {
	// 查询关注作者及未查看更新的作者数量。
	FollowingUpdates(ctx context.Context, userId string) (*FollowingUpdatesInfo, error)
	// 确认查看指定作者快照内的更新，不清除后续新帖。
	ReadFollowingUpdates(ctx context.Context, userId, token string) error
	// 记录有效主动点击，排除作者自身浏览并按天去重。
	RecordDiscoverClick(ctx context.Context, userId string, input userreq.DiscoverClickRequest) error
	// 设置或撤销帖子及作者级不感兴趣。
	SetDiscoverFeedback(ctx context.Context, userId string, input userreq.DiscoverFeedbackRequest) (string, error)
	// 查询时间流、关注流、推荐或热榜并生成下一页游标。
	ListFeed(ctx context.Context, userId string, input userreq.FeedRequest) (*FeedInfo, error)
}

// DiscoverFeedServiceImpl 实现信息流与关注更新业务。
type DiscoverFeedServiceImpl struct {
	discoverDAO      userdao.DiscoverDAO
	distributionDAO  userdao.DiscoverDistributionDAO
	followingReadDAO userdao.FollowingReadDAO
	rankingDAO       userdao.DiscoverRankingDAO
	blockDAO         userdao.BlockDAO
	userInfoDAO      userdao.UserInfoDAO
}

// FeedInfo 信息流展示数据及下一页边界，不对时间流计算总数。
type FeedInfo struct {
	List         []PostInfo
	NextCursor   string
	HasMore      bool
	HasFollowing bool
}

// feedCursor 保存排序边界和查询条件，仅用于翻页，不作为权限依据。
type feedCursor struct {
	AuthorUuid string    `json:"author_uuid,omitempty"`
	Source     string    `json:"source"`
	MediaType  int8      `json:"media_type"`
	UserId     string    `json:"user_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Id         int64     `json:"id"`
	Mode       string    `json:"mode,omitempty"`
	Version    string    `json:"version,omitempty"`
	Position   int       `json:"position,omitempty"`
}

// ErrDiscoverSessionExpired 表示固定候选已过期，需要刷新而不能在翻页途中重新排名。
var ErrDiscoverSessionExpired = errors.New("推荐列表已过期，请刷新后继续浏览")

// rankedFeedSession 保存当前账号固定筛选的候选或热榜引用，不保存权限。
type rankedFeedSession struct {
	UserId    string   `json:"user_id"`
	Source    string   `json:"source"`
	MediaType int8     `json:"media_type"`
	Version   string   `json:"version"`
	Ids       []string `json:"ids,omitempty"`
	HotKey    string   `json:"hot_key,omitempty"`
}

// discoverHotCacheMu 防止本进程并发首屏请求重复生成并覆盖同一版热榜。
var discoverHotCacheMu sync.Mutex

// FollowingAuthorInfo 关注栏作者及确认更新所需的服务端快照。
type FollowingAuthorInfo struct {
	userdao.FollowingAuthorUpdate
	ReadToken string `json:"read_token"`
}

// FollowingUpdatesInfo 未查看更新作者数及关注作者列表。
type FollowingUpdatesInfo struct {
	UnreadAuthors int                   `json:"unread_authors"`
	Authors       []FollowingAuthorInfo `json:"authors"`
}

// ErrFollowingReadExpired 表示更新快照失效，需要重新获取作者状态。
var ErrFollowingReadExpired = errors.New("作者更新快照已失效，请刷新后重试")

// followingReadSnapshot 绑定账号、作者及本次关注关系的已发布作品边界。
type followingReadSnapshot struct {
	User   string `json:"u"`
	Author string `json:"a"`
	Follow int64  `json:"f"`
	Post   int64  `json:"p"`
}

// followingReadKey 优先使用服务端密钥，未配置JWT密钥时生成进程内随机密钥。
var followingReadKey = sync.OnceValue(func() []byte {
	if secret := configs.GetConfig().JWTConfig.Secret; secret != "" {
		return []byte(secret)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("生成关注更新签名密钥失败: " + err.Error())
	}
	return key
})

// NewDiscoverFeedService 构造服务并注入所需依赖。
func NewDiscoverFeedService(
	discoverDAO userdao.DiscoverDAO,
	distributionDAO userdao.DiscoverDistributionDAO,
	followingReadDAO userdao.FollowingReadDAO,
	rankingDAO userdao.DiscoverRankingDAO,
	blockDAO userdao.BlockDAO,
	userInfoDAO userdao.UserInfoDAO,
) DiscoverFeedService {
	return &DiscoverFeedServiceImpl{
		discoverDAO:      discoverDAO,
		distributionDAO:  distributionDAO,
		followingReadDAO: followingReadDAO,
		rankingDAO:       rankingDAO,
		blockDAO:         blockDAO,
		userInfoDAO:      userInfoDAO,
	}
}

// ListFeed 按当前公开范围查询下一批帖子，失败时不返回新游标。
func (s *DiscoverFeedServiceImpl) ListFeed(ctx context.Context, userId string, input userreq.FeedRequest) (*FeedInfo, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录后查看信息流")
	}
	if input.Source == "" {
		input.Source = "all"
	}
	if input.Source != "all" && input.Source != "following" && input.Source != "recommended" && input.Source != "hot" {
		return nil, fmt.Errorf("信息流数据源无效")
	}
	if input.PageSize == 0 {
		input.PageSize = 12
	}
	if input.PageSize < 1 || input.PageSize > 20 {
		return nil, fmt.Errorf("信息流每页数量须为1至20条")
	}
	if len(input.AuthorUuid) > 20 || (input.AuthorUuid != "" && input.Source != "following") {
		return nil, fmt.Errorf("作者筛选只适用于关注信息流")
	}
	boundary := feedCursor{Source: input.Source, MediaType: -1, AuthorUuid: input.AuthorUuid}
	if input.MediaType != nil {
		if *input.MediaType != 0 && *input.MediaType != 1 {
			return nil, fmt.Errorf("信息流媒体类型无效")
		}
		boundary.MediaType = *input.MediaType
	}
	if input.Source == "following" || input.Source == "recommended" || input.Source == "hot" {
		boundary.UserId = userId
	}
	if input.Cursor != "" {
		var err error
		boundary, err = decodeFeedCursor(input.Cursor, boundary)
		if err != nil {
			return nil, err
		}
	}
	if input.Source == "recommended" || input.Source == "hot" {
		return s.listRankedFeed(ctx, userId, input, boundary)
	}
	return s.listTimeFeed(ctx, userId, input, boundary)
}

// RecordDiscoverClick 在同一事务内核查公开帖子并保存按北京时间天去重的点击。
func (s *DiscoverFeedServiceImpl) RecordDiscoverClick(ctx context.Context, userId string, input userreq.DiscoverClickRequest) error {
	if userId == "" {
		return fmt.Errorf("请先登录")
	}
	created := false
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(ctx, input.PostUuid)
		if err != nil {
			return fmt.Errorf("读取点击帖子失败: %w", err)
		}
		if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) || post.Visibility != 0 {
			return fmt.Errorf("帖子不存在或不可见")
		}
		if post.UserId == userId {
			return nil
		}
		now := time.Now()
		created, err = userdao.NewDiscoverRankingDAO(tx).SaveClick(ctx, &models.DiscoverClick{UserId: userId, PostUuid: post.Uuid, Source: input.Source, ClickDay: now.In(time.FixedZone("北京时间", 8*3600)).Format("2006-01-02"), CreatedAt: now})
		if err != nil {
			return fmt.Errorf("保存帖子点击失败: %w", err)
		}
		return nil
	})
	if err == nil && created {
		businessmetrics.RecordDiscoverClick()
	}
	return err
}

// SetDiscoverFeedback 从当前帖子确定作者，在Service事务内锁定并设置幂等反馈。
func (s *DiscoverFeedServiceImpl) SetDiscoverFeedback(ctx context.Context, userId string, input userreq.DiscoverFeedbackRequest) (string, error) {
	if userId == "" || input.Active == nil {
		return "", fmt.Errorf("不感兴趣参数无效")
	}
	if input.TargetType != constants.DiscoverFeedbackPost && input.TargetType != constants.DiscoverFeedbackAuthor {
		return "", fmt.Errorf("不感兴趣目标类型无效")
	}
	authorId := ""
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(ctx, input.PostUuid)
		if err != nil {
			return fmt.Errorf("读取反馈帖子失败: %w", err)
		}
		if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) || post.Visibility != 0 {
			return fmt.Errorf("帖子不存在或不可见")
		}
		if post.UserId == userId {
			return fmt.Errorf("不能对自己的作品设置不感兴趣")
		}
		authorId = post.UserId
		target := post.Uuid
		if input.TargetType == constants.DiscoverFeedbackAuthor {
			target = authorId
		}
		if err := userdao.NewDiscoverRankingDAO(tx).SetFeedback(ctx, &models.DiscoverFeedback{UserId: userId, TargetType: input.TargetType, TargetUuid: target, CreatedAt: time.Now()}, *input.Active); err != nil {
			return fmt.Errorf("保存不感兴趣状态失败: %w", err)
		}
		return nil
	})
	return authorId, err
}

// FollowingUpdates 按作者汇总未查看状态，不依赖异步收件箱是否已写入。
func (s *DiscoverFeedServiceImpl) FollowingUpdates(ctx context.Context, userId string) (*FollowingUpdatesInfo, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录")
	}
	rows, err := s.followingReadDAO.FollowingUpdates(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("查询关注作者更新失败: %w", err)
	}
	result := &FollowingUpdatesInfo{Authors: make([]FollowingAuthorInfo, 0, len(rows))}
	for _, row := range rows {
		item := FollowingAuthorInfo{FollowingAuthorUpdate: row}
		if row.HasUpdate {
			result.UnreadAuthors++
			raw, err := json.Marshal(followingReadSnapshot{userId, row.AuthorUuid, row.FollowId, row.LatestPostId})
			if err != nil {
				return nil, fmt.Errorf("生成作者更新快照失败: %w", err)
			}
			item.ReadToken = base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(signFollowingSnapshot(raw))
		}
		result.Authors = append(result.Authors, item)
	}
	return result, nil
}

// ReadFollowingUpdates 校验签名并锁定关系后前移进度，重复、乱序或重新关注不误清提醒。
func (s *DiscoverFeedServiceImpl) ReadFollowingUpdates(ctx context.Context, userId, token string) error {
	invalid := ErrFollowingReadExpired
	parts := strings.Split(token, ".")
	if len(token) > 2048 || len(parts) != 2 {
		return invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return invalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, signFollowingSnapshot(raw)) {
		return invalid
	}
	var value followingReadSnapshot
	if json.Unmarshal(raw, &value) != nil || userId == "" || value.User != userId || value.Author == "" || value.Post <= 0 {
		return invalid
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		follow, err := userdao.NewFollowDAO(tx).FindFollowForUpdate(ctx, userId, value.Author)
		if err != nil {
			return err
		}
		if follow == nil || follow.Id != value.Follow {
			return ErrFollowingReadExpired
		}
		return userdao.NewFollowingReadDAO(tx).Advance(ctx, follow.Id, value.Post)
	})
}

// encodeFeedCursor 将排序边界和查询条件编码为客户端可传递的游标。
func encodeFeedCursor(value feedCursor) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("生成信息流游标失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// decodeFeedCursor 检查游标格式、排序边界及查询条件，实际权限仍由数据库查询保证。
func decodeFeedCursor(token string, expected feedCursor) (feedCursor, error) {
	invalid := fmt.Errorf("信息流游标无效或与当前筛选不匹配，请刷新列表")
	if len(token) > 2048 {
		return feedCursor{}, invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return feedCursor{}, invalid
	}
	var value feedCursor
	if json.Unmarshal(raw, &value) != nil || value.Source != expected.Source || value.MediaType != expected.MediaType || value.UserId != expected.UserId || value.AuthorUuid != expected.AuthorUuid {
		return feedCursor{}, invalid
	}
	if value.Mode == "ranked" {
		if (expected.Source != "recommended" && expected.Source != "hot") || value.Position < 0 || value.Position > 3*constants.DiscoverRankingPoolLimit {
			return feedCursor{}, invalid
		}
		// 旧版随机会话游标没有版本号，升级后明确引导客户端重新加载。
		if value.Version == "" {
			return feedCursor{}, ErrDiscoverSessionExpired
		}
		if _, err := uuid.Parse(value.Version); err != nil {
			return feedCursor{}, invalid
		}
	} else if (value.Mode != "" && value.Mode != "time") || value.CreatedAt.IsZero() || value.Id <= 0 {
		return feedCursor{}, invalid
	}
	return value, nil
}

// listTimeFeed 复用阶段三时间游标，推荐降级时继续按数据库负反馈过滤。
func (s *DiscoverFeedServiceImpl) listTimeFeed(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor) (*FeedInfo, error) {
	following := input.Source == "following"
	var posts []models.DiscoverPost
	var err error
	if following && input.AuthorUuid != "" {
		posts, err = s.followingReadDAO.ListFollowingAuthorFeed(ctx, userId, input.AuthorUuid, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	} else if input.Source == "recommended" {
		posts, err = s.rankingDAO.ListRecommendationFeed(ctx, userId, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	} else if following {
		started := time.Now()
		var mode string
		posts, mode, err = s.listRedisFollowingFeed(ctx, userId, input, boundary)
		businessmetrics.ObserveDiscoverFollowingQuery(mode, time.Since(started), err)
	} else {
		posts, err = s.discoverDAO.ListFeed(ctx, userId, following, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	}
	if err != nil {
		return nil, fmt.Errorf("查询信息流失败: %w", err)
	}
	result := &FeedInfo{HasMore: len(posts) > input.PageSize}
	if result.HasMore {
		posts = posts[:input.PageSize]
	}
	if following {
		result.HasFollowing = len(posts) > 0
		// 已返回关注作品便可确定存在关注关系，只有空列表需要额外查询。
		if !result.HasFollowing {
			result.HasFollowing, err = s.discoverDAO.HasFollowing(ctx, userId)
			if err != nil {
				return nil, fmt.Errorf("查询关注关系失败: %w", err)
			}
		}
	}
	result.List, _, err = buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, posts, 0)
	if err != nil {
		return nil, err
	}
	if result.HasMore {
		last := posts[len(posts)-1]
		boundary.CreatedAt, boundary.Id = last.CreatedAt, last.Id
		result.NextCursor, err = encodeFeedCursor(boundary)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// listRankedFeed 固定每轮排名，首屏失败降级时间流，后续请求不静默换一轮候选。
func (s *DiscoverFeedServiceImpl) listRankedFeed(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor) (*FeedInfo, error) {
	session := rankedFeedSession{UserId: userId, Source: input.Source, MediaType: boundary.MediaType}
	key := fmt.Sprintf("discover:feed:%s:%s:%d", userId, input.Source, boundary.MediaType)
	if input.Cursor != "" {
		if boundary.Mode == "time" {
			return s.listTimeFeed(ctx, userId, input, boundary)
		}
		if boundary.Mode != "ranked" {
			return nil, fmt.Errorf("推荐游标无效，请刷新列表")
		}
		payload, err := redispkg.GetKeyNilIsErr(key)
		if err != nil || json.Unmarshal([]byte(payload), &session) != nil {
			return nil, ErrDiscoverSessionExpired
		}
		if session.UserId != userId || session.Version != boundary.Version {
			return nil, ErrDiscoverSessionExpired
		}
		if session.Source != input.Source || session.MediaType != boundary.MediaType {
			return nil, fmt.Errorf("推荐游标与当前账号或频道不匹配")
		}
		if session.HotKey != "" {
			payload, err := redispkg.GetKeyNilIsErr(session.HotKey)
			if err != nil || json.Unmarshal([]byte(payload), &session.Ids) != nil {
				return nil, ErrDiscoverSessionExpired
			}
		}
		if boundary.Position > len(session.Ids) {
			return nil, fmt.Errorf("推荐游标位置无效")
		}
	} else {
		var ids []string
		var err error
		if input.Source == "hot" {
			ids, session.HotKey, err = s.createHotRanking(ctx, input.MediaType)
		} else {
			ids, err = s.createRanking(ctx, userId, input.MediaType)
		}
		if err == nil && ids != nil {
			boundary.Mode, boundary.Version = "ranked", uuid.NewString()
			session.Version = boundary.Version
			if session.HotKey == "" {
				session.Ids = ids
			}
			var payload []byte
			payload, err = json.Marshal(session)
			if err == nil {
				err = redispkg.SetKeyEx(key, string(payload), constants.DiscoverRankingSessionTTL)
			}
		}
		if err != nil || ids == nil {
			businessmetrics.RecordDiscoverFeedFallback(input.Source)
			if err != nil {
				zlog.Warn("推荐或热榜首屏降级为时间流，userId=" + userId + ": " + err.Error())
			}
			boundary.Mode = "time"
			boundary.Version, boundary.Position = "", 0
			return s.listTimeFeed(ctx, userId, input, boundary)
		}
		session.Ids = ids
	}
	return s.rankedFeedPage(ctx, userId, input, boundary, session.Ids)
}

// rankedFeedPage 按固定顺序批量核查当前权限，过滤后继续取候选直到填满或耗尽。
func (s *DiscoverFeedServiceImpl) rankedFeedPage(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor, ids []string) (*FeedInfo, error) {
	posts := []models.DiscoverPost{}
	result := &FeedInfo{}
	position := boundary.Position
scanCandidates:
	for position < len(ids) {
		end := min(position+40, len(ids))
		batch, err := s.rankingDAO.FindRankingPosts(ctx, userId, ids[position:end], input.MediaType, input.Source == "recommended")
		if err != nil {
			return nil, fmt.Errorf("核查推荐帖子当前状态失败: %w", err)
		}
		byId := make(map[string]models.DiscoverPost, len(batch))
		for _, post := range batch {
			byId[post.Uuid] = post
		}
		for position < end {
			post, visible := byId[ids[position]]
			if visible && len(posts) == input.PageSize {
				boundary.Position = position
				result.HasMore = true
				var err error
				result.NextCursor, err = encodeFeedCursor(boundary)
				if err != nil {
					return nil, err
				}
				break scanCandidates
			}
			position++
			if visible {
				posts = append(posts, post)
			}
		}
	}
	var err error
	result.List, _, err = buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, posts, 0)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// createHotRanking 首次访问生成共享两小时版本，返回排名和供会话引用的缓存键。
func (s *DiscoverFeedServiceImpl) createHotRanking(ctx context.Context, mediaType *int8) ([]string, string, error) {
	discoverHotCacheMu.Lock()
	defer discoverHotCacheMu.Unlock()
	bucket := time.Now().Truncate(constants.DiscoverHotRefresh)
	media := int8(-1)
	if mediaType != nil {
		media = *mediaType
	}
	key := fmt.Sprintf("discover:hot:%d:%d", media, bucket.Unix())
	if payload, err := redispkg.GetKeyNilIsErr(key); err == nil {
		var ids []string
		if json.Unmarshal([]byte(payload), &ids) == nil && ids != nil {
			// 只延长同版排名的缓存寿命，保证新建的30分钟会话仍可引用，不重新排序。
			err := redispkg.SetKeyEx(key, payload, constants.DiscoverHotRefresh+constants.DiscoverRankingSessionTTL)
			return ids, key, err
		}
	}
	candidates, err := s.rankingDAO.ListRankingCandidates(ctx, "", "hot", nil, mediaType, bucket)
	if err != nil {
		return nil, "", err
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Uuid)
	}
	payload, err := json.Marshal(ids)
	if err != nil {
		return nil, "", fmt.Errorf("编码热榜缓存失败: %w", err)
	}
	err = redispkg.SetKeyEx(key, string(payload), constants.DiscoverHotRefresh+constants.DiscoverRankingSessionTTL)
	return ids, key, err
}

// createRanking 合并近期、热门、兴趣和关注候选，生成当前账号的推荐顺序。
func (s *DiscoverFeedServiceImpl) createRanking(ctx context.Context, userId string, mediaType *int8) ([]string, error) {
	now := time.Now()
	history, err := s.rankingDAO.ListInterestHistory(ctx, userId, now)
	if err != nil {
		return nil, err
	}
	authors, err := s.rankingDAO.ListFollowedAuthorIds(ctx, userId)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 && len(authors) == 0 {
		return nil, nil
	}
	interests := discoverTagInterests(history)
	tags := make([]string, 0, len(interests))
	for tag := range interests {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool {
		if interests[tags[i]] == interests[tags[j]] {
			return tags[i] < tags[j]
		}
		return interests[tags[i]] > interests[tags[j]]
	})
	if len(tags) > 10 {
		tags = tags[:10]
	}
	var candidates []userdao.RankingCandidate
	seen := map[string]bool{}
	for _, kind := range []string{"recent", "hot", "affinity"} {
		rows, err := s.rankingDAO.ListRankingCandidates(ctx, userId, kind, tags, mediaType, now)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if !seen[row.Uuid] {
				seen[row.Uuid] = true
				candidates = append(candidates, row)
			}
		}
	}
	return rankDiscoverCandidates(candidates, interests, authors, now), nil
}

// discoverTagInterests 根据真实互动累计标签权重，按最大权重归一化。
func discoverTagInterests(history []userdao.InterestHistory) map[string]float64 {
	weights := map[string]float64{}
	maxWeight := 0.0
	for _, row := range history {
		var tags []string
		if json.Unmarshal([]byte(row.Tags), &tags) != nil {
			continue
		}
		for _, tag := range tags {
			weights[tag] += row.Weight
			maxWeight = math.Max(maxWeight, weights[tag])
		}
	}
	if maxWeight > 0 {
		for tag := range weights {
			weights[tag] /= maxWeight
		}
	}
	return weights
}

// rankDiscoverCandidates 使用可解释分数并优先错开相邻作者与首标签，不删除同作者候选。
func rankDiscoverCandidates(candidates []userdao.RankingCandidate, interests map[string]float64, authors []string, now time.Time) []string {
	following := map[string]bool{}
	for _, id := range authors {
		following[id] = true
	}
	scores := map[string]float64{}
	topics := map[string]string{}
	for _, row := range candidates {
		var tags []string
		_ = json.Unmarshal(row.Tags, &tags)
		match := 0.0
		for _, tag := range tags {
			match = math.Max(match, interests[tag])
		}
		if len(tags) > 0 {
			topics[row.Uuid] = tags[0]
		}
		ageHours := math.Max(0, now.Sub(row.CreatedAt).Hours())
		score := math.Log1p(row.ActivityScore) + 3*match + 2/(1+ageHours/24)
		if following[row.UserId] {
			score += 1.5
		}
		scores[row.Uuid] = score
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if scores[a.Uuid] != scores[b.Uuid] {
			return scores[a.Uuid] > scores[b.Uuid]
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.Id > b.Id
	})
	ids := make([]string, 0, len(candidates))
	lastAuthor, lastTopic := "", ""
	for len(candidates) > 0 {
		chosen := 0
		// 同话题候选较多时先保证作者错开，再尽量同时错开话题。
		for i, row := range candidates {
			if row.UserId != lastAuthor {
				chosen = i
				break
			}
		}
		for i, row := range candidates {
			if row.UserId != lastAuthor && (lastTopic == "" || topics[row.Uuid] == "" || topics[row.Uuid] != lastTopic) {
				chosen = i
				break
			}
		}
		row := candidates[chosen]
		ids = append(ids, row.Uuid)
		lastAuthor, lastTopic = row.UserId, topics[row.Uuid]
		candidates = append(candidates[:chosen], candidates[chosen+1:]...)
	}
	return ids
}

// signFollowingSnapshot 对阅读快照签名，防止客户端自行确认未来更新。
func signFollowingSnapshot(raw []byte) []byte {
	mac := hmac.New(sha256.New, followingReadKey())
	mac.Write(raw)
	return mac.Sum(nil)
}

// rebuildDiscoverInbox 从帖子和当前关注关系重建近期索引，变更冲突时放弃发布并由请求回源。
func (s *DiscoverFeedServiceImpl) rebuildDiscoverInbox(ctx context.Context, userID string, from time.Time) (bool, error) {
	token := uuid.NewString()
	attempt, cancel := context.WithTimeout(ctx, constants.DiscoverInboxReadTimeout)
	revision, acquired, err := beginDiscoverInboxBuild(attempt, userID, token)
	cancel()
	if err != nil || !acquired {
		return false, err
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), constants.DiscoverInboxReadTimeout)
		defer releaseCancel()
		if err := releaseDiscoverInboxBuild(releaseCtx, userID, token); err != nil {
			zlog.Warn("释放关注收件箱重建锁失败，userId=" + userID + ": " + err.Error())
		}
	}()
	// 不缓存拉黑过滤结果，解除拉黑后仍可从同一索引召回，权限统一在最终查询检查。
	buildCtx, buildCancel := context.WithTimeout(ctx, constants.DiscoverInboxBuildTimeout)
	defer buildCancel()
	posts, err := s.distributionDAO.ListInboxPosts(buildCtx, userID, from, constants.DiscoverInboxCapacity)
	if err != nil {
		return false, err
	}
	members := make([]string, 0, len(posts))
	scores := make([]int64, 0, len(posts))
	for _, post := range posts {
		members = append(members, discoverInboxMember(post))
		scores = append(scores, post.CreatedAt.Unix())
	}
	attempt, cancel = context.WithTimeout(ctx, constants.DiscoverInboxReadTimeout)
	defer cancel()
	return publishDiscoverInboxBuild(attempt, userID, token, revision, members, scores)
}

// listRedisFollowingFeed 尝试Redis混合召回，缓存未就绪、窗口不足或并发变化时直接从MySQL全拉。
func (s *DiscoverFeedServiceImpl) listRedisFollowingFeed(ctx context.Context, userID string, input userreq.FeedRequest, boundary feedCursor) ([]models.DiscoverPost, string, error) {
	mode := "redis"
	fallback := func(reason string) ([]models.DiscoverPost, string, error) {
		businessmetrics.RecordDiscoverInboxRead(reason)
		posts, err := s.discoverDAO.ListFeed(ctx, userID, true, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
		return posts, "fallback", err
	}
	attempt, cancel := context.WithTimeout(ctx, constants.DiscoverInboxReadTimeout)
	snapshot, err := readDiscoverInbox(attempt, userID)
	cancel()
	if err != nil {
		return fallback("error")
	}
	from := discoverDistributionFrom(configs.GetConfig().DiscoverFeedConfig)
	if !snapshot.Ready {
		businessmetrics.RecordDiscoverInboxRead("miss")
		rebuilt, err := s.rebuildDiscoverInbox(ctx, userID, from)
		if err != nil || !rebuilt {
			return fallback("rebuild_fallback")
		}
		attempt, cancel = context.WithTimeout(ctx, constants.DiscoverInboxReadTimeout)
		snapshot, err = readDiscoverInbox(attempt, userID)
		cancel()
		if err != nil || !snapshot.Ready {
			return fallback("error")
		}
		mode = "redis_rebuilt"
	}
	ids := make([]int64, 0, len(snapshot.Members))
	floor, floorID := from, int64(0)
	for i, member := range snapshot.Members {
		id, created, err := parseDiscoverInboxMember(member)
		if err != nil {
			return fallback("invalid")
		}
		ids = append(ids, id)
		if len(snapshot.Members) == constants.DiscoverInboxCapacity && i == len(snapshot.Members)-1 && !created.Before(from) {
			floor, floorID = created, id
		}
	}
	if !boundary.CreatedAt.IsZero() && (boundary.CreatedAt.Before(floor) || (boundary.CreatedAt.Equal(floor) && boundary.Id <= floorID)) {
		return fallback("history")
	}
	posts, err := s.distributionDAO.ListHybridFeed(ctx, userID, ids, input.MediaType, from, floor, boundary.CreatedAt, floorID, boundary.Id, input.PageSize+1)
	if err != nil {
		return fallback("query_error")
	}
	if len(posts) < input.PageSize+1 {
		return fallback("insufficient")
	}
	// 若分发在读取索引后完成，pending分支可能已消失；检测变更后全拉避免这段切换漏帖。
	attempt, cancel = context.WithTimeout(ctx, constants.DiscoverInboxReadTimeout)
	unchanged, err := discoverInboxUnchanged(attempt, userID, snapshot.Revision)
	cancel()
	if err != nil || !unchanged {
		return fallback("changed")
	}
	businessmetrics.RecordDiscoverInboxRead("hit")
	return posts, mode, nil
}
