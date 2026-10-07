package inventory

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) CreateDeviceCredential(ctx context.Context, deviceID uuid.UUID, tokenHash string) error {
	const q = `
		INSERT INTO device_credentials (device_id, token_hash)
		VALUES ($1, $2)
	`

	_, err := s.pool.Exec(ctx, q, deviceID, tokenHash)
	return err
}