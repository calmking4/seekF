package userservice

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"path"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"
	"strings"
	"time"
)

// DiscoverImageMaxBytes 图片上限10MB，DiscoverVideoMaxBytes 视频上限500MB。
const (
	DiscoverImageMaxBytes int64 = 10 * 1024 * 1024
	DiscoverVideoMaxBytes int64 = 500 * 1024 * 1024
)

// ErrInvalidUpload 表示上传文件的格式、内容或大小不符合要求。
var ErrInvalidUpload = errors.New("上传文件校验失败")

// FileService 上传、归属验证和资源清理服务接口。
type FileService interface {
	DiscoverAssetStates(ctx context.Context, userId string, urls []string) (map[string]bool, error)
	OpenDiscoverFile(ctx context.Context, userId, fileURL, rangeHeader string) (*oss.ResourceStream, error)
	UploadFile(ctx context.Context, userId string, file *multipart.FileHeader, fileType oss.FileCategory) (*oss.UploadResult, error)
	InitMultipartUpload(ctx context.Context, userId string, req userreq.InitMultipartUploadRequest) (*oss.MultipartInitResult, error)
	UploadPart(ctx context.Context, userId, objectKey, uploadId string, partNumber int32, file *multipart.FileHeader) (string, error)
	ListUploadParts(ctx context.Context, userId, objectKey, uploadId string) ([]oss.PartInfo, error)
	CompleteMultipartUpload(ctx context.Context, userId string, req userreq.CompleteMultipartUploadRequest) (*oss.UploadResult, error)
	AbortMultipartUpload(ctx context.Context, userId, objectKey, uploadId string) error
	DeleteFile(ctx context.Context, userId, objectKey, fileURL string) error
}

// FileServiceImpl 文件上传服务实现。
type FileServiceImpl struct{}

// NewFileService 创建文件服务实例。
func NewFileService() FileService { return &FileServiceImpl{} }

// UploadFile 验证发现页图片内容和大小并登记资源所有者。
func (s *FileServiceImpl) UploadFile(ctx context.Context, userId string, file *multipart.FileHeader, fileType oss.FileCategory) (*oss.UploadResult, error) {
	if fileType != oss.DiscoverImage && fileType != oss.DiscoverVideo {
		return oss.UploadMultipartFile(ctx, file, fileType)
	}
	if fileType == oss.DiscoverVideo {
		return nil, fmt.Errorf("发现页视频请使用分片上传")
	}
	cfg, format, err := validateDiscoverImage(file)
	if err != nil {
		return nil, err
	}
	// 以真实格式生成对象后缀，兼容PNG/JPEG文件被误命名的情况。
	result, err := oss.PrepareUpload(discoverImageFilename(file.Filename, format), fileType)
	if err != nil {
		return nil, err
	}
	asset := &models.FileAsset{UserId: userId, URL: result.URL, ObjectKey: result.ObjectKey, Category: string(fileType), Size: file.Size, ContentType: "image/" + format, Width: cfg.Width, Height: cfg.Height, Verified: true, State: 0}
	// 先登记上传租约，数据库不可用时不产生无归属对象；标识冲突不能影响旧作品资源。
	if err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewFileDAO(tx)
		references, err := dao.CountAssetReferences(asset)
		if err != nil {
			return err
		}
		if references > 0 {
			return fmt.Errorf("资源标识冲突，请重试上传")
		}
		return dao.CreateAsset(asset)
	}); err != nil {
		return nil, fmt.Errorf("登记上传资源失败: %w", err)
	}
	source, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开图片文件失败: %w", err)
	}
	defer source.Close()
	if err := oss.UploadReaderAtKey(ctx, source, result.ObjectKey, fileType); err != nil {
		return nil, err
	}
	if err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewFileDAO(tx)
		stored, err := dao.FindAssetForUpdate(result.URL)
		if err != nil {
			return err
		}
		if stored == nil || stored.State != 0 {
			return fmt.Errorf("上传资源状态已变化")
		}
		stored.State = 1
		return dao.SaveAsset(stored)
	}); err != nil {
		return nil, fmt.Errorf("完成资源登记失败，未引用对象将自动清理: %w", err)
	}
	return result, nil
}

