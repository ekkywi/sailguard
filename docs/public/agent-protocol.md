# Agent protocol

This document describes the agent ↔ control plane contract at a product level. Field-level examples for implementers are maintained in engineering docs.

**Platform floor:** Windows **7 SP1 or later** and Linux. Windows XP is not supported. See [Deployment overview](deployment-overview.md) for the support matrix.

## Lifecycle

1. **Enroll** with a one-time (or limited-use) enrollment token via `POST /v1/agent/enroll` *(admins mint/revoke tokens in the console or API)*. The agent sends a stable **machine ID** (Windows Cryptography `MachineGuid`, Linux `/etc/machine-id`) and OS version. If that machine ID already exists, a valid token **reclaims** the device (credential rotation) instead of failing with “already enrolled”. Windows endpoints may use an admin **installer package** (ZIP) from a chosen channel: **windows-legacy** (Win7 SP1+, Go 1.20) or **windows-current** (Win10/11, current Go) — `sailguard-agent-setup.exe` installs to Program Files / ProgramData; portable `start.bat` / `start-admin.bat` remain as a fallback. Linux agents typically use `go run` / local binary + env or `config.json` until a download channel ships.

2. Receive a unique **device credential** and recommended sync intervals; persist locally on the agent (`device.json` under the configured data directory)  
3. Authenticate subsequent calls with `Authorization: Bearer <device_token>` *(whoami / events / policy)*  
4. **Submit event batches** via `POST /v1/agent/events` — **heartbeat** refreshes `last_seen_at` and soft-updates hostname/OS/agent version; **`violation_detected`** / **`violation_blocked`** are persisted (including process owner when available) and listed for admins (`GET /v1/events` / console Events)  
5. **Poll / pull effective policy** via `GET /v1/agent/policy` (optional `?hash=` → `unchanged`); replace local cache atomically when content changes  
6. **Observe processes** and evaluate against cached policy:  
   - **Enumerate** via Linux `/proc` or Windows Toolhelp (process owner attached when the OS allows)  
   - **Fast path (Windows):** WMI `__InstanceCreationEvent` for new processes; if WMI is unavailable, fall back to Toolhelp PID-diff (~0.5s)  
   - **Reconciliation:** periodic full scan (default ~3s; `SG_SCAN_INTERVAL`) catches anything the watch missed  
   - **audit** → report `violation_detected`; **enforce** → terminate (Level 1) and report `violation_blocked`  
7. **Heartbeat and policy sync** run on a separate slower timer (default ~30s; `SG_HEARTBEAT_INTERVAL`) — not on the enforce path  
8. Refresh when the server returns a newer version/hash — supported via conditional sync above

## Transport expectations

- HTTPS/TLS in production  
- Device authentication on all post-enroll calls  
- JSON request/response bodies with an explicit `schema_version` for evolution  

## Policy document (effective)

Agents receive a flattened policy containing:

- Schema and policy **version**  
- Content **hash**  
- **Mode**: `audit` or `enforce`  
- **Rules**: concrete matchers only (no unresolved categories)

Agents should replace local policy atomically after validating the payload.

## Events

Supported event classes include:

| Type | Purpose |
|------|---------|
| `heartbeat` | Liveness, agent/OS metadata, current policy version |
| `violation_detected` | Policy match while in audit; stored and listed for administrators (may include process `user_name`) |
| `violation_blocked` | Match under enforce, including enforcement outcome (`terminated` / `terminate_failed`); may include process `user_name` |
| `agent_error` | Agent-side failures worth surfacing to operators (planned persistence) |

Batches include per-event client IDs to support idempotent ingestion. Servers may hint that policy reload is recommended in the batch response.

## Offline behavior

Agents retain a local policy cache and may queue outbound events to durable local storage when the control plane is unreachable, then flush when connectivity returns (subject to size limits).

## Safety constraints

- Agents must not provide a general-purpose remote command channel  
- Policy payloads are structured rules, not shell scripts  
- Built-in protections prevent termination of critical OS processes even if misconfigured remotely  
