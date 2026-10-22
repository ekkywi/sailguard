package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

// ListFilter selects audit rows. Empty fields are ignored.
type ListFilter struct {
	From         *time.Time
	To           *time.Time
	ActionPrefix string // e.g. "policy" → action LIKE 'policy.%'
	Actor        string // substring match on actor_email (case-insensitive)
	ResourceType string
	Q            string // substring on action, summary, actor_email, resource_id
	Limit        int
	Offset       int
}

type ListResult struct {
	Items []Entry
	Total int
}

func clampListLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (f ListFilter) buildWhere() (string, []any) {
	conds := make([]string, 0, 6)
	args := make([]any, 0, 6)

	if f.From != nil {
		args = append(args, *f.From)
		conds = append(conds, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		conds = append(conds, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if p := strings.TrimSpace(f.ActionPrefix); p != "" {
		args = append(args, p+".%")
		conds = append(conds, fmt.Sprintf("action LIKE $%d", len(args)))
	}
	if a := strings.TrimSpace(f.Actor); a != "" {
		args = append(args, "%"+strings.ToLower(a)+"%")
		conds = append(conds, fmt.Sprintf("lower(actor_email) LIKE $%d", len(args)))
	}
	if rt := strings.TrimSpace(f.ResourceType); rt != "" {
		args = append(args, rt)
		conds = append(conds, fmt.Sprintf("resource_type = $%d", len(args)))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		args = append(args, pat)
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(lower(action) LIKE $%d OR lower(summary) LIKE $%d OR lower(actor_email) LIKE $%d OR lower(resource_id) LIKE $%d)",
			n, n, n, n,
		))
	}

	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func (s *Store) List(ctx context.Context, f ListFilter) (ListResult, error) {
	where, args := f.buildWhere()
	limit := clampListLimit(f.Limit)
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	countSQL := `SELECT count(*) FROM admin_audit_logs ` + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	limPos := len(args) + 1
	offPos := len(args) + 2
	listSQL := fmt.Sprintf(`
		SELECT id, actor_user_id, actor_email, action,
				resource_type, resource_id, summary, metadata, created_at
		FROM admin_audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limPos, offPos)

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	out := make([]Entry, 0, limit)
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.ID, &e.ActorUserID, &e.ActorEmail, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Summary, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return ListResult{}, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: out, Total: total}, nil
}

// ListForExport returns up to maxRows matching the filter (newest first).
func (s *Store) ListForExport(ctx context.Context, f ListFilter, maxRows int) ([]Entry, error) {
	if maxRows <= 0 || maxRows > 10000 {
		maxRows = 10000
	}
	f.Limit = maxRows
	f.Offset = 0
	// bypass clampListLimit 200 for export
	where, args := f.buildWhere()
	listArgs := append(append([]any{}, args...), maxRows)
	listSQL := fmt.Sprintf(`
		SELECT id, actor_user_id, actor_email, action,
				resource_type, resource_id, summary, metadata, created_at
		FROM admin_audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d
	`, where, len(args)+1)

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
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
