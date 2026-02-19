package api

import "time"

type User struct {
	Id        int       `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Username  string    `json:"name" gorm:"column:name;unique"`
	Password  string    `json:"password" gorm:"column:password"`
	IsVaild   int       `json:"isVaild" gorm:"column:isVaild"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName 指定 User 结构体对应的表名为 "users"
func (User) TableName() string {
	return "users"
}

type UserDTO struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}

type SignedUP struct {
	Success   bool   `json:"success"`
	SessionId string `json:"session_id"`
}

type VideoInfo struct {
	Id          int       `gorm:"primaryKey;autoIncrement;column:id"`
	Vid         string    `json:"id" gorm:"column:vid"`
	AuthorId    int       `json:"author_id" gorm:"column:author_id"`
	Name        string    `json:"name" gorm:"column:name"`
	CreatedTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	ClickCount  int       `json:"click_count" gorm:"column:click_count"`
}

func (VideoInfo) TableName() string {
	return "video_info"
}

type VideoInfoDTO struct {
	Videos []*VideoInfo `json:"videos"`
}

type VideoDeletionRecord struct {
	Id  int    `gorm:"primaryKey;autoIncrement;column:id"`
	Vid string `json:"vid" gorm:"column:vid"`
}

func (VideoDeletionRecord) TableName() string {
	return "video_delete_record"
}

// comment related db ops
type Comment struct {
	Id         int       `gorm:"primaryKey;autoIncrement;column:id"`
	CommentId  string    `json:"commentId" gorm:"column:comment_id"`
	VideoId    string    `json:"video_id" gorm:"column:video_id"`
	AuthorId   int       `json:"author_id" gorm:"column:author_id"`
	Content    string    `json:"content" gorm:"column:content"`
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
}

func (Comment) TableName() string {
	return "comments"
}

type CommentDTO struct {
	AuthorName string    `json:"authorName"`
	CommentId  string    `json:"comment_id"`
	Content    string    `json:"content"`
	CreateTime time.Time `json:"create_time"`
}

type CommentsDTO struct {
	Comments []*CommentDTO `json:"comments"`
}

type SimpleSession struct {
	Id        int    `gorm:"primaryKey;autoIncrement;column:id"`
	SessionId string `json:"session_id" gorm:"column:session_id"`
	UserId    int    `json:"user_id" gorm:"column:user_id"`
	Username  string `json:"username" gorm:"column:username"`
	TTL       string `json:"ttl" gorm:"column:ttl"`
}

func (SimpleSession) TableName() string {
	return "sessions"
}

type UserAddNewVideoDTO struct {
	AuthorId int    `json:"author_id"`
	Name     string `json:"name"`
}

type PostCommentsDTO struct {
	//VideoId  string `json:"video_id"`
	AuthorId int    `json:"author_id"`
	Content  string `json:"content"`
}
