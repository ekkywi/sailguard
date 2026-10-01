# API overview

SailGuard exposes a versioned HTTP API under `/v1`. Responses use a consistent envelope:

```json
{ "ok": true, "data": { } }
```

```json
{ "ok": false, "error": { "code": "string", "message": "string" } }
```

## Authentication

| Client | Mechanism |
|--------|-----------|
| Administrators | Bearer access token after local login (directory federation may be added later) |
| Agents | Bearer device credential issued at enrollment |

Administrative APIs are further protected by role-based permissions. Agent credentials cannot call administrative routes.

## Capability areas

### Identity & access

Login, session/profile, password change, user administration, and role assignment.

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
- `GET /v1/ready` — dependency readiness (for orchestrators)

## Notes for integrators

- Prefer TLS everywhere in production deployments  
- Treat enrollment tokens as secrets; prefer short TTL and use limits  
- Agent event submissions should be batched and idempotent via client-generated event IDs  

The exhaustive route and permission matrix used during implementation lives in internal engineering docs and is intentionally not duplicated here in full detail.
