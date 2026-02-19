package dbops

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
	api "video-server/api/defs"
	"video-server/api/utils"
)

func AddUser(userName string, pwd string) (user *api.User, err error) {
	// 创建对象
	user = &api.User{
		Username: userName,
		Password: pwd,
		// CreatAt:  time.Now(), // 如果设置了 autoCreateTime tag，这里可以省略
		// IsVaild: 0, // 默认为0的话不用写
	}

	// Create 插入数据
	// GORM 会自动将生成的 ID 填充回 user.Id
	err = Db.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func GetUserByName(userName string) (user *api.User, err error) {
	// 注意：这里返回单个 User，不是 slice
	user = new(api.User)
	err = Db.Where("name = ?", userName).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func DeleteUser(id int, userName string) error {
	if id == 0 && userName == "" {
		return fmt.Errorf("empty userName,can not delete")
	}
	pUser, err := GetUserByName(userName)
	if pUser == nil || err == sql.ErrNoRows {
		return fmt.Errorf("user not exist")
	}
	// Unscoped() 表示物理删除。如果不加，且模型有 DeletedAt 字段，则会软删除
	// 这里的写法等同于 DELETE FROM users WHERE id=? AND name=?
	result := Db.Where("id = ? AND name = ?", id, userName).Delete(&api.User{})
	return result.Error
}

// video info related db ops
func AddNewVideo(aid int, name string) (*api.VideoInfo, error) {
	vid, err := utils.NewUUID()
	if err != nil {
		panic("error to make a new uuid")
	}

	video := &api.VideoInfo{
		Vid:      vid,
		AuthorId: aid,
		Name:     name,
		// CreateTime, ClickCount 会由 Tag 或数据库默认值处理
	}

	if err := Db.Create(video).Error; err != nil {
		return nil, err
	}
	return video, nil
}

func GetVideoInfo(vid string) (video_info *api.VideoInfo, err error) {
	// 注意：这里返回单个 VideoInfo，不是 slice
	videoInfo := &api.VideoInfo{}
	err = Db.Where("vid = ?", vid).First(videoInfo).Error
	if err != nil {
		return nil, err
	}
	return videoInfo, nil
}

func GetUserAllVideos(id int) ([]*api.VideoInfo, error) {
	var videos []*api.VideoInfo
	err := Db.Where("author_id = ?", id).Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func GetAllVideoInfo() ([]*api.VideoInfo, error) {
	var videos []*api.VideoInfo
	// Find 查询多条，Order 排序
	err := Db.Order("create_time DESC").Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func DeleteVideoInfo(vid string) error {
	result := Db.Where("vid = ?", vid).Delete(&api.VideoInfo{})
	return result.Error
}

// comment related db ops
func InsertNewComments(vid string, aid int, content string) error {
	comment_id, err := utils.NewUUID()
	comment := &api.Comment{
		CommentId: comment_id,
		VideoId:   vid,
		AuthorId:  aid,
		Content:   content,
		// CreateTime 会由 Tag 或数据库默认值处理
	}
	if err = Db.Create(comment).Error; err != nil {
		return err
	}
	return nil
}

func ListComments(vid string, from, to time.Time) ([]*api.CommentDTO, error) {
	var comments []*api.CommentDTO
	// GORM 的 Raw SQL 查询映射到非 Model 结构体 (DTO)
	// 这种场景通常用 Raw() + Scan()
	err := Db.Raw(`
        SELECT users.name as author_name, comments.comment_id, comments.content, comments.create_time
        FROM comments 
        INNER JOIN users ON comments.author_id = users.id
        WHERE comments.video_id = ? AND comments.create_time BETWEEN ? AND ?`,
		vid, from, to).Scan(&comments).Error

	if err != nil {
		return nil, err
	}
	return comments, nil
}

// session related db ops
func InsertNewSession(sid string, userId int, userName string, ttl int64) error {
	ttlString := strconv.FormatInt(ttl, 10)
	newSession := &api.SimpleSession{
		SessionId: sid,
		UserId:    userId,
		Username:  userName,
		TTL:       ttlString,
	}
	if err := Db.Create(newSession).Error; err != nil {
		return err
	}
	return nil
}

func LoadSessionsFromDB() ([]api.SimpleSession, error) {
	var result []api.SimpleSession
	err := Db.Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func LoadOneSessionFromDB(sid string) (*api.SimpleSession, error) {
	var s api.SimpleSession
	err := Db.Where("session_id = ?", sid).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteSessionFromDB(sid string) error {
	result := Db.Where("session_id = ?", sid).Delete(&api.SimpleSession{})
	return result.Error
}

// shcheduler related db ops
func InsertNewVideoDeletionRecord(vid string) error {
	record := &api.VideoDeletionRecord{
		Vid: vid,
	}
	if err := Db.Create(record).Error; err != nil {
		return err
	}
	return nil
}

func ReadVideoDeletionRecord(count int) ([]string, error) {
	var vids []string
	err := Db.Model(&api.VideoDeletionRecord{}).Limit(count).Pluck("vid", &vids).Error
	if err != nil {
		return nil, err
	}
	return vids, nil
}

func DeleteVideoDeletionRecord(vid string) error {
	result := Db.Where("vid = ?", vid).Delete(&api.VideoDeletionRecord{})
	return result.Error
}
