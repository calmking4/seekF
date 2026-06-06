package oss

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/util"
)

// FileCategory 表示上传文件的业务分类
type FileCategory string

const (
	// UserAvatar 用户头像
	UserAvatar FileCategory = "user_avatar"
	// GroupAvatar 群头像
	GroupAvatar FileCategory = "group_avatar"
	// MessageImage 消息图片
	MessageImage FileCategory = "message_image"
	// MessageVideo 消息视频
	MessageVideo FileCategory = "message_video"
	// MessageAudio 消息音频
	MessageAudio FileCategory = "message_audio"
	// KnowledgeDoc 知识库文档
	KnowledgeDoc FileCategory = "knowledge_doc"
	// DiscoverImage 发现图片
	DiscoverImage FileCategory = "discover_image"
	// DiscoverVideo 发现视频
	DiscoverVideo FileCategory = "discover_video"
)

// OSS 路径前缀（不以 / 结尾，后面会再拼子目录和文件名）
const (
	avatarDirPrefix       = "common/user_avatars"
	groupAvatarDirPrefix  = "common/group_avatars"
	messageImageDirPrefix = "messages/images"
	messageVideoDirPrefix = "messages/videos"
	messageAudioDirPrefix = "messages/audios"
	knowledgeDirPrefix    = "knowledge"
	discoverImageDirPrefix = "discover/images"
	discoverVideoDirPrefix = "discover/videos"
)

// UploadResult 上传结果
type UploadResult struct {
	ObjectKey string `json:"objectKey"`
	URL       string `json:"url"`
}

var (
	clientOnce sync.Once
	ossClient  *oss.Client
	clientErr  error

	ossBucketName string
	ossBaseURL    string
)

func initOSSClient() {
	cfg := configs.GetConfig()
	region := cfg.OSSConfig.Region
	bucketName := cfg.OSSConfig.Bucket
	baseURL := cfg.OSSConfig.BaseURL
	accessKeyID := cfg.OSSConfig.AccessKeyID
	accessKeySecret := cfg.OSSConfig.AccessKeySecret

	if region == "" || accessKeyID == "" || accessKeySecret == "" || bucketName == "" {
		clientErr = fmt.Errorf("OSS配置缺失，请设置 [ossConfig] AccessKeyID/AccessKeySecret/Region/Bucket")
		return
	}

	provider := credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret)

	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(region)

	ossClient = oss.NewClient(ossCfg)
	ossBucketName = bucketName
	ossBaseURL = strings.TrimRight(baseURL, "/")
}

func getClient() (*oss.Client, error) {
	clientOnce.Do(initOSSClient) //只初始化一次
	if clientErr != nil {
		return nil, clientErr
	}

	if ossClient == nil {
		return nil, fmt.Errorf("OSS客户端未初始化")
	}

	return ossClient, nil
}

func dirForCategory(category FileCategory) string {
	switch category {
	case UserAvatar:
		return avatarDirPrefix
	case GroupAvatar:
		return groupAvatarDirPrefix
	case MessageImage:
		return messageImageDirPrefix
	case MessageVideo:
		return messageVideoDirPrefix
	case MessageAudio:
		return messageAudioDirPrefix
	case KnowledgeDoc:
		return knowledgeDirPrefix
	case DiscoverImage:
		return discoverImageDirPrefix
	case DiscoverVideo:
		return discoverVideoDirPrefix
	default:
		return messageImageDirPrefix
	}
}

// 生成OSS文件的唯一标识（路径+文件名）
func buildObjectKey(category FileCategory, originalFilename string) string {
	dir := dirForCategory(category)
	datePrefix := time.Now().Format("2006/01/02")

	ext := strings.ToLower(path.Ext(originalFilename))
	if ext == "" {
		ext = ".bin"
	}

	randomPart := util.GetNowAndLenRandomString(6)
	baseName := time.Now().Format("150405") + "_" + randomPart + ext // Format("150405")时分秒

	return path.Join(dir, datePrefix, baseName)
}

// 生成文件的访问URL
func buildFileURL(objectKey string) string {
	if ossBaseURL != "" {
		return fmt.Sprintf("%s/%s", ossBaseURL, objectKey)
	}
	return objectKey
}

func UploadReader(ctx context.Context, r io.Reader, filename string, category FileCategory) (*UploadResult, error) {
	if r == nil {
		return nil, fmt.Errorf("读取器为空")
	}

	client, err := getClient()
	if err != nil {
		return nil, err
	}

	objectKey := buildObjectKey(category, filename)

	if ctx == nil {
		ctx = context.Background()
	}

	_, err = client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(ossBucketName),
		Key:    oss.Ptr(objectKey),
		Body:   r,
	})
	if err != nil {
		return nil, fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	return &UploadResult{
		ObjectKey: objectKey,
		URL:       buildFileURL(objectKey),
	}, nil
}

func UploadMultipartFile(ctx context.Context, fileHeader *multipart.FileHeader, category FileCategory) (*UploadResult, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("文件头为空")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	return UploadReader(ctx, src, fileHeader.Filename, category)
}

// ObjectKeyFromURL 从访问 URL 解析 OSS 对象键
func ObjectKeyFromURL(fileURL string) string {
	if fileURL == "" {
		return ""
	}
	if ossBaseURL != "" {
		prefix := ossBaseURL + "/"
		if strings.HasPrefix(fileURL, prefix) {
			return strings.TrimPrefix(fileURL, prefix)
		}
	}
	parsed, err := url.Parse(fileURL)
	if err != nil || parsed.Path == "" {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

// DeleteFile 删除 OSS 上的文件
func DeleteFile(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("对象键为空")
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	_, err = client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(ossBucketName),
		Key:    oss.Ptr(objectKey),
	})
	if err != nil {
		return fmt.Errorf("删除OSS文件失败: %w", err)
	}

	return nil
}

// DeleteFileByURL 根据访问 URL 删除 OSS 文件
func DeleteFileByURL(ctx context.Context, fileURL string) error {
	objectKey := ObjectKeyFromURL(fileURL)
	if objectKey == "" {
		return fmt.Errorf("无法从URL解析对象键")
	}
	return DeleteFile(ctx, objectKey)
}
