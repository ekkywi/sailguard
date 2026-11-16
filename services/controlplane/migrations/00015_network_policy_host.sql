-- +goose Up

ALTER TABLE network_policy_rules
    DROP CONSTRAINT network_policy_rules_dest_type_check;
ALTER TABLE network_policy_rules
    ADD CONSTRAINT network_policy_rules_dest_type_check
    CHECK (dest_type IN ('ip', 'cidr', 'host'));

ALTER TABLE device_network_overrides
    DROP CONSTRAINT device_network_overrides_dest_type_check;
ALTER TABLE device_network_overrides
    ADD CONSTRAINT device_network_overrides_dest_type_check
    CHECK (dest_type IN ('ip', 'cidr', 'host'));

-- +goose Down
ALTER TABLE network_policy_rules
    DROP CONSTRAINT network_policy_rules_dest_type_check;
ALTER TABLE network_policy_rules
    ADD CONSTRAINT network_policy_rules_dest_type_check
    CHECK (dest_type IN ('ip', 'cidr'));

ALTER TABLE device_network_overrides
    DROP CONSTRAINT device_network_overrides_dest_type_check;
ALTER TABLE device_network_overrides
    AND CONSTRAINT device_network_overrides_dest_type_check
    CHECK (dest_type IN ('ip', 'cidr'));