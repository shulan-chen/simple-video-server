package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
	"video-server/api/cache"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/utils"
	"video-server/stream"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	HEADER_FILED_SESSION = "X-Session-Id"
	HEADER_FILED_UNAME   = "X-User-Name"
	HEADER_FILED_UID     = "X-User-Id"
)

// thumbnailSignedURLTTL 封面签名 URL 的有效期。
// 签名 URL 只在响应时现生成，不存入数据库，TTL 只影响客户端缓存时间。
const thumbnailSignedURLTTL = 7 * 24 * time.Hour

// CreateUser 用户注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user  body      api.UserDTO  true  "用户信息"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  utils.AppError
// @Failure      409   {object}  utils.AppError  "用户已存在"
// @Failure      500   {object}  utils.AppError
// @Router       /user [post]
func CreateUser(c *gin.Context) {
	traceID := c.GetString("trace_id")
	loginUser := &api.UserDTO{}

	// 解析请求体
	if err := c.ShouldBindJSON(loginUser); err != nil {
		utils.Logger.Error("解析请求体失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIInvalidRequest, err)
		return
	}

	// 检查用户是否已存在
	exitedUser, err := dbops.GetUserByName(loginUser.Username)
	if err == nil && exitedUser != nil {
		utils.Logger.Warn("用户已存在",
			zap.String("trace_id", traceID),
			zap.String("username", loginUser.Username))
		utils.AbortWithErrorMsg(c, utils.ErrAPIUserExisted, "")
		return
	}

	// 加密密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(loginUser.Password), 10)
	if err != nil {
		utils.Logger.Error("密码加密失败",
			zap.String("trace_id", traceID),
			zap.String("username", loginUser.Username),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIEncryptFailed, err)
		return
	}

	// 保存用户
	_, err = dbops.AddUser(loginUser.Username, string(hashedPwd))
	if err != nil {
		utils.Logger.Error("保存用户失败",
			zap.String("trace_id", traceID),
			zap.String("username", loginUser.Username),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}

	utils.Logger.Info("用户注册成功",
		zap.String("trace_id", traceID),
		zap.String("username", loginUser.Username))
	c.JSON(http.StatusCreated, gin.H{"message": "注册成功"})
}

// Login 用户登录
// @Summary      用户登录
// @Description  用户登录获取Access Token和Refresh Token
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user_name  path      string       true  "用户名"
// @Param        user       body      api.UserDTO  true  "登录信息"
// @Success      200        {object}  map[string]interface{}  "返回access_token、refresh_token等"
// @Failure      400        {object}  utils.AppError
// @Failure      401        {object}  utils.AppError  "用户不存在或密码错误"
// @Failure      500        {object}  utils.AppError
// @Router       /user/{user_name} [post]
func Login(c *gin.Context) {
	traceID := c.GetString("trace_id")
	uname := c.Param("user_name")
	loginUser := &api.UserDTO{}

	// 解析请求体
	if err := c.ShouldBindJSON(loginUser); err != nil {
		utils.Logger.Error("解析请求体失败",
			zap.String("trace_id", traceID),
			zap.String("username", uname),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIInvalidRequest, err)
		return
	}
	loginUser.Username = uname

	// 查询用户
	pUser, err := dbops.GetUserByName(uname)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Logger.Warn("用户不存在",
				zap.String("trace_id", traceID),
				zap.String("username", uname))
			utils.AbortWithErrorMsg(c, utils.ErrAPIUserNotFound, "")
			return
		}
		utils.Logger.Error("查询用户失败",
			zap.String("trace_id", traceID),
			zap.String("username", uname),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(pUser.Password), []byte(loginUser.Password)); err != nil {
		utils.Logger.Warn("密码错误",
			zap.String("trace_id", traceID),
			zap.String("username", uname))
		utils.AbortWithErrorMsg(c, utils.ErrAPIPasswordWrong, "")
		return
	}

	// 生成 Access Token
	accessToken, err := utils.GenerateAccessToken(pUser.Username, pUser.Id)
	if err != nil {
		utils.Logger.Error("生成access token失败",
			zap.String("trace_id", traceID),
			zap.String("username", uname),
			zap.Int("user_id", pUser.Id),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPITokenGenerate, err)
		return
	}

	// 生成 Refresh Token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		utils.Logger.Error("生成refresh token失败",
			zap.String("trace_id", traceID),
			zap.String("username", uname),
			zap.Int("user_id", pUser.Id),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPITokenGenerate, err)
		return
	}

	// 保存 Refresh Token 到 Redis
	if err := cache.SaveRefreshToken(c.Request.Context(), pUser.Id, refreshToken); err != nil {
		utils.Logger.Error("保存refresh token失败",
			zap.String("trace_id", traceID),
			zap.String("username", uname),
			zap.Int("user_id", pUser.Id),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIRedisOperation, err)
		return
	}

	utils.Logger.Info("用户登录成功",
		zap.String("trace_id", traceID),
		zap.String("username", uname),
		zap.Int("user_id", pUser.Id),
		zap.String("ip", c.ClientIP()))

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    utils.GetAccessTokenTTL(),
		"token_type":    "Bearer",
	})
}

