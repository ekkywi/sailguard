-- +goose Up

INSERT INTO permissions (code, description) VALUES
    ('user.read', 'View users and roles'),
    ('user.write', 'Create and update users'),
    ('device.read', 'View devices and groups'),
    ('device.write', 'Manage devices, groups, enrollment'),
    ('policy.read', 'View policies and assignments'),
    ('policy.write', 'Edit and publish policies / overrides'),
    ('event.read', 'View events and dashboards'),
    ('alert.read', 'View alerts'),
    ('alert.manage', 'Retry and manage alerts'),
    ('audit.read', 'View admin audit logs'),
    ('system.read', 'View system settings'),
    ('system.write', 'Change system settings')
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (code, name, description) VALUES
    ('super_admin', 'Super Admin', 'Full access'),
    ('security_admin', 'Security Admin', 'Policies, alerts, read devices/events'),
    ('it_operator', 'IT Operator', 'Devices/groups and operational read'),
    ('auditor', 'Auditor', 'Read-only including audit logs'),
    ('viewer', 'Viewer', 'Limited read of devices and events')
ON CONFLICT (code) DO NOTHING;

-- super_admin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

-- security_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'policy.read', 'policy.write',
    'event.read',
    'alert.read', 'alert.manage',
    'device.read',
    'audit.read'
)
WHERE r.code = 'security_admin'
ON CONFLICT DO NOTHING;

-- it_operator
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'device.read', 'device.write',
    'event.read',
    'policy.read'
)
WHERE r.code = 'it_operator'
ON CONFLICT DO NOTHING;

-- auditor
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'user.read',
    'device.read',
    'policy.read',
    'event.read',
    'alert.read',
    'audit.read',
    'system.read'
)
WHERE r.code = 'auditor'
ON CONFLICT DO NOTHING;

-- viewer
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'device.read',
    'event.read'
)
WHERE r.code = 'viewer'
ON CONFLICT DO NOTHING;

-- +goose Down

DELETE FROM role_permissions;

DELETE FROM permissions
WHERE code IN (
    'user.read', 'user.write',
    'device.read', 'device.write',
    'policy.read', 'policy.write',
    'event.read',
    'alert.read', 'alert.manage',
    'audit.read',
    'system.read', 'system.write'
);

DELETE FROM roles
WHERE code IN (
    'super_admin',
    'security_admin',
    'it_operator',
    'auditor',
    'viewer'
);
