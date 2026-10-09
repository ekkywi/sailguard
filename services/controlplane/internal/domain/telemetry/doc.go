package telemetry

// Package telemetry: agent event ingest, persistence, dedupe, heartbeats.
//
// MVP heartbeat path lives in inventory/httpapi for now
// (POST /v1/agent/events → ApplyHeartbeat). Full event store + Streams = M4.
