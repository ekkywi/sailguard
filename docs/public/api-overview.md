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

- **Available now:** login; current-user profile with roles and permission codes (`/auth/me`); **users admin** (`GET/POST /v1/users`, `GET/PATCH /v1/users/{id}`, `POST …/reset-password`, `GET /v1/roles`, `GET /v1/permissions`; web `/users` gated by `user.read` / `user.write`). Admin console also hides nav links and write controls from `/auth/me` permission codes (API `requirePermission` remains authoritative).
- **Planned:** password change, logout/session revoke (server-side), LDAP/AD, MFA

### Inventory

- **Available now:** list/create/get/**PATCH**/**DELETE** device groups; list/add/remove group members; list/get/**PATCH**/**DELETE** devices (`display_name`, `status` active|disabled); create/list/revoke enrollment tokens (admin Bearer + `device.read` / `device.write`). Create enrollment token returns the plaintext secret once. Admin web console covers Devices and Groups detail actions, plus Tokens pages. Agent auth rejects `disabled` devices with HTTP **403** `device_disabled`; enroll reclaim of a disabled device is also refused (same code) so a token cannot re-enable it.
- **Available now (agent):** on HTTP **401** (invalid/revoked device token), if `enrollment_token` is still in agent config, the agent reclaims via enroll and refreshes `device.json` (cooldown ~1m). **403** `device_disabled` never triggers re-enroll.
- **Planned:** dedicated admin rotate-credential API

### Policy

- **Available now:** list/create/get/**DELETE** policies; **PATCH** mode (`audit`/`enforce`); list/add/**PATCH**/delete rules (including `rule_type=category`); categories CRUD (`GET/POST /v1/categories`, `GET/PATCH/DELETE /v1/categories/{code}`) and signatures (`GET/POST …/signatures`, `DELETE …/signatures/{id}`); list/add/remove assignments; publish (bumps version); effective-policy preview `GET /v1/devices/{id}/policy`; **device overrides** (`GET/POST /v1/devices/{id}/overrides`, `DELETE …/overrides/{overrideId}`; `allow` requires future `expires_at`; list defaults to active only, `?include_expired=true` for history). Admin web: `/policies`, `/policies/:id`, `/categories`, `/categories/:code`, device detail tab `?tab=overrides`.
- **Available now (admin audit):** `GET /v1/audit-logs` (`audit.read`) lists `admin_audit_logs` with filters (`family`, `from`/`to` RFC3339, `actor`, `q`, `resource_type`, `limit`/`offset`) and `total`. `GET /v1/audit-logs/export` downloads CSV (same filters, max 10k rows). Coverage: policy CRUD/rules/assignments/publish; categories + signatures; overrides; users; devices; enrollment tokens; groups + membership; agent package download (channel/kind only — no enrollment token); login success/failure. Secrets never logged. Web console `/audit` (family chips, date range, search, row detail, CSV export).

### Agent channel

- **Available now:** enrollment (`POST /v1/agent/enroll`) including **reclaim** when `machine_guid` already exists (valid enrollment token rotates device credential; **refuses** if device `status=disabled` with `403` `device_disabled`); device auth (`GET /v1/agent/whoami`, events, policy) rejects disabled devices the same way; event batch (`POST /v1/agent/events`) accepts **heartbeat**, **`violation_detected`**, and **`violation_blocked`** (persisted to `device_events`, idempotent by `client_event_id`; violation payloads may include process `user_name`); effective policy download (`GET /v1/agent/policy`, optional `?hash=` for unchanged). Agents report stable machine IDs (Windows MachineGuid / Linux machine-id) and OS version strings. Agents evaluate cached policy: Linux `/proc` + Windows Toolhelp; Windows watch prefers WMI process-creation events with PID-diff + periodic scan fallbacks; enforce mode terminates matched processes (Level 1). While offline, agents keep enforcing from cache and queue failed violation uploads to local `events-queue.json`, then flush on reconnect (idempotent `client_event_id`). Admin **Download agent** packages: `GET/POST /v1/agent-packages/windows` (channels `windows-legacy` / `windows-current` / `linux` placeholder; `package_kind` `install`|`update` — install requires `enrollment_token`, update omits it; ZIP with `sailguard-agent-setup.exe`, channel agent exe, `config.json`, portable start scripts; `SG_AGENT_WIN7_PATH` / `SG_AGENT_WINDOWS_PATH` / `SG_AGENT_SETUP_PATH`).
- **Planned:** ETW-based watch refinements; full async telemetry pipeline (streams/worker); Linux download channel

### Telemetry & operations

- **Available now:** health/ready; agents submit `violation_detected` / `violation_blocked` (persisted, including optional process user); admin `GET /v1/events`; web console **Events** page (`/events`, User column); fleet **dashboard summary** `GET /v1/dashboard/summary` (`event.read` — device online/stale within online window, violations in last 24h, recent violation rows, top processes/hosts 24h, stale device list; Home overview); **violations timeseries** `GET /v1/dashboard/violations-timeseries?range=24h|7d` (filled hourly/daily buckets; Home chart)
- **Planned:** event detail filters, alerts, alert-channel configuration, async stream pipeline

## Health

- `GET /v1/health` — process liveness  
- `GET /v1/ready` — dependency readiness (`data.db` and `data.redis` booleans; unavailable dependencies return HTTP 503)

## Notes for integrators

- Prefer TLS everywhere in production deployments  
- Treat enrollment tokens and JWT signing secrets as secrets; rotate when compromised  
- Prefer short-lived access tokens; do not log raw Bearer tokens  
- Agent event submissions should be batched and idempotent via client-generated event IDs  

The exhaustive route and permission matrix used during implementation lives in internal engineering docs and is intentionally not duplicated here in full detail.
