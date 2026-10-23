package authsession

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func sessionKey(jti string) string { return "sg:admin:session:" + jti }
func userSessionsKey(userID uuid.UUID) string {
	return "sg:admin:user:" + userID.String() + ":sessions"
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return fmt.Errorf("invalid session")
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, sessionKey(jti), userID.String(), ttl)
	pipe.SAdd(ctx, userSessionsKey(userID), jti)
	pipe.Expire(ctx, userSessionsKey(userID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Exists(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, sessionKey(jti)).Result()
	return n > 0, err
}

func (s *Store) Revoke(ctx context.Context, userID uuid.UUID, jti string) error {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, sessionKey(jti))
	pipe.SRem(ctx, userSessionsKey(userID), jti)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	key := userSessionsKey(userID)
	jtis, err := s.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	for _, jti := range jtis {
		pipe.Del(ctx, sessionKey(jti))
	}
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	return err
}