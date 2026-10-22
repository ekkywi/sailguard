package queue

// Package queue wraps Redis Streams produce/consume for events and alerts.
//
// Violation ingest (Phase A): API Produce on sg:events (group sg-workers);
// worker DecodeViolationMessage → telemetry.InsertViolation → Ack.
// Alerts / retention jobs remain later milestones.
