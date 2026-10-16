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
		SELECT c.device_id, d.status
		FROM device_credentials c
		JOIN devices d on d.id = c.device_id
		WHERE c.token_hash = $1
			AND c.revoked_at IS NULL
		LIMIT 1
	`

	var id uuid.UUID
	var status string
	err := s.pool.QueryRow(ctx, q, tokenHash).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	if status == "disabled" {
		return uuid.Nil, ErrDeviceDisabled
	}
	return id, nil
}

func (s *Store) rotateDeviceCredential(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
	newTokenHash string,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE device_credentials
		SET revoked_at = now()
		WHERE device_id = $1
		AND revoked_at IS NULL
	`, deviceID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO device_credentials (device_id, token_hash)
		VALUES ($1, $2)
	`, deviceID, newTokenHash)
	return err
}
