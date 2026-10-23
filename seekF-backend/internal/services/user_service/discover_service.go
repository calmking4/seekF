package userservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	businessmetrics "seekF-backend/internal/pkg/observability/metrics"
	redispkg "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DiscoverService 发现页内容与互动业务接口。
type DiscoverService interface {
	// 根据已上传图片生成标题和标签候选，不保存或发布作品。
	SuggestDiscoverCreation(ctx context.Context, userId string, input userreq.DiscoverAISuggestionsRequest) (*DiscoverCreationSuggestions, error)
	// 查询关注作者及未查看更新的作者数量。
	FollowingUpdates(ctx context.Context, userId string) (*FollowingUpdatesInfo, error)
	// 确认查看指定作者快照内的更新，不清除后续新帖。
	ReadFollowingUpdates(ctx context.Context, userId, token string) error
	// 记录有效主动点击，排除作者自身浏览并按天去重。
	RecordDiscoverClick(ctx context.Context, userId string, input userreq.DiscoverClickRequest) error
	// 设置或撤销帖子及作者级不感兴趣。
	SetDiscoverFeedback(ctx context.Context, userId string, input userreq.DiscoverFeedbackRequest) (string, error)
	// 查询当前作者的筛选分页作品。
	ListOwnWorks(ctx context.Context, userId string, input userreq.ListOwnWorksRequest) ([]PostInfo, int64, map[int8]int64, error)
	// 仅修改作者作品可见性及版本。
	SetPostVisibility(ctx context.Context, userId string, input userreq.SetPostVisibilityRequest) (uint64, error)
	SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error)
	ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error)
	CreatePost(ctx context.Context, userId string, input userreq.CreatePostRequest) (*PostInfo, error)
	UpdatePost(ctx context.Context, userId, postUuid string, input userreq.CreatePostRequest) (*PostInfo, error)
	DeletePost(ctx context.Context, userId, postUuid string) error
	// 查询时间流、关注流、推荐或热榜并生成下一页游标。
	ListFeed(ctx context.Context, userId string, input userreq.FeedRequest) (*FeedInfo, error)
	ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error)
	AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error)
	ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error)
	AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error)
	// 搜索帖子（ES全文搜索）
	SearchPosts(ctx context.Context, userId string, keyword string, page, pageSize int) ([]PostInfo, int64, error)
	// 获取用户主页信息
	GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error)

	// 收藏夹
	CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error)
	UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error
	DeleteFolder(ctx context.Context, userId, folderUuid string) error
	ListFolders(ctx context.Context, userId, viewerId string) ([]FolderInfo, error)
	GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error)
	ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int) ([]PostInfo, int64, error)

	// 收藏
	CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error)
}

type DiscoverServiceImpl struct {
	discoverDAO     userdao.DiscoverDAO
	userInfoDAO     userdao.UserInfoDAO
	followDAO       userdao.FollowDAO
	notificationDAO userdao.NotificationDAO
}

// MediaSize 媒体宽高，单位像素，零表示未知。
type MediaSize struct {
	Width  int
	Height int
}

// PostInfo 帖子列表展示信息。
type PostInfo struct {
	IsBlockedBy      bool
	ReviewNote       string
	SupportOnly      int8
	CanPlay          bool
	LockReason       string
	SupportExpiresAt string
	Status           int8
	ViewCount        int
	Visibility       int8
	Version          uint64
	Width            int
	Height           int
	Uuid             string
	UserId           string
	Nickname         string
	Avatar           string
	Title            string
	Content          string
	MediaType        int8
	CoverUrl         string
	Tags             []string
	FirstUrl         string
	LikeCount        int
	CommentCount     int
	CollectCount     int
	IsLiked          bool
	IsCollected      bool
	CreatedAt        string
}

type PostDetailInfo struct {
	SupportOnly      int8
	CanPlay          bool
	LockReason       string
	SupportExpiresAt string
	Visibility       int8
	Version          uint64
	Uuid             string
	UserId           string
	Nickname         string
	Avatar           string
	Title            string
	Content          string
	MediaType        int8
	CoverUrl         string
	Tags             []string
	Urls             []string
	LikeCount        int
	CommentCount     int
	CollectCount     int
	ViewCount        int
	IsLiked          bool
	IsCollected      bool
	CreatedAt        string
}

// CommentInfo 评论展示及回复数量信息。
type CommentInfo struct {
	ReplyCount      int64
	Uuid            string
	UserId          string
	Nickname        string
	Avatar          string
	ParentId        string
	ReplyToUserId   string
	ReplyToNickname string
	Content         string
	LikeCount       int
	IsLiked         bool
	CreatedAt       string
}

type FolderInfo struct {
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CoverUrl    string
	CreatedAt   string
}

type FolderDetailInfo struct {
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CreatedAt   string
}

// NewDiscoverService 构造发现页业务服务并注入数据访问依赖。
func NewDiscoverService(discoverDAO userdao.DiscoverDAO, userInfoDAO userdao.UserInfoDAO, followDAO userdao.FollowDAO, notificationDAO userdao.NotificationDAO) DiscoverService {
	return &DiscoverServiceImpl{
		discoverDAO:     discoverDAO,
		userInfoDAO:     userInfoDAO,
		followDAO:       followDAO,
		notificationDAO: notificationDAO,
	}
}

// CreatePost 在事务中校验资源归属并幂等发布作品。
func (s *DiscoverServiceImpl) CreatePost(ctx context.Context, userId string, input userreq.CreatePostRequest) (*PostInfo, error) {
	return s.saveDiscoverPost(ctx, userId, "", input)
}

// ========== 信息流、推荐与行为反馈 ==========

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

