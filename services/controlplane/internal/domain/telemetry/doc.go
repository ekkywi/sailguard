package telemetry

// Package telemetry: agent event persistence and dashboard summary queries.
//
// Heartbeat still updates devices.last_seen_at via inventory.
// Violations are stored in device_events (00005_events.sql).
// Dashboard Summary aggregates device online/stale + 24h violation counts.
// ViolationsTimeseries fills hourly/daily buckets for charts.