// Logout 用户登出
// @Summary      用户登出
// @Description  删除Refresh Token，使用户退出登录
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user_name      path      string  true  "用户名"
// @Param        refresh_token  body      object  true  "Refresh Token"  example({"refresh_token": "xxx"})
// @Success      200            {object}  map[string]string
// @Failure      400            {object}  utils.AppError
// @Security     Bearer
// @Router       /user/{user_name}/logout [post]
func Logout(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userName := c.GetString("user_name")
	userID := c.GetString("user_id")

	userIDInt, _ := strconv.Atoi(userID)

	// 删除 Refresh Token（根据 userId）
	if err := cache.DeleteRefreshToken(c.Request.Context(), userIDInt); err != nil {
		utils.Logger.Error("删除refresh token失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("user_id", userID),
			zap.Error(err))
		// 即使失败也继续（token可能已不存在）
	}

	utils.Logger.Info("用户登出成功",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.String("user_id", userID))

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// RefreshToken 刷新Access Token
// @Summary      刷新Token
// @Description  使用 userId 自动获取 Refresh Token 并刷新 Access Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  utils.AppError  "Refresh Token无效"
// @Security     Bearer
// @Router       /auth/refresh [post]
func RefreshToken(c *gin.Context) {
	traceID := c.GetString("trace_id")

	// 从 access_token 中解析 userId（即使过期也能解析）
	token := c.GetHeader("X-Session-Id")
	if token == "" {
		utils.Logger.Error("缺少 access token",
			zap.String("trace_id", traceID))
		utils.AbortWithErrorMsg(c, utils.ErrAPIUnauthorized, "")
		return
	}

	// 使用宽容模式解析（允许过期的 token）
	claims, err := utils.ParseTokenLenient(token)
	if err != nil {
		utils.Logger.Error("解析 token 失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPITokenExpired, err)
		return
	}

	userId := claims.UserId

	// 从 Redis 获取该用户的 Refresh Token
	refreshToken, err := cache.GetRefreshToken(c.Request.Context(), userId)
	if err != nil {
		utils.Logger.Warn("Refresh token不存在或已过期",
			zap.String("trace_id", traceID),
			zap.Int("user_id", userId),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIRefreshTokenInvalid, err)
		return
	}

	// 1. 先查缓存（根据userId）
	user, err := cache.GetUserById(c.Request.Context(), userId)
	if err == nil {
		utils.Logger.Debug("用户信息缓存命中",
			zap.String("trace_id", traceID),
			zap.Int("user_id", userId))
	} else {
		// 2. 缓存未命中，查数据库
		user, err = dbops.GetUserById(userId)
		if err != nil {
			utils.Logger.Error("获取用户信息失败",
				zap.String("trace_id", traceID),
				zap.Int("user_id", userId),
				zap.Error(err))
			utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
			return
		}

		// 3. 写入缓存
		if err := cache.SetUser(c.Request.Context(), user); err != nil {
			utils.Logger.Warn("写入用户缓存失败",
				zap.String("trace_id", traceID),
				zap.Int("user_id", userId),
				zap.Error(err))
		}
	}

	// 生成新的 Access Token
	newAccessToken, err := utils.GenerateAccessToken(user.Username, userId)
	if err != nil {
		utils.Logger.Error("生成新access token失败",
			zap.String("trace_id", traceID),
			zap.String("username", user.Username),
			zap.Int("user_id", userId),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPITokenGenerate, err)
		return
	}

	// Refresh Token 轮换（可选：生成新的 refresh token）
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		utils.Logger.Error("生成新refresh token失败",
			zap.String("trace_id", traceID),
			zap.String("username", user.Username),
			zap.Int("user_id", userId),
			zap.Error(err))
		newRefreshToken = refreshToken // 失败则使用旧的
	} else {
		if err := cache.RotateRefreshToken(c.Request.Context(), userId, newRefreshToken); err != nil {
			utils.Logger.Error("轮换refresh token失败",
				zap.String("trace_id", traceID),
				zap.String("username", user.Username),
				zap.Int("user_id", userId),
				zap.Error(err))
			newRefreshToken = refreshToken // 失败则使用旧的
		}
	}

	utils.Logger.Info("刷新token成功",
		zap.String("trace_id", traceID),
		zap.String("username", user.Username),
		zap.Int("user_id", userId))

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    utils.GetAccessTokenTTL(),
		"token_type":    "Bearer",
	})
}