// ListFeed 按当前公开范围查询下一批帖子，失败时不返回新游标。
func (s *DiscoverServiceImpl) ListFeed(ctx context.Context, userId string, input userreq.FeedRequest) (*FeedInfo, error) {
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

// listTimeFeed 复用阶段三时间游标，推荐降级时继续按数据库负反馈过滤。
func (s *DiscoverServiceImpl) listTimeFeed(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor) (*FeedInfo, error) {
	following := input.Source == "following"
	var posts []models.DiscoverPost
	var err error
	if following && input.AuthorUuid != "" {
		posts, err = s.discoverDAO.ListFollowingAuthorFeed(ctx, userId, input.AuthorUuid, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	} else if input.Source == "recommended" {
		posts, err = s.discoverDAO.ListRecommendationFeed(ctx, userId, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	} else if following && configs.GetConfig().DiscoverFeedConfig.Enabled {
		started := time.Now()
		posts, err = s.discoverDAO.ListHybridFeed(ctx, userId, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
		mode := "hybrid"
		if err != nil {
			zlog.Warn("关注混合流查询失败，回退读时查询，userId=" + userId + ": " + err.Error())
			mode = "fallback"
			posts, err = s.discoverDAO.ListFeed(ctx, userId, true, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
		}
		businessmetrics.ObserveDiscoverFollowingQuery(mode, time.Since(started), err)
	} else {
		started := time.Now()
		posts, err = s.discoverDAO.ListFeed(ctx, userId, following, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
		if following {
			businessmetrics.ObserveDiscoverFollowingQuery("pull", time.Since(started), err)
		}
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
	result.List, _, err = s.buildPostInfoList(ctx, userId, posts, 0)
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

// RecordDiscoverClick 在同一事务内核查公开帖子并保存按北京时间天去重的点击。
func (s *DiscoverServiceImpl) RecordDiscoverClick(ctx context.Context, userId string, input userreq.DiscoverClickRequest) error {
	if userId == "" {
		return fmt.Errorf("请先登录")
	}
	created := false
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(input.PostUuid)
		if err != nil {
			return fmt.Errorf("读取点击帖子失败: %w", err)
		}
		if !canViewUnblockedDiscoverPost(tx, post, userId) || post.Visibility != 0 {
			return fmt.Errorf("帖子不存在或不可见")
		}
		if post.UserId == userId {
			return nil
		}
		now := time.Now()
		created, err = dao.SaveClick(&models.DiscoverClick{UserId: userId, PostUuid: post.Uuid, Source: input.Source, ClickDay: now.In(time.FixedZone("北京时间", 8*3600)).Format("2006-01-02"), CreatedAt: now})
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
func (s *DiscoverServiceImpl) SetDiscoverFeedback(ctx context.Context, userId string, input userreq.DiscoverFeedbackRequest) (string, error) {
	if userId == "" || input.Active == nil {
		return "", fmt.Errorf("不感兴趣参数无效")
	}
	if input.TargetType != constants.DiscoverFeedbackPost && input.TargetType != constants.DiscoverFeedbackAuthor {
		return "", fmt.Errorf("不感兴趣目标类型无效")
	}
	authorId := ""
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(input.PostUuid)
		if err != nil {
			return fmt.Errorf("读取反馈帖子失败: %w", err)
		}
		if !canViewUnblockedDiscoverPost(tx, post, userId) || post.Visibility != 0 {
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
		if err := dao.SetFeedback(&models.DiscoverFeedback{UserId: userId, TargetType: input.TargetType, TargetUuid: target, CreatedAt: time.Now()}, *input.Active); err != nil {
			return fmt.Errorf("保存不感兴趣状态失败: %w", err)
		}
		return nil
	})
	return authorId, err
}

// listRankedFeed 固定每轮排名，首屏失败降级时间流，后续请求不静默换一轮候选。
func (s *DiscoverServiceImpl) listRankedFeed(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor) (*FeedInfo, error) {
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
func (s *DiscoverServiceImpl) rankedFeedPage(ctx context.Context, userId string, input userreq.FeedRequest, boundary feedCursor, ids []string) (*FeedInfo, error) {
	posts := []models.DiscoverPost{}
	result := &FeedInfo{}
	position := boundary.Position
scanCandidates:
	for position < len(ids) {
		end := min(position+40, len(ids))
		batch, err := s.discoverDAO.FindRankingPosts(ctx, userId, ids[position:end], input.MediaType, input.Source == "recommended")
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
	result.List, _, err = s.buildPostInfoList(ctx, userId, posts, 0)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// createHotRanking 首次访问生成共享两小时版本，返回排名和供会话引用的缓存键。
func (s *DiscoverServiceImpl) createHotRanking(ctx context.Context, mediaType *int8) ([]string, string, error) {
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
	candidates, err := s.discoverDAO.ListRankingCandidates(ctx, "", "hot", nil, mediaType, bucket)
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
func (s *DiscoverServiceImpl) createRanking(ctx context.Context, userId string, mediaType *int8) ([]string, error) {
	now := time.Now()
	history, err := s.discoverDAO.ListInterestHistory(ctx, userId, now)
	if err != nil {
		return nil, err
	}
	authors, err := s.discoverDAO.ListFollowedAuthorIds(ctx, userId)
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
		rows, err := s.discoverDAO.ListRankingCandidates(ctx, userId, kind, tags, mediaType, now)
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

// ========== 点赞作品 ==========

// ListLikedPosts 查询用户点赞的帖子及媒体信息。
func (s *DiscoverServiceImpl) ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListLikedPosts(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountLikedPosts(userId)
	if err != nil {
		return nil, 0, err
	}

	return s.buildPostInfoList(ctx, userId, posts, total)
}

// GetPostDetail 查询可用帖子详情及当前用户互动状态。
func (s *DiscoverServiceImpl) GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(uuid)
	if err != nil {
		return nil, err
	}
	// 作者可以在管理页查看待审、驳回或下架作品，其他用户仍仅能查看正常公开作品。
	if post != nil && post.UserId != userId {
		blocked, by, err := userdao.NewBlockDAO(db.GormDB.WithContext(ctx)).Status(userId, post.UserId)
		if err != nil {
			return nil, fmt.Errorf("查询帖子拉黑权限失败: %w", err)
		}
		if by {
			return nil, fmt.Errorf("已被拉黑")
		}
		if blocked {
			return nil, fmt.Errorf("你已拉黑该用户")
		}
	}
	if !canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), post, userId) && (post == nil || post.UserId != userId || post.DeletedAt.Valid) {
		return nil, fmt.Errorf("帖子不存在")
	}
	expiries, err := discoverSupportExpiries(ctx, userId, []models.DiscoverPost{*post})
	if err != nil {
		return nil, err
	}
	canPlay, lockReason, expiresAt := discoverPlaybackAccess(*post, userId, expiries[post.UserId], time.Now())
	// 无权益时仅返回助力提示所需信息，不装配正文、媒体或互动数据。
	if !canPlay {
		return &PostDetailInfo{Uuid: post.Uuid, UserId: post.UserId, SupportOnly: post.SupportOnly, CanPlay: false, LockReason: lockReason, SupportExpiresAt: expiresAt}, nil
	}

	// 异步增加浏览量
	go func() {
		if err := s.discoverDAO.IncrementViewCount(post.Id); err != nil {
			zlog.Error("增加帖子浏览量失败: " + err.Error())
		}
	}()

	mediaList, err := s.discoverDAO.FindMediaByPostId(post.Id)
	if err != nil {
		return nil, fmt.Errorf("查询帖子媒体失败: %w", err)
	}
	urls := make([]string, 0, len(mediaList))
	for _, m := range mediaList {
		if canPlay {
			urls = append(urls, m.Url)
		}
	}

	var tags []string
	if len(post.Tags) > 0 {
		json.Unmarshal(post.Tags, &tags)
	}

	user, _ := s.userInfoDAO.FindUserByUuid(post.UserId)
	nickname := ""
	avatar := ""
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	isLiked := false
	like, _ := s.discoverDAO.FindLike(userId, uuid)
	if like != nil {
		isLiked = true
	}

	isCollected := false
	col, _ := s.discoverDAO.FindCollectionByUserAndTarget(userId, uuid)
	if col != nil {
		isCollected = true
	}

	return &PostDetailInfo{
		SupportOnly: post.SupportOnly, CanPlay: canPlay, LockReason: lockReason, SupportExpiresAt: expiresAt,
		Visibility: post.Visibility, Version: post.Version,
		Uuid:         post.Uuid,
		UserId:       post.UserId,
		Nickname:     nickname,
		Avatar:       avatar,
		Title:        post.Title,
		Content:      post.Content,
		MediaType:    post.MediaType,
		CoverUrl:     post.CoverUrl,
		Tags:         tags,
		Urls:         urls,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		CollectCount: post.CollectCount,
		ViewCount:    post.ViewCount,
		IsLiked:      isLiked,
		IsCollected:  isCollected,
		CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// SetLike 在同一事务中设置点赞状态、校准计数并写入通知，重复请求保持幂等。
func (s *DiscoverServiceImpl) SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error) {
	var count int
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		var targetId int64
		var owner string
		if isComment {
			comment, err := txDiscoverDAO.FindCommentByUuidForUpdate(targetUuid)
			if err != nil {
				return fmt.Errorf("查询评论失败: %w", err)
			}
			if comment == nil {
				return fmt.Errorf("评论不存在")
			}
			post, err := txDiscoverDAO.FindPostById(comment.PostId)
			if err != nil {
				return fmt.Errorf("查询评论所属帖子失败: %w", err)
			}
			if !canViewUnblockedDiscoverPost(tx, post, userId) {
				return fmt.Errorf("评论所属帖子不可用")
			}
			if err := requireDiscoverSupport(ctx, tx, userId, post); err != nil {
				return err
			}
			targetId, owner = comment.Id, comment.UserId
		} else {
			post, err := txDiscoverDAO.FindPostByUuidForUpdate(targetUuid)
			if err != nil {
				return fmt.Errorf("查询帖子失败: %w", err)
			}
			if !canViewUnblockedDiscoverPost(tx, post, userId) {
				return fmt.Errorf("帖子不存在或已下架")
			}
			targetId, owner = post.Id, post.UserId
		}
		// 先锁定目标再读取当前状态，避免同一目标并发请求重复增删和通知。
		existing, err := txDiscoverDAO.FindLike(userId, targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞状态失败: %w", err)
		}
		changed := liked != (existing != nil)
		if changed {
			if liked {
				if err := txDiscoverDAO.CreateLike(&models.DiscoverLike{UserId: userId, TargetUuid: targetUuid}); err != nil {
					return fmt.Errorf("保存点赞失败: %w", err)
				}
			} else {
				if err := txDiscoverDAO.DeleteLike(userId, targetUuid); err != nil {
					return fmt.Errorf("取消点赞失败: %w", err)
				}
			}
		}
		// 保留按真实唯一用户数校准计数的行为，兼容遗留重复记录。
		count, err = txDiscoverDAO.CountLikesByTargetUuid(targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞数量失败: %w", err)
		}
		if isComment {
			err = txDiscoverDAO.UpdateCommentLikeCount(targetId, count)
		} else {
			err = txDiscoverDAO.UpdatePostLikeCount(targetId, count)
		}
		if err != nil {
			return fmt.Errorf("更新点赞数量失败: %w", err)
		}
		if changed && liked && owner != userId {
			notificationType, content := NotificationTypeLikePost, "赞了你的帖子"
			if isComment {
				notificationType, content = NotificationTypeLikeComment, "赞了你的评论"
			}
			// 通知写入失败时一起回滚点赞与计数，避免业务成功但通知丢失。
			if err := userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: owner, ActorId: userId, Type: notificationType, TargetUuid: targetUuid, Content: content}); err != nil {
				return fmt.Errorf("创建点赞通知失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

// AddComment 保存评论并通知作者或被回复者。
func (s *DiscoverServiceImpl) AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error) {
	var post *models.DiscoverPost
	replyRecipient := replyToUserId

	commentUUID := "C" + util.GetNowAndLenRandomString(11)
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		UserId:        userId,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
	}

	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		var err error
		post, err = txDiscoverDAO.FindPostByUuidForUpdate(postUuid)
		if err != nil {
			return err
		}
		if !canViewUnblockedDiscoverPost(tx, post, userId) {
			return fmt.Errorf("帖子不存在或已下架")
		}
		if err := requireDiscoverSupport(ctx, tx, userId, post); err != nil {
			return err
		}
		if parentUuid != "" {
			parent, err := txDiscoverDAO.FindCommentByUuid(parentUuid)
			if err != nil {
				return fmt.Errorf("查询被回复评论失败: %w", err)
			}
			if parent == nil || parent.PostId != post.Id {
				return fmt.Errorf("被回复的评论不存在或不属于当前帖子")
			}
			if replyRecipient == "" {
				replyRecipient = parent.UserId
			}
		}
		comment.PostId = post.Id
		if err := txDiscoverDAO.CreateComment(comment); err != nil {
			return fmt.Errorf("评论失败: %v", err)
		}
		if err := txDiscoverDAO.IncrementCommentCount(post.Id); err != nil {
			return err
		}
		recipient, notificationType, text := post.UserId, NotificationTypeCommentPost, "评论了你的帖子："
		if parentUuid != "" {
			recipient, notificationType, text = replyRecipient, NotificationTypeReply, "回复了你的评论："
		}
		if recipient != "" && recipient != userId {
			return userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: recipient, ActorId: userId, Type: notificationType, TargetUuid: commentUUID, Content: text + truncateContent(content, 50)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	user, _ := s.userInfoDAO.FindUserByUuid(userId)
	nickname := ""
	avatar := ""
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	return &CommentInfo{
		Uuid:          commentUUID,
		UserId:        userId,
		Nickname:      nickname,
		Avatar:        avatar,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
		CreatedAt:     comment.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// truncateContent 截断内容，超过 maxLen 加省略号
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// ListComments 兼容旧客户端的全部评论分页。
func (s *DiscoverServiceImpl) ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if !canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), post, userId) {
		return nil, fmt.Errorf("帖子不存在或已下架")
	}
	if err := requireDiscoverSupport(ctx, db.GormDB, userId, post); err != nil {
		return nil, err
	}
	comments, err := s.discoverDAO.FindCommentsByPostId(post.Id, page, pageSize)
	if err != nil {
		return nil, err
	}
	return s.buildCommentInfoList(userId, comments)
}

// ListCommentThread 分页查询顶级评论或指定评论的回复，校验父评论归属。
func (s *DiscoverServiceImpl) ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, false, err
	}
	if !canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), post, userId) {
		return nil, false, fmt.Errorf("帖子不存在或已下架")
	}
	if err := requireDiscoverSupport(ctx, db.GormDB, userId, post); err != nil {
		return nil, false, err
	}
	if parentUuid != "" {
		parent, err := s.discoverDAO.FindCommentByUuid(parentUuid)
		if err != nil {
			return nil, false, err
		}
		if parent == nil || parent.PostId != post.Id || parent.ParentId != "" {
			return nil, false, fmt.Errorf("父评论不存在或不属于当前帖子")
		}
	}
	comments, hasMore, err := s.discoverDAO.FindCommentThread(post.Id, parentUuid, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	infos, err := s.buildCommentInfoList(userId, comments)
	return infos, hasMore, err
}

// buildCommentInfoList 批量装配评论作者、回复数量与当前用户点赞状态。
func (s *DiscoverServiceImpl) buildCommentInfoList(userId string, comments []models.DiscoverComment) ([]CommentInfo, error) {
	result := make([]CommentInfo, 0, len(comments))
	if len(comments) == 0 {
		return result, nil
	}
	userIds, uuids := make([]string, 0), make([]string, 0, len(comments))
	for _, c := range comments {
		userIds = append(userIds, c.UserId)
		if c.ReplyToUserId != "" {
			userIds = append(userIds, c.ReplyToUserId)
		}
		uuids = append(uuids, c.Uuid)
	}
	users, err := s.userInfoDAO.FindUsersByUuids(userIds)
	if err != nil {
		return nil, fmt.Errorf("查询评论用户失败: %w", err)
	}
	userMap := make(map[string]models.UserInfo)
	for _, u := range users {
		userMap[u.Uuid] = u
	}
	likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, uuids)
	if err != nil {
		return nil, fmt.Errorf("查询评论点赞失败: %w", err)
	}
	liked := make(map[string]bool)
	for _, like := range likes {
		liked[like.TargetUuid] = true
	}
	replies, err := s.discoverDAO.CountReplies(uuids)
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		user, replyUser := userMap[c.UserId], userMap[c.ReplyToUserId]
		result = append(result, CommentInfo{Uuid: c.Uuid, UserId: c.UserId, Nickname: user.Nickname, Avatar: user.Avatar, ParentId: c.ParentId, ReplyToUserId: c.ReplyToUserId, ReplyToNickname: replyUser.Nickname, Content: c.Content, LikeCount: c.LikeCount, IsLiked: liked[c.Uuid], ReplyCount: replies[c.Uuid], CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return result, nil
}

// AddAIComment 保存用户评论并触发 AI 回复任务。
func (s *DiscoverServiceImpl) AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error) {
	// 1. 保存用户评论（完整内容含@AI助手前缀）
	userComment, err := s.AddComment(ctx, userId, postUuid, parentUuid, replyToUserId, content)
	if err != nil {
		return nil, err
	}

	// 2. 确定AI评论的父级和回复目标
	aiParentUuid := parentUuid
	aiReplyToUserId := ""
	if aiParentUuid == "" {
		// 用户发的是顶级评论，AI回复嵌套在用户评论下，不显示"回复 XXX:"
		aiParentUuid = userComment.Uuid
	} else {
		// 用户回复别人时@AI，AI回复显示"回复 [用户昵称]:"
		aiReplyToUserId = userId
	}

	// 3. 发送Kafka任务
	aipkg.SendAICommentTask(aipkg.AICommentPayload{
		PostUuid:       postUuid,
		Content:        aiQuestion,
		ParentUuid:     aiParentUuid,
		ReplyToUserId:  aiReplyToUserId,
		ReplyToContent: replyToContent,
	})

	return userComment, nil
}

// ========== 收藏夹 ==========

func (s *DiscoverServiceImpl) CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error) {
	folderUUID := "F" + util.GetNowAndLenRandomString(11)
	folder := &models.DiscoverCollectionFolder{
		Uuid:        folderUUID,
		UserId:      userId,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := s.discoverDAO.CreateFolder(folder); err != nil {
		return nil, fmt.Errorf("创建收藏夹失败: %v", err)
	}
	return &FolderInfo{
		Uuid:        folderUUID,
		Name:        name,
		Description: description,
		IsPublic:    isPublic == 1,
		PostCount:   0,
		CreatedAt:   folder.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权修改此收藏夹")
	}
	return s.discoverDAO.UpdateFolder(folderUuid, name, description, isPublic)
}

func (s *DiscoverServiceImpl) DeleteFolder(ctx context.Context, userId, folderUuid string) error {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权删除此收藏夹")
	}
	return s.discoverDAO.DeleteFolder(folderUuid)
}

// ListFolders 按查看者权限装配收藏夹封面与列表。
func (s *DiscoverServiceImpl) ListFolders(ctx context.Context, userId, viewerId string) ([]FolderInfo, error) {
	folders, err := s.discoverDAO.ListFoldersByUserId(userId)
	if err != nil {
		return nil, err
	}
	var result []FolderInfo
	for _, f := range folders {
		// 获取收藏夹第一张图片作为封面
		coverUrl := ""
		posts, _ := s.discoverDAO.ListCollectedPostsByFolder(f.Id, viewerId, 1, 1)
		if len(posts) > 0 && canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), &posts[0], viewerId) {
			coverUrl = posts[0].CoverUrl
			if coverUrl == "" && posts[0].MediaType != 1 {
				mediaList, _ := s.discoverDAO.FindMediaByPostId(posts[0].Id)
				if len(mediaList) > 0 {
					coverUrl = mediaList[0].Url
				}
			}
		}
		result = append(result, FolderInfo{
			Uuid:        f.Uuid,
			Name:        f.Name,
			Description: f.Description,
			IsPublic:    f.IsPublic == 1,
			PostCount:   f.PostCount,
			CoverUrl:    coverUrl,
			CreatedAt:   f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func (s *DiscoverServiceImpl) GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, fmt.Errorf("无权查看私密收藏夹")
	}
	return &FolderDetailInfo{
		Uuid:        folder.Uuid,
		Name:        folder.Name,
		Description: folder.Description,
		IsPublic:    folder.IsPublic == 1,
		PostCount:   folder.PostCount,
		CreatedAt:   folder.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ListCollectedPosts 查询收藏夹内的帖子及互动状态。
func (s *DiscoverServiceImpl) ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int) ([]PostInfo, int64, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return nil, 0, err
	}
	if folder == nil {
		return nil, 0, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, 0, fmt.Errorf("无权查看私密收藏夹")
	}

	posts, err := s.discoverDAO.ListCollectedPostsByFolder(folder.Id, userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountCollectedPostsByFolder(folder.Id, userId)
	if err != nil {
		return nil, 0, err
	}

	return s.buildPostInfoList(ctx, userId, posts, total)
}

// ========== 收藏/取消收藏 ==========

// CollectPost 保存收藏及通知，并在事务内维护收藏夹和帖子计数。
func (s *DiscoverServiceImpl) CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return false, 0, err
	}
	if !canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), post, userId) {
		return false, 0, fmt.Errorf("帖子不存在")
	}

	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return false, 0, err
	}
	if folder == nil {
		return false, 0, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return false, 0, fmt.Errorf("无权操作此收藏夹")
	}

	// 检查是否已在该收藏夹中收藏
	existing, _ := s.discoverDAO.FindCollectionInFolder(userId, folder.Id, postUuid)
	if existing != nil {
		return false, 0, fmt.Errorf("该帖子已在此收藏夹中")
	}
	previousCollection, err := s.discoverDAO.FindCollectionByUserAndTarget(userId, postUuid)
	if err != nil {
		return false, 0, fmt.Errorf("查询收藏状态失败: %w", err)
	}

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		col := &models.DiscoverCollection{
			UserId:     userId,
			FolderId:   folder.Id,
			TargetUuid: postUuid,
		}
		if err := txDiscoverDAO.CreateCollection(col); err != nil {
			return err
		}
		// 收藏多个文件夹时只在首次收藏发送通知，与收藏写入保持同一事务。
		if previousCollection == nil && post.UserId != userId {
			if err := userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{
				UserId: post.UserId, ActorId: userId, Type: NotificationTypeCollectPost,
				TargetUuid: postUuid, Content: "收藏了你的帖子：" + truncateContent(post.Title, 50),
			}); err != nil {
				return fmt.Errorf("创建收藏通知失败: %w", err)
			}
		}

		if err := txDiscoverDAO.IncrementFolderPostCount(folder.Id); err != nil {
			return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
		}
		if err := txDiscoverDAO.IncrementCollectCount(post.Id); err != nil {
			return fmt.Errorf("更新帖子收藏数失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
	collectCount := 0
	if updatedPost != nil {
		collectCount = updatedPost.CollectCount
	}
	return true, collectCount, nil
}

// UncollectPost 删除收藏并在事务内维护收藏夹和帖子计数。
func (s *DiscoverServiceImpl) UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return false, 0, err
	}
	if folder == nil {
		return false, 0, fmt.Errorf("收藏夹不存在")
	}

	existing, _ := s.discoverDAO.FindCollectionInFolder(userId, folder.Id, postUuid)
	if existing == nil {
		return false, 0, fmt.Errorf("未收藏该帖子")
	}

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		if err := txDiscoverDAO.DeleteCollection(userId, folder.Id, postUuid); err != nil {
			return err
		}

		if err := txDiscoverDAO.DecrementFolderPostCount(folder.Id); err != nil {
			return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
		}

		post, err := txDiscoverDAO.FindPostByUuid(postUuid)
		if err != nil {
			return fmt.Errorf("查询收藏帖子失败: %w", err)
		}
		if post != nil {
			if err := txDiscoverDAO.DecrementCollectCount(post.Id); err != nil {
				return fmt.Errorf("更新帖子收藏数失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
	if canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), updatedPost, userId) {
		return false, updatedPost.CollectCount, nil
	}
	return false, 0, nil
}

// CheckCollected 仅对当前身份可访问的作品查询收藏状态。
func (s *DiscoverServiceImpl) CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return false, "", err
	}
	if !canViewUnblockedDiscoverPost(db.GormDB.WithContext(ctx), post, userId) {
		return false, "", fmt.Errorf("作品不存在或不可见")
	}
	col, _ := s.discoverDAO.FindCollectionByUserAndTarget(userId, postUuid)
	if col == nil {
		return false, "", nil
	}
	// 通过 folderId 查找收藏夹
	var folder models.DiscoverCollectionFolder
	result := db.GormDB.Where("id = ?", col.FolderId).First(&folder)
	if result.Error != nil {
		return true, "", nil
	}
	return true, folder.Uuid, nil
}

// SearchPosts 合并全文和实时作者昵称候选后核对权限并分页，积压或ES故障时回退数据库。
func (s *DiscoverServiceImpl) SearchPosts(ctx context.Context, userId, keyword string, page, pageSize int) ([]PostInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		pending, pendingErr := userdao.NewDiscoverOutboxDAO(db.GormDB.WithContext(ctx)).HasIncomplete()
		if pendingErr == nil && !pending {
			searchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			ids, searchErr := userdao.NewESDiscoverDAO().SearchPostIDs(searchCtx, keyword)
			cancel()
			if searchErr == nil {
				authorIDs, authorErr := s.discoverDAO.SearchAuthorPostIDs(ctx, keyword, userId)
				if authorErr != nil {
					return nil, 0, authorErr
				}
				// 候选过多时不截断作者作品，改用数据库完整分页；去重在权限核对后完成。
				if len(authorIDs) <= constants.DiscoverSearchCandidateLimit && len(ids)+len(authorIDs) > 0 {
					posts, total, verifyErr := s.filterDiscoverSearch(ctx, userId, append(ids, authorIDs...), page, pageSize)
					if verifyErr != nil {
						return nil, 0, verifyErr
					}
					return s.buildPostInfoList(ctx, userId, posts, total)
				}
			}
			if searchErr != nil && ctx.Err() == nil {
				zlog.Warn("全文搜索降级为数据库查询: " + searchErr.Error())
			}
		}
	}
	posts, total, err := s.discoverDAO.SearchPostsPage(keyword, page, pageSize, userId)
	if err != nil {
		return nil, 0, err
	}
	return s.buildPostInfoList(ctx, userId, posts, total)
}

