---
name: sailguard-dev
description: >-
  Guide SailGuard on-prem endpoint monitoring development (Go API/agent,
  React admin, goose, Postgres/Redis). Use when working in the sailguard
  repo, continuing M1+ milestones, auth, policy, agent, or when the user
  pastes the SailGuard continue prompt / mentions SailGuard handoff.
---

# SailGuard development guide

## First actions

1. Read `docs/internal/handoff.md` if present (folder is gitignored — may be missing on fresh clones).
2. Skim `docs/internal/checklist-mvp.md` when available — do not delete unchecked items.
3. Read `docs/internal/academic-path.md` when discussing scope, titles, magang vs skripsi — keep technical work aligned with the internship→thesis path.
4. Respect `docs/internal/ports.md` or `docs/public/deployment-overview.md` + README ports table.
5. If `docs/internal/` is missing, ask the user to restore it or rely on code + `docs/public/`.

## How to work with this user

- Prefer **step-by-step guidance** the user types themselves.
- Explain **why** each snippet exists (short).
- Only implement large changes in Agent mode when explicitly asked.
- Do not shrink product scope to “solo MVP lite”; defer items stay on checklist/roadmap.
- Keep lab secrets out of `docs/public/`.

## Architecture reminders

- Control plane: `services/controlplane` (`cmd/api`, `cmd/worker`).
- Shared policy engine: `pkgs/policy` (merge allow>block at equal weight; safety rail).
- Agent contracts: `pkgs/agentcontract`.
- Agent OS split: `internal/host` + `windows|linux|darwin` + `platform_*` build tags.
- Migrations: goose under `services/controlplane/migrations/`.
- System catalog seeds (RBAC): goose migrations.
- Dev/demo seeds (admin user): `seeds/development/` only — never production goose.
- Module path: `github.com/ekkywi/sailguard/...`.

## Dev commands

```bash
make dev-up
make migrate-up
go run ./services/controlplane/cmd/api
cd web && npm run dev
curl http://127.0.0.1:18080/v1/health
curl -s http://127.0.0.1:18080/v1/ready   # db + redis true
```

## Milestone orientation

- M0–M2 **done**.
- M3 **done for magang core**: policy APIs + web policy UI + agent sync + audit match MVP.
- M4 **partial**: persist `device_events` + `GET /v1/events` + web `/events` **done**; remaining real process enum, enforce Level 1, streams/dashboard.
- Later: alerts, users admin, LDAP, etc.

## When resuming

State the next concrete step from `handoff.md` (currently: real process enum or enforce Level 1). Prefer one complete method per guidance step. If handoff conflicts with the repo, trust the repo + checklist and update handoff.

## Lab smoke (audit → Events)

```bash
export SG_DATA_DIR=$HOME/.sailguard-agent-lab
export SG_LAB_FAKE_PROCESS=notepad.exe
# policy audit + block notepad.exe published+assigned, then:
go run ./agent/cmd/sailguard-agent
# expect: audit scan … violations=1 accepted=1
# Admin: http://127.0.0.1:15180/events  or  GET /v1/events with Bearer
```
