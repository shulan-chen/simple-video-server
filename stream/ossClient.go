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

func init() {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AppConfig.OssKey, config.AppConfig.OssSecret)).
		WithRegion(config.AppConfig.OssRegion)

	ossClient = oss.NewClient(cfg)
}

func UploadToOSS(ctx context.Context, fileName string, fileData io.Reader, contentType string) error {

	putRequest := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(config.AppConfig.OssBucket),
		Key:         oss.Ptr(OSS_VIDEO_DIR + fileName),
		Body:        fileData,
		ContentType: oss.Ptr(contentType),
	}

	result, err := ossClient.PutObject(ctx, putRequest)
	if err != nil {
		utils.Logger.Error("上传到OSS失败",
			zap.String("file", fileName),
			zap.String("bucket", config.AppConfig.OssBucket),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("上传到OSS成功",
		zap.String("file", fileName),
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
		utils.Logger.Error("从OSS删除失败",
			zap.String("file", fileName),
			zap.String("bucket", config.AppConfig.OssBucket),
			zap.Error(err))
		return err
	}

	utils.Logger.Info("从OSS删除成功",
		zap.String("file", fileName))

	return nil
}

// 新增：获取预签名 URL
func GetOssVideoURL(ctx context.Context, fileName string) (string, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(config.AppConfig.OssBucket),
		Key:    oss.Ptr(OSS_VIDEO_DIR + fileName),
	}

	// 生成预签名 URL，有效期设置为 1 小时 (3600秒)
	result, err := ossClient.Presign(ctx, request, oss.PresignExpires(1*time.Hour))
	if err != nil {
		utils.Logger.Error("Sign URL error", zap.Error(err))
		return "", err
	}

	return result.URL, nil
}
