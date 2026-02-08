package api

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/session"
	"video-server/api/utils"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

func CreateUser(w http.ResponseWriter, req *http.Request, param httprouter.Params) {

	reqBody, _ := ioutil.ReadAll(req.Body)
	loginUser := &api.UserDTO{}

	err := json.Unmarshal(reqBody, loginUser)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}
	exitedUser, err := dbops.GetUserByName(loginUser.Username)
	if err == nil && exitedUser != nil {
		sendErrorResponse(w, api.ErrorUserExisted)
		return
	}

	_, err = dbops.AddUser(loginUser.Username, loginUser.Password)
	if err != nil {
		utils.Logger.Error("AddUser failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	sendNormalResponse(w, "register success", 201)
}

func Login(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	uname := param.ByName("user_name")
	res, _ := ioutil.ReadAll(req.Body)
	loginUser := &api.UserDTO{}

	err := json.Unmarshal(res, loginUser)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}
	loginUser.Username = uname
	//fmt.Println(*loginUser)

	pUser, err := dbops.GetUserByName(uname)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, api.ErrorNotAuthUser)
			return
		}
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	if pUser.Password != loginUser.Password {
		sendErrorResponse(w, api.ErrorLoginPasswordWrong)
		return
	}

	simplesession, err := session.AddNewSession(pUser.Id, pUser.Username)
	su := api.SignedUP{
		SessionId: simplesession.SessionId,
		Success:   true,
	}
	if resp, err := json.Marshal(su); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}
}

func Logout(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	sid := req.Header.Get(HEADER_FILED_SESSION)
	session.DeleteSession(sid)
	sendNormalResponse(w, "logout success", 200)
}

func GetUserInfo(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}

	user, err := dbops.GetUserByName(param.ByName("user_name"))
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}

	if resp, err := json.Marshal(user); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}

}

func AddNewVideo(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}
	//uname := param.ByName("user_name")

	reqBody, _ := ioutil.ReadAll(req.Body)
	userNewVideoDTO := &api.UserAddNewVideoDTO{}
	err := json.Unmarshal(reqBody, userNewVideoDTO)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}
	videoInfo, err := dbops.AddNewVideo(userNewVideoDTO.AuthorId, userNewVideoDTO.Name)
	if err != nil {
		utils.Logger.Error("AddNewVideo failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	if resp, err := json.Marshal(videoInfo); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 201)
	}
}

func ListUserAllVideos(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}

	uname := param.ByName("user_name")
	user, err := dbops.GetUserByName(uname)
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	videoInfos, err := dbops.GetUserAllVideos(user.Id)
	if err != nil {
		utils.Logger.Error("GetUserAllVideos failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	videoInfoDTO := &api.VideoInfoDTO{}
	videoInfoDTO.Videos = videoInfos
	if resp, err := json.Marshal(videoInfoDTO); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}

}

func ListAllVideos(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}
	videoInfos, err := dbops.GetAllVideoInfo()
	if err != nil {
		utils.Logger.Error("ListAllVideos failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	videoInfoDTO := &api.VideoInfoDTO{}
	videoInfoDTO.Videos = videoInfos
	if resp, err := json.Marshal(videoInfoDTO); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}
}

func DeleteVideoInfo(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}
	userName := param.ByName("user_name")
	user, err := dbops.GetUserByName(userName)
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	vid := param.ByName("vid")
	existedVideo, err := dbops.GetVideoInfo(vid)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, api.ErrorVideoNotExisted)
			return
		}
	}
	if existedVideo.AuthorId != user.Id {
		sendErrorResponse(w, api.ErrorVideoNotMatchToUser)
		return
	}
	err = dbops.DeleteVideoInfo(vid)
	if err != nil {
		utils.Logger.Error("DeleteVideoInfo failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}

	err = dbops.InsertNewVideoDeletionRecord(vid)
	if err != nil {
		http.Error(w, "Failed to schedule video delete record task", http.StatusInternalServerError)
		return
	}
	sendNormalResponse(w, "ok", 200)
}

func PostComments(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}
	vid := param.ByName("vid")
	reqBody, _ := ioutil.ReadAll(req.Body)
	userComment := &api.PostCommentsDTO{}
	err := json.Unmarshal(reqBody, userComment)
	if err != nil {
		utils.Logger.Error("PostComments failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}
	err = dbops.InsertNewComments(vid, userComment.AuthorId, userComment.Content)
	if err != nil {
		utils.Logger.Error("InsertNewComments failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	sendNormalResponse(w, "ok", 201)
}

func ListComments(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if !ValidateUser(w, req) {
		return
	}
	vid := param.ByName("vid")

	comments, err := dbops.ListComments(vid, time.Unix(0, 0), time.Now())
	if err != nil {
		utils.Logger.Error("ListComments failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	commentsDTO := &api.CommentsDTO{}
	commentsDTO.Comments = comments
	if resp, err := json.Marshal(commentsDTO); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}
}
