# SailGuard

On-prem IT endpoint monitoring: detect, report, and block disallowed applications (Windows-first).

## Stack (locked)

| Layer | Choice |
|-------|--------|
| Agent | Go (Windows first; Linux/macOS stubs) |
| Control plane | Go (`api` + `worker`) |
| Frontend | React + TypeScript + **npm** |
| Data | PostgreSQL + Redis (Streams from day one) |
| Deploy | On-prem / self-hosted |

## Repo layout

```text
pkgs/                  shared Go (policy merge + agent contracts)
services/controlplane/ API + worker
agent/                 multi-OS agent
web/                   admin UI
docs/public/           shareable product documentation (in git)
docs/internal/         engineering notes (local only — gitignored)
deploy/docker/         production images
```

## Development

Docker runs **only Postgres + Redis**. API, worker, and web run on the host so code changes are fast.

Dev Docker / host ports (SailGuard-specific, loopback): see [docs/public/deployment-overview.md](docs/public/deployment-overview.md) and local `docs/internal/ports.md` if present.

| Service | Address |
|---------|---------|
| API | `127.0.0.1:18080` |
| Web | `127.0.0.1:15180` |
| Postgres | `127.0.0.1:15433` |
| Redis | `127.0.0.1:16380` |

```bash
cp .env.example .env
make dev-up
make test-pkgs
make build-api && ./bin/sailguard-api   # or: go run ./services/controlplane/cmd/api
make build-worker                        # separate terminal
cd web && npm install && npm run dev
```

Health: `curl http://127.0.0.1:18080/v1/health`  
Ready: `curl http://127.0.0.1:18080/v1/ready` (expects `db` and `redis` true when Compose deps are up)  
Auth (after seed): `POST /v1/auth/login` then `GET /v1/auth/me` with Bearer token — see [docs/public/api-overview.md](docs/public/api-overview.md).  
Web console: http://127.0.0.1:15180/login (lab admin after seed).

More detail: [docs/internal/dev-workflow.md](docs/internal/dev-workflow.md).

## Production

```bash
docker compose up -d --build
```

Agents install on endpoints; they are not part of the server compose stack.

## Documentation

| Kind | Start here |
|------|------------|
| **Public** (product, in git) | [docs/public/](docs/public/) |
| **Internal** (local only, gitignored) | `docs/internal/` on your machine — checklist, handoff, ports |
| **Resume chat** | Copy `docs/internal/CONTINUE_PROMPT.md` (keep that folder synced privately across machines) |

Index: [docs/README.md](docs/README.md).

## License

Proprietary / TBD.
