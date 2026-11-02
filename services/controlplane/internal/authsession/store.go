package authsession

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

type SessionMeta struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type Session struct {
	ID     string
	Meta   SessionMeta
	Legacy bool
}

func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func sessionKey(jti string) string { return "sg:admin:session:" + jti }
func userSessionsKey(userID uuid.UUID) string {
	return "sg:admin:user:" + userID.String() + ":sessions"
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, jti string, ttl time.Duration, meta SessionMeta) error {
	if jti == "" || ttl <= 0 {
		return fmt.Errorf("invalid session")
	}
	meta.UserID = userID.String()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, sessionKey(jti), raw, ttl)
	pipe.SAdd(ctx, userSessionsKey(userID), jti)
	pipe.Expire(ctx, userSessionsKey(userID), ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) Exists(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, sessionKey(jti)).Result()
	return n > 0, err
}

func (s *Store) Get(ctx context.Context, jti string) (*Session, error) {
	if jti == "" {
		return nil, fmt.Errorf("empty jti")
	}
	raw, err := s.rdb.Get(ctx, sessionKey(jti)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var meta SessionMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return &Session{
			ID: jti,
			Meta: SessionMeta{
				UserID: strings.Trim(string(raw), `"`),
			},
			Legacy: true,
		}, nil
	}
	return &Session{
		ID:   jti,
		Meta: meta,
	}, nil
}

func (s *Store) List(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	jtis, err := s.rdb.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(jtis))
	for _, jti := range jtis {
		sess, err := s.Get(ctx, jti)
		if err != nil {
			return nil, err
		}
		if sess == nil {
			_ = s.rdb.SRem(ctx, userSessionsKey(userID), jti).Err()
			continue
		}
		out = append(out, *sess)
	}
	return out, nil
}

func (s *Store) RevokeOthers(ctx context.Context, userID uuid.UUID, keepJTI string) error {
	jtis, err := s.rdb.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	for _, jti := range jtis {
		if jti == keepJTI {
			continue
		}
		pipe.Del(ctx, sessionKey(jti))
		pipe.SRem(ctx, userSessionsKey(userID), jti)
	}
	_, err = pipe.Exec(ctx)
	return err
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
