package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	api "video-server/api/defs"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// InitRedis 初始化Redis客户端
func InitRedis(addr, password string, db int) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	return err
}

// 缓存键前缀（数据缓存）
const (
	prefixUser       = "data:user:"         // 用户信息（使用userId）
	prefixVideo      = "data:video:"        // 视频信息
	prefixVideoList  = "data:videos:all:"   // 全部视频列表
	prefixUserVideos = "data:videos:user:"  // 用户视频列表
	prefixComments   = "data:comments:"     // 评论列表
)

// 缓存过期时间
const (
	userCacheTTL      = 1 * time.Hour     // 用户信息：1小时
	videoCacheTTL     = 30 * time.Minute  // 视频信息：30分钟
	videoListCacheTTL = 10 * time.Minute  // 视频列表：10分钟（热门数据）
	commentsCacheTTL  = 5 * time.Minute   // 评论列表：5分钟
)

// GetUserById 根据用户ID获取用户缓存
func GetUserById(ctx context.Context, userId int) (*api.User, error) {
	key := fmt.Sprintf("%s%d", prefixUser, userId)
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var user api.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// SetUser 设置用户缓存（使用userId作为key）
func SetUser(ctx context.Context, user *api.User) error {
	key := fmt.Sprintf("%s%d", prefixUser, user.Id)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, userCacheTTL).Err()
}

// DeleteUserById 根据用户ID删除用户缓存
func DeleteUserById(ctx context.Context, userId int) error {
	key := fmt.Sprintf("%s%d", prefixUser, userId)
	return redisClient.Del(ctx, key).Err()
}

// GetVideo 获取视频缓存
func GetVideo(ctx context.Context, vid string) (*api.VideoInfo, error) {
	key := prefixVideo + vid
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var video api.VideoInfo
	if err := json.Unmarshal([]byte(data), &video); err != nil {
		return nil, err
	}
	return &video, nil
}

// SetVideo 设置视频缓存
func SetVideo(ctx context.Context, video *api.VideoInfo) error {
	key := prefixVideo + video.Vid
	data, err := json.Marshal(video)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, videoCacheTTL).Err()
}

// DeleteVideo 删除视频缓存
func DeleteVideo(ctx context.Context, vid string) error {
	key := prefixVideo + vid
	return redisClient.Del(ctx, key).Err()
}

// GetAllVideos 获取所有视频列表缓存
func GetAllVideos(ctx context.Context) ([]*api.VideoInfo, error) {
	key := prefixVideoList + "latest"
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var videos []*api.VideoInfo
	if err := json.Unmarshal([]byte(data), &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// SetAllVideos 设置所有视频列表缓存
func SetAllVideos(ctx context.Context, videos []*api.VideoInfo) error {
	key := prefixVideoList + "latest"
	data, err := json.Marshal(videos)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, videoListCacheTTL).Err()
}

// InvalidateAllVideos 使全部视频列表缓存失效
func InvalidateAllVideos(ctx context.Context) error {
	key := prefixVideoList + "latest"
	return redisClient.Del(ctx, key).Err()
}

// GetUserVideos 获取用户视频列表缓存
func GetUserVideos(ctx context.Context, userId int) ([]*api.VideoInfo, error) {
	key := fmt.Sprintf("%s%d", prefixUserVideos, userId)
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var videos []*api.VideoInfo
	if err := json.Unmarshal([]byte(data), &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// SetUserVideos 设置用户视频列表缓存
func SetUserVideos(ctx context.Context, userId int, videos []*api.VideoInfo) error {
	key := fmt.Sprintf("%s%d", prefixUserVideos, userId)
	data, err := json.Marshal(videos)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, videoListCacheTTL).Err()
}

// InvalidateUserVideos 使用户视频列表缓存失效
func InvalidateUserVideos(ctx context.Context, userId int) error {
	key := fmt.Sprintf("%s%d", prefixUserVideos, userId)
	return redisClient.Del(ctx, key).Err()
}

// GetComments 获取评论列表缓存
func GetComments(ctx context.Context, vid string) ([]*api.CommentDTO, error) {
	key := prefixComments + vid
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var comments []*api.CommentDTO
	if err := json.Unmarshal([]byte(data), &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// SetComments 设置评论列表缓存
func SetComments(ctx context.Context, vid string, comments []*api.CommentDTO) error {
	key := prefixComments + vid
	data, err := json.Marshal(comments)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, data, commentsCacheTTL).Err()
}

// InvalidateComments 使评论列表缓存失效
func InvalidateComments(ctx context.Context, vid string) error {
	key := prefixComments + vid
	return redisClient.Del(ctx, key).Err()
}