// filterDiscoverSearch 对候选先核对当前权限再分页，避免私密候选占位导致短页或错误总数。
func (s *DiscoverServiceImpl) filterDiscoverSearch(ctx context.Context, userId string, ids []string, page, pageSize int) ([]models.DiscoverPost, int64, error) {
	visible := make(map[string]models.DiscoverPost, len(ids))
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		posts, err := s.discoverDAO.FindPublicPostsByUuids(ctx, ids[start:end], userId)
		if err != nil {
			return nil, 0, fmt.Errorf("核对搜索作品公开范围失败: %w", err)
		}
		for _, post := range posts {
			visible[post.Uuid] = post
		}
	}
	ordered := make([]models.DiscoverPost, 0, len(visible))
	for _, id := range ids {
		if post, ok := visible[id]; ok {
			ordered = append(ordered, post)
			delete(visible, id)
		}
	}
	total := int64(len(ordered))
	if int64(page) > total/int64(pageSize)+1 {
		return []models.DiscoverPost{}, total, nil
	}
	offset := (int64(page) - 1) * int64(pageSize)
	if offset >= total {
		return []models.DiscoverPost{}, total, nil
	}
	end := offset + int64(pageSize)
	if end > total {
		end = total
	}
	return ordered[offset:end], total, nil
}

