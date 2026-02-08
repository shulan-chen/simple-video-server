package session

import (
	"sync"
	"time"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/utils"
)

var ttl = time.Duration(30 * time.Minute)
var sessionMap *sync.Map

func init() {
	sessionMap = &sync.Map{}
	LoadSessions()
}

func AddNewSession(userId int, userName string) (api.SimpleSession, error) {
	sid, _ := utils.NewUUID()
	expire := time.Now().Add(ttl).Unix()
	session := api.SimpleSession{SessionId: sid, UserId: userId, Username: userName, TTL: expire}
	err := AddSessionToRedis(sid, session)
	if err != nil {
		return session, err
	}
	sessionMap.Store(sid, session)
	return session, nil
}

func LoadSessions() {
	sessions, err := dbops.LoadSessionsFromDB()
	if err != nil {
		panic(err)
	}
	for _, s := range sessions {
		sessionMap.Store(s.SessionId, s)
	}
}

func IsSessionExpired(sid string) (userName string, ok bool) {
	session, ok := sessionMap.Load(sid)
	if !ok {
		//本地cache没有还要去db搂，防止分布式不一致的情况
		existSession, err := dbops.LoadOneSessionFromDB(sid)
		if err != nil || existSession.SessionId == "" {
			return "", true
		}
		sessionMap.Store(existSession.SessionId, existSession)
		return existSession.Username, false
	}
	s := session.(api.SimpleSession)
	if s.TTL < time.Now().Unix() {
		DeleteSession(sid)
		return s.Username, true
	}
	return s.Username, false
}

func DeleteSession(sid string) {
	sessionMap.Delete(sid)
	dbops.DeleteSessionFromDB(sid)
}
