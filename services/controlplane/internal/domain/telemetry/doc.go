package telemetry

// Package telemetry: agent event persistence and (later) query/dashboard.
//
// Heartbeat still updates devices.last_seen_at via inventory.
// Violations are stored in device_events (00005_events.sql).
