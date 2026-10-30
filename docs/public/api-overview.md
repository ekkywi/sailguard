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
| Login | `POST /v1/auth/login` | JSON body: `email`, `password` → `access_token` (Bearer), `expires_in`, basic `user`. Issues a server session (`jti`) in Redis (TTL = `SG_ADMIN_TOKEN_TTL`, default 8h). |
| Current user | `GET /v1/auth/me` | Header: `Authorization: Bearer <access_token>`; token must have a live Redis session |
| Logout | `POST /v1/auth/logout` | Revokes the current session (`jti`); subsequent calls with that token return 401 |

Tokens are signed access tokens (HMAC) bound to a revocable server session. The admin console also applies a **10-minute idle** logout in the browser. Directory federation (for example LDAP/AD) may be added later without changing the Bearer pattern for API clients.

| Client | Mechanism |
|--------|-----------|
| Administrators | Bearer access token after local login |
| Agents | Bearer device credential issued at enrollment |

Administrative APIs are further protected by role-based permissions as routes come online. Agent credentials must not call administrative routes.

## Capability areas

### Identity & access

- **Available now:** login; current-user profile with roles and permission codes (`/auth/me`); **users admin** (`GET/POST /v1/users`, `GET/PATCH /v1/users/{id}`, `POST …/reset-password`, `GET /v1/roles`, `GET /v1/permissions`; web `/users` gated by `user.read` / `user.write`). Admin console also hides nav links and write controls from `/auth/me` permission codes (API `requirePermission` remains authoritative).
- **Available now (sessions):** configurable admin token TTL (`SG_ADMIN_TOKEN_TTL`); Redis-backed session revoke on logout; console idle timeout (10m) and 401 → re-login.
- **Planned:** password change, session list / revoke-all UI, LDAP/AD, MFA

### Inventory

- **Available now:** list/create/get/**PATCH**/**DELETE** device groups; list/add/remove group members; list/get/**PATCH**/**DELETE** devices (`display_name`, `status` active|disabled, plus asset fields `pic`, `location`, `asset_tag`, `notes`). Device list supports filters `status`, `os`, search `q` (hostname/PIC/location/asset tag/…), and `limit`/`offset` with `total`. Devices also expose agent-reported hardware inventory (`cpu_model`, `cpu_cores`, `ram_mb`, `disk_gb`, `hw_manufacturer`, `hw_model`, `hardware_reported_at`) updated from heartbeat — not admin PATCH. Create/list/revoke enrollment tokens (admin Bearer + `device.read` / `device.write`). Create enrollment token returns the plaintext secret once. Admin web console covers Devices (filter/search/pagination; asset edit modal; Hardware panel on detail) and Groups detail actions, plus Tokens pages. Agent auth rejects `disabled` devices with HTTP **403** `device_disabled`; enroll reclaim of a disabled device is also refused (same code) so a token cannot re-enable it. Device `status` is control-plane activation (not connectivity); connectivity still uses `last_seen_at`.
- **Available now (agent):** on HTTP **401** (invalid/revoked device token), if `enrollment_token` is still in agent config, the agent reclaims via enroll and refreshes `device.json` (cooldown ~1m). **403** `device_disabled` never triggers re-enroll.
- **Planned:** dedicated admin rotate-credential API

### Policy

- **Available now:** list/create/get/**DELETE** policies; **PATCH** mode (`audit`/`enforce`); list/add/**PATCH**/delete rules (including `rule_type=category`); categories CRUD (`GET/POST /v1/categories`, `GET/PATCH/DELETE /v1/categories/{code}`) and signatures (`GET/POST …/signatures`, `DELETE …/signatures/{id}`); list/add/remove assignments; publish (bumps version); effective-policy preview `GET /v1/devices/{id}/policy`; **device overrides** (`GET/POST /v1/devices/{id}/overrides`, `DELETE …/overrides/{overrideId}`; `allow` requires future `expires_at`; list defaults to active only, `?include_expired=true` for history). Admin web: `/policies`, `/policies/:id`, `/categories`, `/categories/:code`, device detail tab `?tab=overrides`.
- **Available now (admin audit):** `GET /v1/audit-logs` (`audit.read`) lists `admin_audit_logs` with filters (`family`, `from`/`to` RFC3339, `actor`, `q`, `resource_type`, `limit`/`offset`) and `total`. `GET /v1/audit-logs/export` downloads CSV (same filters, max 10k rows). Coverage: policy CRUD/rules/assignments/publish; categories + signatures; overrides; users; devices; enrollment tokens; groups + membership; agent package download (channel/kind only — no enrollment token); login success/failure. Secrets never logged. Web console `/audit` (family chips, date range, search, row detail, CSV export).

### Agent channel

