package queue

// Package queue wraps Redis Streams produce/consume for events and alerts.
//
// Violations: API Produce on sg:events (group sg-workers);
// worker DecodeViolationMessage → InsertViolation → Ack.
//
// Alerts: worker Produce on sg:alerts (group sg-alert-workers) after enqueueing
// alert_deliveries; alert consumer DecodeAlertMessage → webhook → Ack.
