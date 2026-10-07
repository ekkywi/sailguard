package inventory

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListEnrollmentTokens(ctx context.Context) ([]EnrollmentToken, error) {
	const q = `
		SELECT id, label, token_hash, max_uses, use_count,
		       expires_at, revoked_at, created_by, created_at
		FROM enrollment_tokens
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EnrollmentToken, 0)
	for rows.Next() {
		var t EnrollmentToken
		if err := rows.Scan(
			&t.ID, &t.Label, &t.TokenHash, &t.MaxUses, &t.UseCount,
			&t.ExpiresAt, &t.RevokedAt, &t.CreatedBy, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateEnrollmentToken(
	ctx context.Context,
	label string,
	tokenHash string,
	maxUses int,
	expiresAt *time.Time,
	createdBy *uuid.UUID,
) (*EnrollmentToken, error) {
	const q = `
		INSERT INTO enrollment_tokens (label, token_hash, max_uses, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, label, token_hash, max_uses, use_count,
			expires_at, revoked_at, created_by, created_at
	`

	var t EnrollmentToken
	err := s.pool.QueryRow(ctx, q, label, tokenHash, maxUses, expiresAt, createdBy).Scan(
		&t.ID, &t.Label, &t.TokenHash, &t.MaxUses, &t.UseCount,
		&t.ExpiresAt, &t.RevokedAt, &t.CreatedBy, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) FindEnrollmentTokenByID(ctx context.Context, id uuid.UUID) (*EnrollmentToken, error) {
	const q = `
		SELECT id, label, token_hash, max_uses, use_count,
			expires_at, revoked_at, created_by, created_at
		FROM enrollment_tokens
		WHERE id = $1
	`

	var t EnrollmentToken
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.Label, &t.TokenHash, &t.MaxUses, &t.UseCount,
		&t.ExpiresAt, &t.RevokedAt, &t.CreatedBy, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) RevokeEnrollmentToken(ctx context.Context, id uuid.UUID) (*EnrollmentToken, error) {
	const q = `
		UPDATE enrollment_tokens
		SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, label, token_hash, max_uses, use_count,
			expires_at, revoked_at, created_by, created_at
	`

	var t EnrollmentToken
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.Label, &t.TokenHash, &t.MaxUses, &t.UseCount,
		&t.ExpiresAt, &t.RevokedAt, &t.CreatedBy, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}