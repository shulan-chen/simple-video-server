package stream

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"time"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var VIDEO_DIR = "./videos/"
var MAX_UPLOAD_SIZE int64 = 1024 * 1024 * 100 // 100MB

func streamLocalHandler(c *gin.Context) {
	traceID := c.GetString("trace_id")
	vid := c.Param("vid-id")
	video_storePath := VIDEO_DIR + vid

	video, err := os.Open(video_storePath)
	if err != nil {
		utils.Logger.Error("打开视频文件失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.String("path", video_storePath),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamFileRead, err)
		return
	}
	defer video.Close()

	c.Header("Content-Type", "video/mp4")
	http.ServeContent(c.Writer, c.Request, "", time.Now(), video)
}

func streamOssHandler(c *gin.Context) {
	traceID := c.GetString("trace_id")
	vid := c.Param("vid-id")

	targetUrl, err := GetOssVideoURL(c.Request.Context(), vid)
	if err != nil {
		utils.Logger.Error("生成OSS URL失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamOSSSignURL, err)
		return
	}

	c.Redirect(http.StatusMovedPermanently, targetUrl)
}

func uploadLocalHandler(c *gin.Context) {
	traceID := c.GetString("trace_id")
	req := c.Request
	vid := c.Param("vid-id")

	req.Body = http.MaxBytesReader(c.Writer, req.Body, MAX_UPLOAD_SIZE)
	if err := req.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		utils.Logger.Error("解析表单失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamFileTooLarge, err)
		return
	}

	file, _, err := req.FormFile("file")
	if err != nil {
		utils.Logger.Error("获取上传文件失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamMissingFile, err)
		return
	}
	defer file.Close()

	video_storePath := VIDEO_DIR + vid
	out, err := os.Create(video_storePath)
	if err != nil {
		utils.Logger.Error("创建文件失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.String("path", video_storePath),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamFileCreate, err)
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		utils.Logger.Error("写入文件失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamFileWrite, err)
		return
	}

	utils.Logger.Info("上传视频成功",
		zap.String("trace_id", traceID),
		zap.String("video_id", vid))

	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}

func uploadOssHandler(c *gin.Context) {
	traceID := c.GetString("trace_id")
	req := c.Request
	vid := c.Param("vid-id")

	req.Body = http.MaxBytesReader(c.Writer, req.Body, MAX_UPLOAD_SIZE)
	if err := req.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		utils.Logger.Error("解析表单失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamFileTooLarge, err)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		utils.Logger.Error("获取上传文件失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamMissingFile, err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if err = UploadToOSS(req.Context(), vid, file, contentType); err != nil {
		utils.Logger.Error("上传到OSS失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.String("content_type", contentType),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrStreamOSSUpload, err)
		return
	}

	utils.Logger.Info("上传视频到OSS成功",
		zap.String("trace_id", traceID),
		zap.String("video_id", vid))

	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}

func testPageHandler(c *gin.Context) {
	//page, err := os.Open("./videos/test_video.html")
	t, _ := template.ParseFiles("./videos/test_video.html")
	t.Execute(c.Writer, nil)
}
