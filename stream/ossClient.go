package stream

import (
	"context"
	"io"
	"time"
	"video-server/api/utils"
	"video-server/internal/config"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"go.uber.org/zap"
)

const (
	OSS_VIDEO_DIR = "videos/"
)

var ossClient *oss.Client

// InitOSSClient 初始化 OSS 客户端（必须在 config 加载后调用）
func InitOSSClient() {
	// 构建完整的 Endpoint URL
	endpoint := "https://" + config.AppConfig.OssAddr

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AppConfig.OssKey, config.AppConfig.OssSecret)).
		WithRegion(config.AppConfig.OssRegion).
		WithEndpoint(endpoint)

	ossClient = oss.NewClient(cfg)

	utils.Logger.Info("OSS 客户端初始化成功",
		zap.String("endpoint", endpoint),
		zap.String("region", config.AppConfig.OssRegion),
		zap.String("bucket", config.AppConfig.OssBucket))
}

func UploadToOSS(ctx context.Context, objectKey string, fileData io.Reader, contentType string) error {
	// objectKey 直接使用传入的完整路径（videos/xxx 或 titlePage/xxx）
	putRequest := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(config.AppConfig.OssBucket),
		Key:         oss.Ptr(objectKey),
		Body:        fileData,
		ContentType: oss.Ptr(contentType),
	}

	result, err := ossClient.PutObject(ctx, putRequest)
	if err != nil {
		utils.Logger.Error("上传到OSS失败",
			zap.String("object_key", objectKey),
			zap.String("bucket", config.AppConfig.OssBucket),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("上传到OSS成功",
		zap.String("object_key", objectKey),
		zap.String("etag", *result.ETag))

	return nil
}


func DeleteFromOSS(ctx context.Context, fileName string) error {
	deleteRequest := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(config.AppConfig.OssBucket),
		Key:    oss.Ptr(OSS_VIDEO_DIR + fileName),
	}

	_, err := ossClient.DeleteObject(ctx, deleteRequest)
	if err != nil {
		utils.Logger.Error("从OSS删除视频失败",
			zap.String("file", fileName),
			zap.String("bucket", config.AppConfig.OssBucket),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("从OSS删除视频成功",
		zap.String("file", fileName))

	return nil
}

// DeleteThumbnailFromOSS 从 OSS 删除封面图片（titlePage/{vid}.jpg）
func DeleteThumbnailFromOSS(ctx context.Context, vid string) error {
	objectKey := "titlePage/" + vid + ".jpg"
	deleteRequest := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(config.AppConfig.OssBucket),
		Key:    oss.Ptr(objectKey),
	}

	_, err := ossClient.DeleteObject(ctx, deleteRequest)
	if err != nil {
		utils.Logger.Error("从OSS删除封面失败",
			zap.String("vid", vid),
			zap.String("object_key", objectKey),
			zap.String("bucket", config.AppConfig.OssBucket),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("从OSS删除封面成功",
		zap.String("vid", vid),
		zap.String("object_key", objectKey))

	return nil
}

// 获取预签名 URL（通用方法）
func GetOSSSignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(config.AppConfig.OssBucket),
		Key:    oss.Ptr(objectKey),
	}

	// 生成预签名 URL
	result, err := ossClient.Presign(ctx, request, oss.PresignExpires(expiry))
	if err != nil {
		utils.Logger.Error("生成签名URL失败",
			zap.String("object_key", objectKey),
			zap.Error(err))
		return "", err
	}

	return result.URL, nil
}

// GetOssVideoURL 获取视频的预签名 URL（兼容旧接口）
func GetOssVideoURL(ctx context.Context, fileName string) (string, error) {
	return GetOSSSignedURL(ctx, OSS_VIDEO_DIR+fileName, 12*time.Hour)
}

// RenameThumbnailInOSS 将封面从 oldKey 复制到 newKey 后删除 oldKey。
// 用于将上传时的临时路径（titlePage/temp_TIMESTAMP.jpg）规范化为 titlePage/{vid}.jpg。
func RenameThumbnailInOSS(ctx context.Context, oldKey, newKey string) error {
	_, err := ossClient.CopyObject(ctx, &oss.CopyObjectRequest{
		Bucket:    oss.Ptr(config.AppConfig.OssBucket),
		Key:       oss.Ptr(newKey),
		SourceKey: oss.Ptr(oldKey),
	})
	if err != nil {
		utils.Logger.Error("复制封面对象失败",
			zap.String("old_key", oldKey),
			zap.String("new_key", newKey),
			zap.Error(err))
		return err
	}

	_, err = ossClient.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(config.AppConfig.OssBucket),
		Key:    oss.Ptr(oldKey),
	})
	if err != nil {
		utils.Logger.Warn("删除旧封面对象失败（复制已成功）",
			zap.String("old_key", oldKey),
			zap.Error(err))
	}

	utils.Logger.Info("封面重命名成功",
		zap.String("old_key", oldKey),
		zap.String("new_key", newKey))
	return nil
}
