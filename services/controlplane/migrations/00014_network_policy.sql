-- +goose Up

CREATE TABLE network_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    mode            TEXT NOT NULL DEFAULT 'audit',
    priority        INT NOT NULL DEFAULT 100,
    version         BIGINT NOT NULL DEFAULT 0,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT network_policies_mode_check
        CHECK (mode IN ('audit', 'enforce')),
    CONSTRAINT network_policies_name_nonempty_check
        CHECK (length(trim(name)) > 0)
);

CREATE TRIGGER network_policies_set_updated_at
    BEFORE UPDATE ON network_policies
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE network_policy_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id   UUID NOT NULL REFERENCES network_policies (id) ON DELETE CASCADE,
    action      TEXT NOT NULL,
    dest_type   TEXT NOT NULL,
    dest_value  TEXT NOT NULL,
    port        INT,
    label       TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT network_policy_rules_action_check
        CHECK (action IN ('allow', 'deny')),
    CONSTRAINT network_policy_rules_dest_type_check
        CHECK (dest_type IN ('ip', 'cidr')),
    CONSTRAINT network_policy_rules_dest_value_check
        CHECK (length(trim(dest_value)) > 0),
    CONSTRAINT network_policy_rules_port_check
    CHECK (port IS NULL OR (port >= 1 AND port <= 65535))
);

CREATE INDEX network_policy_rules_policy_id_idx
    ON network_policy_rules (policy_id);

CREATE TRIGGER network_policy_rules_set_updated_at
    BEFORE UPDATE ON network_policy_rules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE network_policy_assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id   UUID NOT NULL REFERENCES network_policies (id) ON DELETE CASCADE,
    scope       TEXT NOT NULL,
    group_id    UUID REFERENCES device_groups (id) ON DELETE CASCADE,
    device_id   UUID REFERENCES devices (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT network_policy_assignments_policy_id_scope_check
        CHECK (scope IN ('org', 'group', 'device')),
    CONSTRAINT network_policy_assignments_target_check
        CHECK (
            (scope = 'org' AND group_id IS NULL AND device_id IS NULL)
            OR (scope = 'group' AND group_id IS NOT NULL AND device_id IS NULL)
            OR (scope = 'device' AND device_id IS NOT NULL AND group_id IS NULL)
        )
);

CREATE INDEX network_policy_assignments_policy_id_idx
    ON network_policy_assignments (policy_id);
CREATE INDEX network_policy_assignments_group_id_idx
    ON network_policy_assignments (group_id);
CREATE INDEX network_policy_assignments_device_id_idx
    ON network_policy_assignments (device_id);

CREATE TABLE device_network_overrides (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    action      TEXT NOT NULL,
    dest_type   TEXT NOT NULL,
    dest_value  TEXT NOT NULL,
    port        INT,
    label       TEXT NOT NULL DEFAULT '',
    reason      TEXT NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT device_network_overrides_action_check
        CHECK (action IN ('allow', 'deny')),
    CONSTRAINT device_network_overrides_dest_type_check
        CHECK (dest_type IN ('ip', 'cidr')),
    CONSTRAINT device_network_overrides_dest_value_check
        CHECK (length(trim(dest_value)) > 0),
    CONSTRAINT device_network_overrides_port_check
    CHECK (port IS NULL OR (port >= 1 AND port <= 65535))
);

CREATE INDEX device_network_overrides_device_id_idx
    ON device_network_overrides (device_id);

CREATE TRIGGER device_network_overrides_set_updated_at
    BEFORE UPDATE ON device_network_overrides
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

INSERT INTO permissions (code, description) VALUES
    ('network.read', 'View network policies and assignments'),
    ('network.write', 'Edit and publish network policies / overrides')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('network.read', 'network.write')
WHERE r.code = 'security_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('network.read')
WHERE r.code IN ('it_operator', 'auditor')
ON CONFLICT DO NOTHING;

-- +goose Down

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('network.read', 'network.write')
);

DELETE FROM permissions
WHERE code IN ('network.read', 'network.write');

DROP TABLE IF EXISTS device_network_overrides;
DROP TABLE IF EXISTS network_policy_assignments;
DROP TABLE IF EXISTS network_policy_rules;
DROP TABLE IF EXISTS network_policies;