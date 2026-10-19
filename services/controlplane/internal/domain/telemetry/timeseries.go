package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TimeseriesPoint is one bucket in the violations timeseries.
type TimeseriesPoint struct {
	T        time.Time
	Detected int64
	Blocked  int64
}

// ViolationsTimeseries is the response shape for GET /v1/dashboard/violations-timeseries.
type ViolationsTimeseries struct {
	Range  string
	Bucket string
	From   time.Time
	To     time.Time
	Points []TimeseriesPoint
}

// ViolationsTimeseries aggregates detected/blocked counts into filled buckets.
// rangeKey: "24h" (hourly) or "7d" (daily). Default "24h".
func (s *Store) ViolationsTimeseries(ctx context.Context, rangeKey string) (ViolationsTimeseries, error) {
	rangeKey = strings.ToLower(strings.TrimSpace(rangeKey))
	if rangeKey == "" {
		rangeKey = "24h"
	}

	now := time.Now().UTC()
	var (
		bucket string
		from   time.Time
		to     time.Time
		step   time.Duration
		trunc  string
	)

	switch rangeKey {
	case "7d":
		bucket = "day"
		trunc = "day"
		step = 24 * time.Hour
		to = now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		from = to.Add(-7 * 24 * time.Hour)
	case "24h":
		bucket = "hour"
		trunc = "hour"
		step = time.Hour
		to = now.Truncate(time.Hour).Add(time.Hour)
		from = to.Add(-24 * time.Hour)
	default:
		return ViolationsTimeseries{}, fmt.Errorf("invalid range %q (want 24h or 7d)", rangeKey)
	}

	out := ViolationsTimeseries{
		Range:  rangeKey,
		Bucket: bucket,
		From:   from,
		To:     to,
		Points: make([]TimeseriesPoint, 0),
	}

	const q = `
		SELECT
			date_trunc($1, occurred_at) AS bucket,
			COUNT(*) FILTER (WHERE event_type = 'violation_detected')::bigint AS detected,
			COUNT(*) FILTER (WHERE event_type = 'violation_blocked')::bigint AS blocked
		FROM device_events
		WHERE occurred_at >= $2
		  AND occurred_at < $3
		  AND event_type IN ('violation_detected', 'violation_blocked')
		GROUP BY 1
		ORDER BY 1
	`
	rows, err := s.pool.Query(ctx, q, trunc, from, to)
	if err != nil {
		return ViolationsTimeseries{}, fmt.Errorf("timeseries query: %w", err)
	}
	defer rows.Close()

	byBucket := make(map[int64]TimeseriesPoint)
	for rows.Next() {
		var p TimeseriesPoint
		if err := rows.Scan(&p.T, &p.Detected, &p.Blocked); err != nil {
			return ViolationsTimeseries{}, fmt.Errorf("scan timeseries: %w", err)
		}
		p.T = p.T.UTC()
		byBucket[p.T.Unix()] = p
	}
	if err := rows.Err(); err != nil {
		return ViolationsTimeseries{}, err
	}

	for t := from; t.Before(to); t = t.Add(step) {
		if p, ok := byBucket[t.Unix()]; ok {
			out.Points = append(out.Points, p)
			continue
		}
		out.Points = append(out.Points, TimeseriesPoint{T: t})
	}

	return out, nil
}
