-- +goose Up

CREATE TABLE device_events (
    id UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID          NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    client_event_id         TEXT NOT NULL,
    event_type              TEXT NOT NULL,
    occurred_at             TIMESTAMPTZ NOT NULL,
    received_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    dedupe_key              TEXT,

    -- process (violation_*)
    process_pid             INT,
    process_name            TEXT,
    process_path            TEXT,
    process_hash_sha256     TEXT,
    process_publisher       TEXT,
    process_product_name    TEXT,
    process_user_name       TEXT,

    -- match
    match_rule_id           TEXT,
    match_action            TEXT,
    match_rule_type         TEXT,
    match_value             TEXT,
    match_category_code     TEXT,

    -- policy snapshot at report time
    policy_version          BIGINT,
    policy_mode             TEXT,

    CONSTRAINT device_events_type_check
        CHECK (event_type IN (
            'violation_detected',
            'violation_blocked',
            'agent_error'
        )),

    CONSTRAINT device_events_client_id_unique
        UNIQUE (device_id, client_event_id)
);

CREATE INDEX device_events_device_occurred_idx
    ON device_events (device_id, occurred_at DESC);

CREATE INDEX device_events_type_occurred_idx
    ON device_events (event_type, occurred_at DESC);

-- +goose Down

DROP TABLE IF EXISTS device_events;