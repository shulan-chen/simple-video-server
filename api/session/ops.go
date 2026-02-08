package session

import (
	"sync"
	"time"
	api "video-server/api/defs"
	"video-server/api/utils"

	"go.uber.org/zap"
)

var ttl = time.Duration(30 * time.Minute)
var sessionMap *sync.Map
var sessionKey = "user_sessions"

func init() {
	initRedis()
	sessionMap = &sync.Map{}
	LoadSessions()
}

func getAllKeys(m *sync.Map) []string {
	var keys []string
	m.Range(func(key, value interface{}) bool {
		// 假设 key 是 string 类型
		keys = append(keys, key.(string))
		return true // 继续遍历
	})
	return keys
}
func AddNewSession(userId int, userName string) (api.SimpleSession, error) {
	sid, _ := utils.NewUUID()
	expire := time.Now().Add(ttl).Unix()
	session := api.SimpleSession{SessionId: sid, UserId: userId, Username: userName, TTL: expire}
	sessionMap.Store(sid, session)
	err := AddSessionToRedis(sid, session)
	if err != nil {
		utils.Logger.Error("AddSessionToRedis failed", zap.Error(err))
		return session, err
	}

	UpdateSessions(sessionKey, getAllKeys(sessionMap))
	return session, nil
}

func LoadSessions() {
	sessions, err := LoadSessionsFromRedis(sessionKey)
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
		//本地cache没有还要去redis搂，防止分布式不一致的情况
		existSession, err := GetSessionFromRedis(sid)
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
	DeleteSessionFromRedis(sid)
	UpdateSessions(sessionKey, getAllKeys(sessionMap))
}
