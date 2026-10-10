package policy

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) ListRulesByPolicyID(ctx context.Context, policyID uuid.UUID) ([]PolicyRule, error) {
	const q = `
		SELECT id, policy_id, action, rule_type, operator, value, category_code,
		       enabled, created_at, updated_at
		FROM policy_rules
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
			&r.ID, &r.PolicyID, &r.Action, &r.RuleType, &r.Operator, &r.Value, &r.CategoryCode,
			&r.Enabled, &r.CreatedAt, &r.UpdatedAt,
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

	action := strings.TrimSpace(strings.ToLower(in.Action))
	ruleType := strings.TrimSpace(strings.ToLower(in.RuleType))
	op := strings.TrimSpace(strings.ToLower(in.Operator))
	if op == "" {
		op = "equals"
	}
	value := strings.TrimSpace(in.Value)
	cat := strings.TrimSpace(in.CategoryCode)

	switch action {
	case "allow", "block":
	default:
		return nil, ErrInvalid
	}
	switch ruleType {
	case "process_name", "path", "hash_sha256", "publisher", "product_name", "category":
	default:
		return nil, ErrInvalid
	}
	switch op {
	case "equals", "prefix", "contains":
	default:
		return nil, ErrInvalid
	}

	var categoryCode *string
	if ruleType == "category" {
		if cat == "" {
			return nil, ErrInvalid
		}
		categoryCode = &cat
		if value == "" {
			value = cat
		}
	} else if value == "" {
		return nil, ErrInvalid
	}

	const q = `
		INSERT INTO policy_rules (
			policy_id, action, rule_type, operator, value, category_code
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, policy_id, action, rule_type, operator, value, category_code,
		          enabled, created_at, updated_at
	`
	var r PolicyRule
	err := s.pool.QueryRow(ctx, q, policyID, action, ruleType, op, value, categoryCode).Scan(
		&r.ID, &r.PolicyID, &r.Action, &r.RuleType, &r.Operator, &r.Value, &r.CategoryCode,
		&r.Enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
