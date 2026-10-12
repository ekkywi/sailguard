package policy

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListAssignments(ctx context.Context, policyID uuid.UUID) ([]PolicyAssignment, error) {
	const q = `
		SELECT id, policy_id, scope, group_id, device_id, created_at
		FROM policy_assignments
		WHERE policy_id = $1
		ORDER BY created_at
	`

	rows, err := s.pool.Query(ctx, q, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PolicyAssignment, 0)
	for rows.Next() {
		var a PolicyAssignment
		if err := rows.Scan(
			&a.ID, &a.PolicyID, &a.Scope, &a.GroupID, &a.DeviceID, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AddAssignment(ctx context.Context, policyID uuid.UUID, in AddAssignmentInput) (*PolicyAssignment, error) {
	if _, err := s.FindPolicyByID(ctx, policyID); err != nil {
		return nil, err
	}

	scope := strings.TrimSpace(strings.ToLower(in.Scope))
	var groupID, deviceID *uuid.UUID

	switch scope {
	case "org":
		// no targets
	case "group":
		id, err := uuid.Parse(strings.TrimSpace(in.GroupID))
		if err != nil {
			return nil, ErrInvalid
		}
		groupID = &id
	case "device":
		id, err := uuid.Parse(strings.TrimSpace(in.DeviceID))
		if err != nil {
			return nil, ErrInvalid
		}
		deviceID = &id
	default:
		return nil, ErrInvalid
	}

	const q = `
		INSERT INTO policy_assignments (policy_id, scope, group_id, device_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, policy_id, scope, group_id, device_id, created_at
	`

	var a PolicyAssignment
	err := s.pool.QueryRow(ctx, q, policyID, scope, groupID, deviceID).Scan(
		&a.ID, &a.PolicyID, &a.Scope, &a.GroupID, &a.DeviceID, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) RemoveAssignment(ctx context.Context, policyID, assignmentID uuid.UUID) error {
	const q = `
		DELETE FROM policy_assignments
		WHERE id = $1 AND policy_id = $2
	`

	tag, err := s.pool.Exec(ctx, q, assignmentID, policyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) PublishPolicy(ctx context.Context, policyID uuid.UUID) (*Policy, error) {
	const q = `
		UPDATE policies
		SET version = version + 1,
		    published_at = now()
		WHERE id = $1
		RETURNING id, name, description, mode, priority, version, published_at,
		          created_at, updated_at
	`
	var p Policy
	err := s.pool.QueryRow(ctx, q, policyID).Scan(
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