// InitMultipartUpload 验证视频参数并登记续传任务归属。
func (s *FileServiceImpl) InitMultipartUpload(ctx context.Context, userId string, req userreq.InitMultipartUploadRequest) (*oss.MultipartInitResult, error) {
	category := oss.ParseFileCategory(req.FileType)
	if !oss.IsVideoCategory(category) {
		return nil, fmt.Errorf("仅视频支持分片上传")
	}
	if req.FileSize <= 0 || (category == oss.DiscoverVideo && (req.FileSize > DiscoverVideoMaxBytes || strings.ToLower(path.Ext(req.FileName)) != ".mp4" || req.ContentType != "video/mp4")) {
		return nil, fmt.Errorf("发现页视频仅支持不超过500MB的MP4")
	}
	result, err := oss.MultipartInit(ctx, req.FileName, category, req.ContentType)
	if err != nil {
		return nil, err
	}
	asset := &models.FileAsset{UserId: userId, URL: result.URL, ObjectKey: result.ObjectKey, UploadId: result.UploadId, Category: string(category), Size: req.FileSize, ContentType: req.ContentType}
	if err := userdao.NewFileDAO(db.GormDB.WithContext(ctx)).CreateAsset(asset); err != nil {
		if cleanup := oss.MultipartAbort(context.Background(), result.ObjectKey, result.UploadId); cleanup != nil {
			zlog.Error("取消登记失败的分片任务失败: " + cleanup.Error())
		}
		return nil, fmt.Errorf("登记上传任务失败: %w", err)
	}
	return result, nil
}

// withUploadAsset 在事务中锁定并验证分片上传归属，防止任务被其他用户操作。
func withUploadAsset(ctx context.Context, userId, objectKey, uploadId string, action func(*models.FileAsset) error) error {
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewFileDAO(tx)
		asset, err := dao.FindAssetForUpdate(oss.FileURL(objectKey))
		if err != nil {
			return err
		}
		if asset == nil || asset.UserId != userId || asset.ObjectKey != objectKey || asset.UploadId != uploadId || asset.State >= 2 {
			return fmt.Errorf("上传任务不存在或无权访问")
		}
		if err := action(asset); err != nil {
			return err
		}
		return dao.SaveAsset(asset)
	})
}

