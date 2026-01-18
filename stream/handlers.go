package stream

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"time"
	"video-server/api/utils"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var VIDEO_DIR = "./videos/"
var MAX_UPLOAD_SIZE int64 = 1024 * 1024 * 100 // 100MB

func streamHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	vid := param.ByName("vid-id")
	video_storePath := VIDEO_DIR + vid

	video, err := os.Open(video_storePath)
	if err != nil {
		utils.Logger.Error("Open file error", zap.Error(err))
		sendErrorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer video.Close()
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, req, "", time.Now(), video)
}

func uploadHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	req.Body = http.MaxBytesReader(w, req.Body, MAX_UPLOAD_SIZE)
	err := req.ParseMultipartForm(MAX_UPLOAD_SIZE)
	if err != nil {
		utils.Logger.Error("Parse multipart form error", zap.Error(err))
		sendErrorResponse(w, http.StatusBadRequest, "File too large")
		return
	}

	file, _, err := req.FormFile("file")
	if err != nil {
		utils.Logger.Error("Get form file error", zap.Error(err))
		sendErrorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	vid := param.ByName("vid-id")
	video_storePath := VIDEO_DIR + vid
	out, err := os.Create(video_storePath)
	if err != nil {
		utils.Logger.Error("Create file error", zap.Error(err))
		sendErrorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		utils.Logger.Error("Write file error", zap.Error(err))
		sendErrorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Upload success")
}

func testPageHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	//page, err := os.Open("./videos/test_video.html")
	t, _ := template.ParseFiles("./videos/test_video.html")
	t.Execute(w, nil)
}
