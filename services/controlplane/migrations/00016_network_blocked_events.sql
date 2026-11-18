-- +goose Up

ALTER TABLE device_events
    DROP CONSTRAINT IF EXISTS device_events_type_check;

ALTER TABLE device_events
    ADD CONSTRAINT device_events_type_check
    CHECK (event_type IN (
        'violation_detected',
        'violation_blocked',
        'network_blocked',
        'agent_error'
    ));

ALTER TABLE device_events
    ADD COLUMN IF NOT EXISTS network_dest_type  TEXT,
    ADD COLUMN IF NOT EXISTS network_dest_value TEXT,
    ADD COLUMN IF NOT EXISTS network_dest_ip    TEXT,
    ADD COLUMN IF NOT EXISTS network_dest_port  INT,
    ADD COLUMN IF NOT EXISTS network_protocol   TEXT,
    ADD COLUMN IF NOT EXISTS network_direction  TEXT;

-- +goose Down

ALTER TABLE device_events
    DROP COLUMN IF EXISTS network_direction,
    DROP COLUMN IF EXISTS network_protocol,
    DROP COLUMN IF EXISTS network_dest_port,
    DROP COLUMN IF EXISTS network_dest_ip,
    DROP COLUMN IF EXISTS network_dest_value,
    DROP COLUMN IF EXISTS network_dest_type;

ALTER TABLE device_events
    DROP CONSTRAINT IF EXISTS device_events_type_check;

ALTER TABLE device_events
    ADD CONSTRAINT device_events_type_check
    CHECK (event_type IN (
        'violation_detected',
        'violation_blocked',
        'agent_error'
    ));