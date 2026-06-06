package oss

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

const (
	// MinPartSize OSS 分片最小 100KB（最后一片除外）
	MinPartSize = 100 * 1024
	// DefaultPartSize 默认分片大小 5MB
	DefaultPartSize = 5 * 1024 * 1024
)

// MultipartInitResult 分片上传初始化结果
type MultipartInitResult struct {
	UploadId  string `json:"uploadId"`
	ObjectKey string `json:"objectKey"`
	URL       string `json:"url"`
	PartSize  int64  `json:"partSize"`
}

// PartInfo 已上传分片信息
type PartInfo struct {
	PartNumber int32  `json:"partNumber"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

// MultipartInit 初始化分片上传任务
func MultipartInit(ctx context.Context, filename string, category FileCategory, contentType string) (*MultipartInitResult, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	objectKey := buildObjectKey(category, filename)

	if ctx == nil {
		ctx = context.Background()
	}

	req := &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(ossBucketName),
		Key:    oss.Ptr(objectKey),
	}
	if contentType != "" {
		req.ContentType = oss.Ptr(contentType)
	}

	result, err := client.InitiateMultipartUpload(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("初始化分片上传失败: %w", err)
	}

	uploadId := ""
	if result.UploadId != nil {
		uploadId = *result.UploadId
	}

	return &MultipartInitResult{
		UploadId:  uploadId,
		ObjectKey: objectKey,
		URL:       buildFileURL(objectKey),
		PartSize:  DefaultPartSize,
	}, nil
}

// MultipartUploadPart 上传单个分片
func MultipartUploadPart(ctx context.Context, objectKey, uploadId string, partNumber int32, body io.Reader, contentLength int64) (string, error) {
	client, err := getClient()
	if err != nil {
		return "", err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	result, err := client.UploadPart(ctx, &oss.UploadPartRequest{
		Bucket:        oss.Ptr(ossBucketName),
		Key:           oss.Ptr(objectKey),
		UploadId:      oss.Ptr(uploadId),
		PartNumber:    partNumber,
		Body:          body,
		ContentLength: oss.Ptr(contentLength),
	})
	if err != nil {
		return "", fmt.Errorf("上传分片失败(part=%d): %w", partNumber, err)
	}

	if result.ETag == nil {
		return "", fmt.Errorf("上传分片失败(part=%d): 未返回ETag", partNumber)
	}

	return strings.Trim(*result.ETag, `"`), nil
}

// MultipartListParts 查询已上传的分片列表（用于断点续传）
func MultipartListParts(ctx context.Context, objectKey, uploadId string) ([]PartInfo, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var allParts []PartInfo
	partNumberMarker := int32(0)

	for {
		result, err := client.ListParts(ctx, &oss.ListPartsRequest{
			Bucket:           oss.Ptr(ossBucketName),
			Key:              oss.Ptr(objectKey),
			UploadId:         oss.Ptr(uploadId),
			MaxParts:         1000,
			PartNumberMarker: partNumberMarker,
		})
		if err != nil {
			return nil, fmt.Errorf("查询已上传分片失败: %w", err)
		}

		for _, p := range result.Parts {
			etag := ""
			if p.ETag != nil {
				etag = strings.Trim(*p.ETag, `"`)
			}
			allParts = append(allParts, PartInfo{
				PartNumber: p.PartNumber,
				ETag:       etag,
				Size:       p.Size,
			})
		}

		if !result.IsTruncated {
			break
		}
		partNumberMarker = result.NextPartNumberMarker
	}

	sort.Slice(allParts, func(i, j int) bool {
		return allParts[i].PartNumber < allParts[j].PartNumber
	})

	return allParts, nil
}

// MultipartComplete 完成分片上传并合并文件
func MultipartComplete(ctx context.Context, objectKey, uploadId string, parts []PartInfo) (*UploadResult, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	uploadParts := make([]oss.UploadPart, len(parts))
	for i, p := range parts {
		etag := p.ETag
		uploadParts[i] = oss.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       oss.Ptr(etag),
		}
	}

	_, err = client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(ossBucketName),
		Key:      oss.Ptr(objectKey),
		UploadId: oss.Ptr(uploadId),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: uploadParts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("合并分片失败: %w", err)
	}

	return &UploadResult{
		ObjectKey: objectKey,
		URL:       buildFileURL(objectKey),
	}, nil
}

// MultipartAbort 取消分片上传任务
func MultipartAbort(ctx context.Context, objectKey, uploadId string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	_, err = client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
		Bucket:   oss.Ptr(ossBucketName),
		Key:      oss.Ptr(objectKey),
		UploadId: oss.Ptr(uploadId),
	})
	if err != nil {
		return fmt.Errorf("取消分片上传失败: %w", err)
	}

	return nil
}

// ParseFileCategory 解析文件业务类型
func ParseFileCategory(fileTypeStr string) FileCategory {
	switch fileTypeStr {
	case "user_avatar":
		return UserAvatar
	case "group_avatar":
		return GroupAvatar
	case "message_image":
		return MessageImage
	case "message_video":
		return MessageVideo
	case "message_audio":
		return MessageAudio
	case "knowledge_doc":
		return KnowledgeDoc
	case "discover_image":
		return DiscoverImage
	case "discover_video":
		return DiscoverVideo
	default:
		return MessageImage
	}
}

// IsVideoCategory 是否为视频类型（需走分片上传）
func IsVideoCategory(category FileCategory) bool {
	return category == MessageVideo || category == DiscoverVideo
}
