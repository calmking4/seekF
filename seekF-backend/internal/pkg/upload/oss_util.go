package oss

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
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
	// CategoryAvatar 用户头像
	CategoryAvatar FileCategory = "avatar"
	// CategoryImage 一般图片
	CategoryImage FileCategory = "image"
	// CategoryVideo 视频
	CategoryVideo FileCategory = "video"
)

// OSS 路径前缀（不以 / 结尾，后面会再拼子目录和文件名）
const (
	avatarDirPrefix = "images/avatars"
	imageDirPrefix  = "images/common"
	videoDirPrefix  = "videos"
)

// UploadResult 上传结果
type UploadResult struct {
	// ObjectKey 是 OSS 中的对象键（不带域名）
	ObjectKey string `json:"objectKey"`
	// URL 是可直接访问的完整 URL（取决于你配置的 OSS_BASE_URL）
	URL string `json:"url"`
}

var (
	clientOnce sync.Once
	ossClient  *oss.Client
	clientErr  error

	ossBucketName string
	ossBaseURL    string
)

// initOSSClient 使用配置文件初始化 OSS v2 Client。
func initOSSClient() {
	//获取配置
	cfg := configs.GetConfig()
	region := cfg.OSSConfig.Region
	bucketName := cfg.OSSConfig.Bucket
	baseURL := cfg.OSSConfig.BaseURL
	accessKeyID := cfg.OSSConfig.AccessKeyID
	accessKeySecret := cfg.OSSConfig.AccessKeySecret

	if region == "" || accessKeyID == "" || accessKeySecret == "" || bucketName == "" {
		clientErr = fmt.Errorf("OSS config missing, please set [ossConfig] AccessKeyID/AccessKeySecret/Region/Bucket")
		return
	}

	// 使用静态凭证提供器，AccessKey 取自 ossConfig
	provider := credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret)

	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(region)

	ossClient = oss.NewClient(ossCfg)
	ossBucketName = bucketName
	ossBaseURL = strings.TrimRight(baseURL, "/")
}

func getClient() (*oss.Client, error) {
	clientOnce.Do(initOSSClient)
	if clientErr != nil {
		return nil, clientErr
	}

	if ossClient == nil {
		return nil, fmt.Errorf("OSS client not initialized")
	}

	return ossClient, nil
}

// dirForCategory 根据文件分类返回目录前缀
func dirForCategory(cat FileCategory) string {
	switch cat {
	case CategoryAvatar:
		return avatarDirPrefix
	case CategoryVideo:
		return videoDirPrefix
	case CategoryImage:
		fallthrough
	default:
		return imageDirPrefix
	}
}

// buildObjectKey 构造 OSS 对象键，包含：业务前缀 / 日期 / 随机文件名
func buildObjectKey(cat FileCategory, originalFilename string) string {
	dir := dirForCategory(cat)
	datePrefix := time.Now().Format("2006/01/02") // 例如 2026/03/09

	ext := strings.ToLower(path.Ext(originalFilename))
	if ext == "" {
		ext = ".bin"
	}

	// 使用时间 + 随机串生成文件名，避免冲突
	randomPart := util.GetNowAndLenRandomString(6)
	baseName := time.Now().Format("150405") + "_" + randomPart + ext

	return path.Join(dir, datePrefix, baseName)
}

// buildFileURL 根据对象键拼接可访问 URL
// 优先使用 OSS_BASE_URL，其次回退到 bucket.endpoint 形式（如果可用）。
func buildFileURL(objectKey string) string {
	if ossBaseURL != "" {
		return fmt.Sprintf("%s/%s", ossBaseURL, objectKey)
	}

	// 未配置自定义域名时，仅返回对象键，交由上层拼接域名或直接用于带签名的访问
	return objectKey
}

// UploadReader 将任意 io.Reader 上传到 OSS。
// filename 用于推断扩展名和生成友好的文件名；cat 用于选择保存目录。
func UploadReader(ctx context.Context, r io.Reader, filename string, cat FileCategory) (*UploadResult, error) {
	if r == nil {
		return nil, fmt.Errorf("reader is nil")
	}

	client, err := getClient()
	if err != nil {
		return nil, err
	}

	objectKey := buildObjectKey(cat, filename)

	if ctx == nil {
		ctx = context.Background()
	}

	_, err = client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(ossBucketName),
		Key:    oss.Ptr(objectKey),
		Body:   r,
	})
	if err != nil {
		return nil, fmt.Errorf("put object to OSS failed: %w", err)
	}

	return &UploadResult{
		ObjectKey: objectKey,
		URL:       buildFileURL(objectKey),
	}, nil
}

// UploadMultipartFile 方便在 Gin 控制器中直接上传 multipart.FileHeader。
func UploadMultipartFile(ctx context.Context, fileHeader *multipart.FileHeader, cat FileCategory) (*UploadResult, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file header is nil")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open multipart file failed: %w", err)
	}
	defer src.Close()

	return UploadReader(ctx, src, fileHeader.Filename, cat)
}
