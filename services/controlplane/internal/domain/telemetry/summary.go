package telemetry

import (
	"context"
	"fmt"
	"time"
)

const (
	DefaultOnlineWindowSec = 120
	DefaultRecentLimit     = 10
	DefaultTopLimit        = 5
	DefaultStaleLimit      = 10
)

// DeviceCounts is fleet inventory + online/stale among active devices.
type DeviceCounts struct {
	Total    int64
	Active   int64
	Disabled int64
	Pending  int64
	Online   int64
	Stale    int64
}

// ViolationCounts is violation totals in a time window.
type ViolationCounts struct {
	Detected int64
	Blocked  int64
}

// RecentViolation is a compact row for the dashboard feed.
type RecentViolation struct {
	ID          string
	DeviceID    string
	Hostname    string
	EventType   string
	ProcessName *string
	OccurredAt  time.Time
}

// TopProcessCount is a process name ranked by violation volume (24h).
type TopProcessCount struct {
	ProcessName string
	Count       int64
}

// TopHostCount is a device ranked by violation volume (24h).
type TopHostCount struct {
	DeviceID string
	Hostname string
	Count    int64
}

// StaleDevice is an active device outside the online window.
type StaleDevice struct {
	DeviceID   string
	Hostname   string
	LastSeenAt *time.Time
}

// DashboardSummary is the fleet snapshot for GET /v1/dashboard/summary.
type DashboardSummary struct {
	GeneratedAt      time.Time
	OnlineWindowSec  int
	Devices          DeviceCounts
	Violations24h    ViolationCounts
	RecentViolations []RecentViolation
	TopProcesses     []TopProcessCount
	TopHosts         []TopHostCount
	StaleDevices     []StaleDevice
}

