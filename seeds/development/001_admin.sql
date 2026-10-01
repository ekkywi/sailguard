-- Development-only bootstrap admin.
-- DO NOT run in production.
-- Password (plain): ChangeMe!SailGuard1
-- Change after first login.

INSERT INTO users (email, name, password_hash, auth_provider, is_active)
VALUES (
    'admin@sailguard.local',
    'SailGuard Admin',
    '$2y$05$BCTMIaoWp4GSa6utBkQKu.kxdxWV3ee7AzmflB7Bv04Q.wdcf7CGa',
    'local',
    true
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'super_admin'
WHERE u.email = 'admin@sailguard.local'
ON CONFLICT DO NOTHING;