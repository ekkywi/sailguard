package networkpolicy

import (
	"context"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

func (s *Store) ListOverridesByDeviceID(
	ctx context.Context,
	deviceID uuid.UUID,
	includeExpired bool,
) ([]DeviceOverride, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return nil, err
	}

	q := `
		SELECT id, device_id, action, dest_type, dest_value, port, label, reason, expires_at, created_at, updated_at
		FROM device_network_overrides
		WHERE device_id = $1
	`
	if !includeExpired {
		q += ` AND (expires_at IS NULL OR expires_at > now())`
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceOverride, 0)
	for rows.Next() {
		var o DeviceOverride
		if err := rows.Scan(
			&o.ID,
			&o.DeviceID,
			&o.Action,
			&o.DestType,
			&o.DestValue,
			&o.Port,
			&o.Label,
			&o.Reason,
			&o.ExpiresAt,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) CreateOverride(
	ctx context.Context,
	deviceID uuid.UUID,
	in CreateOverrideInput,
) (*DeviceOverride, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return nil, err
	}

	action, destType, destValue, label, port, err := normalizeRuleFields(
		in.Action,
		in.DestType,
		in.DestValue,
		in.Label,
		in.Port,
	)
	if err != nil {
		return nil, err
	}

	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, ErrInvalid
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if in.ExpiresAt != nil {
		t := in.ExpiresAt.UTC()
		if !t.After(now) {
			return nil, ErrInvalid
		}
		expiresAt = &t
	}
	if action == "allow" && expiresAt == nil {
		return nil, ErrInvalid
	}

	const q = `
		INSERT INTO device_network_overrides (
			device_id, action, dest_type, dest_value, port, label, reason, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, device_id, action, dest_type, dest_value, port, label, reason, expires_at, created_at, updated_at
	`
	var o DeviceOverride
	err = s.pool.QueryRow(
		ctx, q, deviceID, action, destType, destValue, port, label, reason, expiresAt,
	).Scan(
		&o.ID,
		&o.DeviceID,
		&o.Action,
		&o.DestType,
		&o.DestValue,
		&o.Port,
		&o.Label,
		&o.Reason,
		&o.ExpiresAt,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) DeleteOverride(
	ctx context.Context,
	deviceID, overrideID uuid.UUID,
) error {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return err
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM device_network_overrides
		WHERE id = $1 AND device_id = $2
	`, overrideID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
