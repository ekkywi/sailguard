package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) FindByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT id, email, name, password_hash, auth_provider, is_active, last_login_at
		FROM users
		WHERE email = $1
	`

	var u User
	err := s.pool.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.IsActive,
		&u.LastLoginAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
