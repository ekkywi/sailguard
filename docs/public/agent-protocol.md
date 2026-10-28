# Agent protocol

This document describes the agent ↔ control plane contract at a product level. Field-level examples for implementers are maintained in engineering docs.

**Platform floor:** Windows **7 SP1 or later** and Linux. Windows XP is not supported. See [Deployment overview](deployment-overview.md) for the support matrix.

## Lifecycle

1. **Enroll** with a one-time (or limited-use) enrollment token via `POST /v1/agent/enroll` *(admins mint/revoke tokens in the console or API)*. The agent sends a stable **machine ID** (Windows Cryptography `MachineGuid`, Linux `/etc/machine-id`) and OS version. If that machine ID already exists, a valid token **reclaims** the device (credential rotation) instead of failing with “already enrolled”. Reclaim is **refused** when the existing device `status` is `disabled` (`403` `device_disabled`) so enrollment cannot silently re-enable it. Administrators download endpoint packages from the console (`windows-legacy` / `windows-current` / `linux`): **first-install** packages bake an enrollment token into `config.json`; **update** packages omit the token and reuse the existing data-directory `device.json`. **Windows:** run `sailguard-agent-setup.exe` as Administrator — installs to Program Files / ProgramData and registers **SailGuardAgent** (with a scheduled-task / detached-process fallback when SCM start is blocked). **Linux:** unzip and run `sudo ./install.sh` — installs `/usr/local/bin/sailguard-agent`, writes `config.json` beside the binary, creates the data directory (default `/var/lib/sailguard/agent`), and enables `sailguard-agent.service`.

2. Receive a unique **device credential** and recommended sync intervals; persist locally on the agent (`device.json` under the configured data directory)  
3. Authenticate subsequent calls with `Authorization: Bearer <device_token>` *(whoami / events / policy)*. If an operator has set the device to **disabled**, these calls return **403** with `device_disabled`; the agent should log and wait for admin re-enable (it must **not** treat this as a cue to re-enroll). If the device token is **invalid/revoked** (**401** `unauthorized`), and a valid `enrollment_token` remains in local config, the agent may **re-enroll / reclaim** (same machine ID) to obtain a new device credential, then retry. A bad enrollment token returns **401** `enrollment_invalid` (distinct from device unauthorized) and does not rotate credentials.  
4. **Submit event batches** via `POST /v1/agent/events` — **heartbeat** refreshes `last_seen_at` and soft-updates hostname/OS/agent version; **`violation_detected`** / **`violation_blocked`** are accepted into the control-plane stream and persisted asynchronously by the worker (including process owner when available), then listed for admins (`GET /v1/events` / console Events)  
5. **Poll / pull effective policy** via `GET /v1/agent/policy` (optional `?hash=` → `unchanged`); replace local cache atomically when content changes  
6. **Observe processes** and evaluate against cached policy:  
   - **Enumerate** via Linux `/proc` or Windows Toolhelp (process owner attached when the OS allows)  
   - **Linux process name:** `/proc/<pid>/comm` truncates names longer than 15 characters; the agent prefers the executable basename from `cmdline` when `comm` looks truncated so `process_name` rules (for example `gnome-calculator`) match the full name. Path rules still use the executable path from `cmdline`  
   - **Fast path (Windows):** WMI `__InstanceCreationEvent` for new processes; if WMI is unavailable, fall back to Toolhelp PID-diff (~0.5s)  
   - **Linux observe path:** poll-only today (`Watch` unsupported); enforcement relies on the periodic scan  
   - **Reconciliation:** periodic full scan (default ~3s; `SG_SCAN_INTERVAL`) catches anything the watch missed  
   - **audit** → report `violation_detected`; **enforce** → terminate (Level 1) and report `violation_blocked`  
7. **Heartbeat and policy sync** run on a separate slower timer (default ~30s; `SG_HEARTBEAT_INTERVAL`) — not on the enforce path  
8. Refresh when the server returns a newer version/hash — supported via conditional sync above

## Transport expectations

- HTTPS/TLS in production  
- Device authentication on all post-enroll calls  
- JSON request/response bodies with an explicit `schema_version` for evolution  
- Enroll and event ingest may return **429** `rate_limited` (per IP / per device); agents should back off and rely on the offline violation queue when uploads are throttled  
- Oversized request bodies or event batches are rejected with **400** (server-side size and batch caps)  

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

Batches include per-event client IDs to support idempotent ingestion (safe under agent retry and worker replay). A successful HTTP accept means the control plane enqueued the violation; durable listing in Events requires the worker to be running. Servers may hint that policy reload is recommended in the batch response.

## Offline behavior

Agents retain a local policy cache so audit/enforce continues while the control plane is unreachable. **Violation** events (`violation_detected` / `violation_blocked`) that fail to upload are appended to a durable local file (`events-queue.json` under the agent data directory, capped at a few hundred entries). On the heartbeat timer (and at startup), the agent flushes queued batches; the server’s `client_event_id` idempotency makes retries safe. Heartbeats are not queued.

## Safety constraints

- Agents must not provide a general-purpose remote command channel  
- Policy payloads are structured rules, not shell scripts  
- Built-in protections prevent termination of critical OS processes even if misconfigured remotely  
