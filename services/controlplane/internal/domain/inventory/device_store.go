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

func (s *Store) CreateDevice(
	ctx context.Context,
	hostname, displayName, os, osVersion, agentVersion string,
	machineGUID *string,
) (*Device, error) {
	const q = `
		INSERT INTO devices (
			hostname, display_name, os, os_version, agent_version,
			machine_guid, status, enrolled_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'active', now())
		 RETURNING id, hostname, display_name, os, os_version, agent_version,
		 	machine_guid, status, last_seen_at, enrolled_at, created_at, updated_at
	`
	var d Device
	err := s.pool.QueryRow(
		ctx, q, hostname, displayName, os, osVersion, agentVersion, machineGUID,
	).Scan(
		&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
		&d.MachineGUID, &d.Status, &d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) TouchLastSeen(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE devices
		SET last_seen_at = now()
		WHERE id = $1
	`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ApplyHeartbeat updates last_seen_at and optionally soft-refreshes identity fields.
func (s *Store) ApplyHeartbeat(
	ctx context.Context,
	id uuid.UUID,
	hostname, osVersion, agentVersion string,
) error {
	const q = `
		UPDATE devices
		SET last_seen_at = now(),
		    hostname = CASE WHEN $2 <> '' THEN $2 ELSE hostname END,
		    os_version = CASE WHEN $3 <> '' THEN $3 ELSE os_version END,
		    agent_version = CASE WHEN $4 <> '' THEN $4 ELSE agent_version END
		WHERE id = $1
	`
	tag, err := s.pool.Exec(ctx, q, id, hostname, osVersion, agentVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}