package inventory

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListDevices(ctx context.Context) ([]Device, error) {
	const q = `
		SELECT id, hostname, display_name, os, os_version, agent_version,
		       machine_guid, status, last_seen_at, enrolled_at, created_at, updated_at
		FROM devices
		ORDER BY hostname, created_at
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(
			&d.ID,
			&d.Hostname,
			&d.DisplayName,
			&d.OS,
			&d.OSVersion,
			&d.AgentVersion,
			&d.MachineGUID,
			&d.Status,
			&d.LastSeenAt,
			&d.EnrolledAt,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) FindDeviceByID(ctx context.Context, id uuid.UUID) (*Device, error) {
	const q = `
		SELECT id, hostname, display_name, os, os_version, agent_version,
		       machine_guid, status, last_seen_at, enrolled_at, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	var d Device
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&d.ID,
		&d.Hostname,
		&d.DisplayName,
		&d.OS,
		&d.OSVersion,
		&d.AgentVersion,
		&d.MachineGUID,
		&d.Status,
		&d.LastSeenAt,
		&d.EnrolledAt,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}
