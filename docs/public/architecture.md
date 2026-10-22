# Architecture

SailGuard is an on-premises platform for monitoring and controlling disallowed applications on managed endpoints. Supported agents target **Windows 7 SP1+** and **Linux**; a multi-platform agent layout reserves Darwin stubs for later. Windows XP is out of scope.

## Design goals

- Detect and report use of applications that violate organizational policy
- Enforce policy by preventing matched processes from remaining running (process termination model)
- Express policy with multiple match dimensions (executable name, path, hash, publisher, product metadata, and curated categories)
- Assign policy at organization, group, and device scope, with time-bounded device exceptions
- Operate fully self-hosted for environments that require data residency and network isolation
- Separate administrative access by role (RBAC)

## Logical architecture

```text
Managed endpoint                 Control plane (on-prem)              Administrators
────────────────                 ───────────────────────              ──────────────
SailGuard Agent  ── TLS ──────►  API service                          Web console
  • local policy cache           Worker service                         (HTTPS)
  • process evaluation           PostgreSQL (system of record)
  • enforcement + reporting      Redis (job streams & short-lived cache)
```

Agents evaluate policy locally so enforcement continues during brief control-plane unavailability. The control plane remains the system of record for configuration, identity, and historical events.

## Components

| Component | Responsibility |
|-----------|----------------|
| **Agent** | Device enrollment (including reclaim by machine ID; refuses disabled reclaim), policy sync, process observation (Linux `/proc`; Windows Toolhelp + WMI watch with PID-diff fallback), audit/enforce Level 1 terminate, event submission with optional process owner; durable offline violation queue (`events-queue.json`) flushed on reconnect; Windows **SailGuardAgent** service auto-start; heartbeat timer separate from enforce path; logs and waits when the server reports `device_disabled` |
| **API** | Authentication (local admin login + Bearer tokens; device Bearer with reject when `status=disabled`), authorization, configuration APIs, agent-facing endpoints (heartbeat sync; violation batches enqueued to Redis), Windows agent package download |
| **Worker** | Consumes Redis Stream `sg:events` and persists violation rows to PostgreSQL; later: notifications and retention jobs |
| **Web console** | Administrative UI (local sign-in; overview; Devices and Groups with edit/delete/disable; Tokens; Download agent install/update packages; Policies list/create/delete, rules, assign, publish; Categories; Users; Audit log with filter/export; Events with process user; toast feedback on key actions). Visual language: calm operational console — product mark + limited gold accent (nav active, enforce mode); factual labels/CTAs (not marketing copy) |
| **PostgreSQL** | Durable storage for configuration, identity, and events |
| **Redis** | Job streams (`sg:events` for violation ingest) and ephemeral cache (not the system of record for business data) |

## Policy evaluation model

Effective policy is computed **per device**:

1. Organization-level assignments  
2. Group-level assignments for groups that include the device  
3. Device-level assignments  
4. Time-bounded device overrides  

Category-based rules are expanded on the server into concrete technical signatures before delivery. Agents receive a flattened rule set and a policy mode (`audit` or `enforce`).

See [Policy model](policy-model.md) for semantics.

## Enforcement approach

SailGuard’s primary enforcement approach is **endpoint process control**: detect a matching process and terminate it when the device is in enforce mode.

Detection uses a **layered path**: Windows prefers WMI process-creation events, falls back to Toolhelp PID-diff if WMI fails, and always runs a **periodic full scan** as reconciliation. Heartbeat and policy pull stay on a slower cadence so enforcement latency is not tied to telemetry intervals. (ETW remains an optional future refinement.)

Stronger OS-native application control mechanisms (for example publisher-based OS policy packs) may be evaluated as optional advanced capabilities without changing the control-plane model.

## Trust boundaries

- Endpoints trust the control plane only over authenticated, encrypted channels  
- Agent credentials are unique per device  
- Administrative actions that change enforce posture are authorization-gated and auditable  
- Datastores are intended to run on private infrastructure networks, not on the public internet  

Details of security commitments are described in [Security overview](security-overview.md).
