package inventory

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateDeviceCredential(ctx context.Context, deviceID uuid.UUID, tokenHash string) error {
	const q = `
		INSERT INTO device_credentials (device_id, token_hash)
		VALUES ($1, $2)
	`

	_, err := s.pool.Exec(ctx, q, deviceID, tokenHash)
	return err
}

func (s *Store) FindDeviceIDByActiveTokenHash(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	const q = `
		SELECT device_id
		FROM device_credentials
		WHERE token_hash = $1
			AND revoked_at IS NULL
		LIMIT 1
	`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, q, tokenHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}