// buildPostInfoList 批量复核反向拉黑并构造正常帖子或不含原始内容的占位列表。
func (s *DiscoverServiceImpl) buildPostInfoList(ctx context.Context, userId string, posts []models.DiscoverPost, total int64) ([]PostInfo, int64, error) {
	if len(posts) == 0 {
		return []PostInfo{}, total, nil
	}
	expiries, err := discoverSupportExpiries(ctx, userId, posts)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()

	postIds := make([]int64, 0, len(posts))
	userIds := make([]string, 0, len(posts))
	postUuids := make([]string, 0, len(posts))
	for _, post := range posts {
		postIds = append(postIds, post.Id)
		userIds = append(userIds, post.UserId)
		postUuids = append(postUuids, post.Uuid)
	}

	blockedBy, err := userdao.NewBlockDAO(db.GormDB.WithContext(ctx)).BlockedByAuthors(userId, userIds)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子占位权限失败: %w", err)
	}

	// 批量查询媒体
	mediaList, err := s.discoverDAO.FindMediaByPostIds(postIds)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子媒体失败: %w", err)
	}
	mediaMap := make(map[int64]userdao.DiscoverMediaInfo)
	for _, media := range mediaList {
		if _, exists := mediaMap[media.PostId]; !exists {
			mediaMap[media.PostId] = media
		}
	}

	// 批量查询用户信息
	users, err := s.userInfoDAO.FindUsersByUuids(userIds)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子作者失败: %w", err)
	}
	userMap := make(map[string]*models.UserInfo)
	for i := range users {
		userMap[users[i].Uuid] = &users[i]
	}

	// 批量查询点赞状态
	likedMap := make(map[string]bool)
	if userId != "" {
		likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, fmt.Errorf("查询帖子点赞状态失败: %w", err)
		}
		for _, like := range likes {
			likedMap[like.TargetUuid] = true
		}
	}

	// 批量查询收藏状态
	collectedMap := make(map[string]bool)
	if userId != "" {
		collections, err := s.discoverDAO.FindCollectionsByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, fmt.Errorf("查询帖子收藏状态失败: %w", err)
		}
		for _, col := range collections {
			collectedMap[col.TargetUuid] = true
		}
	}

	var result []PostInfo
	for _, post := range posts {
		if blockedBy[post.UserId] {
			nickname, avatar := "", ""
			if author := userMap[post.UserId]; author != nil {
				nickname, avatar = author.Nickname, author.Avatar
			}
			result = append(result, PostInfo{Uuid: post.Uuid, UserId: post.UserId, Nickname: nickname, Avatar: avatar, Title: "已被拉黑", IsBlockedBy: true, LockReason: "已被拉黑", Width: 4, Height: 3, CreatedAt: post.CreatedAt.Format("2006-01-02 15:04:05")})
			continue
		}
		firstMedia := mediaMap[post.Id]
		firstUrl := firstMedia.Url
		canPlay, lockReason, expiresAt := discoverPlaybackAccess(post, userId, expiries[post.UserId], now)
		content := post.Content
		if !canPlay {
			firstUrl = ""
			if post.SupportOnly == 1 {
				content = ""
			}
		}
		cardSize := discoverCardSize(post, firstMedia)

		var tags []string
		if len(post.Tags) > 0 {
			json.Unmarshal(post.Tags, &tags)
		}

		nickname := ""
		avatar := ""
		if user, exists := userMap[post.UserId]; exists {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		reviewNote := ""
		if post.UserId == userId {
			reviewNote = post.ReviewNote
		}
		result = append(result, PostInfo{
			ReviewNote:  reviewNote,
			SupportOnly: post.SupportOnly, CanPlay: canPlay, LockReason: lockReason, SupportExpiresAt: expiresAt,
			Visibility: post.Visibility, Version: post.Version, ViewCount: post.ViewCount, Status: post.Status,
			Uuid:         post.Uuid,
			UserId:       post.UserId,
			Nickname:     nickname,
			Avatar:       avatar,
			Title:        post.Title,
			Content:      content,
			MediaType:    post.MediaType,
			CoverUrl:     post.CoverUrl,
			Tags:         tags,
			FirstUrl:     firstUrl,
			Width:        cardSize.Width,
			Height:       cardSize.Height,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CollectCount: post.CollectCount,
			IsLiked:      likedMap[post.Uuid],
			IsCollected:  collectedMap[post.Uuid],
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

// ========== 用户主页 ==========

type UserProfileInfo struct {
	Uuid           string
	Nickname       string
	Avatar         string
	Signature      string
	PostCount      int64
	TotalLikes     int
	FollowingCount int64
	FollowerCount  int64
	IsFollowed     bool
	IsBlocked      bool
	IsBlockedBy    bool
	Posts          []PostInfo
	Total          int64
}

// GetUserProfile 查询用户主页及帖子列表。
func (s *DiscoverServiceImpl) GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error) {
	// 查询用户信息
	user, err := s.userInfoDAO.FindUserByUuid(profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	blocked, blockedBy, err := userdao.NewBlockDAO(db.GormDB.WithContext(ctx)).Status(currentUserId, profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询主页拉黑状态失败: %w", err)
	}
	// 查询帖子列表
	posts, err := s.discoverDAO.ListPostsByUserId(profileUserId, currentUserId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询帖子列表失败: %v", err)
	}

	total, err := s.discoverDAO.CountPostsByUserId(profileUserId, currentUserId)
	if err != nil {
		return nil, fmt.Errorf("查询帖子总数失败: %v", err)
	}

	totalLikes, err := s.discoverDAO.GetUserTotalLikes(profileUserId, currentUserId)
	if err != nil {
		totalLikes = 0
	}

	postInfos, _, err := s.buildPostInfoList(ctx, currentUserId, posts, total)
	if err != nil {
		return nil, err
	}

	// 查询当前用户是否关注了该用户
	isFollowed := false
	if currentUserId != "" && currentUserId != profileUserId {
		follow, _ := s.followDAO.FindFollow(currentUserId, profileUserId)
		if follow != nil {
			isFollowed = true
		}
	}

	return &UserProfileInfo{
		Uuid:           user.Uuid,
		Nickname:       user.Nickname,
		Avatar:         user.Avatar,
		Signature:      user.Signature,
		PostCount:      total,
		TotalLikes:     totalLikes,
		FollowingCount: user.FollowingCount,
		FollowerCount:  user.FollowerCount,
		IsFollowed:     isFollowed,
		IsBlocked:      blocked,
		IsBlockedBy:    blockedBy,
		Posts:          postInfos,
		Total:          total,
	}, nil
}

// discoverCardSize 返回卡片展示尺寸，视频自定义封面优先，媒体原尺寸仍独立保存。
func discoverCardSize(post models.DiscoverPost, media userdao.DiscoverMediaInfo) MediaSize {
	if post.MediaType == 1 && post.CoverUrl != "" && post.CoverWidth > 0 && post.CoverHeight > 0 {
		return MediaSize{Width: post.CoverWidth, Height: post.CoverHeight}
	}
	return MediaSize{Width: media.Width, Height: media.Height}
}

// ========== 作品编辑、可见性与幂等 ==========

// ErrDiscoverConflict 表示幂等键内容冲突或编辑版本过期。
var ErrDiscoverConflict = errors.New("作品已变化，请刷新后重试")

// CanViewDiscoverPost 校验当前身份能否访问正常且未删除的作品。
func CanViewDiscoverPost(post *models.DiscoverPost, userId string) bool {
	return post != nil && !post.DeletedAt.Valid && post.Status == 0 && (post.Visibility == 0 || post.UserId == userId)
}

// validateDiscoverPost 统一校验并规范化标题、正文、标签、媒体和可见性。
func validateDiscoverPost(input *userreq.CreatePostRequest, creating bool) error {
	for i, value := range input.Urls {
		input.Urls[i] = normalizeDiscoverMediaURL(value)
	}
	input.CoverUrl = normalizeDiscoverMediaURL(input.CoverUrl)
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if utf8.RuneCountInString(input.Title) < 1 || utf8.RuneCountInString(input.Title) > 50 {
		return fmt.Errorf("标题需为1至50字")
	}
	if utf8.RuneCountInString(input.Content) > 5000 {
		return fmt.Errorf("正文不能超过5000字")
	}
	if input.Visibility != 0 && input.Visibility != 1 {
		return fmt.Errorf("可见性无效")
	}
	if input.MediaType != 0 && input.MediaType != 1 {
		return fmt.Errorf("媒体类型无效")
	}
	if input.SupportOnly != 0 && input.SupportOnly != 1 {
		return fmt.Errorf("助力属性无效")
	}
	if len(input.Urls) == 0 || len(input.Urls) > 9 || (input.MediaType == 1 && len(input.Urls) != 1) {
		return fmt.Errorf("图片需为1至9张，视频仅限1个")
	}
	if input.MediaType == 1 && input.CoverUrl != "" && input.CoverUrl == input.Urls[0] {
		return fmt.Errorf("视频封面必须单独上传图片")
	}
	if len(input.Tags) > 10 {
		return fmt.Errorf("标签最多10个")
	}
	seen := map[string]bool{}
	for i, tag := range input.Tags {
		tag = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(tag), "#"))
		if tag == "" || utf8.RuneCountInString(tag) > 20 || seen[tag] {
			return fmt.Errorf("标签需为1至20字且不能重复")
		}
		seen[tag] = true
		input.Tags[i] = tag
	}
	seen = map[string]bool{}
	for _, fileURL := range input.Urls {
		if fileURL == "" || len(fileURL) > 500 || seen[fileURL] {
			return fmt.Errorf("媒体地址为空、过长或重复")
		}
		seen[fileURL] = true
	}
	if len(input.CoverUrl) > 500 {
		return fmt.Errorf("封面地址过长")
	}
	if input.MediaType == 0 && input.CoverUrl != "" && !seen[input.CoverUrl] {
		return fmt.Errorf("图片封面必须来自当前图片列表")
	}
	if len(input.MediaSizes) != 0 && len(input.MediaSizes) != len(input.Urls) {
		return fmt.Errorf("媒体尺寸与数量不一致")
	}
	sizes := append([]userreq.MediaSizeRequest{}, input.MediaSizes...)
	if input.CoverSize != nil {
		sizes = append(sizes, *input.CoverSize)
	}
	for i, size := range sizes {
		// 图片及封面不限制像素宽高，视频原尺寸沿用已有校验。
		isVideoSize := input.MediaType == 1 && i < len(input.MediaSizes)
		if size.Width < 0 || size.Height < 0 || (isVideoSize && (size.Width > 65535 || size.Height > 65535)) || (size.Width == 0) != (size.Height == 0) {
			return fmt.Errorf("媒体尺寸无效")
		}
	}
	if creating && (len(input.RequestKey) < 8 || len(input.RequestKey) > 64) {
		return fmt.Errorf("请提供8至64字符的发布请求标识")
	}
	return nil
}

// UpdatePost 校验作者与版本后更新作品内容和资源引用。
func (s *DiscoverServiceImpl) UpdatePost(ctx context.Context, userId, postUuid string, input userreq.CreatePostRequest) (*PostInfo, error) {
	if postUuid == "" {
		return nil, fmt.Errorf("请选择需要编辑的作品")
	}
	return s.saveDiscoverPost(ctx, userId, postUuid, input)
}

// saveDiscoverPost 在同一事务中维护作品、媒体关联与帖子内的发布防重字段。
func (s *DiscoverServiceImpl) saveDiscoverPost(ctx context.Context, userId, postUuid string, input userreq.CreatePostRequest) (*PostInfo, error) {
	if err := validateDiscoverPost(&input, postUuid == ""); err != nil {
		return nil, err
	}
	payload := input
	payload.RequestKey = ""
	payload.Version = 0
	// 保留旧摘要结构中的空post_uuid，避免已保存的发布请求重试发生摘要冲突。
	raw, err := json.Marshal(struct {
		PostUuid string `json:"post_uuid"`
		userreq.CreatePostRequest
	}{CreatePostRequest: payload})
	if err != nil {
		return nil, fmt.Errorf("处理发布参数失败: %w", err)
	}
	digest := sha256.Sum256(raw)
	hash := hex.EncodeToString(digest[:])
	var saved *models.DiscoverPost
	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		files := userdao.NewFileDAO(tx)
		if postUuid == "" {
			key := input.RequestKey
			saved = &models.DiscoverPost{Uuid: "D" + util.GetNowAndLenRandomString(11), UserId: userId, Title: input.Title, Status: constants.DiscoverPostPending, Version: 1, PublishRequestKey: &key, PublishPayloadHash: hash}
			created, err := dao.ClaimPublishPost(saved)
			if err != nil {
				return fmt.Errorf("创建发布记录失败: %w", err)
			}
			if !created {
				saved, err = dao.FindPostByPublishKeyForUpdate(userId, key)
				if err != nil {
					return fmt.Errorf("读取发布记录失败: %w", err)
				}
				if saved == nil {
					return fmt.Errorf("发布标识冲突，请重试")
				}
				if saved.PublishPayloadHash != hash {
					return fmt.Errorf("相同发布标识对应不同内容: %w", ErrDiscoverConflict)
				}
				if saved.DeletedAt.Valid {
					return fmt.Errorf("该请求发布的作品已删除")
				}
				return nil
			}

		} else {
			var err error
			saved, err = dao.FindPostByUuidForUpdate(postUuid)
			if err != nil {
				return err
			}
			if saved == nil || saved.UserId != userId {
				return fmt.Errorf("作品不存在或无权编辑")
			}
			if input.Version != saved.Version {
				return fmt.Errorf("编辑版本已过期: %w", ErrDiscoverConflict)
			}
			saved.Version++
		}
		if postUuid != "" && input.SupportOnly != saved.SupportOnly {
			return fmt.Errorf("作品发布后不能切换免费或助力属性")
		}
		if input.SupportOnly == 1 {
			activate, err := userdao.NewSupportDAO(tx).FindActivate(userId)
			if err != nil {
				return fmt.Errorf("查询助力开通状态失败: %w", err)
			}
			if activate == nil {
				return fmt.Errorf("请先开通创作者助力")
			}
		}
		urls := append([]string{}, input.Urls...)
		if input.CoverUrl != "" {
			urls = append(urls, input.CoverUrl)
		}
		sort.Strings(urls)
		assets := map[string]*models.FileAsset{}
		for _, fileURL := range urls {
			if assets[fileURL] != nil {
				continue
			}
			asset, err := files.FindAssetForUpdate(fileURL)
			if err != nil {
				return fmt.Errorf("查询媒体归属失败: %w", err)
			}
			if asset == nil || asset.UserId != userId || asset.State != 1 || !asset.Verified {
				return fmt.Errorf("媒体未上传完成、已过期或不属于当前用户")
			}
			expected := "discover_image"
			if input.MediaType == 1 && fileURL == input.Urls[0] {
				expected = "discover_video"
			}
			if asset.Category != expected {
				return fmt.Errorf("媒体类型与上传资源不一致")
			}
			if input.SupportOnly == 1 && (!isDiscoverObjectKey(asset.ObjectKey) || asset.URL != oss.FileURL(asset.ObjectKey)) {
				return fmt.Errorf("助力作品必须使用本站上传的私有资源")
			}
			// 编辑历史OSS资源时沿用私有ACL保护，外部历史链接只保留展示关联。
			if (postUuid != "" || input.SupportOnly == 1) && isDiscoverObjectKey(asset.ObjectKey) && asset.URL == oss.FileURL(asset.ObjectKey) {
				if err := oss.SetDiscoverPrivateACL(ctx, asset.ObjectKey); err != nil {
					return err
				}
			}
			if fileURL == input.CoverUrl && asset.Width == 0 && input.CoverSize != nil {
				asset.Width, asset.Height = input.CoverSize.Width, input.CoverSize.Height
			}
			if err := files.SaveAsset(asset); err != nil {
				return err
			}
			assets[fileURL] = asset
		}
		tags, err := json.Marshal(input.Tags)
		if err != nil {
			return err
		}
		saved.Title, saved.Content, saved.Tags = input.Title, input.Content, tags
		saved.MediaType, saved.Visibility, saved.CoverUrl = input.MediaType, input.Visibility, input.CoverUrl
		saved.SupportOnly = input.SupportOnly
		// 内容变更后重新审核，原驳回或下架原因不带入新版本。
		saved.Status, saved.ReviewNote = constants.DiscoverPostPending, ""
		saved.CoverWidth, saved.CoverHeight = 0, 0
		saved.CoverAssetId = nil
		if cover := assets[input.CoverUrl]; cover != nil {
			saved.CoverAssetId = &cover.Id
			saved.CoverWidth, saved.CoverHeight = cover.Width, cover.Height
		}
		if saved.Id == 0 {
			if err := dao.CreatePost(saved); err != nil {
				return err
			}
		} else {
			if err := dao.SavePost(saved); err != nil {
				return err
			}
			if err := dao.DeletePostMedia(saved.Id); err != nil {
				return err
			}
		}
		for i, fileURL := range input.Urls {
			asset := assets[fileURL]
			// 未知视频尺寸由前端元数据补齐一次，后续统一从资源读取，不覆盖已验证图片尺寸。
			if asset.Width == 0 && len(input.MediaSizes) > i {
				asset.Width, asset.Height = input.MediaSizes[i].Width, input.MediaSizes[i].Height
				if err := files.SaveAsset(asset); err != nil {
					return fmt.Errorf("保存媒体尺寸失败: %w", err)
				}
			}
			if err := dao.CreateMedia(&models.DiscoverMedia{PostId: saved.Id, AssetId: asset.Id, Url: fileURL, Type: input.MediaType, SortOrder: i}); err != nil {
				return err
			}
		}

		// 待审核内容仅同步搜索状态，审核通过后再分发至关注流。
		return userdao.NewDiscoverOutboxDAO(tx).Create(saved, false)
	})
	if err != nil {
		return nil, err
	}
	posts, _, err := s.buildPostInfoList(ctx, userId, []models.DiscoverPost{*saved}, 1)
	if err != nil {
		return nil, err
	}
	if len(posts) != 1 {
		return nil, fmt.Errorf("读取已保存作品失败")
	}
	return &posts[0], nil
}

