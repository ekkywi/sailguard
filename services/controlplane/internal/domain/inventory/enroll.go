package inventory

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) EnrollDevice(
	ctx context.Context,
	enrollmentTokenPlain string,
	hostname, osFamily, osVersion, agentVersion, machineGUID string,
) (*Device, string, error) {
	hash := HashToken(strings.TrimSpace(enrollmentTokenPlain))
	tok, err := s.FindEnrollmentTokenByHash(ctx, hash)
	if errors.Is(err, ErrNotFound) {
		return nil, "", ErrEnrollmentInvalid
	}
	if err != nil {
		return nil, "", err
	}

	if err := tok.ValidateForEnroll(time.Now()); err != nil {
		return nil, "", err
	}

	deviceTokenPlain, err := GenerateEnrollmentSecret()
	if err != nil {
		return nil, "", err
	}
	credHash := HashToken(deviceTokenPlain)

	hostname = strings.TrimSpace(hostname)
	osFamily = strings.TrimSpace(strings.ToLower(osFamily))
	switch osFamily {
	case "windows", "linux", "darwin":
	default:
		osFamily = "windows"
	}
	displayName := hostname

	var mg *string
	if strings.TrimSpace(machineGUID) != "" {
		g := strings.TrimSpace(machineGUID)
		mg = &g
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback(ctx)

	if err := s.IncrementEnrollmentTokenUse(ctx, tx, tok.ID); err != nil {
		return nil, "", err
	}

	const insertDevice = `
	INSERT INTO devices (
		hostname, display_name, os, os_version, agent_version,
		machine_guid, status, enrolled_at
	) VALUES ($1, $2, $3, $4, $5, $6, 'active', now())
	RETURNING id, hostname, display_name, os, os_version, agent_version,
	          machine_guid, status, last_seen_at, enrolled_at, created_at, updated_at
	`
	var d Device
	err = tx.QueryRow(ctx, insertDevice,
		hostname, displayName, osFamily, osVersion, agentVersion, mg,
	).Scan(
		&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
		&d.MachineGUID, &d.Status, &d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, "", ErrDeviceExists
		}
		return nil, "", err
	}

	const insertCred = `
	INSERT INTO device_credentials (device_id, token_hash)
	VALUES ($1, $2)
`
	if _, err := tx.Exec(ctx, insertCred, d.ID, credHash); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}
	return &d, deviceTokenPlain, nil
}