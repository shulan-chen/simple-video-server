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

func AddNewSession(userId int, userName string) api.SimpleSession {
	sid, _ := utils.NewUUID()
	expire := time.Now().Add(ttl).Unix()
	err := dbops.InsertNewSession(sid, userId, userName, expire)
	if err != nil {
		panic(err)
	}
	session := api.SimpleSession{SessionId: sid, UserId: userId, Username: userName, TTL: expire}
	sessionMap.Store(sid, session)
	return session
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