// DeletePost 仅允许作者软删除作品，并在同一事务中记录搜索同步事件。
func (s *DiscoverServiceImpl) DeletePost(ctx context.Context, userId, postUuid string) error {
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(postUuid)
		if err != nil {
			return err
		}
		if post == nil {
			return nil
		}
		if post.UserId != userId {
			return fmt.Errorf("无权删除此作品")
		}
		post.Version++
		if err := dao.SavePost(post); err != nil {
			return err
		}
		if err := dao.DeletePost(post); err != nil {
			return err
		}
		return userdao.NewDiscoverOutboxDAO(tx).Create(post, false)
	})
}

// normalizeDiscoverMediaURL 从受控媒体入口恢复资源原地址，统一发布摘要及资源归属查询。
func normalizeDiscoverMediaURL(value string) string {
	parsed, err := url.Parse(value)
	if err == nil && strings.HasSuffix(parsed.Path, "/user/file/view") {
		return parsed.Query().Get("url")
	}
	return value
}

// ListOwnWorks 校验作者作品筛选参数并装配展示信息。
func (s *DiscoverServiceImpl) ListOwnWorks(ctx context.Context, userId string, input userreq.ListOwnWorksRequest) ([]PostInfo, int64, map[int8]int64, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 50 {
		input.PageSize = 20
	}
	input.Keyword = strings.TrimSpace(input.Keyword)
	if utf8.RuneCountInString(input.Keyword) > 100 {
		return nil, 0, nil, fmt.Errorf("搜索关键词不能超过100字")
	}
	if input.Status != nil && (*input.Status < 0 || *input.Status > 3) {
		return nil, 0, nil, fmt.Errorf("作品状态无效")
	}
	if input.Visibility != nil && *input.Visibility != 0 && *input.Visibility != 1 {
		return nil, 0, nil, fmt.Errorf("可见性无效")
	}
	posts, total, err := s.discoverDAO.ListOwnWorks(userId, input.Keyword, input.Status, input.Visibility, input.Page, input.PageSize)
	if err != nil {
		return nil, 0, nil, err
	}
	counts, err := s.discoverDAO.CountOwnWorksByStatus(userId, input.Keyword, input.Visibility)
	if err != nil {
		return nil, 0, nil, err
	}
	items, total, err := s.buildPostInfoList(ctx, userId, posts, total)
	return items, total, counts, err
}

// SetPostVisibility 在事务中只更新作者作品可见性和版本，不重写媒体或上传文件。
func (s *DiscoverServiceImpl) SetPostVisibility(ctx context.Context, userId string, input userreq.SetPostVisibilityRequest) (uint64, error) {
	if input.Visibility == nil || (*input.Visibility != 0 && *input.Visibility != 1) {
		return 0, fmt.Errorf("可见性无效")
	}
	var saved *models.DiscoverPost
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(input.PostUuid)
		if err != nil {
			return err
		}
		if post == nil || post.UserId != userId {
			return fmt.Errorf("作品不存在或无权修改")
		}
		if post.Version != input.Version {
			return fmt.Errorf("编辑版本已过期: %w", ErrDiscoverConflict)
		}
		if post.Visibility != *input.Visibility {
			post.Visibility = *input.Visibility
			post.Version++
			if err := dao.SavePost(post); err != nil {
				return err
			}
			if err := userdao.NewDiscoverOutboxDAO(tx).Create(post, false); err != nil {
				return err
			}
		}
		saved = post
		return nil
	})
	if err != nil {
		return 0, err
	}
	return saved.Version, nil
}
