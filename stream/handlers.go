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
	vid := c.Param("vid-id")
	video_storePath := VIDEO_DIR + vid

	video, err := os.Open(video_storePath)
	if err != nil {
		utils.Logger.Error("Open file error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	defer video.Close()
	c.Header("Content-Type", "video/mp4")
	http.ServeContent(c.Writer, c.Request, "", time.Now(), video)
}

func streamOssHandler(c *gin.Context) {
	vid := c.Param("vid-id")
	// 调用 GetOssVideoURL 获取带签名的 URL
	targetUrl, err := GetOssVideoURL(c.Request.Context(), vid)
	if err != nil {
		utils.Logger.Error("Get OSS URL error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	c.Redirect(http.StatusMovedPermanently, targetUrl)
}

func uploadLocalHandler(c *gin.Context) {
	req := c.Request
	req.Body = http.MaxBytesReader(c.Writer, req.Body, MAX_UPLOAD_SIZE)
	err := req.ParseMultipartForm(MAX_UPLOAD_SIZE)
	if err != nil {
		utils.Logger.Error("Parse multipart form error", zap.Error(err))
		c.String(http.StatusBadRequest, "File too large")
		return
	}

	file, _, err := req.FormFile("file")
	if err != nil {
		utils.Logger.Error("Get form file error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	vid := c.Param("vid-id")
	video_storePath := VIDEO_DIR + vid
	out, err := os.Create(video_storePath)
	if err != nil {
		utils.Logger.Error("Create file error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		utils.Logger.Error("Write file error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	c.String(http.StatusOK, "Upload success")
}

func uploadOssHandler(c *gin.Context) {
	req := c.Request
	req.Body = http.MaxBytesReader(c.Writer, req.Body, MAX_UPLOAD_SIZE)
	err := req.ParseMultipartForm(MAX_UPLOAD_SIZE)
	if err != nil {
		utils.Logger.Error("Parse multipart form error", zap.Error(err))
		c.String(http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		utils.Logger.Error("Get form file error", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")

	vid := c.Param("vid-id")
	err = UploadToOSS(req.Context(), vid, file, contentType)
	if err != nil {
		utils.Logger.Error("Upload to OSS error", zap.Error(err))
		c.String(http.StatusInternalServerError, "upload to OSS error")
		return
	}
	c.String(http.StatusOK, "Upload success")
}

func testPageHandler(c *gin.Context) {
	//page, err := os.Open("./videos/test_video.html")
	t, _ := template.ParseFiles("./videos/test_video.html")
	t.Execute(c.Writer, nil)
}
