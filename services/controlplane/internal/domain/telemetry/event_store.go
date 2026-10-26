package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EventListFilter selects device_events rows. Empty fields are ignored.
type EventListFilter struct {
	From      *time.Time
	To        *time.Time
	EventType string // violation_detected | violation_blocked
	DeviceID  *uuid.UUID
	Q         string // substring: hostname, process, user, match value
	Limit     int
	Offset    int
}

type EventListResult struct {
	Items []DeviceEvent
	Total int
}

type ViolationRow struct {
	ID              uuid.UUID
	DeviceID        uuid.UUID
	ClientEventID   string
	EventType       string
	OccurredAt      time.Time
	ProcessName     string
	ProcessPath     string
	ProcessUserName string
	MatchAction     string
	MatchValue      string
	PolicyMode      string
}

func clampEventListLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (f EventListFilter) buildWhere() (string, []any) {
	conds := make([]string, 0, 6)
	args := make([]any, 0, 6)

	if f.From != nil {
		args = append(args, *f.From)
		conds = append(conds, fmt.Sprintf("e.occurred_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		conds = append(conds, fmt.Sprintf("e.occurred_at <= $%d", len(args)))
	}
	if et := strings.TrimSpace(f.EventType); et != "" {
		args = append(args, et)
		conds = append(conds, fmt.Sprintf("e.event_type = $%d", len(args)))
	}
	if f.DeviceID != nil {
		args = append(args, *f.DeviceID)
		conds = append(conds, fmt.Sprintf("e.device_id = $%d", len(args)))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		args = append(args, pat)
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			`(lower(COALESCE(d.hostname, '')) LIKE $%d
			 OR lower(COALESCE(e.process_name, '')) LIKE $%d
			 OR lower(COALESCE(e.process_path, '')) LIKE $%d
			 OR lower(COALESCE(e.process_user_name, '')) LIKE $%d
			 OR lower(COALESCE(e.match_value, '')) LIKE $%d
			 OR lower(COALESCE(e.match_category_code, '')) LIKE $%d)`,
			n, n, n, n, n, n,
		))
	}

	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

const eventSelectCols = `
	e.id, e.device_id, COALESCE(d.hostname, ''),
	e.client_event_id, e.event_type, e.occurred_at, e.received_at, e.dedupe_key,
	e.process_pid, e.process_name, e.process_path, e.process_hash_sha256,
	e.process_publisher, e.process_product_name, e.process_user_name,
	e.match_rule_id, e.match_action, e.match_rule_type, e.match_value, e.match_category_code,
	e.policy_version, e.policy_mode
`

func scanDeviceEvent(rows pgx.Rows) (DeviceEvent, error) {
	var e DeviceEvent
	err := rows.Scan(
		&e.ID,
		&e.DeviceID,
		&e.Hostname,
		&e.ClientEventID,
		&e.EventType,
		&e.OccurredAt,
		&e.ReceivedAt,
		&e.DedupeKey,
		&e.ProcessPID,
		&e.ProcessName,
		&e.ProcessPath,
		&e.ProcessHashSHA256,
		&e.ProcessPublisher,
		&e.ProcessProductName,
		&e.ProcessUserName,
		&e.MatchRuleID,
		&e.MatchAction,
		&e.MatchRuleType,
		&e.MatchValue,
		&e.MatchCategoryCode,
		&e.PolicyVersion,
		&e.PolicyMode,
	)
	return e, err
}

// ListEvents returns newest matching device events with total count.
func (s *Store) ListEvents(ctx context.Context, f EventListFilter) (EventListResult, error) {
	where, args := f.buildWhere()
	limit := clampEventListLimit(f.Limit)
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	countSQL := `
		SELECT count(*)
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
	` + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return EventListResult{}, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	listSQL := fmt.Sprintf(`
		SELECT %s
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
		%s
		ORDER BY e.occurred_at DESC, e.received_at DESC
		LIMIT $%d OFFSET $%d
	`, eventSelectCols, where, len(args)+1, len(args)+2)

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return EventListResult{}, err
	}
	defer rows.Close()

	out := make([]DeviceEvent, 0)
	for rows.Next() {
		e, err := scanDeviceEvent(rows)
		if err != nil {
			return EventListResult{}, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return EventListResult{}, err
	}
	return EventListResult{Items: out, Total: total}, nil
}

// ListEventsForExport returns up to maxRows matching the filter (newest first).
func (s *Store) ListEventsForExport(ctx context.Context, f EventListFilter, maxRows int) ([]DeviceEvent, error) {
	if maxRows <= 0 || maxRows > 10000 {
		maxRows = 10000
	}
	where, args := f.buildWhere()
	listArgs := append(append([]any{}, args...), maxRows)
	listSQL := fmt.Sprintf(`
		SELECT %s
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
		%s
		ORDER BY e.occurred_at DESC, e.received_at DESC
		LIMIT $%d
	`, eventSelectCols, where, len(args)+1)

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceEvent, 0)
	for rows.Next() {
		e, err := scanDeviceEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// InsertViolation inserts a violation row.
// Returns inserted=false when (device_id, client_event_id) already exists (idempotent).
func (s *Store) InsertViolation(
	ctx context.Context,
	in InsertViolationInput,
) (inserted bool, id uuid.UUID, err error) {
	eventType := strings.TrimSpace(in.EventType)
	if eventType != "violation_detected" && eventType != "violation_blocked" {
		return false, uuid.Nil, fmt.Errorf("invalid event_type: %q", eventType)
	}
	if strings.TrimSpace(in.ClientEventID) == "" {
		return false, uuid.Nil, fmt.Errorf("client_event_id is required")
	}
	if strings.TrimSpace(in.ProcessName) == "" {
		return false, uuid.Nil, fmt.Errorf("process_name is required")
	}
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}

	const q = `
		INSERT INTO device_events (
			device_id, client_event_id, event_type, occurred_at, dedupe_key,
			process_pid, process_name, process_path, process_hash_sha256,
			process_publisher, process_product_name, process_user_name,
			match_rule_id, match_action, match_rule_type, match_value, match_category_code,
			policy_version, policy_mode
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''),
			$6, $7, NULLIF($8, ''), NULLIF($9, ''),
			NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''),
			NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, ''),
			$18, NULLIF($19, '')
		)
		ON CONFLICT (device_id, client_event_id) DO NOTHING
		RETURNING id
	`

	var pid *int32
	if in.ProcessPID != 0 {
		v := int32(in.ProcessPID)
		pid = &v
	}
	var polVer any
	if in.PolicyVersion != 0 {
		polVer = in.PolicyVersion
	}

	err = s.pool.QueryRow(
		ctx, q,
		in.DeviceID,
		strings.TrimSpace(in.ClientEventID),
		eventType,
		in.OccurredAt.UTC(),
		strings.TrimSpace(in.DedupeKey),
		pid,
		strings.TrimSpace(in.ProcessName),
		in.ProcessPath,
		in.ProcessHashSHA256,
		in.ProcessPublisher,
		in.ProcessProductName,
		in.ProcessUserName,
		in.MatchRuleID,
		in.MatchAction,
		in.MatchRuleType,
		in.MatchValue,
		in.MatchCategoryCode,
		polVer,
		in.PolicyMode,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, uuid.Nil, nil
		}
		return false, uuid.Nil, err
	}
	return true, id, nil
}

func (s *Store) GetViolationByID(ctx context.Context, id uuid.UUID) (*ViolationRow, error) {
	const q = `
		SELECT id, device_id, client_event_id, event_type, occurred_at,
				COALESCE(process_name, ''), COALESCE(process_path, ''),
				COALESCE(process_user_name, ''),
				COALESCE(match_action, ''), COALESCE(match_value, ''),
				COALESCE(policy_mode, '')
		FROM device_events
		WHERE id = $1
	`
	var r ViolationRow
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&r.ID, &r.DeviceID, &r.ClientEventID, &r.EventType, &r.OccurredAt,
		&r.ProcessName, &r.ProcessPath, &r.ProcessUserName,
		&r.MatchAction, &r.MatchValue, &r.PolicyMode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}