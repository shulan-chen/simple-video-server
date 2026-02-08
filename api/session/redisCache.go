package session

import (
	"context"
	api "video-server/api/defs"

	"github.com/go-redis/redis/v8"
)

var rconn *redis.Client
var ctx context.Context

func init() {
	ctx = context.Background()
	rconn = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})
}

func AddSessionToRedis(sid string, session api.SimpleSession) error {
	err := rconn.Set(ctx, sid, session, ttl).Err()
	return err
}

func GetSessionFromRedis(sid string) (*api.SimpleSession, error) {
	var session *api.SimpleSession
	err := rconn.Get(ctx, sid).Scan(session)
	return session, err
}

func DeleteSessionFromRedis(sid string) error {
	err := rconn.Del(ctx, sid).Err()
	return err
}
