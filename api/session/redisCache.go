package session

import (
	"context"
	"encoding/json"
	api "video-server/api/defs"

	"video-server/config"

	"github.com/go-redis/redis/v8"
)

var rconn *redis.Client
var ctx context.Context

func initRedis() {
	ctx = context.Background()
	rconn = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPwd,
		DB:       config.AppConfig.RedisDB,
	})
}

func UpdateSessions(sessionKey string, sessionIds []string) error {
	data, err := json.Marshal(sessionIds)
	err = rconn.Set(ctx, sessionKey, data, ttl).Err()
	return err
}
func LoadSessionsFromRedis(sessionKey string) ([]api.SimpleSession, error) {
	var sessionIds []string
	data, err := rconn.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []api.SimpleSession{}, nil
		}
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &sessionIds)
	var sessions []api.SimpleSession
	for _, sid := range sessionIds {
		session, err := GetSessionFromRedis(sid)
		if err != nil {
			continue
		}
		sessions = append(sessions, *session)
	}
	return sessions, nil
}

func AddSessionToRedis(sid string, session api.SimpleSession) error {
	data, err := json.Marshal(session)
	err = rconn.Set(ctx, sid, data, ttl).Err()
	return err
}

func GetSessionFromRedis(sid string) (*api.SimpleSession, error) {
	session := &api.SimpleSession{}
	data, err := rconn.Get(ctx, sid).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, session)
	return session, err
}

func DeleteSessionFromRedis(sid string) error {
	err := rconn.Del(ctx, sid).Err()
	return err
}
