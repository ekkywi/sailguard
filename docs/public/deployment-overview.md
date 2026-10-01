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

## Configuration

Runtime configuration is supplied through environment variables (database URL, Redis URL, HTTP bind address, token signing material, and similar). Example templates ship without production secrets.

## Capacity guidance (indicative)

The architecture targets mid-scale estates (on the order of **hundreds to low thousands** of Windows endpoints) with batched agent telemetry and asynchronous workers. Exact sizing depends on event volume, retention policy, and hardware.

## Development vs production runtimes

For engineering workstations, dependencies may be started alone (database and Redis) while API, worker, and UI processes run directly on the host for fast iteration. Production deployments should run the full control-plane service set under the organization’s process supervisor or container platform.

See internal developer workflow notes for day-to-day commands.
