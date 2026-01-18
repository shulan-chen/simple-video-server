package dbops

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
	api "video-server/api/defs"
	"video-server/api/utils"

	"go.uber.org/zap"
)

func AddUser(userName string, pwd string) (user *api.User, err error) {
	ctime := time.Now()
	insert, err := Db.Prepare("insert into users (name,password,creatAt) values(?,?,?)")
	if err != nil {
		return nil, err
	}
	defer insert.Close()
	_, err = insert.Exec(userName, pwd, ctime)
	if err != nil {
		return nil, err
	}
	user, err = GetUserByName(userName)
	return user, err
}

func GetUserByName(userName string) (user *api.User, err error) {
	// 注意：这里返回单个 User，不是 slice
	user = new(api.User)
	err = Db.QueryRow("SELECT * FROM users WHERE name = ?", userName).
		Scan(&user.Id, &user.Username, &user.Password, &user.IsVaild, &user.CreatAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		utils.Logger.Error("Query error:", zap.Error(err))
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
	deleteStatment, err := Db.Prepare("delete from users where id = ? and name = ?")
	if err != nil {
		panic(err)
	}
	defer deleteStatment.Close()

	_, err = deleteStatment.Exec(pUser.Id, userName)
	if err != nil {
		return err
	}
	return nil
}

// video info related db ops
func AddNewVideo(aid int, name string) (*api.VideoInfo, error) {
	vid, err := utils.NewUUID()
	if err != nil {
		panic("error to make a new uuid")
	}

	ctime := time.Now()
	insert, err := Db.Prepare("insert into video_info (vid,author_id,name,create_time) values(?,?,?,?)")
	if err != nil {
		return nil, err
	}
	defer insert.Close()
	_, err = insert.Exec(vid, aid, name, ctime)
	if err != nil {
		return nil, err
	}

	return &api.VideoInfo{Vid: vid, AuthorId: aid, Name: name, CreateTime: ctime}, nil
}

func GetVideoInfo(vid string) (*api.VideoInfo, error) {
	// 注意：这里返回单个 VideoInfo，不是 slice
	videoInfo := new(api.VideoInfo)
	stmtQuery, err := Db.Prepare("SELECT * FROM video_info WHERE vid = ?")
	defer stmtQuery.Close()
	if err != nil {
		return nil, err
	}
	err = stmtQuery.QueryRow(vid).Scan(&videoInfo.Vid, &videoInfo.AuthorId, &videoInfo.Name, &videoInfo.CreateTime, &videoInfo.ClickCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
	}
	return videoInfo, nil
}

func GetUserAllVideos(id int) ([]*api.VideoInfo, error) {
	stmtOut, err := Db.Prepare("SELECT * FROM video_info WHERE author_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmtOut.Close()
	rows, err := stmtOut.Query(id)
	if err != nil {
		return nil, err
	}
	var res []*api.VideoInfo
	for rows.Next() {
		videoInfo := new(api.VideoInfo)
		err := rows.Scan(&videoInfo.Vid, &videoInfo.AuthorId, &videoInfo.Name, &videoInfo.CreateTime, &videoInfo.ClickCount)
		if err != nil {
			return nil, err
		}
		res = append(res, videoInfo)
	}
	return res, nil
}

func DeleteVideoInfo(vid string) error {
	stmtDel, err := Db.Prepare("delete from video_info where vid = ?")
	if err != nil {
		return err
	}
	defer stmtDel.Close()
	_, err = stmtDel.Exec(vid)
	if err != nil {
		return err
	}
	return nil
}

// comment related db ops
func InsertNewComments(vid string, aid int, content string) error {
	stmtIns, err := Db.Prepare("insert into comments (comment_id,video_id,author_id,content,create_time) values(?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmtIns.Close()
	comment_id, err := utils.NewUUID()
	ctime := time.Now()
	_, err = stmtIns.Exec(comment_id, vid, aid, content, ctime)
	if err != nil {
		return err
	}
	return nil
}

func ListComments(vid string, from, to time.Time) ([]*api.CommentDTO, error) {
	stmtOut, err := Db.Prepare(`select users.name,comments.comment_id,comments.content,comments.create_time
	from comments inner join users on comments.author_id = users.id
	where video_id = ? and create_time >= ? and create_time <= ?`)
	if err != nil {
		return nil, err
	}
	defer stmtOut.Close()
	rows, err := stmtOut.Query(vid, from, to)
	if err != nil {
		return nil, err
	}
	var res []*api.CommentDTO
	for rows.Next() {
		commentDTO := new(api.CommentDTO)
		err := rows.Scan(&commentDTO.AuthorName, &commentDTO.CommentId, &commentDTO.Content, &commentDTO.CreateTime)
		if err != nil {
			return nil, err
		}
		res = append(res, commentDTO)
	}
	return res, nil
}

// session related db ops
func InsertNewSession(sid string, userId int, userName string, ttl int64) error {
	ttlString := strconv.FormatInt(ttl, 10)
	insert, err := Db.Prepare("insert into sessions (session_id,user_id, user_name, ttl) values(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer insert.Close()
	_, err = insert.Exec(sid, userId, userName, ttlString)
	return err
}

func LoadSessionsFromDB() ([]api.SimpleSession, error) {
	stmtOut, err := Db.Prepare("select * from sessions")
	if err != nil {
		panic(err)
	}
	defer stmtOut.Close()
	rows, err := stmtOut.Query()
	if err != nil {
		panic(err)
	}
	var result []api.SimpleSession
	for rows.Next() {
		var s api.SimpleSession
		ttl := ""
		var id int64
		err := rows.Scan(&id, &s.SessionId, &s.UserId, &s.Username, &ttl)
		if err != nil {
			panic(err)
		}
		s.TTL, _ = strconv.ParseInt(ttl, 10, 64)
		result = append(result, s)
	}
	return result, nil
}

func LoadOneSessionFromDB(sid string) (*api.SimpleSession, error) {
	stmtOut, err := Db.Prepare("select * from sessions where session_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmtOut.Close()
	var s api.SimpleSession
	ttl := ""
	err = stmtOut.QueryRow(sid).Scan(&s.SessionId, &s.UserId, &s.Username, &ttl)
	if err != nil {
		return nil, err
	}
	s.TTL, _ = strconv.ParseInt(ttl, 10, 64)
	return &s, nil
}

func DeleteSessionFromDB(sid string) error {
	stmtDel, err := Db.Prepare("delete from sessions where session_id = ?")
	if err != nil {
		return err
	}
	defer stmtDel.Close()
	_, err = stmtDel.Exec(sid)
	if err != nil {
		return err
	}
	return nil
}

// shcheduler related db ops
func InsertNewVideoDeletionRecord(vid string) error {
	insert, err := Db.Prepare("insert into video_delete_record (vid) values(?)")
	if err != nil {
		return err
	}
	defer insert.Close()
	_, err = insert.Exec(vid)
	if err != nil {
		utils.Logger.Error("InsertNewVideoDeletionRecord failed", zap.Error(err))
		return err
	}
	return nil
}

func ReadVideoDeletionRecord(count int) ([]string, error) {
	stmtOut, err := Db.Prepare("select vid from video_delete_record limit ?")
	if err != nil {
		return nil, err
	}
	defer stmtOut.Close()
	rows, err := stmtOut.Query(count)
	if err != nil {
		utils.Logger.Error("ReadVideoDeletionRecord failed", zap.Error(err))
		return nil, err
	}
	var vids []string
	for rows.Next() {
		var vid string
		var id int64
		err := rows.Scan(&id, &vid)
		if err != nil {
			return nil, err
		}
		vids = append(vids, vid)
	}
	return vids, nil
}

func DeleteVideoDeletionRecord(vid string) error {
	stmtDel, err := Db.Prepare("delete from video_delete_record where vid = ?")
	if err != nil {
		return err
	}
	defer stmtDel.Close()
	_, err = stmtDel.Exec(vid)
	if err != nil {
		utils.Logger.Error("DeleteVideoDeletionRecord failed", zap.Error(err))
		return err
	}
	return nil
}
