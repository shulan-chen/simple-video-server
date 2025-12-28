package api

import "time"

type User struct {
	Id       int       `json:"id"`
	Username string    `json:"name"`
	Password string    `json:"password"`
	IsVaild  int       `json:"isVaild"`
	CreatAt  time.Time `json:createAt`
	//UpdateAt int64  `json:updateAt`
}

type SignedUP struct {
	Success   bool   `json:"success"`
	SessionId string `json:"session_id"`
}

type VideoInfo struct {
	Vid        string    `json:"id"`
	AuthorId   int       `json:"author_id"`
	Name       string    `json:"name"`
	CreateTime time.Time `json:"create_time"`
	ClickCount int       `json:"click_count"`
}

type Comment struct {
	CommentId  int       `json:"commentId"`
	VideoId    string    `json:"video_id"`
	AuthorId   int       `json:"author_id"`
	Content    string    `json:"content"`
	CreateTime time.Time `json:"create_time"`
}

type CommentDTO struct {
	AuthorName string    `json:"authorName"`
	CommentId  string    `json:"comment_id"`
	Content    string    `json:"content"`
	CreateTime time.Time `json:"create_time"`
}

type SimpleSession struct {
	SessionId string `json:"session_id"`
	UserId    int    `json:"user_id"`
	Username  string `json:"username"`
	TTL       int64  `json:"ttl"`
}
