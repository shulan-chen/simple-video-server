package api

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
	"video-server/api/utils"
	"video-server/internal/config"

	"go.uber.org/zap"
)

var httpClient *http.Client

func init() {
	httpClient = &http.Client{
		Timeout: 300 * time.Second, // 上传大文件需要较长超时
	}
}

// UploadVideoToStream 调用 Stream 服务上传视频文件
func UploadVideoToStream(videoID string, fileReader io.Reader, contentType string) error {
	// 构建 Stream 服务地址
	streamAddr := getStreamAddr()
	url := fmt.Sprintf("%s/videos/upload/%s", streamAddr, videoID)

	// 创建 multipart 请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建文件字段
	part, err := writer.CreateFormFile("file", videoID)
	if err != nil {
		return fmt.Errorf("创建表单字段失败: %w", err)
	}

	// 复制文件内容
	if _, err := io.Copy(part, fileReader); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 关闭 multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭表单失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	utils.Logger.Info("开始上传文件到Stream服务",
		zap.String("video_id", videoID),
		zap.String("url", url))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求Stream服务失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Stream服务返回错误: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	utils.Logger.Info("文件上传到Stream服务成功",
		zap.String("video_id", videoID))

	return nil
}

// getStreamAddr 获取 Stream 服务地址
func getStreamAddr() string {
	if config.AppConfig.StreamAddr != "" {
		addr := config.AppConfig.StreamAddr
		if addr[0] == ':' {
			return "http://localhost" + addr
		}
		if len(addr) > 7 && addr[:7] != "http://" && (len(addr) <= 8 || addr[:8] != "https://") {
			return "http://" + addr
		}
		return addr
	}
	return "http://localhost:9090"
}
