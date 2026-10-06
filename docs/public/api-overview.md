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
| Agents | Bearer device credential issued at enrollment (planned) |

Administrative APIs are further protected by role-based permissions as routes come online. Agent credentials must not call administrative routes.

## Capability areas

### Identity & access

- **Available now:** login; current-user profile with roles and permission codes (`/auth/me`); admin web console sign-in against these endpoints
- **Planned:** password change, logout/session revoke (server-side), user administration and role assignment APIs

### Inventory

Devices, groups, membership, enrollment tokens, and credential rotation.

### Policy

Policies, rules, assignments, categories, device overrides, and publish.

### Agent channel

- Enrollment  
- Effective policy download  
- Event batch upload (heartbeat and violations)

### Telemetry & operations

Event query, alerts, dashboard summaries, administrative audit logs, health endpoints, and alert-channel configuration.

## Health

- `GET /v1/health` — process liveness  
- `GET /v1/ready` — dependency readiness (`data.db` and `data.redis` booleans; unavailable dependencies return HTTP 503)

## Notes for integrators

- Prefer TLS everywhere in production deployments  
- Treat enrollment tokens and JWT signing secrets as secrets; rotate when compromised  
- Prefer short-lived access tokens; do not log raw Bearer tokens  
- Agent event submissions should be batched and idempotent via client-generated event IDs  

The exhaustive route and permission matrix used during implementation lives in internal engineering docs and is intentionally not duplicated here in full detail.
