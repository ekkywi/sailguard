-- +goose Up

-- Notifications channels (v1: webhook URL only).
CREATE TABLE alert_channels (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        TEXT NOT NULL DEFAULT 'webhook',
    name        TEXT NOT NULL DEFAULT 'Default webhook',
    enabled     BOOLEAN NOT NULL DEFAULT false,
    config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT  alert_channels_kind_check CHECK (kind IN ('webhook'))
);

-- One default row so the UI always has something to edit.
INSERT INTO alert_channels (kind, name, enabled, config)
VALUES (
    'webhook',
    'Default webhook',
    false,
    '{"url": ""}'::jsonb
);

-- +goose Down

DROP TABLE IF EXISTS alert_channels;