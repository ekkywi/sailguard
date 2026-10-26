package alerting

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrDeliveryNotFound = errors.New("alert delivery not found")

// EnqueueDelivery creates a pending row if (channel, event) is new.
// On conflict, returns the existing row (idempotent).
func (s *Store) EnqueueDelivery(
	ctx context.Context,
	channelID, eventID uuid.UUID,
) (*Delivery, error) {
	const q = `
		INSERT INTO alert_deliveries (channel_id, event_id, status, attempts, max_attempts)
		VALUES ($1, $2, 'pending', 0, 5)
		ON CONFLICT (channel_id, event_id) DO UPDATE
			SET updated_at = alert_deliveries.updated_at
		RETURNING id, channel_id, event_id, status, attempts, max_attempts,
		          last_error, next_retry_at, delivered_at, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q, channelID, eventID)
	d, err := scanDeliveryRow(row)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) GetDelivery(ctx context.Context, id uuid.UUID) (*Delivery, error) {
	const q = `
		SELECT id, channel_id, event_id, status, attempts, max_attempts,
		       last_error, next_retry_at, delivered_at, created_at, updated_at
		FROM alert_deliveries
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, q, id)
	d, err := scanDeliveryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) MarkDelivered(ctx context.Context, id uuid.UUID) (*Delivery, error) {
	const q = `
		UPDATE alert_deliveries
		SET status = 'delivered',
		    last_error = NULL,
		    next_retry_at = NULL,
		    delivered_at = now(),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, channel_id, event_id, status, attempts, max_attempts,
		          last_error, next_retry_at, delivered_at, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q, id)
	d, err := scanDeliveryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// MarkAttemptFailed increments attempts. If attempts >= max → dead; else failed + next_retry_at.
func (s *Store) MarkAttemptFailed(
	ctx context.Context,
	id uuid.UUID,
	errMsg string,
	backoff time.Duration,
) (*Delivery, error) {
	const q = `
		UPDATE alert_deliveries
		SET attempts = attempts + 1,
		    last_error = $2,
		    status = CASE
		        WHEN attempts + 1 >= max_attempts THEN 'dead'
		        ELSE 'failed'
		    END,
		    next_retry_at = CASE
		        WHEN attempts + 1 >= max_attempts THEN NULL
		        ELSE now() + $3::interval
		    END,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, channel_id, event_id, status, attempts, max_attempts,
		          last_error, next_retry_at, delivered_at, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q, id, errMsg, backoff.String())
	d, err := scanDeliveryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// RequestRetry sets pending so a worker/admin can send again (from failed or dead).
func (s *Store) RequestRetry(ctx context.Context, id uuid.UUID) (*Delivery, error) {
	const q = `
		UPDATE alert_deliveries
		SET status = 'pending',
		    next_retry_at = now(),
		    updated_at = now()
		WHERE id = $1
		  AND status IN ('failed', 'dead')
		RETURNING id, channel_id, event_id, status, attempts, max_attempts,
		          last_error, next_retry_at, delivered_at, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q, id)
	d, err := scanDeliveryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) ListDeliveries(
	ctx context.Context,
	status string,
	limit int,
) ([]Delivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var (
		rows pgx.Rows
		err  error
	)
	if status == "" {
		const q = `
			SELECT id, channel_id, event_id, status, attempts, max_attempts,
			       last_error, next_retry_at, delivered_at, created_at, updated_at
			FROM alert_deliveries
			ORDER BY created_at DESC
			LIMIT $1
		`
		rows, err = s.pool.Query(ctx, q, limit)
	} else {
		const q = `
			SELECT id, channel_id, event_id, status, attempts, max_attempts,
			       last_error, next_retry_at, delivered_at, created_at, updated_at
			FROM alert_deliveries
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = s.pool.Query(ctx, q, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Delivery, 0)
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ClaimDueRetries flips due failed rows to pending and returns them so the
// worker can Produce once without re-selecting every loop iteration.
func (s *Store) ClaimDueRetries(ctx context.Context, limit int) ([]Delivery, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	const q = `
		UPDATE alert_deliveries
		SET status = 'pending',
		    next_retry_at = NULL,
		    updated_at = now()
		WHERE id IN (
			SELECT id
			FROM alert_deliveries
			WHERE status = 'failed'
			  AND next_retry_at IS NOT NULL
			  AND next_retry_at <= now()
			ORDER BY next_retry_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, channel_id, event_id, status, attempts, max_attempts,
		          last_error, next_retry_at, delivered_at, created_at, updated_at
	`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Delivery, 0)
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanDeliveryRow(row scannable) (Delivery, error) {
	var d Delivery
	err := row.Scan(
		&d.ID, &d.ChannelID, &d.EventID, &d.Status, &d.Attempts, &d.MaxAttempts,
		&d.LastError, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}
