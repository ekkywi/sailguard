package networkpolicy

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

const (
	scopeWeightOrg      = 100
	scopeWeightGroup    = 200
	scopeWeightDevice   = 300
	scopeWeightOverride = 400
)

type assignedNetworkPolicy struct {
	ID        uuid.UUID
	Mode      string
	Priority  int
	Version   int64
	UpdatedAt time.Time
	Scope     string
}

type netCandidate struct {
	ID          string
	Action      string
	DestType    string
	DestValue   string
	Port        *int
	Label       string
	Source      string
	ScopeWeight int
	PolicyPrio  int
	PolicyMode  string
	UpdatedUnix int64
}

// EffectiveForDevice builds the flattened network destination policy for one device.
// Only published network policies assigned to org / the device's groups / the device
// are included, plus non-expired device overrides. Evaluation order for the agent:
// safety_allow → rules allow → rules deny → fail-open.
func (s *Store) EffectiveForDevice(ctx context.Context, deviceID uuid.UUID) (EffectivePolicy, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return EffectivePolicy{}, err
	}

	groupIDs, err := inv.ListGroupIDsForDevice(ctx, deviceID)
	if err != nil {
		return EffectivePolicy{}, err
	}
	if groupIDs == nil {
		groupIDs = []uuid.UUID{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT
			p.id, p.mode, p.priority, p.version, p.updated_at, a.scope
		FROM network_policies p
		JOIN network_policy_assignments a ON a.policy_id = p.id
		WHERE p.published_at IS NOT NULL
		  AND p.version > 0
		  AND (
		        a.scope = 'org'
		     OR (a.scope = 'group' AND a.group_id = ANY($1::uuid[]))
		     OR (a.scope = 'device' AND a.device_id = $2)
		  )
	`, groupIDs, deviceID)
	if err != nil {
		return EffectivePolicy{}, err
	}
	defer rows.Close()

	assigned := make([]assignedNetworkPolicy, 0)
	var maxVersion int64
	for rows.Next() {
		var ap assignedNetworkPolicy
		if err := rows.Scan(
			&ap.ID, &ap.Mode, &ap.Priority, &ap.Version, &ap.UpdatedAt, &ap.Scope,
		); err != nil {
			return EffectivePolicy{}, err
		}
		assigned = append(assigned, ap)
		if ap.Version > maxVersion {
			maxVersion = ap.Version
		}
	}
	if err := rows.Err(); err != nil {
		return EffectivePolicy{}, err
	}

	candidates := make([]netCandidate, 0)
	for _, ap := range assigned {
		rules, err := s.ListRulesByPolicyID(ctx, ap.ID)
		if err != nil {
			return EffectivePolicy{}, err
		}
		for _, r := range rules {
			if !r.Enabled {
				continue
			}
			candidates = append(candidates, netCandidate{
				ID:          r.ID.String(),
				Action:      r.Action,
				DestType:    r.DestType,
				DestValue:   r.DestValue,
				Port:        r.Port,
				Label:       r.Label,
				Source:      ap.Scope,
				ScopeWeight: scopeWeight(ap.Scope),
				PolicyPrio:  ap.Priority,
				PolicyMode:  ap.Mode,
				UpdatedUnix: ap.UpdatedAt.Unix(),
			})
		}
	}

	overrides, err := s.overrideCandidates(ctx, deviceID)
	if err != nil {
		return EffectivePolicy{}, err
	}
	candidates = append(candidates, overrides...)

	rules, mode := mergeNetworkCandidates(candidates)
	ep := EffectivePolicy{
		SchemaVersion: 1,
		Version:       maxVersion,
		ComputedAt:    time.Now().UTC().Format(time.RFC3339),
		Mode:          mode,
		SafetyAllow:   defaultSafetyAllow(),
		Rules:         rules,
	}
	ep.Hash = HashEffective(ep)
	return ep, nil
}

func (s *Store) overrideCandidates(ctx context.Context, deviceID uuid.UUID) ([]netCandidate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, dest_type, dest_value, port, label, updated_at
		FROM device_network_overrides
		WHERE device_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]netCandidate, 0)
	for rows.Next() {
		var (
			id                     uuid.UUID
			action, destType, dest string
			port                   *int
			label                  string
			updatedAt              time.Time
		)
		if err := rows.Scan(&id, &action, &destType, &dest, &port, &label, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, netCandidate{
			ID:          id.String(),
			Action:      action,
			DestType:    destType,
			DestValue:   dest,
			Port:        port,
			Label:       label,
			Source:      "device_override",
			ScopeWeight: scopeWeightOverride,
			PolicyPrio:  0,
			PolicyMode:  "audit", // overrides do not force enforce
			UpdatedUnix: updatedAt.Unix(),
		})
	}
	return out, rows.Err()
}

func scopeWeight(scope string) int {
	switch scope {
	case "org":
		return scopeWeightOrg
	case "group":
		return scopeWeightGroup
	case "device":
		return scopeWeightDevice
	case "device_override", "override":
		return scopeWeightOverride
	default:
		return scopeWeightOrg
	}
}

func destKey(destType, destValue string, port *int) string {
	if port == nil {
		return destType + "|" + destValue + "|*"
	}
	return fmt.Sprintf("%s|%s|%d", destType, destValue, *port)
}

func better(a, b netCandidate) bool {
	if a.ScopeWeight != b.ScopeWeight {
		return a.ScopeWeight > b.ScopeWeight
	}
	if a.Action != b.Action {
		return a.Action == "allow" && b.Action == "deny"
	}
	if a.PolicyPrio != b.PolicyPrio {
		return a.PolicyPrio > b.PolicyPrio
	}
	if a.UpdatedUnix != b.UpdatedUnix {
		return a.UpdatedUnix > b.UpdatedUnix
	}
	return a.ID > b.ID
}

func pickWinner(group []netCandidate) netCandidate {
	best := group[0]
	for _, c := range group[1:] {
		if better(c, best) {
			best = c
		}
	}
	return best
}

// mergeNetworkCandidates resolves destination conflicts and returns sorted rules
// (allow first, then deny) plus B1 mode: enforce if any winning policy was enforce.
func mergeNetworkCandidates(candidates []netCandidate) ([]EffectiveRule, string) {
	if len(candidates) == 0 {
		return make([]EffectiveRule, 0), "audit"
	}

	byKey := make(map[string][]netCandidate)
	for _, c := range candidates {
		k := destKey(c.DestType, c.DestValue, c.Port)
		byKey[k] = append(byKey[k], c)
	}

	winners := make([]netCandidate, 0, len(byKey))
	mode := "audit"
	for _, group := range byKey {
		w := pickWinner(group)
		winners = append(winners, w)
		if w.PolicyMode == "enforce" {
			mode = "enforce"
		}
	}

	sort.SliceStable(winners, func(i, j int) bool {
		if winners[i].Action != winners[j].Action {
			if winners[i].Action == "allow" {
				return true
			}
			if winners[j].Action == "allow" {
				return false
			}
		}
		if winners[i].DestValue != winners[j].DestValue {
			return winners[i].DestValue < winners[j].DestValue
		}
		return winners[i].ID < winners[j].ID
	})

	out := make([]EffectiveRule, 0, len(winners))
	for _, w := range winners {
		out = append(out, EffectiveRule{
			ID:        w.ID,
			Action:    w.Action,
			DestType:  w.DestType,
			DestValue: w.DestValue,
			Port:      w.Port,
			Label:     w.Label,
			Source:    w.Source,
		})
	}
	return out, mode
}
