package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/session"
	"video-server/api/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
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
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(loginUser.Password), 10)
	if err != nil {
		c.JSON(500, gin.H{"error": "加密失败"})
		return
	}

	_, err = dbops.AddUser(loginUser.Username, string(hashedPwd))
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
	if err := bcrypt.CompareHashAndPassword([]byte(pUser.Password), []byte(loginUser.Password)); err != nil {
		sendErrorResponse(c.Writer, api.ErrorLoginPasswordWrong)
		return
	}

	// 生成 Access Token (15分钟)
	accessToken, err := utils.GenerateAccessToken(pUser.Username, pUser.Id)
	if err != nil {
		utils.Logger.Error("生成access token失败", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorInternalFaults)
		return
	}

	// 生成 Refresh Token (7天)
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		utils.Logger.Error("生成refresh token失败", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorInternalFaults)
		return
	}

	// 将 Refresh Token 存入 Redis
	if err := session.SaveRefreshToken(refreshToken, pUser.Id); err != nil {
		utils.Logger.Error("保存refresh token失败", zap.Error(err))
		sendErrorResponse(c.Writer, api.ErrorInternalFaults)
		return
	}

	// 返回双token
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    utils.GetAccessTokenTTL(), // 秒
		"token_type":    "Bearer",
	})
}

func Logout(c *gin.Context) {
	// 从请求体中获取 refresh_token
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendErrorResponse(c.Writer, api.ErrorRequestBodyParseFailed)
		return
	}

	// 从Redis删除 Refresh Token
	if err := session.DeleteRefreshToken(req.RefreshToken); err != nil {
		utils.Logger.Error("删除refresh token失败", zap.Error(err))
		// 即使删除失败也返回成功（可能token已经不存在）
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logout success",
	})
}

// RefreshToken 刷新 Access Token
func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	// 验证 Refresh Token
	userId, err := session.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// 从数据库获取用户信息
	user, err := dbops.GetUserById(userId)
	if err != nil {
		utils.Logger.Error("获取用户信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	// 生成新的 Access Token
	newAccessToken, err := utils.GenerateAccessToken(user.Username, userId)
	if err != nil {
		utils.Logger.Error("生成新access token失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// (可选) Refresh Token 轮换：生成新的 Refresh Token
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		utils.Logger.Error("生成新refresh token失败", zap.Error(err))
		// 不影响主流程，继续使用旧的 refresh token
		newRefreshToken = req.RefreshToken
	} else {
		// 轮换：删除旧的，保存新的
		if err := session.RotateRefreshToken(req.RefreshToken, newRefreshToken, userId); err != nil {
			utils.Logger.Error("轮换refresh token失败", zap.Error(err))
			// 失败时仍使用旧token
			newRefreshToken = req.RefreshToken
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    utils.GetAccessTokenTTL(),
		"token_type":    "Bearer",
	})
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
	uid := c.Request.Header.Get(HEADER_FILED_UID)
	uidInt, _ := strconv.Atoi(uid)
	fmt.Printf("ListUserAllVideos, uid %d\n", uidInt)
	videoInfos, err := dbops.GetUserAllVideos(uidInt)
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
	uid := c.Request.Header.Get(HEADER_FILED_UID)
	uidInt, _ := strconv.Atoi(uid)
	vid := c.Param("vid")
	existedVideo, err := dbops.GetVideoInfo(vid)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(c.Writer, api.ErrorVideoNotExisted)
			return
		}
	}
	if existedVideo.AuthorId != uidInt {
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
