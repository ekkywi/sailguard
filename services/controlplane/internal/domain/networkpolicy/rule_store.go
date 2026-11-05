package networkpolicy

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListRulesByPolicyID(ctx context.Context, policyID uuid.UUID) ([]PolicyRule, error) {
	const q = `
		SELECT id, policy_id, action, dest_type, dest_value, port, label, enabled, created_at, updated_at
		FROM network_policy_rules
		WHERE policy_id = $1
		ORDER BY created_at
	`
	rows, err := s.pool.Query(ctx, q, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PolicyRule, 0)
	for rows.Next() {
		var r PolicyRule
		if err := rows.Scan(
			&r.ID,
			&r.PolicyID,
			&r.Action,
			&r.DestType,
			&r.DestValue,
			&r.Port,
			&r.Label,
			&r.Enabled,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AddRule(ctx context.Context, policyID uuid.UUID, in AddRuleInput) (*PolicyRule, error) {
	if _, err := s.FindPolicyByID(ctx, policyID); err != nil {
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

	const q = `
		INSERT INTO network_policy_rules (
			policy_id, action, dest_type, dest_value, port, label
		) VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, policy_id, action, dest_type, dest_value, port, label, enabled, created_at, updated_at
	`
	var r PolicyRule
	err = s.pool.QueryRow(ctx, q, policyID, action, destType, destValue, port, label).Scan(
		&r.ID,
		&r.PolicyID,
		&r.Action,
		&r.DestType,
		&r.DestValue,
		&r.Port,
		&r.Label,
		&r.Enabled,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) UpdateRule(ctx context.Context, policyID, ruleID uuid.UUID, in UpdateRuleInput) (*PolicyRule, error) {
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

	const q = `
		UPDATE network_policy_rules
		SET action = $3,
			dest_type = $4,
			dest_value = $5,
			port = $6,
			label = $7,
			updated_at = now()
		WHERE id = $1 AND policy_id = $2
		RETURNING id, policy_id, action, dest_type, dest_value, port, label, enabled, created_at, updated_at
	`
	var r PolicyRule
	err = s.pool.QueryRow(ctx, q, ruleID, policyID, action, destType, destValue, port, label).Scan(
		&r.ID,
		&r.PolicyID,
		&r.Action,
		&r.DestType,
		&r.DestValue,
		&r.Port,
		&r.Label,
		&r.Enabled,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) DeleteRule(ctx context.Context, policyID, ruleID uuid.UUID) error {
	const q = `
		DELETE FROM network_policy_rules
		WHERE id = $1 AND policy_id = $2
	`
	tag, err := s.pool.Exec(ctx, q, ruleID, policyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
