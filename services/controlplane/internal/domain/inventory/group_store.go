package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListGroups(ctx context.Context) ([]DeviceGroup, error) {
	const q = `
		SELECT id, name, description, created_at, updated_at
		FROM device_groups
		ORDER BY name
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceGroup, 0)
	for rows.Next() {
		var g DeviceGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateGroup(ctx context.Context, name, description string) (*DeviceGroup, error) {
	const q = `
		INSERT INTO device_groups (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`

	var g DeviceGroup
	err := s.pool.QueryRow(ctx, q, name, description).Scan(
		&g.ID,
		&g.Name,
		&g.Description,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) FindGroupByID(ctx context.Context, id uuid.UUID) (*DeviceGroup, error) {
	const q = `
		SELECT id, name, description, created_at, updated_at
		FROM device_groups
		WHERE id = $1
	`

	var g DeviceGroup
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&g.ID,
		&g.Name,
		&g.Description,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) UpdateGroup(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	description *string,
) (*DeviceGroup, error) {
	cur, err := s.FindGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return nil, ErrInvalid
		}
		cur.Name = n
	}
	if description != nil {
		cur.Description = strings.TrimSpace(*description)
	}

	const q = `
		UPDATE device_groups
		SET name = $2,
			description = $3,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, description, created_at, updated_at
	`

	var g DeviceGroup
	err = s.pool.QueryRow(ctx, q, id, cur.Name, cur.Description).Scan(
		&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM device_groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
