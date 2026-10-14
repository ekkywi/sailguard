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
- Agent OS floor: **Windows 7 SP1+** and **Linux** (Darwin stub; **XP unsupported**). Prefer classic Win32 APIs for enum/terminate.
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

- M0–M3 magang core **done** (policy UI, audit match, Events persist/UI; mode PATCH + delete rules).
- M4 **partial**: Linux `/proc` List+Terminate **done**; Windows Toolhelp List+Terminate+PID-diff Watch+scan fallback **done**; Win7 Go 1.20 build **done**. Remaining: WMI/ETW Watch (6B), streams/dashboard, MachineGuid.
- Later: alerts, users admin, LDAP, etc.

## Agent timers / dual-path

- `SG_SCAN_INTERVAL` default `3s` — full list reconciliation (and Linux enforce path today).
- `SG_HEARTBEAT_INTERVAL` default `30s` — policy sync + heartbeat (not enforce).
- Windows `Watch`: PID-diff every ~500ms; on `ErrWatchUnsupported` → poll only.
- Shared `recentSet` dedupe (~45s) across watch + scan to avoid double kill/report.

## When resuming

State the next concrete step from `handoff.md` (currently: Windows WMI/ETW Watch 6B, or MachineGuid). Prefer one complete method per guidance step. If handoff conflicts with the repo, trust the repo + checklist and update handoff.

## Lab smoke

```bash
# Linux real enum (no fake) — expect watch unsupported, poll fallback:
unset SG_LAB_FAKE_PROCESS
export SG_DATA_DIR=$HOME/.sailguard-agent-lab
go run ./agent/cmd/sailguard-agent
# expect: process watch: unsupported — poll fallback only
# expect: audit scan: processes=<large N> …

# Win7 enforce: ./scripts/build-agent-win7.sh → SG_SERVER_URL=http://<lan-ip>:18080
# API LAN bind: SG_HTTP_ADDR=0.0.0.0:18080 (source .env)

# Optional lab fake match:
# export SG_LAB_FAKE_PROCESS=notepad.exe
```
