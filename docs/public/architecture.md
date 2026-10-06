# Architecture

SailGuard is an on-premises platform for monitoring and controlling disallowed applications on managed endpoints. The initial supported platform is Microsoft Windows, with a multi-platform agent architecture prepared for additional operating systems.

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
| **Agent** | Device enrollment, policy synchronization, process observation, enforcement, event submission |
| **API** | Authentication (local admin login + Bearer tokens), authorization, configuration APIs, agent-facing endpoints |
| **Worker** | Asynchronous processing via Redis Streams (event pipelines, notifications, retention jobs) |
| **Web console** | Administrative UI (local sign-in against the API; devices, policies, events, and users expand over time) |
| **PostgreSQL** | Durable storage for configuration, identity, and events |
| **Redis** | Job streams (for example event ingest) and ephemeral cache (not the system of record for business data) |

## Policy evaluation model

Effective policy is computed **per device**:

1. Organization-level assignments  
2. Group-level assignments for groups that include the device  
3. Device-level assignments  
4. Time-bounded device overrides  

Category-based rules are expanded on the server into concrete technical signatures before delivery. Agents receive a flattened rule set and a policy mode (`audit` or `enforce`).

See [Policy model](policy-model.md) for semantics.

## Enforcement approach

SailGuard’s primary enforcement approach is **endpoint process control**: detect a matching process and terminate it when the device is in enforce mode. Stronger OS-native application control mechanisms (for example publisher-based OS policy packs) may be evaluated as optional advanced capabilities without changing the control-plane model.

## Trust boundaries

- Endpoints trust the control plane only over authenticated, encrypted channels  
- Agent credentials are unique per device  
- Administrative actions that change enforce posture are authorization-gated and auditable  
- Datastores are intended to run on private infrastructure networks, not on the public internet  

Details of security commitments are described in [Security overview](security-overview.md).
