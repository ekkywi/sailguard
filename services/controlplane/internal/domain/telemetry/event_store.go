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

// ListEvents return newest devices events.
func (s *Store) ListEvents(ctx context.Context, limit int) ([]DeviceEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	const q = `
		SELECT
			e.id, e.device_id, COALESCE(d.hostname, ''),
			e.client_event_id, e.event_type, e.occurred_at, e.received_at, e.dedupe_key,
			e.process_pid, e.process_name, e.process_path, e.process_hash_sha256,
			e.process_publisher, e.process_product_name, e.process_user_name,
			e.match_rule_id, e.match_action, e.match_rule_type, e.match_value, e.match_category_code,
			e.policy_version, e.policy_mode
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
		ORDER BY e.occurred_at DESC, e.received_at DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceEvent, 0)
	for rows.Next() {
		var e DeviceEvent
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
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
