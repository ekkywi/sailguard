-- +goose Up

-- One row per (channel, event) delivery attempt lifecycle.
CREATE TABLE alert_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES alert_channels (id) ON DELETE CASCADE,
    event_id        UUID NOT NULL REFERENCES device_events (id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending',
    attempts        INT  NOT NULL DEFAULT 0,
    max_attempts    INT  NOT NULL DEFAULT 5,
    last_error      TEXT,
    next_retry_at   TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT alert_deliveries_status_check
        CHECK (status IN ('pending', 'delivered', 'failed', 'dead')),

    -- Same violation should not create duplicate open deliveries per channel.
    CONSTRAINT alert_deliveries_channel_event_unique
        UNIQUE (channel_id, event_id)
);

CREATE INDEX alert_deliveries_status_retry_idx
    ON alert_deliveries (status, next_retry_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX alert_deliveries_event_id_idx
    ON alert_deliveries (event_id);

-- +goose Down

DROP TABLE IF EXISTS alert_deliveries;