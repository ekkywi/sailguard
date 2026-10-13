# API overview

SailGuard exposes a versioned HTTP API under `/v1`. Responses use a consistent envelope:

```json
{ "ok": true, "data": { } }
```

```json
{ "ok": false, "error": { "code": "string", "message": "string" } }
```

## Authentication (administrators)

| Step | Endpoint | Notes |
|------|----------|--------|
| Login | `POST /v1/auth/login` | JSON body: `email`, `password` → `access_token` (Bearer), `expires_in`, basic `user` |
| Current user | `GET /v1/auth/me` | Header: `Authorization: Bearer <access_token>` |

Tokens are signed access tokens (HMAC). Directory federation (for example LDAP/AD) may be added later without changing the Bearer pattern for API clients.

| Client | Mechanism |
|--------|-----------|
| Administrators | Bearer access token after local login |
| Agents | Bearer device credential issued at enrollment |

Administrative APIs are further protected by role-based permissions as routes come online. Agent credentials must not call administrative routes.

## Capability areas

### Identity & access

- **Available now:** login; current-user profile with roles and permission codes (`/auth/me`); admin web console sign-in against these endpoints
- **Planned:** password change, logout/session revoke (server-side), user administration and role assignment APIs

### Inventory

- **Available now:** list/create/get device groups; list/add/remove group members; list/get devices; create/list/revoke enrollment tokens (admin Bearer + `device.read` / `device.write`). Create enrollment token returns the plaintext secret once. Admin web console covers Devices, Groups (with membership), and Tokens pages.
- **Planned:** device PATCH, credential rotation

### Policy

- **Available now:** list/create/get policies; list/add rules; list/add/remove assignments; publish (bumps version); effective-policy preview `GET /v1/devices/{id}/policy`. Admin web: `/policies` and `/policies/:id`. Schema also includes categories and overrides.
- **Planned:** category signature admin, device override APIs, richer policy editor (edit/delete rules)

### Agent channel

- **Available now:** enrollment (`POST /v1/agent/enroll`); device auth (`GET /v1/agent/whoami`); event batch (`POST /v1/agent/events`) accepts **heartbeat** (updates `last_seen_at`) and **`violation_detected`** (persisted to `device_events`, idempotent by `client_event_id`); effective policy download (`GET /v1/agent/policy`, optional `?hash=` for unchanged). Agents evaluate cached policy in **audit** mode and report matches. On **Linux**, agents enumerate live processes via `/proc` (Windows Toolhelp enumeration planned next).
- **Planned:** Windows process enumeration; enforce-mode termination (`violation_blocked`) on Windows and Linux; full async telemetry pipeline (streams/worker)

### Telemetry & operations

- **Available now:** health/ready; agents submit `violation_detected` (persisted); admin `GET /v1/events`; web console **Events** page (`/events`)
- **Planned:** event detail filters, alerts, dashboard summaries, administrative audit logs, alert-channel configuration, async stream pipeline

## Health

- `GET /v1/health` — process liveness  
- `GET /v1/ready` — dependency readiness (`data.db` and `data.redis` booleans; unavailable dependencies return HTTP 503)

## Notes for integrators

- Prefer TLS everywhere in production deployments  
- Treat enrollment tokens and JWT signing secrets as secrets; rotate when compromised  
- Prefer short-lived access tokens; do not log raw Bearer tokens  
- Agent event submissions should be batched and idempotent via client-generated event IDs  

The exhaustive route and permission matrix used during implementation lives in internal engineering docs and is intentionally not duplicated here in full detail.
