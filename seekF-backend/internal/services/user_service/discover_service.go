package userservice

import (
	"context"
	"encoding/json"
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/util"
)

type DiscoverService interface {
	CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string) (*PostInfo, error)
	ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error)
	ToggleLike(ctx context.Context, userId, targetUuid string) (bool, int, error)
	AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error)
	ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error)
	ToggleCommentLike(ctx context.Context, userId, commentUuid string) (bool, int, error)
}

type DiscoverServiceImpl struct {
	discoverDAO  userdao.DiscoverDAO
	userInfoDAO  userdao.UserInfoDAO
}

type PostInfo struct {
	Uuid         string
	UserId       string
	Nickname     string
	Avatar       string
	Title        string
	Content      string
	MediaType    int8
	Tags         []string
	FirstUrl     string
	LikeCount    int
	CommentCount int
	IsLiked      bool
	CreatedAt    string
}

type PostDetailInfo struct {
	Uuid         string
	UserId       string
	Nickname     string
	Avatar       string
	Title        string
	Content      string
	MediaType    int8
	Tags         []string
	Urls         []string
	LikeCount    int
	CommentCount int
	IsLiked      bool
	CreatedAt    string
}

type CommentInfo struct {
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

func NewDiscoverService(discoverDAO userdao.DiscoverDAO, userInfoDAO userdao.UserInfoDAO) DiscoverService {
	return &DiscoverServiceImpl{
		discoverDAO: discoverDAO,
		userInfoDAO: userInfoDAO,
	}
}

func (s *DiscoverServiceImpl) CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string) (*PostInfo, error) {
	postUUID := "D" + util.GetNowAndLenRandomString(11)

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	post := &models.DiscoverPost{
		Uuid:      postUUID,
		UserId:    userId,
		Title:     title,
		Content:   content,
		MediaType: mediaType,
		Tags:      tagsJSON,
		Status:    0,
	}

	if err := s.discoverDAO.CreatePost(post); err != nil {
		return nil, fmt.Errorf("创建帖子失败: %v", err)
	}

	for i, url := range urls {
		media := &models.DiscoverMedia{
			PostId:    post.Id,
			Type:      mediaType,
			Url:       url,
			SortOrder: i,
		}
		if err := s.discoverDAO.CreateMedia(media); err != nil {
			return nil, fmt.Errorf("保存媒体失败: %v", err)
		}
	}

	user, _ := s.userInfoDAO.FindUserByUuid(userId)

	firstUrl := ""
	if len(urls) > 0 {
		firstUrl = urls[0]
	}

	return &PostInfo{
		Uuid:      postUUID,
		UserId:    userId,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Title:     title,
		Content:   content,
		MediaType: mediaType,
		Tags:      tags,
		FirstUrl:  firstUrl,
		CreatedAt: post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListPosts(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountPosts()
	if err != nil {
		return nil, 0, err
	}

	var result []PostInfo
	for _, post := range posts {
		mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
		firstUrl := ""
		if len(mediaList) > 0 {
			firstUrl = mediaList[0].Url
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

		// 查询当前用户是否点赞了该帖子
		isLiked := false
		if userId != "" {
			like, _ := s.discoverDAO.FindLike(userId, post.Uuid)
			if like != nil {
				isLiked = true
			}
		}

		result = append(result, PostInfo{
			Uuid:         post.Uuid,
			UserId:       post.UserId,
			Nickname:     nickname,
			Avatar:       avatar,
			Title:        post.Title,
			Content:      post.Content,
			MediaType:    post.MediaType,
			Tags:         tags,
			FirstUrl:     firstUrl,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			IsLiked:      isLiked,
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

func (s *DiscoverServiceImpl) GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(uuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
	var urls []string
	for _, m := range mediaList {
		urls = append(urls, m.Url)
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

	return &PostDetailInfo{
		Uuid:         post.Uuid,
		UserId:       post.UserId,
		Nickname:     nickname,
		Avatar:       avatar,
		Title:        post.Title,
		Content:      post.Content,
		MediaType:    post.MediaType,
		Tags:         tags,
		Urls:         urls,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		IsLiked:      isLiked,
		CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ToggleLike(ctx context.Context, userId, targetUuid string) (bool, int, error) {
	existing, _ := s.discoverDAO.FindLike(userId, targetUuid)
	if existing != nil {
		if err := s.discoverDAO.DeleteLike(userId, targetUuid); err != nil {
			return false, 0, err
		}
		post, _ := s.discoverDAO.FindPostByUuid(targetUuid)
		if post != nil {
			s.discoverDAO.DecrementLikeCount(post.Id)
			// 重新获取帖子以获取更新后的点赞数
			updatedPost, _ := s.discoverDAO.FindPostByUuid(targetUuid)
			if updatedPost != nil {
				return false, updatedPost.LikeCount, nil
			}
		}
		return false, 0, nil
	}

	like := &models.DiscoverLike{
		UserId:     userId,
		TargetUuid: targetUuid,
	}
	if err := s.discoverDAO.CreateLike(like); err != nil {
		return false, 0, err
	}

	post, _ := s.discoverDAO.FindPostByUuid(targetUuid)
	if post != nil {
		s.discoverDAO.IncrementLikeCount(post.Id)
		// 重新获取帖子以获取更新后的点赞数
		updatedPost, _ := s.discoverDAO.FindPostByUuid(targetUuid)
		if updatedPost != nil {
			return true, updatedPost.LikeCount, nil
		}
	}
	return true, 0, nil
}

func (s *DiscoverServiceImpl) AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	commentUUID := "C" + util.GetNowAndLenRandomString(11)
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		PostId:        post.Id,
		UserId:        userId,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
	}

	if err := s.discoverDAO.CreateComment(comment); err != nil {
		return nil, fmt.Errorf("评论失败: %v", err)
	}

	s.discoverDAO.IncrementCommentCount(post.Id)

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

func (s *DiscoverServiceImpl) ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	comments, err := s.discoverDAO.FindCommentsByPostId(post.Id, page, pageSize)
	if err != nil {
		return nil, err
	}

	var result []CommentInfo
	for _, c := range comments {
		user, _ := s.userInfoDAO.FindUserByUuid(c.UserId)
		nickname := ""
		avatar := ""
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		replyToNickname := ""
		if c.ReplyToUserId != "" {
			replyToUser, _ := s.userInfoDAO.FindUserByUuid(c.ReplyToUserId)
			if replyToUser != nil {
				replyToNickname = replyToUser.Nickname
			}
		}

		// 查询当前用户是否点赞了该评论
		isLiked := false
		if userId != "" {
			like, _ := s.discoverDAO.FindLike(userId, c.Uuid)
			if like != nil {
				isLiked = true
			}
		}

		result = append(result, CommentInfo{
			Uuid:            c.Uuid,
			UserId:          c.UserId,
			Nickname:        nickname,
			Avatar:          avatar,
			ParentId:        c.ParentId,
			ReplyToUserId:   c.ReplyToUserId,
			ReplyToNickname: replyToNickname,
			Content:         c.Content,
			LikeCount:       c.LikeCount,
			IsLiked:         isLiked,
			CreatedAt:       c.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

func (s *DiscoverServiceImpl) ToggleCommentLike(ctx context.Context, userId, commentUuid string) (bool, int, error) {
	existing, _ := s.discoverDAO.FindLike(userId, commentUuid)
	if existing != nil {
		if err := s.discoverDAO.DeleteLike(userId, commentUuid); err != nil {
			return false, 0, err
		}
		// 通过 UUID 查询评论
		comment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
		if comment != nil {
			s.discoverDAO.DecrementCommentLikeCount(comment.Id)
			// 重新查询获取更新后的点赞数
			updatedComment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
			if updatedComment != nil {
				return false, updatedComment.LikeCount, nil
			}
		}
		return false, 0, nil
	}

	like := &models.DiscoverLike{
		UserId:     userId,
		TargetUuid: commentUuid,
	}
	if err := s.discoverDAO.CreateLike(like); err != nil {
		return false, 0, err
	}
	// 通过 UUID 查询评论
	comment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
	if comment != nil {
		s.discoverDAO.IncrementCommentLikeCount(comment.Id)
		// 重新查询获取更新后的点赞数
		updatedComment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
		if updatedComment != nil {
			return true, updatedComment.LikeCount, nil
		}
	}
	return true, 0, nil
}
