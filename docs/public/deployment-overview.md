# Deployment overview

SailGuard is designed for **self-hosted / on-premises** installation. Agents run on managed endpoints; the control plane runs on customer infrastructure.

## Reference topology

```text
┌─────────────────────────────────────────────┐
│                 Private network             │
│  ┌─────────┐  ┌────────┐  ┌──────┐  ┌─────┐ │
│  │   Web   │  │  API   │  │Worker│  │Redis│ │
│  └────┬────┘  └───┬────┘  └──┬───┘  └──┬──┘ │
│       │           │          │         │    │
│       └───────────┴─────┬────┴─────────┘    │
│                         │                   │
│                   ┌─────▼─────┐             │
│                   │ PostgreSQL│             │
│                   └───────────┘             │
└───────────────────────▲─────────────────────┘
                        │ HTTPS
              ┌─────────┴─────────┐
              │  Managed endpoints │
              │  (SailGuard Agent) │
              └───────────────────┘
```

## Production packaging

A full-stack Compose definition is provided for typical single-host or small-cluster on-prem pilots:

- PostgreSQL  
- Redis  
- API  
- Worker  
- Web (static UI behind a reverse proxy)

Operators should override default credentials, inject strong secrets, terminate TLS at a reverse proxy or load balancer, and keep datastores on the private Docker network only. Default published host ports are **18080** (API) and **15180** (web); see engineering `ports.md`.

Agents are installed on endpoints and are **not** packaged inside the control-plane Compose stack.

## Supported endpoint platforms

| Platform | Support level | Notes |
|----------|---------------|--------|
| **Windows 7 SP1 and later** (including 8.1, 10, 11) | Supported target | Prefer 64-bit; TLS 1.2; Toolhelp list/terminate + PID-diff watch; build Win7 with Go 1.20 |
| **Linux** | Supported target | Process enumeration via `/proc`; Level 1 terminate; poll-based observe (event watch later) |
| **Windows XP / Vista** | **Not supported** | Out of scope for the current Go agent |
| **macOS** | Later | Agent stubs only |

Primary validation focus: Windows 10/11 and Linux; Windows 7 is in scope and should be regression-tested when agent binaries change. Use classic Win32 APIs for process list/terminate where possible so Windows 7–11 stay on one code path.

**Windows 7 binaries:** Go 1.21+ does not run on Windows 7. Build with **Go 1.20** via `scripts/build-agent-win7.sh` → `bin/sailguard-agent-win7.exe`. Day-to-day development keeps the newer Go toolchain for Windows 10/11 and Linux.

## Configuration

Runtime configuration is supplied through environment variables (database URL, Redis URL, HTTP bind address, token signing material, and similar). Example templates ship without production secrets.

## Capacity guidance (indicative)

The architecture targets mid-scale estates (on the order of **hundreds to low thousands** of Windows 7 SP1+ and Linux endpoints) with batched agent telemetry and asynchronous workers. Exact sizing depends on event volume, retention policy, and hardware.

## Development vs production runtimes

For engineering workstations, dependencies may be started alone (database and Redis) while API, worker, and UI processes run directly on the host for fast iteration. Production deployments should run the full control-plane service set under the organization’s process supervisor or container platform.

See internal developer workflow notes for day-to-day commands.
