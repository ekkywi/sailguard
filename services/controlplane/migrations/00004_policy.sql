-- +goose Up

-- Catalog: business categories (expanded to signatures before agent sync)
CREATE TABLE categories (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE category_signatures (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_code TEXT NOT NULL REFERENCES categories (code) ON DELETE CASCADE,
    rule_type     TEXT NOT NULL,
    operator      TEXT NOT NULL DEFAULT 'equals',
    value         TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT category_signatures_rule_type_check
        CHECK (rule_type IN (
            'process_name', 'path', 'hash_sha256', 'publisher', 'product_name'
        )),
    CONSTRAINT category_signatures_operator_check
        CHECK (operator IN ('equals', 'prefix', 'contains'))
);

CREATE INDEX category_signatures_category_code_idx
    ON category_signatures (category_code);

-- Policies (draft edited in place; version bumps on publish)
CREATE TABLE policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    mode        TEXT NOT NULL DEFAULT 'audit',
    priority    INT NOT NULL DEFAULT 100,
    version     BIGINT NOT NULL DEFAULT 0,
    published_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT policies_mode_check
        CHECK (mode IN ('audit', 'enforce')),
    CONSTRAINT policies_name_nonempty_check
        CHECK (length(trim(name)) > 0)
);

CREATE TRIGGER policies_set_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE policy_rules (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id     UUID NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    action        TEXT NOT NULL,
    rule_type     TEXT NOT NULL,
    operator      TEXT NOT NULL DEFAULT 'equals',
    value         TEXT NOT NULL,
    category_code TEXT REFERENCES categories (code) ON DELETE RESTRICT,
    enabled       BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT policy_rules_action_check
        CHECK (action IN ('allow', 'block')),
    CONSTRAINT policy_rules_rule_type_check
        CHECK (rule_type IN (
            'process_name', 'path', 'hash_sha256',
            'publisher', 'product_name', 'category'
        )),
    CONSTRAINT policy_rules_operator_check
        CHECK (operator IN ('equals', 'prefix', 'contains')),
    CONSTRAINT policy_rules_category_value_check
        CHECK (
            (rule_type = 'category' AND category_code IS NOT NULL)
            OR (rule_type <> 'category' AND category_code IS NULL AND length(trim(value)) > 0)
        )
);

CREATE INDEX policy_rules_policy_id_idx ON policy_rules (policy_id);

CREATE TRIGGER policy_rules_set_updated_at
    BEFORE UPDATE ON policy_rules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Assignments: org | group | device
CREATE TABLE policy_assignments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id  UUID NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    scope      TEXT NOT NULL,
    group_id   UUID REFERENCES device_groups (id) ON DELETE CASCADE,
    device_id  UUID REFERENCES devices (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT policy_assignments_scope_check
        CHECK (scope IN ('org', 'group', 'device')),
    CONSTRAINT policy_assignments_target_check
        CHECK (
            (scope = 'org' AND group_id IS NULL AND device_id IS NULL)
            OR (scope = 'group' AND group_id IS NOT NULL AND device_id IS NULL)
            OR (scope = 'device' AND device_id IS NOT NULL AND group_id IS NULL)
        )
);

CREATE INDEX policy_assignments_policy_id_idx ON policy_assignments (policy_id);
CREATE INDEX policy_assignments_group_id_idx ON policy_assignments (group_id);
CREATE INDEX policy_assignments_device_id_idx ON policy_assignments (device_id);

-- Time-bounded device overrides (highest merge weight)
CREATE TABLE device_overrides (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id  UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    rule_type  TEXT NOT NULL,
    operator   TEXT NOT NULL DEFAULT 'equals',
    value      TEXT NOT NULL,
    reason     TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT device_overrides_action_check
        CHECK (action IN ('allow', 'block')),
    CONSTRAINT device_overrides_rule_type_check
        CHECK (rule_type IN (
            'process_name', 'path', 'hash_sha256', 'publisher', 'product_name'
        )),
    CONSTRAINT device_overrides_operator_check
        CHECK (operator IN ('equals', 'prefix', 'contains'))
);

CREATE INDEX device_overrides_device_id_idx ON device_overrides (device_id);

CREATE TRIGGER device_overrides_set_updated_at
    BEFORE UPDATE ON device_overrides
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Minimal category seeds (signatures can be added later via API)
INSERT INTO categories (code, name, description) VALUES
    ('messaging', 'Messaging', 'Consumer chat / messaging apps'),
    ('p2p', 'P2P', 'Peer-to-peer file sharing'),
    ('game', 'Games', 'Games and game launchers'),
    ('remote_desktop', 'Remote desktop', 'Remote access tools')
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS device_overrides;
DROP TABLE IF EXISTS policy_assignments;
DROP TABLE IF EXISTS policy_rules;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS category_signatures;
DROP TABLE IF EXISTS categories;