// Summary aggregates fleet health and recent violations from Postgres.
// onlineWindowSec defaults to DefaultOnlineWindowSec when <= 0.
func (s *Store) Summary(ctx context.Context, onlineWindowSec int) (DashboardSummary, error) {
	if onlineWindowSec <= 0 {
		onlineWindowSec = DefaultOnlineWindowSec
	}

	out := DashboardSummary{
		GeneratedAt:      time.Now().UTC(),
		OnlineWindowSec:  onlineWindowSec,
		RecentViolations: make([]RecentViolation, 0),
		TopProcesses:     make([]TopProcessCount, 0),
		TopHosts:         make([]TopHostCount, 0),
		StaleDevices:     make([]StaleDevice, 0),
	}

	const deviceQ = `
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE status = 'active')::bigint AS active,
			COUNT(*) FILTER (WHERE status = 'disabled')::bigint AS disabled,
			COUNT(*) FILTER (WHERE status = 'pending')::bigint AS pending,
			COUNT(*) FILTER (
				WHERE status = 'active'
				  AND last_seen_at IS NOT NULL
				  AND last_seen_at >= now() - make_interval(secs => $1)
			)::bigint AS online,
			COUNT(*) FILTER (
				WHERE status = 'active'
				  AND (last_seen_at IS NULL OR last_seen_at < now() - make_interval(secs => $1))
			)::bigint AS stale
		FROM devices
	`
	if err := s.pool.QueryRow(ctx, deviceQ, onlineWindowSec).Scan(
		&out.Devices.Total,
		&out.Devices.Active,
		&out.Devices.Disabled,
		&out.Devices.Pending,
		&out.Devices.Online,
		&out.Devices.Stale,
	); err != nil {
		return DashboardSummary{}, fmt.Errorf("device counts: %w", err)
	}

	const violQ = `
		SELECT
			COUNT(*) FILTER (WHERE event_type = 'violation_detected')::bigint AS detected,
			COUNT(*) FILTER (WHERE event_type = 'violation_blocked')::bigint AS blocked
		FROM device_events
		WHERE occurred_at >= now() - interval '24 hours'
		  AND event_type IN ('violation_detected', 'violation_blocked')
	`
	if err := s.pool.QueryRow(ctx, violQ).Scan(
		&out.Violations24h.Detected,
		&out.Violations24h.Blocked,
	); err != nil {
		return DashboardSummary{}, fmt.Errorf("violation counts: %w", err)
	}

	const recentQ = `
		SELECT
			e.id::text,
			e.device_id::text,
			COALESCE(d.hostname, ''),
			e.event_type,
			e.process_name,
			e.occurred_at
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
		WHERE e.event_type IN ('violation_detected', 'violation_blocked')
		ORDER BY e.occurred_at DESC, e.received_at DESC
		LIMIT $1
	`
	rows, err := s.pool.Query(ctx, recentQ, DefaultRecentLimit)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("recent violations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v RecentViolation
		if err := rows.Scan(
			&v.ID,
			&v.DeviceID,
			&v.Hostname,
			&v.EventType,
			&v.ProcessName,
			&v.OccurredAt,
		); err != nil {
			return DashboardSummary{}, fmt.Errorf("scan recent: %w", err)
		}
		out.RecentViolations = append(out.RecentViolations, v)
	}
	if err := rows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	const topProcQ = `
		SELECT COALESCE(process_name, ''), COUNT(*)::bigint AS cnt
		FROM device_events
		WHERE occurred_at >= now() - interval '24 hours'
		  AND event_type IN ('violation_detected', 'violation_blocked')
		  AND COALESCE(process_name, '') <> ''
		GROUP BY COALESCE(process_name, '')
		ORDER BY cnt DESC, 1 ASC
		LIMIT $1
	`
	procRows, err := s.pool.Query(ctx, topProcQ, DefaultTopLimit)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("top processes: %w", err)
	}
	defer procRows.Close()
	for procRows.Next() {
		var p TopProcessCount
		if err := procRows.Scan(&p.ProcessName, &p.Count); err != nil {
			return DashboardSummary{}, fmt.Errorf("scan top process: %w", err)
		}
		out.TopProcesses = append(out.TopProcesses, p)
	}
	if err := procRows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	const topHostQ = `
		SELECT e.device_id::text, COALESCE(d.hostname, ''), COUNT(*)::bigint AS cnt
		FROM device_events e
		LEFT JOIN devices d ON d.id = e.device_id
		WHERE e.occurred_at >= now() - interval '24 hours'
		  AND e.event_type IN ('violation_detected', 'violation_blocked')
		GROUP BY e.device_id, d.hostname
		ORDER BY cnt DESC, 2 ASC
		LIMIT $1
	`
	hostRows, err := s.pool.Query(ctx, topHostQ, DefaultTopLimit)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("top hosts: %w", err)
	}
	defer hostRows.Close()
	for hostRows.Next() {
		var h TopHostCount
		if err := hostRows.Scan(&h.DeviceID, &h.Hostname, &h.Count); err != nil {
			return DashboardSummary{}, fmt.Errorf("scan top host: %w", err)
		}
		out.TopHosts = append(out.TopHosts, h)
	}
	if err := hostRows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	const staleQ = `
		SELECT id::text, COALESCE(hostname, ''), last_seen_at
		FROM devices
		WHERE status = 'active'
		  AND (last_seen_at IS NULL OR last_seen_at < now() - make_interval(secs => $1))
		ORDER BY last_seen_at ASC NULLS FIRST, hostname ASC
		LIMIT $2
	`
	staleRows, err := s.pool.Query(ctx, staleQ, onlineWindowSec, DefaultStaleLimit)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("stale devices: %w", err)
	}
	defer staleRows.Close()
	for staleRows.Next() {
		var d StaleDevice
		if err := staleRows.Scan(&d.DeviceID, &d.Hostname, &d.LastSeenAt); err != nil {
			return DashboardSummary{}, fmt.Errorf("scan stale: %w", err)
		}
		out.StaleDevices = append(out.StaleDevices, d)
	}
	if err := staleRows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	return out, nil
}