// GetUserInfo 获取用户信息
// @Summary      获取用户信息
// @Description  根据用户名获取用户详细信息
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        user_name  path      string  true  "用户名"
// @Success      200        {object}  api.User
// @Failure      404        {object}  utils.AppError  "用户不存在"
// @Failure      500        {object}  utils.AppError
// @Security     Bearer
// @Router       /user/{user_name} [get]
func GetUserInfo(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userName := c.Param("user_name")

	// GetUserInfo接口按username查询，不适合缓存（因为key是userId）
	// 如果要缓存需要建立username->userId映射，增加复杂度
	// 此接口不是高频调用，直接查数据库
	user, err := dbops.GetUserByName(userName)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Logger.Warn("用户不存在",
				zap.String("trace_id", traceID),
				zap.String("username", userName))
			utils.AbortWithErrorMsg(c, utils.ErrAPIUserNotFound, "")
			return
		}
		utils.Logger.Error("查询用户失败",
			zap.String("trace_id", traceID),
			zap.String("username", userName),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// AddNewVideo 添加视频（事务性上传：视频文件 + 封面 + 元数据，全部由后端处理）
// @Summary      上传视频
// @Description  一次请求上传视频文件和封面，后端内部生成 vid、上传 OSS、写入数据库
// @Tags         视频管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        user_name   path      string  true  "用户名"
// @Param        file        formData  file    true  "视频文件"
// @Param        thumbnail   formData  file    true  "封面图片"
// @Param        video_name  formData  string  true  "视频名称"
// @Success      200         {object}  api.VideoInfo
// @Failure      400         {object}  utils.AppError
// @Failure      401         {object}  utils.AppError  "未登录或登录过期"
// @Failure      500         {object}  utils.AppError
// @Security     Bearer
// @Router       /user/{user_name}/videos [post]
func AddNewVideo(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")

	utils.Logger.Debug("开始事务性视频上传",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.String("user_id", userID))

	if err := c.Request.ParseMultipartForm(200 << 20); err != nil { // 200MB（视频+封面）
		utils.AbortWithError(c, utils.ErrAPIInvalidRequest, err)
		return
	}

	// 获取视频文件
	videoFile, videoHeader, err := c.Request.FormFile("file")
	if err != nil {
		utils.AbortWithErrorMsg(c, utils.ErrAPIInvalidRequest, "缺少视频文件")
		return
	}
	defer videoFile.Close()

	// 获取封面文件
	thumbnailFile, _, err := c.Request.FormFile("thumbnail")
	if err != nil {
		utils.AbortWithErrorMsg(c, utils.ErrAPIInvalidRequest, "缺少封面图片")
		return
	}
	defer thumbnailFile.Close()

	// 获取视频名称
	videoName := c.PostForm("video_name")
	if videoName == "" {
		utils.AbortWithErrorMsg(c, utils.ErrAPIInvalidRequest, "缺少 video_name 参数")
		return
	}

	userIDInt, _ := strconv.Atoi(userID)

	// ========== 步骤1：生成 vid，写入数据库 ==========
	videoInfo, err := dbops.AddNewVideo("", userIDInt, videoName, "")
	if err != nil {
		utils.Logger.Error("创建视频记录失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}
	videoID := videoInfo.Vid

	rollback := func() {
		if deleteErr := dbops.DeleteVideoInfo(videoID); deleteErr != nil {
			utils.Logger.Error("回滚删除DB记录失败",
				zap.String("trace_id", traceID),
				zap.String("video_id", videoID),
				zap.Error(deleteErr))
		}
	}

	// ========== 步骤2：上传视频到 Stream 服务（保留限流能力）==========
	videoFile.Seek(0, 0)
	contentType := videoHeader.Header.Get("Content-Type")
	if err := UploadVideoToStream(videoID, videoFile, contentType); err != nil {
		utils.Logger.Error("上传视频到OSS失败，回滚",
			zap.String("trace_id", traceID),
			zap.String("video_id", videoID),
			zap.Error(err))
		rollback()
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}

	// ========== 步骤3：直接上传封面到 OSS（API 服务内部调用，不经过 HTTP）==========
	thumbnailKey := "titlePage/" + videoID + ".jpg"
	if err := stream.UploadToOSS(c.Request.Context(), thumbnailKey, thumbnailFile, "image/jpeg"); err != nil {
		utils.Logger.Error("上传封面到OSS失败，回滚",
			zap.String("trace_id", traceID),
			zap.String("video_id", videoID),
			zap.Error(err))
		rollback()
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}

	// ========== 步骤4：更新封面 object key 到数据库 ==========
	if err := dbops.UpdateVideoThumbnail(videoID, thumbnailKey); err != nil {
		utils.Logger.Error("更新封面记录失败，回滚",
			zap.String("trace_id", traceID),
			zap.String("video_id", videoID),
			zap.Error(err))
		rollback()
		utils.AbortWithError(c, utils.ErrAPIDBUpdate, err)
		return
	}

	ctx := c.Request.Context()
	cache.InvalidateAllVideos(ctx)
	cache.InvalidateUserVideos(ctx, userIDInt)

	utils.Logger.Info("视频上传完成",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.String("video_id", videoID),
		zap.String("video_name", videoName))

	videoInfo.ThumbnailUrl = thumbnailKey
	c.JSON(http.StatusOK, videoInfo)
}

// enrichThumbnailURLs 将视频列表中存储的 object key 转换为签名 URL。
// DB 存的是 object key（如 "titlePage/xxx.jpg"），响应时现生成签名 URL 避免过期问题。
func enrichThumbnailURLs(ctx context.Context, videos []*api.VideoInfo) {
	for _, v := range videos {
		if v.ThumbnailUrl == "" || strings.HasPrefix(v.ThumbnailUrl, "http") {
			continue
		}
		signedURL, err := stream.GetOSSSignedURL(ctx, v.ThumbnailUrl, thumbnailSignedURLTTL)
		if err != nil {
			utils.Logger.Warn("生成封面签名URL失败", zap.String("vid", v.Vid), zap.Error(err))
			v.ThumbnailUrl = ""
			continue
		}
		v.ThumbnailUrl = signedURL
	}
}

// ListUserAllVideos 获取用户视频列表
// @Summary      获取用户视频列表
// @Description  获取指定用户的所有视频
// @Tags         视频管理
// @Accept       json
// @Produce      json
// @Param        user_name  path      string  true  "用户名"
// @Success      200        {object}  api.VideoInfoDTO
// @Failure      401        {object}  utils.AppError
// @Failure      500        {object}  utils.AppError
// @Security     Bearer
// @Router       /user/{user_name}/videos [get]
func ListUserAllVideos(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userName := c.GetString("user_name")

	uid := c.GetString("user_id")
	uidInt, _ := strconv.Atoi(uid)
	utils.Logger.Info("获取用户视频列表",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.Int("user_id", uidInt))

	// 1. 先查缓存
	videoInfos, err := cache.GetUserVideos(c.Request.Context(), uidInt)
	if err == nil {
		utils.Logger.Debug("用户视频列表缓存命中",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.Int("user_id", uidInt))
		videoInfoDTO := &api.VideoInfoDTO{Videos: videoInfos}
		c.JSON(http.StatusOK, videoInfoDTO)
		return
	}

	// 2. 缓存未命中，查数据库
	videoInfos, err = dbops.GetUserAllVideos(uidInt)
	if err != nil {
		utils.Logger.Error("查询用户视频列表失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.Int("user_id", uidInt),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}
	//需要查数据库，也就需要再次签名生成URL
	enrichThumbnailURLs(c.Request.Context(), videoInfos)

	// 3. 写入缓存（存签名 URL，但必须保证签名时长大于缓存时长）
	if err := cache.SetUserVideos(c.Request.Context(), uidInt, videoInfos); err != nil {
		utils.Logger.Warn("写入用户视频列表缓存失败",
			zap.String("trace_id", traceID),
			zap.Int("user_id", uidInt),
			zap.Error(err))
	}

	videoInfoDTO := &api.VideoInfoDTO{Videos: videoInfos}
	c.JSON(http.StatusOK, videoInfoDTO)
}

// ListAllVideos 获取所有视频列表
// @Summary      获取所有视频
// @Description  获取系统中所有视频列表（热门接口）
// @Tags         视频管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.VideoInfoDTO
// @Failure      401  {object}  utils.AppError
// @Failure      500  {object}  utils.AppError
// @Security     Bearer
// @Router       /videos [get]
func ListAllVideos(c *gin.Context) {
	traceID := c.GetString("trace_id")
	// 1. 先查缓存（热门数据）
	videoInfos, err := cache.GetAllVideos(c.Request.Context())
	if err == nil {
		utils.Logger.Debug("全部视频列表缓存命中",
			zap.String("trace_id", traceID))
		videoInfoDTO := &api.VideoInfoDTO{Videos: videoInfos}
		c.JSON(http.StatusOK, videoInfoDTO)
		return
	}

	// 2. 缓存未命中，查数据库
	videoInfos, err = dbops.GetAllVideoInfo()
	if err != nil {
		utils.Logger.Error("查询所有视频失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}
	enrichThumbnailURLs(c.Request.Context(), videoInfos)

	// 3. 写入缓存（存签名 URL，但必须保证签名时长大于缓存时长）
	if err := cache.SetAllVideos(c.Request.Context(), videoInfos); err != nil {
		utils.Logger.Warn("写入全部视频列表缓存失败",
			zap.String("trace_id", traceID),
			zap.Error(err))
	}

	videoInfoDTO := &api.VideoInfoDTO{Videos: videoInfos}
	c.JSON(http.StatusOK, videoInfoDTO)
}

// DeleteVideoInfo 删除视频
// @Summary      删除视频
// @Description  软删除视频（异步删除OSS文件）
// @Tags         视频管理
// @Accept       json
// @Produce      json
// @Param        user_name  path      string  true  "用户名"
// @Param        vid        path      string  true  "视频ID"
// @Success      200        {object}  map[string]string
// @Failure      403        {object}  utils.AppError  "无权删除该视频"
// @Failure      404        {object}  utils.AppError  "视频不存在"
// @Failure      500        {object}  utils.AppError
// @Security     Bearer
// @Router       /user/{user_name}/videos/{vid} [delete]
func DeleteVideoInfo(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userName := c.GetString("user_name")

	uid := c.GetString("user_id")
	uidInt, _ := strconv.Atoi(uid)
	vid := c.Param("vid")

	// 检查视频是否存在
	existedVideo, err := dbops.GetVideoInfo(vid)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Logger.Warn("视频不存在",
				zap.String("trace_id", traceID),
				zap.String("user_name", userName),
				zap.String("video_id", vid))
			utils.AbortWithErrorMsg(c, utils.ErrAPIVideoNotExisted, "")
			return
		}
		utils.Logger.Error("查询视频失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}

	// 检查权限
	if existedVideo.AuthorId != uidInt {
		utils.Logger.Warn("无权删除该视频",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.Int("user_id", uidInt),
			zap.String("video_id", vid),
			zap.Int("video_author_id", existedVideo.AuthorId))
		utils.AbortWithErrorMsg(c, utils.ErrAPIVideoNotBelongToUser, "")
		return
	}

	// 软删除视频
	if err = dbops.DeleteVideoInfo(vid); err != nil {
		utils.Logger.Error("删除视频失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBDelete, err)
		return
	}

	// 删除视频的所有评论
	if err = dbops.DeleteVideoComments(vid); err != nil {
		utils.Logger.Error("删除视频评论失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		// 只记录日志，不阻塞删除流程
	}

	// 添加到删除记录（用于scheduler异步删除OSS文件）
	if err = dbops.InsertNewVideoDeletionRecord(vid); err != nil {
		utils.Logger.Error("添加删除记录失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}

	// 使相关缓存失效
	ctx := c.Request.Context()
	cache.InvalidateAllVideos(ctx)          // 全部视频列表
	cache.InvalidateUserVideos(ctx, uidInt) // 用户视频列表
	cache.DeleteVideo(ctx, vid)             // 视频详情
	cache.InvalidateComments(ctx, vid)      // 视频评论

	utils.Logger.Info("删除视频成功",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.String("video_id", vid))

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// PostComments 发表评论
// @Summary      发表评论
// @Description  在视频下发表评论
// @Tags         评论管理
// @Accept       json
// @Produce      json
// @Param        vid      path      string                true  "视频ID"
// @Param        comment  body      api.PostCommentsDTO   true  "评论内容"
// @Success      201      {object}  map[string]string
// @Failure      400      {object}  utils.AppError
// @Failure      429      {object}  utils.AppError  "评论过于频繁"
// @Failure      500      {object}  utils.AppError
// @Security     Bearer
// @Router       /videos/{vid}/comments [post]
func PostComments(c *gin.Context) {
	traceID := c.GetString("trace_id")
	userName := c.GetString("user_name")
	vid := c.Param("vid")
	user_id := c.GetString("user_id")
	author_id, _ := strconv.Atoi(user_id)
	userComment := &api.PostCommentsDTO{}

	if err := c.ShouldBindJSON(userComment); err != nil {
		utils.Logger.Error("解析请求体失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIInvalidRequest, err)
		return
	}

	if err := dbops.InsertNewComments(vid, author_id, userComment.Content); err != nil {
		utils.Logger.Error("添加评论失败",
			zap.String("trace_id", traceID),
			zap.String("user_name", userName),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBInsert, err)
		return
	}

	// 使评论列表缓存失效
	cache.InvalidateComments(c.Request.Context(), vid)

	utils.Logger.Info("添加评论成功",
		zap.String("trace_id", traceID),
		zap.String("user_name", userName),
		zap.String("video_id", vid))

	c.JSON(http.StatusCreated, gin.H{"message": "评论成功"})
}

// ListComments 获取评论列表
// @Summary      获取评论列表
// @Description  获取视频的所有评论
// @Tags         评论管理
// @Accept       json
// @Produce      json
// @Param        vid  path      string  true  "视频ID"
// @Success      200  {object}  api.CommentsDTO
// @Failure      401  {object}  utils.AppError
// @Failure      500  {object}  utils.AppError
// @Security     Bearer
// @Router       /videos/{vid}/comments [get]
func ListComments(c *gin.Context) {
	traceID := c.GetString("trace_id")
	vid := c.Param("vid")

	// 1. 先查缓存
	comments, err := cache.GetComments(c.Request.Context(), vid)
	if err == nil {
		utils.Logger.Debug("评论列表缓存命中",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid))
		commentsDTO := &api.CommentsDTO{Comments: comments}
		c.JSON(http.StatusOK, commentsDTO)
		return
	}

	// 2. 缓存未命中，查数据库
	comments, err = dbops.ListComments(vid, time.Unix(0, 0), time.Now())
	if err != nil {
		utils.Logger.Error("查询评论列表失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
		utils.AbortWithError(c, utils.ErrAPIDBQuery, err)
		return
	}

	// 3. 写入缓存
	if err := cache.SetComments(c.Request.Context(), vid, comments); err != nil {
		utils.Logger.Warn("写入评论列表缓存失败",
			zap.String("trace_id", traceID),
			zap.String("video_id", vid),
			zap.Error(err))
	}

	commentsDTO := &api.CommentsDTO{Comments: comments}
	c.JSON(http.StatusOK, commentsDTO)
}
