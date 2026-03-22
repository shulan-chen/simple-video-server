package session

// ⚠️ 此文件已废弃
// 新的session管理使用 JWT + Refresh Token 方式
// 所有Redis缓存操作已迁移到 api/cache/ 目录
// 保留此文件仅为了兼容性，未来版本将删除

import (
	"context"
	"strconv"
	"sync"
	"time"
	"video-server/api/cache"
	api "video-server/api/defs"
	"video-server/api/utils"

	"go.uber.org/zap"
)

var ttl = time.Duration(30 * time.Minute)
var sessionMap *sync.Map
var sessionKey = "user_sessions"
var ctx = context.Background()

func init() {
	sessionMap = &sync.Map{}
	// 不再主动加载sessions，因为已经使用JWT方式
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
	expireStr := strconv.FormatInt(expire, 10)
	session := api.SimpleSession{SessionId: sid, UserId: userId, Username: userName, TTL: expireStr}
	sessionMap.Store(sid, session)

	err := cache.AddSession(ctx, sid, &session)
	if err != nil {
		utils.Logger.Error("添加Session到Redis失败", zap.Error(err))
		return session, err
	}

	// 更新session列表（已不再需要，保留兼容）
	return session, nil
}

func LoadSessions() {
	// 已废弃，不再需要加载sessions
	// 使用JWT方式，无需预加载
}

func IsSessionExpired(sid string) (userName string, ok bool) {
	session, ok := sessionMap.Load(sid)
	if !ok {
		// 本地cache没有，查Redis
		existSession, err := cache.GetSession(ctx, sid)
		if err != nil || existSession.SessionId == "" {
			return "", true
		}
		sessionMap.Store(existSession.SessionId, existSession)
		return existSession.Username, false
	}

	s := session.(api.SimpleSession)
	ttlInt64, err := strconv.ParseInt(s.TTL, 10, 64)
	if err != nil {
		return s.Username, true
	}

	if ttlInt64 < time.Now().Unix() {
		DeleteSession(sid)
		return s.Username, true
	}

	return s.Username, false
}

func DeleteSession(sid string) {
	sessionMap.Delete(sid)
	cache.DeleteSession(ctx, sid)
}
