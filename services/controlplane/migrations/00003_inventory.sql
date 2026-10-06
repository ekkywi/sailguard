-- +goose Up
-- device groups
CREATE TABLE device_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER device_groups_set_updated_at
    BEFORE UPDATE ON device_groups
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- devices
CREATE TABLE devices (
    id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname        TEXT NOT NULL DEFAULT '',
    display_name    TEXT NOT NULL DEFAULT '',
    os              TEXT NOT NULL DEFAULT 'windows',
    os_version      TEXT NOT NULL DEFAULT '',
    agent_version   TEXT NOT NULL DEFAULT '',
    machine_guid    TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    last_seen_at    TIMESTAMPTZ,
    enrolled_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT devices_os_check
        CHECK (os IN ('windows', 'linux', 'darwin')),
    CONSTRAINT devices_status_check
        CHECK (status IN ('pending', 'active', 'disabled'))
);

CREATE UNIQUE INDEX devices_machine_guid_uidx
    ON devices (machine_guid)
    WHERE machine_guid IS NOT NULL;

CREATE INDEX devices_status_idx ON devices (status);
CREATE INDEX devices_last_seen_at_idx ON devices (last_seen_at);

CREATE TRIGGER devices_set_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- device group members
CREATE TABLE device_group_members (
    group_id UUID NOT NULL REFERENCES device_groups (id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, device_id)
);

CREATE INDEX device_group_members_device_id_idx ON device_group_members (device_id);

-- enrollment tokens
CREATE TABLE enrollment_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label       TEXT NOT NULL DEFAULT '',
    token_hash  TEXT NOT NULL,
    max_uses    INT NOT NULL DEFAULT 1,
    use_count   INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    created_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT enrollment_tokens_max_uses_check CHECK (max_uses > 0),
    CONSTRAINT enrollment_tokens_use_count_check check (use_count >= 0)
);

CREATE INDEX enrollment_tokens_revoked_at_idx ON enrollment_tokens (revoked_at);

-- device credentials
CREATE TABLE device_credentials (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at  TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX device_credentials_active_uidx
    ON device_credentials (device_id)
    WHERE revoked_at IS NULL;
    
-- +goose Down
-- DROP TABLE
DROP TABLE IF EXISTS device_credentials;
DROP TABLE IF EXISTS enrollment_tokens;
DROP TABLE IF EXISTS device_group_members;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS device_groups;
