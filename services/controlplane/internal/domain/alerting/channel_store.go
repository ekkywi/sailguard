package alerting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("alert channel not found")

func (s *Store) ListChannels(ctx context.Context) ([]Channel, error) {
	const q = `
		SELECT id, kind, name, enabled, config, created_at, updated_at
		FROM alert_channels
		ORDER BY created_at ASC
	`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Channel, 0)
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) GetChannel(ctx context.Context, id uuid.UUID) (*Channel, error) {
	const q = `
		SELECT id, kind, name, enabled, config, created_at, updated_at
		FROM alert_channels
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, q, id)
	ch, err := scanChannelRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

type UpdateChannelInput struct {
	Name    *string
	Enabled *bool
	URL     *string
}

func (s *Store) UpdateChannel(ctx context.Context, id uuid.UUID, in UpdateChannelInput) (*Channel, error) {
	cur, err := s.GetChannel(ctx, id)
	if err != nil {
		return nil, err
	}

	name := cur.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" {
			name = cur.Name
		}
	}
	enabled := cur.Enabled
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	cfg := cur.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	if in.URL != nil {
		cfg["url"] = strings.TrimSpace(*in.URL)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	const q = `
		UPDATE alert_channels
		SET name = $2, enabled = $3, config = $4::jsonb, updated_at = now()
		WHERE id = $1
		RETURNING id, kind, name, enabled, config, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q, id, name, enabled, raw)
	ch, err := scanChannelRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) ListEnabledWebhooks(ctx context.Context) ([]Channel, error) {
	all, err := s.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Channel, 0)
	for _, ch := range all {
		if ch.Kind == "webhook" && ch.Enabled && strings.TrimSpace(ch.WebhookURL()) != "" {
			out = append(out, ch)
		}
	}
	return out, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanChannelRow(row scannable) (Channel, error) {
	var ch Channel
	var raw []byte
	err := row.Scan(
		&ch.ID, &ch.Kind, &ch.Name, &ch.Enabled, &raw, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return Channel{}, err
	}
	cfg := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &cfg)
	}
	ch.Config = cfg
	return ch, nil
}

func scanChannel(rows pgx.Rows) (Channel, error) {
	return scanChannelRow(rows)
}
