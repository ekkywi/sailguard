package audit

import (
	"context"
	"encoding/json"
	"strings"
)

func (s *Store) Record(ctx context.Context, in RecordInput) error {
	action := strings.TrimSpace(in.Action)
	if action == "" {
		return nil
	}
	email := strings.TrimSpace(in.ActorEmail)
	resourceType := strings.TrimSpace(in.ResourceType)
	resourceID := strings.TrimSpace(in.ResourceID)
	summary := strings.TrimSpace(in.Summary)

	meta := in.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		raw = []byte("{}")
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO admin_audit_logs (
			actor_user_id, actor_email, action,
			resource_type, resource_id, summary, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, in.ActorUserID, email, action, resourceType, resourceID, summary, raw)
	return err
}

func (s *Store) List(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor_user_id, actor_email, action,
				resource_type,resource_id, summary, metadata, created_at
		FROM admin_audit_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Entry, 0)
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.ID, &e.ActorUserID, &e.ActorEmail, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Summary, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}