- **Available now:** enrollment (`POST /v1/agent/enroll`) including **reclaim** when `machine_guid` already exists (valid enrollment token rotates device credential; **refuses** if device `status=disabled` with `403` `device_disabled`); device auth (`GET /v1/agent/whoami`, events, policy) rejects disabled devices the same way; event batch (`POST /v1/agent/events`) accepts **heartbeat** (updates `last_seen_at` synchronously) and **`violation_detected`** / **`violation_blocked`** (enqueued to Redis Stream `sg:events`; the **worker** persists to `device_events`, idempotent by `client_event_id`; violation payloads may include process `user_name` — the worker process must be running or violations will not appear in admin Events); effective policy download (`GET /v1/agent/policy`, optional `?hash=` for unchanged). Agents report stable machine IDs (Windows MachineGuid / Linux machine-id) and OS version strings. Agents evaluate cached policy: Linux `/proc` + Windows Toolhelp; Windows watch prefers WMI process-creation events with PID-diff + periodic scan fallbacks; enforce mode terminates matched processes (Level 1). While offline, agents keep enforcing from cache and queue failed violation uploads to local `events-queue.json`, then flush on reconnect (idempotent `client_event_id`). Admin **Download agent** packages: `GET/POST /v1/agent-packages/windows` (channels `windows-legacy` / `windows-current` / `linux`; `package_kind` `install`|`update` — install requires `enrollment_token`, update omits it; Windows ZIP includes `sailguard-agent-setup.exe` + agent exe + scripts; Linux ZIP includes `sailguard-agent`, `install.sh`, `sailguard-agent.service`, `config.json`; paths `SG_AGENT_WIN7_PATH` / `SG_AGENT_WINDOWS_PATH` / `SG_AGENT_SETUP_PATH` / `SG_AGENT_LINUX_PATH`).
- **Available now (ingest limits):** `POST /v1/agent/enroll` is rate-limited per client IP (default 10/min, `SG_RATE_ENROLL_PER_MIN`); `POST /v1/agent/events` is rate-limited per device after auth (default 120/min, `SG_RATE_EVENTS_PER_MIN`). Over limit → HTTP **429** `rate_limited` with `Retry-After`. Body size caps (`SG_ENROLL_MAX_BODY` / `SG_EVENTS_MAX_BODY`) and max events per batch (`SG_EVENTS_MAX_BATCH`, default 100) return **400** `validation_error`. Counters live in Redis (1-minute fixed window).
- **Available now (stream worker):** consumer group `sg-workers` on `sg:events`; in-process insert retries; **XAUTOCLAIM** reclaims PEL entries idle ≥60s so a crashed/stuck worker does not drop violations permanently; on **new** inserts the worker may POST a JSON webhook to enabled `alert_channels`; hourly (and on start) **event retention** deletes `device_events` older than `SG_EVENTS_RETENTION_DAYS` (default 180; `0` = off).
- **Planned:** ETW-based watch refinements; Linux process Watch (poll `/proc` today); richer alert fan-out (email/multiple channels)

### Telemetry & operations

- **Available now:** health/ready; agents submit `violation_detected` / `violation_blocked` (async via Redis Streams + worker persist, including optional process user); admin `GET /v1/events`; web console **Events** page (`/events`, User column); fleet **dashboard summary** `GET /v1/dashboard/summary` (`event.read` — device online/stale within online window, violations in last 24h, recent violation rows, top processes/hosts 24h, stale device list; Home overview); **violations timeseries** `GET /v1/dashboard/violations-timeseries?range=24h|7d` (filled hourly/daily buckets; Home chart)
- **Available now (events):** `GET /v1/events` supports filters (`event_type`, `device_id`, `from`/`to` RFC3339, `q`, `limit`/`offset`) and returns `total`; `GET /v1/events/export` downloads CSV (same filters, max 10k rows). Admin console Events page mirrors Audit-style chips, date range, search, and CSV export.
- **Available now (alerts):** `GET /v1/alert-channels` (`alert.read`); `PATCH /v1/alert-channels/{id}` and `POST /v1/alert-channels/{id}/test` (`alert.manage`) — v1 is a single webhook URL + enabled flag; console **Settings → Notifications** (`/settings/notifications`). Deliveries: worker enqueues `alert_deliveries` + Redis `sg:alerts` on new inserts; retries with backoff then `dead`; `GET /v1/alert-deliveries` (`alert.read`); `POST /v1/alert-deliveries/{id}/retry` (`alert.manage`)

## Health

- `GET /v1/health` — process liveness  
- `GET /v1/ready` — dependency readiness (`data.db` and `data.redis` booleans; unavailable dependencies return HTTP 503)

## Notes for integrators

- Prefer TLS everywhere in production deployments  
- Treat enrollment tokens and JWT signing secrets as secrets; rotate when compromised  
- Prefer short-lived access tokens; do not log raw Bearer tokens  
- Agent event submissions should be batched and idempotent via client-generated event IDs  
- Expect **429** on enroll/events when rate limits are exceeded; back off using `Retry-After` (and local agent queue for violations)  

The exhaustive route and permission matrix used during implementation lives in internal engineering docs and is intentionally not duplicated here in full detail.