// UploadPart 先提交任务归属和首片格式校验，再上传数据，避免取消请求导致已上传首片无法恢复。
func (s *FileServiceImpl) UploadPart(ctx context.Context, userId, objectKey, uploadId string, partNumber int32, file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size <= 0 || file.Size > oss.DefaultPartSize {
		return "", fmt.Errorf("分片大小无效")
	}
	source, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开分片文件失败: %w", err)
	}
	defer source.Close()
	err = withUploadAsset(ctx, userId, objectKey, uploadId, func(asset *models.FileAsset) error {
		if asset.State != 0 || partNumber < 1 {
			return fmt.Errorf("上传状态或分片无效")
		}
		expected := oss.DefaultPartSize
		remaining := asset.Size - int64(partNumber-1)*int64(expected)
		if remaining <= 0 {
			return fmt.Errorf("分片序号超出文件大小")
		}
		if remaining < int64(expected) {
			expected = int(remaining)
		}
		if file.Size != int64(expected) {
			return fmt.Errorf("分片大小与上传任务不一致")
		}
		if asset.Category == string(oss.DiscoverVideo) && partNumber == 1 {
			header := make([]byte, 12)
			if _, err := io.ReadFull(source, header); err != nil || string(header[4:8]) != "ftyp" {
				return fmt.Errorf("视频内容不是有效MP4")
			}
			if _, err := source.Seek(0, io.SeekStart); err != nil {
				return err
			}
			asset.Verified = true
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return oss.MultipartUploadPart(ctx, objectKey, uploadId, partNumber, source, file.Size)
}

// ListUploadParts 校验所有者后读取已上传分片。
func (s *FileServiceImpl) ListUploadParts(ctx context.Context, userId, objectKey, uploadId string) ([]oss.PartInfo, error) {
	var parts []oss.PartInfo
	err := withUploadAsset(ctx, userId, objectKey, uploadId, func(asset *models.FileAsset) error {
		var err error
		parts, err = oss.MultipartListParts(ctx, objectKey, uploadId)
		return err
	})
	return parts, err
}

// CompleteMultipartUpload 校验实际分片大小和摘要后完成上传并登记可用状态。
func (s *FileServiceImpl) CompleteMultipartUpload(ctx context.Context, userId string, req userreq.CompleteMultipartUploadRequest) (*oss.UploadResult, error) {
	var result *oss.UploadResult
	err := withUploadAsset(ctx, userId, req.ObjectKey, req.UploadId, func(asset *models.FileAsset) error {
		if asset.State == 1 {
			result = &oss.UploadResult{URL: asset.URL, ObjectKey: asset.ObjectKey}
			return nil
		}
		parts, err := oss.MultipartListParts(ctx, req.ObjectKey, req.UploadId)
		if err != nil {
			// OSS合并已成功但数据库提交失败时，依据已验证任务和真实对象大小恢复登记。
			if strings.Contains(err.Error(), "NoSuchUpload") && (asset.Verified || asset.Category != string(oss.DiscoverVideo)) {
				size, headErr := oss.ResourceSize(ctx, asset.ObjectKey)
				if headErr == nil && size == asset.Size {
					asset.State = 1
					result = &oss.UploadResult{URL: asset.URL, ObjectKey: asset.ObjectKey}
					return nil
				}
			}
			return err
		}
		if len(parts) != len(req.Parts) || len(parts) == 0 {
			return fmt.Errorf("上传分片未完成")
		}
		actual := map[int32]oss.PartInfo{}
		var total int64
		for _, p := range parts {
			actual[p.PartNumber] = p
			total += p.Size
		}
		for i, p := range req.Parts {
			found, ok := actual[p.PartNumber]
			if !ok || p.PartNumber != int32(i+1) || found.ETag != strings.Trim(p.ETag, "\"") {
				return fmt.Errorf("分片顺序或摘要不一致")
			}
		}
		if total != asset.Size || (asset.Category == string(oss.DiscoverVideo) && !asset.Verified) {
			return fmt.Errorf("视频大小或格式校验未通过")
		}
		result, err = oss.MultipartComplete(ctx, req.ObjectKey, req.UploadId, parts)
		if err != nil {
			return err
		}
		asset.State = 1
		asset.Verified = true
		return nil
	})
	return result, err
}

// AbortMultipartUpload 仅取消当前用户尚未完成的分片任务。
func (s *FileServiceImpl) AbortMultipartUpload(ctx context.Context, userId, objectKey, uploadId string) error {
	return withUploadAsset(ctx, userId, objectKey, uploadId, func(asset *models.FileAsset) error {
		if asset.State != 0 {
			return fmt.Errorf("已完成上传不能取消分片")
		}
		if err := oss.MultipartAbort(ctx, objectKey, uploadId); err != nil {
			return err
		}
		asset.State = 2
		return nil
	})
}

// DeleteFile 发现页资源仅允许所有者在无有效引用时删除。
func (s *FileServiceImpl) DeleteFile(ctx context.Context, userId, objectKey, fileURL string) error {
	fileURL = normalizeDiscoverMediaURL(fileURL)
	if fileURL != "" && objectKey != "" && oss.ObjectKeyFromURL(fileURL) != objectKey {
		return fmt.Errorf("资源地址与对象标识不一致")
	}
	if fileURL == "" {
		fileURL = oss.FileURL(objectKey)
	}
	if objectKey == "" {
		objectKey = oss.ObjectKeyFromURL(fileURL)
	}
	if strings.HasPrefix(objectKey, "discover/") {
		return deleteDiscoverAsset(ctx, userId, fileURL, false)
	}
	if objectKey == "" || path.Clean(objectKey) != objectKey || strings.Contains(objectKey, "\\") {
		return fmt.Errorf("资源对象标识无效")
	}
	return oss.DeleteFile(ctx, objectKey)
}

// deleteDiscoverAsset 先锁定并检查引用，再提交删除中状态；远程删除失败可重试且不能再发布引用。
func deleteDiscoverAsset(ctx context.Context, userId, fileURL string, expiredOnly bool) error {
	var deleting *models.FileAsset
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewFileDAO(tx)
		asset, err := dao.FindAssetForUpdate(fileURL)
		if err != nil {
			return err
		}
		if asset == nil || asset.UserId != userId {
			return fmt.Errorf("资源不存在或无权删除")
		}
		if !isDiscoverObjectKey(asset.ObjectKey) || asset.URL != oss.FileURL(asset.ObjectKey) {
			return fmt.Errorf("历史外部资源不由当前存储服务管理")
		}
		if asset.State == 2 {
			return nil
		}
		if expiredOnly && asset.State != 3 && asset.UpdatedAt.After(time.Now().Add(-7*24*time.Hour)) {
			return nil
		}
		references, err := dao.CountAssetReferences(asset)
		if err != nil {
			return err
		}
		if references > 0 {
			if expiredOnly {
				asset.UpdatedAt = time.Now()
				return dao.SaveAsset(asset)
			}
			return fmt.Errorf("资源已被作品引用，不能删除")
		}
		deleting = asset
		// 删除中状态先提交，禁止新发布引用；OSS删除成功但后续数据库写入失败时也不会产生坏引用。
		asset.State = 3
		return dao.SaveAsset(asset)
	})
	if err != nil || deleting == nil {
		return err
	}
	if deleting.UploadId != "" {
		// 已合并的任务不存在分片，取消后仍执行对象删除；重复删除保持可重试。
		abortErr := oss.MultipartAbort(ctx, deleting.ObjectKey, deleting.UploadId)
		if abortErr != nil && !strings.Contains(abortErr.Error(), "NoSuchUpload") {
			return fmt.Errorf("清理视频分片失败: %w", abortErr)
		}
	}
	err = oss.DeleteFile(ctx, deleting.ObjectKey)
	if err != nil {
		return fmt.Errorf("删除云端资源失败，可稍后重试: %w", err)
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewFileDAO(tx)
		asset, err := dao.FindAssetForUpdate(fileURL)
		if err != nil {
			return err
		}
		if asset == nil {
			return fmt.Errorf("资源记录不存在")
		}
		asset.State = 2
		return dao.SaveAsset(asset)
	})
}

