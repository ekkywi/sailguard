package audit

// Package audit: append-only admin audit logs.
// Table: admin_audit_logs (00007_audit_logs.sql).
//
// Writers (Batches 1–4): policy.*, category.*, override.*, user.*,
// device.*, enrollment_token.*, group.*, agent_package.download,
// auth.login / auth.login.fail.
//
// List/Export: ListFilter (family prefix, time range, actor, q, resource_type)
// with limit/offset; CSV export capped at 10k rows.
