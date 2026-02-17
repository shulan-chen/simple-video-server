package api

import (
	"database/sql"
	"net/http"
	"time"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/session"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CreateUser(c *gin.Context) {
	loginUser := &api.UserDTO{}

	err := c.ShouldBindJSON(loginUser)
	if err != nil {
		sendErrorResponse(c.Writer, api.ErrorRequestBodyParseFailed)
		return
	}
	exitedUser, err := dbops.GetUserByName(loginUser.Username)
	if err == nil && exitedUser != nil {
		sendErrorResponse(c.Writer, api.ErrorUserExisted)
		return
	}

	_, err = dbops.AddUser(loginUser.Username, loginUser.Password)
	if err != nil {
		utils.Logger.Error("AddUser failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	sendNormalResponse(c.Writer, "register success", 201)
}

func Login(c *gin.Context) {
	uname := c.Param("user_name")
	loginUser := &api.UserDTO{}

	err := c.ShouldBindJSON(loginUser)
	if err != nil {
		sendErrorResponse(c.Writer, api.ErrorRequestBodyParseFailed)
		return
	}
	loginUser.Username = uname
	//fmt.Println(*loginUser)

	pUser, err := dbops.GetUserByName(uname)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(c.Writer, api.ErrorNotAuthUser)
			return
		}
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	if pUser.Password != loginUser.Password {
		sendErrorResponse(c.Writer, api.ErrorLoginPasswordWrong)
		return
	}

	simplesession, err := session.AddNewSession(pUser.Id, pUser.Username)
	su := api.SignedUP{
		SessionId: simplesession.SessionId,
		Success:   true,
	}
	c.JSON(http.StatusOK, su)
}

func Logout(c *gin.Context) {
	sid := c.Request.Header.Get(HEADER_FILED_SESSION)
	session.DeleteSession(sid)
	sendNormalResponse(c.Writer, "logout success", 200)
}

func GetUserInfo(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	user, err := dbops.GetUserByName(c.Param("user_name"))
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	c.JSON(http.StatusOK, user)
}

func AddNewVideo(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	userNewVideoDTO := &api.UserAddNewVideoDTO{}
	err := c.ShouldBindJSON(userNewVideoDTO)
	if err != nil {
		sendErrorResponse(c.Writer, api.ErrorRequestBodyParseFailed)
		return
	}
	videoInfo, err := dbops.AddNewVideo(userNewVideoDTO.AuthorId, userNewVideoDTO.Name)
	if err != nil {
		utils.Logger.Error("AddNewVideo failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	c.JSON(http.StatusOK, videoInfo)
}

func ListUserAllVideos(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}

	uname := c.Param("user_name")
	user, err := dbops.GetUserByName(uname)
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	videoInfos, err := dbops.GetUserAllVideos(user.Id)
	if err != nil {
		utils.Logger.Error("GetUserAllVideos failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	videoInfoDTO := &api.VideoInfoDTO{}
	videoInfoDTO.Videos = videoInfos
	c.JSON(http.StatusOK, videoInfoDTO)
}

func ListAllVideos(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	videoInfos, err := dbops.GetAllVideoInfo()
	if err != nil {
		utils.Logger.Error("ListAllVideos failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	videoInfoDTO := &api.VideoInfoDTO{}
	videoInfoDTO.Videos = videoInfos
	c.JSON(http.StatusOK, videoInfoDTO)
}

func DeleteVideoInfo(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	userName := c.Param("user_name")
	user, err := dbops.GetUserByName(userName)
	if err != nil {
		utils.Logger.Error("GetUserByName failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	vid := c.Param("vid")
	existedVideo, err := dbops.GetVideoInfo(vid)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(c.Writer, api.ErrorVideoNotExisted)
			return
		}
	}
	if existedVideo.AuthorId != user.Id {
		sendErrorResponse(c.Writer, api.ErrorVideoNotMatchToUser)
		return
	}
	err = dbops.DeleteVideoInfo(vid)
	if err != nil {
		utils.Logger.Error("DeleteVideoInfo failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}

	err = dbops.InsertNewVideoDeletionRecord(vid)
	if err != nil {
		sendErrorResponse(c.Writer, api.ErrorInternalFaults)
		return
	}
	sendNormalResponse(c.Writer, "ok", 200)
}

func PostComments(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	vid := c.Param("vid")
	userComment := &api.PostCommentsDTO{}
	err := c.ShouldBindJSON(userComment)
	if err != nil {
		utils.Logger.Error("PostComments failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorRequestBodyParseFailed)
		return
	}
	err = dbops.InsertNewComments(vid, userComment.AuthorId, userComment.Content)
	if err != nil {
		utils.Logger.Error("InsertNewComments failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	sendNormalResponse(c.Writer, "ok", 201)
}

func ListComments(c *gin.Context) {
	if !ValidateUser(c.Writer, c.Request) {
		return
	}
	vid := c.Param("vid")

	comments, err := dbops.ListComments(vid, time.Unix(0, 0), time.Now())
	if err != nil {
		utils.Logger.Error("ListComments failed", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorDBError)
		return
	}
	commentsDTO := &api.CommentsDTO{}
	commentsDTO.Comments = comments
	c.JSON(http.StatusOK, commentsDTO)
}
