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

- M0–M3 magang core **done** (policy UI, audit match, Events persist/UI; mode PATCH + add/edit/delete rules).
- M4 **partial**: Linux `/proc` List+Terminate **done**; Windows Toolhelp List+Terminate+WMI Watch+scan **done**; Win7 Go 1.20 build **done**; installer ZIP + setup **done**; install vs update download packages **done**; Windows Service `SailGuardAgent` auto-start **done** (Task Scheduler + detached fallback if SCM Access denied); MachineGuid/os-release **done**; process user on events **done**; enroll reclaim **done**; device/group/policy DELETE (and device/group PATCH) **done**; disabled-device auth gate (403 `device_disabled` + enroll refuse) **done**; offline violation queue + flush **done**; edit-rule in place (**PATCH** rules + UI) **done**; category rules in builder (`GET /v1/categories` + type-driven form) + lab signature seeds + **Categories admin UI** **done**; re-enroll on 401 **done** (lab smoke deferred); **dashboard summary** (`GET /v1/dashboard/summary` + Home; top processes/hosts + stale devices) **done**; **violations timeseries** (`GET /v1/dashboard/violations-timeseries?range=24h|7d` + Home chart) **done**. Remaining: Redis stream/worker; optional ETW/Linux Watch; credential ACL; signed fleet auto-update.
- Later: alerts, LDAP, Redis stream/worker, credential ACL, etc. (users admin + RBAC UI gates done).

## Agent timers / dual-path

- `SG_SCAN_INTERVAL` default `3s` — full list reconciliation (and Linux enforce path today).
- `SG_HEARTBEAT_INTERVAL` default `30s` — policy sync + heartbeat (not enforce).
- Windows `Watch`: WMI `__InstanceCreationEvent` first; on failure → Toolhelp PID-diff ~500ms; Linux → `ErrWatchUnsupported` (poll only).
- Shared `recentSet` dedupe (~45s) across watch + scan to avoid double kill/report.
- Win7 build: `./scripts/build-agent-win7.sh` (sed go 1.20 + `go mod tidy` with Go 1.20).
- Windows setup: `./scripts/build-agent-setup.sh` → `bin/sailguard-agent-setup.exe` (Go 1.20; required for Download agent packages).

## When resuming

State the next concrete step from `handoff.md` (currently: Redis stream/worker, credential ACL, or rotate-credential). Prefer one complete method per guidance step. If handoff conflicts with the repo, trust the repo + checklist and update handoff.

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