// StartDiscoverAssetCleanup 每小时清理七天未使用且无有效引用的发现页资源。
func StartDiscoverAssetCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupDiscoverAssets(ctx)
			}
		}
	}()
}

// cleanupDiscoverAssets 分批清理过期资源，每项删除前重新锁定并检查引用。
func cleanupDiscoverAssets(ctx context.Context) {
	var afterId int64
	before := time.Now().Add(-7 * 24 * time.Hour)
	for {
		if ctx.Err() != nil {
			return
		}
		assets, err := userdao.NewFileDAO(db.GormDB.WithContext(ctx)).ListExpiredAssets(before, afterId, 100)
		if err != nil {
			zlog.Error("查询过期发现页资源失败: " + err.Error())
			return
		}
		if len(assets) == 0 {
			return
		}
		for _, asset := range assets {
			// 无论删除是否成功均推进游标，失败项下一轮重试，不堵住后续资源。
			afterId = asset.Id
			task, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := deleteDiscoverAsset(task, asset.UserId, asset.URL, true)
			cancel()
			if err != nil {
				zlog.Warn("清理发现页资源失败，资源=" + asset.ObjectKey + ": " + err.Error())
			}
		}
	}
}

// OpenDiscoverFile 每次读取媒体都校验资源归属或当前公开作品引用，禁止缓存绕过权限。
func (s *FileServiceImpl) OpenDiscoverFile(ctx context.Context, userId, fileURL, rangeHeader string) (*oss.ResourceStream, error) {
	fileURL = normalizeDiscoverMediaURL(fileURL)
	key := oss.ObjectKeyFromURL(fileURL)
	if userId == "" || !isDiscoverObjectKey(key) || fileURL != oss.FileURL(key) {
		return nil, fmt.Errorf("媒体地址无效或未登录")
	}
	dao := userdao.NewFileDAO(db.GormDB.WithContext(ctx))
	asset, err := dao.FindAsset(fileURL)
	if err != nil {
		return nil, err
	}
	if asset != nil && asset.State != 1 {
		return nil, fmt.Errorf("媒体已失效或正在上传")
	}
	if asset == nil || asset.UserId != userId {
		references, err := dao.CountReadableAssetReferences(fileURL, userId)
		if err != nil {
			return nil, err
		}
		if references == 0 {
			return nil, fmt.Errorf("无权查看此媒体")
		}
	}
	return oss.OpenResource(ctx, key, rangeHeader)
}

