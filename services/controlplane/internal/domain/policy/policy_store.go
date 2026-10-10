package policy

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListPolicies(ctx context.Context) ([]Policy, error) {
	const q = `
		SELECT id, name, description, mode, priority, version, published_at,
		       created_at, updated_at
		FROM policies
		ORDER BY name, created_at
	`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Policy, 0)
	for rows.Next() {
		var p Policy
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Mode, &p.Priority, &p.Version, &p.PublishedAt,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreatePolicy(ctx context.Context, name, description, mode string, priority int) (*Policy, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalid
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "audit"
	}
	if mode != "audit" && mode != "enforce" {
		return nil, ErrInvalid
	}
	if priority <= 0 {
		priority = 100
	}

	const q = `
		INSERT INTO policies (name, description, mode, priority)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, description, mode, priority, version, published_at,
		          created_at, updated_at
	`
	var p Policy
	err := s.pool.QueryRow(ctx, q, name, strings.TrimSpace(description), mode, priority).Scan(
		&p.ID, &p.Name, &p.Description, &p.Mode, &p.Priority, &p.Version, &p.PublishedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) FindPolicyByID(ctx context.Context, id uuid.UUID) (*Policy, error) {
	const q = `
		SELECT id, name, description, mode, priority, version, published_at,
		       created_at, updated_at
		FROM policies
		WHERE id = $1
	`
	var p Policy
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Mode, &p.Priority, &p.Version, &p.PublishedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
