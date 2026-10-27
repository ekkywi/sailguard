-- Development-only lab users for RBAC smoke (one per role).
-- DO NOT run in production.
-- Password (plain) for all: ChangeMe!SailGuard1

INSERT INTO users (email, name, password_hash, auth_provider, is_active)
VALUES
    (
        'secadmin@sailguard.local',
        'Lab Security Admin',
        '$2y$05$BCTMIaoWp4GSa6utBkQKu.kxdxWV3ee7AzmflB7Bv04Q.wdcf7CGa',
        'local',
        true
    ),
    (
        'itoperator@sailguard.local',
        'Lab IT Operator',
        '$2y$05$BCTMIaoWp4GSa6utBkQKu.kxdxWV3ee7AzmflB7Bv04Q.wdcf7CGa',
        'local',
        true
    ),
    (
        'auditor@sailguard.local',
        'Lab Auditor',
        '$2y$05$BCTMIaoWp4GSa6utBkQKu.kxdxWV3ee7AzmflB7Bv04Q.wdcf7CGa',
        'local',
        true
    ),
    (
        'viewer@sailguard.local',
        'Lab Viewer',
        '$2y$05$BCTMIaoWp4GSa6utBkQKu.kxdxWV3ee7AzmflB7Bv04Q.wdcf7CGa',
        'local',
        true
    )
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'security_admin'
WHERE u.email = 'secadmin@sailguard.local'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'it_operator'
WHERE u.email = 'itoperator@sailguard.local'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'auditor'
WHERE u.email = 'auditor@sailguard.local'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'viewer'
WHERE u.email = 'viewer@sailguard.local'
ON CONFLICT DO NOTHING;