// DiscoverAssetStates 校验草稿中的远程文件是否仍属于当前用户且可用。
func (s *FileServiceImpl) DiscoverAssetStates(ctx context.Context, userId string, urls []string) (map[string]bool, error) {
	if len(urls) > 10 {
		return nil, fmt.Errorf("一次最多检查10个资源")
	}
	states := map[string]bool{}
	dao := userdao.NewFileDAO(db.GormDB.WithContext(ctx))
	for _, value := range urls {
		fileURL := normalizeDiscoverMediaURL(value)
		asset, err := dao.FindAsset(fileURL)
		if err != nil {
			return nil, err
		}
		states[value] = asset != nil && asset.UserId == userId && asset.State == 1
		if asset == nil {
			count, err := dao.CountOwnedAssetReferences(fileURL, userId)
			if err != nil {
				return nil, err
			}
			states[value] = count > 0
		}
	}
	return states, nil
}

// isDiscoverObjectKey 限制发现页对象目录并拒绝路径穿越标识。
func isDiscoverObjectKey(key string) bool {
	return (strings.HasPrefix(key, "discover/images/") || strings.HasPrefix(key, "discover/videos/")) && path.Clean(key) == key && !strings.Contains(key, "\\")
}

// validateDiscoverImage 校验真实图片格式并读取尺寸，失败时保留具体解码原因。
func validateDiscoverImage(file *multipart.FileHeader) (image.Config, string, error) {
	if file == nil || file.Size <= 0 || file.Size > DiscoverImageMaxBytes {
		return image.Config{}, "", fmt.Errorf("%w：图片大小需为1字节至10MB", ErrInvalidUpload)
	}
	ext := strings.ToLower(path.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return image.Config{}, "", fmt.Errorf("%w：仅支持JPG和PNG图片", ErrInvalidUpload)
	}
	source, err := file.Open()
	if err != nil {
		return image.Config{}, "", fmt.Errorf("读取图片失败: %w", err)
	}
	defer source.Close()
	cfg, format, err := image.DecodeConfig(source)
	if err != nil {
		return image.Config{}, "", fmt.Errorf("%w：无法解析图片内容，请确认文件为完整的PNG或JPEG图片: %w", ErrInvalidUpload, err)
	}
	if format != "jpeg" && format != "png" {
		return image.Config{}, "", fmt.Errorf("%w：文件实际格式为%s，仅支持PNG和JPEG", ErrInvalidUpload, format)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return image.Config{}, "", fmt.Errorf("%w：图片宽高必须为正数，当前为%dx%d", ErrInvalidUpload, cfg.Width, cfg.Height)
	}
	return cfg, format, nil
}

// discoverImageFilename 根据真实图片格式规范化对象文件后缀，不修改图片内容。
func discoverImageFilename(filename, format string) string {
	extension := ".png"
	if format == "jpeg" {
		extension = ".jpg"
	}
	return strings.TrimSuffix(filename, path.Ext(filename)) + extension
}
