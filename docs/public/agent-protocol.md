# Agent protocol

This document describes the agent ↔ control plane contract at a product level. Field-level examples for implementers are maintained in engineering docs.

## Lifecycle

1. **Enroll** with a one-time (or limited-use) enrollment token via `POST /v1/agent/enroll` *(admins mint/revoke tokens in the console or API)*  
2. Receive a unique **device credential** and recommended sync intervals; persist locally on the agent  
3. Authenticate subsequent calls with `Authorization: Bearer <device_token>` *(e.g. whoami / events)*  
4. **Submit event batches** via `POST /v1/agent/events` — **heartbeat** is accepted now and refreshes `last_seen_at`; violation persistence expands later  
5. **Poll / pull effective policy** (version + content hash) — planned  
6. Continuously evaluate local processes against cached policy — planned  
7. Refresh policy when the server indicates a newer version — planned  

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
| `violation_detected` | Policy match while in audit (or pre-enforcement reporting) |
| `violation_blocked` | Match under enforce, including enforcement outcome |
| `agent_error` | Agent-side failures worth surfacing to operators |

Batches include per-event client IDs to support idempotent ingestion. Servers may hint that policy reload is recommended in the batch response.

## Offline behavior

Agents retain a local policy cache and may queue outbound events to durable local storage when the control plane is unreachable, then flush when connectivity returns (subject to size limits).

## Safety constraints

- Agents must not provide a general-purpose remote command channel  
- Policy payloads are structured rules, not shell scripts  
- Built-in protections prevent termination of critical OS processes even if misconfigured remotely  
