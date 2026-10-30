-- +goose Up

ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS primary_ip TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ip_reported_at TIMESTAMPTZ NULL;

-- +goose Down

ALTER TABLE devices
    DROP COLUMN IF EXISTS ip_reported_at,
    DROP COLUMN IF EXISTS primary_ip;
