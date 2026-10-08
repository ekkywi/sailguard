package inventory

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]Device, error) {
	const q = `
		SELECT d.id, d.hostname, d.display_name, d.os, d.os_version, d.agent_version,
			d.machine_guid, d.status, d.last_seen_at, d.enrolled_at, d.created_at, d.updated_at
		FROM device_group_members m
		JOIN devices d ON d.id = m.device_id
		WHERE m.group_id = $1
		ORDER BY d.hostname, d.created_at
	`

	rows, err := s.pool.Query(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(
			&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
			&d.MachineGUID, &d.Status, &d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
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

func (s *Store) AddGroupMember(ctx context.Context, groupID, deviceID uuid.UUID) error {
	if _, err := s.FindGroupByID(ctx, groupID); err != nil {
		return err
	}
	if _, err := s.FindDeviceByID(ctx, deviceID); err != nil {
		return err
	}

	const q = `
		INSERT INTO device_group_members (group_id, device_id)
		VALUES ($1, $2)
	`

	_, err := s.pool.Exec(ctx, q, groupID, deviceID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyMember
		}
		return err
	}
	return nil
}

func (s *Store) RemoveGroupMember(ctx context.Context, groupID, deviceID uuid.UUID) error {
	const q = `
		DELETE FROM device_group_members
		WHERE group_id = $1 AND device_id = $2
	`

	tag, err := s.pool.Exec(ctx, q, groupID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
