package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DeviceListFilter selects devices. Empty fields are ignored.
type DeviceListFilter struct {
	Status string // active | disabled
	OS     string // windows | linux | darwin
	Q      string // hostname, display_name, pic, location, asset_tag, machine_guid
	Limit  int
	Offset int
}

type DeviceListResult struct {
	Items []Device
	Total int
}

func clampDeviceListLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (f DeviceListFilter) buildWhere() (string, []any) {
	conds := make([]string, 0, 4)
	args := make([]any, 0, 4)

	if st := strings.TrimSpace(strings.ToLower(f.Status)); st != "" {
		args = append(args, st)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if os := strings.TrimSpace(strings.ToLower(f.OS)); os != "" {
		args = append(args, os)
		conds = append(conds, fmt.Sprintf("os = $%d", len(args)))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		args = append(args, pat)
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			`(lower(hostname) LIKE $%d
			 OR lower(display_name) LIKE $%d
			 OR lower(pic) LIKE $%d
			 OR lower(location) LIKE $%d
			 OR lower(asset_tag) LIKE $%d
			 OR lower(COALESCE(machine_guid, '')) LIKE $%d
			 OR lower(notes) LIKE $%d)`,
			n, n, n, n, n, n, n,
		))
	}

	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

const deviceSelectCols = `
	id, hostname, display_name, os, os_version, agent_version,
	machine_guid, status,
	pic, location, asset_tag, notes,
	last_seen_at, enrolled_at, created_at, updated_at
`

func scanDevice(row pgx.Row) (Device, error) {
	var d Device
	err := row.Scan(
		&d.ID,
		&d.Hostname,
		&d.DisplayName,
		&d.OS,
		&d.OSVersion,
		&d.AgentVersion,
		&d.MachineGUID,
		&d.Status,
		&d.PIC,
		&d.Location,
		&d.AssetTag,
		&d.Notes,
		&d.LastSeenAt,
		&d.EnrolledAt,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	return d, err
}

func (s *Store) ListDevices(ctx context.Context, f DeviceListFilter) (DeviceListResult, error) {
	where, args := f.buildWhere()
	limit := clampDeviceListLimit(f.Limit)
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	countSQL := `SELECT count(*) FROM devices ` + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return DeviceListResult{}, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	listSQL := `
		SELECT ` + deviceSelectCols + `
		FROM devices
		` + where + `
		ORDER BY hostname, created_at
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
		OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return DeviceListResult{}, err
	}
	defer rows.Close()

	out := make([]Device, 0)
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return DeviceListResult{}, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return DeviceListResult{}, err
	}
	return DeviceListResult{Items: out, Total: total}, nil
}

func (s *Store) FindDeviceByID(ctx context.Context, id uuid.UUID) (*Device, error) {
	const q = `
		SELECT id, hostname, display_name, os, os_version, agent_version,
		       machine_guid, status,
			   pic, location, asset_tag, notes,
			   last_seen_at, enrolled_at, created_at, updated_at
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
		&d.PIC,
		&d.Location,
		&d.AssetTag,
		&d.Notes,
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
		 	machine_guid, status,
			pic, location, asset_tag, notes,
			last_seen_at, enrolled_at, created_at, updated_at
	`
	var d Device
	err := s.pool.QueryRow(
		ctx, q, hostname, displayName, os, osVersion, agentVersion, machineGUID,
	).Scan(
		&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
		&d.MachineGUID, &d.Status,
		&d.PIC, &d.Location, &d.AssetTag, &d.Notes,
		&d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) UpdateDevice(
	ctx context.Context,
	id uuid.UUID,
	displayName *string,
	status *string,
	pic *string,
	location *string,
	assetTag *string,
	notes *string,
) (*Device, error) {
	cur, err := s.FindDeviceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		cur.DisplayName = strings.TrimSpace(*displayName)
	}
	if status != nil {
		st := strings.TrimSpace(strings.ToLower(*status))
		switch st {
		case "active", "disabled":
			cur.Status = st
		default:
			return nil, ErrInvalid
		}
	}
	if pic != nil {
		cur.PIC = strings.TrimSpace(*pic)
	}
	if location != nil {
		cur.Location = strings.TrimSpace(*location)
	}
	if assetTag != nil {
		cur.AssetTag = strings.TrimSpace(*assetTag)
	}
	if notes != nil {
		cur.Notes = strings.TrimSpace(*notes)
	}

	const q = `
		UPDATE devices
		SET display_name = $2,
		    status = $3,
		    pic = $4,
		    location = $5,
		    asset_tag = $6,
		    notes = $7,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, hostname, display_name, os, os_version, agent_version,
		          machine_guid, status,
		          pic, location, asset_tag, notes,
		          last_seen_at, enrolled_at, created_at, updated_at
	`
	var d Device
	err = s.pool.QueryRow(
		ctx, q,
		id, cur.DisplayName, cur.Status,
		cur.PIC, cur.Location, cur.AssetTag, cur.Notes,
	).Scan(
		&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
		&d.MachineGUID, &d.Status,
		&d.PIC, &d.Location, &d.AssetTag, &d.Notes,
		&d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM devices WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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

func (s *Store) FindDeviceMachineGUID(ctx context.Context, machineGUID string) (*Device, error) {
	const q = `
		SELECT id, hostname, display_name, os, os_version, agent_version,
		       machine_guid, status,
			   pic, location, asset_tag, notes,
			   last_seen_at, enrolled_at, created_at, updated_at
		FROM devices
		WHERE machine_guid = $1
		LIMIT 1
	`

	var d Device
	err := s.pool.QueryRow(ctx, q, machineGUID).Scan(
		&d.ID, &d.Hostname, &d.DisplayName, &d.OS, &d.OSVersion, &d.AgentVersion,
		&d.MachineGUID, &d.Status,
		&d.PIC, &d.Location, &d.AssetTag, &d.Notes,
		&d.LastSeenAt, &d.EnrolledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}