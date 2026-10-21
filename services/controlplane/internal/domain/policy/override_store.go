package policy

import (
	"context"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

type DeviceOverride struct {
	ID        uuid.UUID
	DeviceID  uuid.UUID
	Action    string
	RuleType  string
	Operator  string
	Value     string
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Expired reports whether the override is past expires_at (nil = never expires).
func (o DeviceOverride) Expired(now time.Time) bool {
	return o.ExpiresAt != nil && !o.ExpiresAt.After(now)
}

type CreateOverrideInput struct {
	Action    string
	RuleType  string
	Operator  string
	Value     string
	Reason    string
	ExpiresAt *time.Time
}

func (s *Store) ListOverridesByDeviceID(
	ctx context.Context,
	deviceID uuid.UUID,
	includeExpired bool,
) ([]DeviceOverride, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return nil, err
	}

	q := `
		SELECT id, device_id, action, rule_type, operator, value, reason,
		       expires_at, created_at, updated_at
		FROM device_overrides
		WHERE device_id = $1
	`
	if !includeExpired {
		q += ` AND (expires_at IS NULL OR expires_at > now())`
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceOverride, 0)
	for rows.Next() {
		var o DeviceOverride
		if err := rows.Scan(
			&o.ID, &o.DeviceID, &o.Action, &o.RuleType, &o.Operator, &o.Value, &o.Reason,
			&o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) CreateOverride(
	ctx context.Context,
	deviceID uuid.UUID,
	in CreateOverrideInput,
) (*DeviceOverride, error) {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return nil, err
	}

	action := strings.TrimSpace(strings.ToLower(in.Action))
	ruleType := strings.TrimSpace(strings.ToLower(in.RuleType))
	op := strings.TrimSpace(strings.ToLower(in.Operator))
	if op == "" {
		op = "equals"
	}
	value := strings.TrimSpace(in.Value)
	reason := strings.TrimSpace(in.Reason)

	switch action {
	case "allow", "block":
	default:
		return nil, ErrInvalid
	}
	switch ruleType {
	case "process_name", "path", "hash_sha256", "publisher", "product_name":
	default:
		return nil, ErrInvalid
	}
	switch op {
	case "equals", "prefix", "contains":
	default:
		return nil, ErrInvalid
	}
	if value == "" || reason == "" {
		return nil, ErrInvalid
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if in.ExpiresAt != nil {
		t := in.ExpiresAt.UTC()
		if !t.After(now) {
			return nil, ErrInvalid
		}
		expiresAt = &t
	}
	// Emergency allow must be time-bounded (MVP product rule).
	if action == "allow" && expiresAt == nil {
		return nil, ErrInvalid
	}

	const q = `
		INSERT INTO device_overrides (
			device_id, action, rule_type, operator, value, reason, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, device_id, action, rule_type, operator, value, reason,
		          expires_at, created_at, updated_at
	`
	var o DeviceOverride
	err := s.pool.QueryRow(
		ctx, q, deviceID, action, ruleType, op, value, reason, expiresAt,
	).Scan(
		&o.ID, &o.DeviceID, &o.Action, &o.RuleType, &o.Operator, &o.Value, &o.Reason,
		&o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) DeleteOverride(
	ctx context.Context,
	deviceID, overrideID uuid.UUID,
) error {
	inv := inventory.NewStore(s.pool)
	if _, err := inv.FindDeviceByID(ctx, deviceID); err != nil {
		return err
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM device_overrides
		WHERE id = $1 AND device_id = $2
	`, overrideID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
