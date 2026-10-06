# Agent protocol

This document describes the agent ↔ control plane contract at a product level. Field-level examples for implementers are maintained in engineering docs.

## Lifecycle

1. **Enroll** with a one-time (or limited-use) enrollment token *(control-plane enroll endpoint planned; inventory tables for tokens/credentials exist)*  
2. Receive a unique **device credential** and recommended sync intervals  
3. **Poll / pull effective policy** (version + content hash)  
4. Continuously evaluate local processes against cached policy  
5. **Submit event batches** (heartbeats, violations, agent errors)  
6. Refresh policy when the server indicates a newer version  

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
