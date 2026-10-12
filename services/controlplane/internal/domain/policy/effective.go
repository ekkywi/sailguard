package policy

import (
	"context"
	"fmt"
	"time"

	policypkg "github.com/ekkywi/sailguard/pkgs/policy"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

type assignedPolicy struct {
	ID        uuid.UUID
	Mode      string
	Priority  int
	Version   int64
	UpdatedAt time.Time
	Scope     string
}

// EffectiveForDevice builds the flattened effective policy for one device.
// Only published policies assigned to org / the device's groups / the device
// are included. Category rules are expanded via category_signatures.
func (s *Store) EffectiveForDevice(ctx context.Context, deviceID uuid.UUID) (policypkg.EffectivePolicy, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return policypkg.EffectivePolicy{}, err
	}

	groupIDs, err := inv.ListGroupIDsForDevice(ctx, deviceID)
	if err != nil {
		return policypkg.EffectivePolicy{}, err
	}
	if groupIDs == nil {
		groupIDs = []uuid.UUID{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT
			p.id, p.mode, p.priority, p.version, p.updated_at, a.scope
		FROM policies p
		JOIN policy_assignments a ON a.policy_id = p.id
		WHERE p.published_at IS NOT NULL
		  AND p.version > 0
		  AND (
		        a.scope = 'org'
		     OR (a.scope = 'group' AND a.group_id = ANY($1::uuid[]))
		     OR (a.scope = 'device' AND a.device_id = $2)
		  )
	`, groupIDs, deviceID)
	if err != nil {
		return policypkg.EffectivePolicy{}, err
	}
	defer rows.Close()

	assigned := make([]assignedPolicy, 0)
	var maxVersion int64
	for rows.Next() {
		var ap assignedPolicy
		if err := rows.Scan(
			&ap.ID, &ap.Mode, &ap.Priority, &ap.Version, &ap.UpdatedAt, &ap.Scope,
		); err != nil {
			return policypkg.EffectivePolicy{}, err
		}
		assigned = append(assigned, ap)
		if ap.Version > maxVersion {
			maxVersion = ap.Version
		}
	}
	if err := rows.Err(); err != nil {
		return policypkg.EffectivePolicy{}, err
	}

	candidates := make([]policypkg.Candidate, 0)
	for _, ap := range assigned {
		rules, err := s.ListRulesByPolicyID(ctx, ap.ID)
		if err != nil {
			return policypkg.EffectivePolicy{}, err
		}
		for _, r := range rules {
			if !r.Enabled {
				continue
			}
			part, err := s.candidatesFromRule(ctx, ap, r)
			if err != nil {
				return policypkg.EffectivePolicy{}, err
			}
			candidates = append(candidates, part...)
		}
	}

	overrides, err := s.overrideCandidates(ctx, deviceID)
	if err != nil {
		return policypkg.EffectivePolicy{}, err
	}
	candidates = append(candidates, overrides...)

	return policypkg.Merge(policypkg.MergeInput{
		Candidates: candidates,
		Version:    maxVersion,
	}), nil
}

func (s *Store) candidatesFromRule(ctx context.Context, ap assignedPolicy, r PolicyRule) ([]policypkg.Candidate, error) {
	base := policypkg.Candidate{
		PolicyID:    ap.ID.String(),
		Action:      policypkg.Action(r.Action),
		Operator:    policypkg.Operator(r.Operator),
		Source:      "policy",
		Scope:       scopeFromDB(ap.Scope),
		PolicyMode:  policypkg.Mode(ap.Mode),
		PolicyPrio:  ap.Priority,
		UpdatedUnix: ap.UpdatedAt.Unix(),
	}

	if r.RuleType != "category" {
		base.ID = r.ID.String()
		base.RuleType = policypkg.RuleType(r.RuleType)
		base.Value = r.Value
		return []policypkg.Candidate{base}, nil
	}

	if r.CategoryCode == nil || *r.CategoryCode == "" {
		return nil, fmt.Errorf("%w: category rule missing category_code", ErrInvalid)
	}
	code := *r.CategoryCode

	sigRows, err := s.pool.Query(ctx, `
		SELECT id, rule_type, operator, value
		FROM category_signatures
		WHERE category_code = $1
	`, code)
	if err != nil {
		return nil, err
	}
	defer sigRows.Close()

	out := make([]policypkg.Candidate, 0)
	for sigRows.Next() {
		var sigID uuid.UUID
		var ruleType, operator, value string
		if err := sigRows.Scan(&sigID, &ruleType, &operator, &value); err != nil {
			return nil, err
		}
		c := base
		c.ID = sigID.String()
		c.RuleType = policypkg.RuleType(ruleType)
		c.Operator = policypkg.Operator(operator)
		c.Value = value
		c.CategoryCode = code
		c.Source = "category"
		out = append(out, c)
	}
	return out, sigRows.Err()
}

func (s *Store) overrideCandidates(ctx context.Context, deviceID uuid.UUID) ([]policypkg.Candidate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, rule_type, operator, value, reason, expires_at, updated_at
		FROM device_overrides
		WHERE device_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]policypkg.Candidate, 0)
	for rows.Next() {
		var (
			id                                uuid.UUID
			action, ruleType, operator, value string
			reason                            string
			expiresAt                         *time.Time
			updatedAt                         time.Time
		)
		if err := rows.Scan(
			&id, &action, &ruleType, &operator, &value, &reason, &expiresAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		c := policypkg.Candidate{
			ID:          id.String(),
			Action:      policypkg.Action(action),
			RuleType:    policypkg.RuleType(ruleType),
			Operator:    policypkg.Operator(operator),
			Value:       value,
			Source:      "override",
			Scope:       policypkg.ScopeOverride,
			PolicyMode:  policypkg.ModeAudit,
			UpdatedUnix: updatedAt.Unix(),
			Reason:      reason,
		}
		if expiresAt != nil {
			c.ExpiresUnix = expiresAt.Unix()
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scopeFromDB(scope string) policypkg.Scope {
	switch scope {
	case "org":
		return policypkg.ScopeOrg
	case "group":
		return policypkg.ScopeGroup
	case "device":
		return policypkg.ScopeDevice
	default:
		return policypkg.ScopeOrg
	}
}
