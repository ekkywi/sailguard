package queue

// Package queue wraps Redis Streams produce/consume for events and alerts.
// M1 skeleton: Produce / EnsureGroup / ReadGroup / Ack on sg:events (group sg-workers).
// Persist-to-Postgres and alert pipelines land in later milestones (M4+).
