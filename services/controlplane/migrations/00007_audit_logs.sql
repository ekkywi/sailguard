-- +goose Up

CREATE TABLE admin_audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID REFERENCES users (id) ON DELETE SET NULL,
    actor_email     TEXT NOT NULL DEFAULT '',
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL DEFAULT '',
    resource_id     TEXT NOT NULL DEFAULT '',
    summary         TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_logs_created_at_idx
    ON admin_audit_logs (created_at DESC);

CREATE INDEX admin_audit_logs_action_idx
    ON admin_audit_logs (action);

CREATE INDEX admin_audit_logs_resource_idx
    ON admin_audit_logs (resource_type, resource_id);

-- +goose Down

DROP TABLE IF EXISTS admin_audit_logs;