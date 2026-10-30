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
3. Read `docs/internal/academic-path.md` when discussing scope, titles, magang vs skripsi — keep technical work aligned with the internship→thesis path. Locked titles: magang = prototipe agent–server untuk monitoring **dan** pemblokiran; skripsi = pengembangan kebijakan bertingkat. Frame as **aplikasi/RPL**, not cybersecurity/networking; on-premise only in deploy/batasan, not title. Draft laporan: `docs/internal/laporan-magang-draft.md` (update bab there; no lab secrets).
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
make run-api       # terminal 1
make run-worker    # terminal 2 — required for violation → Postgres
cd web && npm run dev
curl http://127.0.0.1:18080/v1/health
curl -s http://127.0.0.1:18080/v1/ready   # db + redis true
```

## Milestone orientation

- M0–M3 magang core **done** (policy UI, audit match, Events persist/UI; mode PATCH + add/edit/delete rules).
- M4 **partial**: Linux `/proc` List+Terminate **done**; Windows Toolhelp List+Terminate+WMI Watch+scan **done**; Win7 Go 1.20 build **done**; installer ZIP + setup **done**; install vs update download packages **done**; Windows Service `SailGuardAgent` auto-start **done** (Task Scheduler + detached fallback if SCM Access denied); MachineGuid/os-release **done**; process user on events **done**; enroll reclaim **done**; device/group/policy DELETE (and device/group PATCH) **done**; disabled-device auth gate (403 `device_disabled` + enroll refuse) **done**; offline violation queue + flush **done**; edit-rule in place (**PATCH** rules + UI) **done**; category rules in builder (`GET /v1/categories` + type-driven form) + lab signature seeds + **Categories admin UI** **done**; re-enroll on 401 **done** (lab smoke deferred); **dashboard summary** (`GET /v1/dashboard/summary` + Home; top processes/hosts + stale devices) **done**; **violations timeseries** (`GET /v1/dashboard/violations-timeseries?range=24h|7d` + Home chart) **done**; **Redis stream/worker Phase A** (`sg:events` enqueue + worker `InsertViolation`; heartbeat sync) **done**; **XAUTOCLAIM** PEL reclaim **done**; admin session TTL/revoke/idle **done**; **Events filters + CSV export** **done**; **alerts webhook v1** + **retry/dead-letter** (`alert_deliveries` + `sg:alerts`; Settings deliveries + admin retry) **done**; **rate limits + body/batch caps** **done**; **event retention** **done** (`SG_EVENTS_RETENTION_DAYS` default 180; `0` = off). **Linux package** **done** (`build-agent-linux.sh`, channel download + systemd `install.sh`); **Linux `/proc` name truncate fix** **done** (prefer cmdline basename when `comm` is 15 chars); **device asset Fase 1–4** **done** (admin fields, hardware, resource last-sample, primary IP + Online/Stale). Remaining: optional ETW/Linux Watch; signed fleet auto-update; email alerts; prod compose E2E / TLS docs.
- Later: LDAP, etc. (users admin + RBAC UI gates + device overrides + audit Batch 1–4 done; credential ACL + revoke done).
- Admin console visual rules: `docs/internal/ui-console.md` (limited `--sg-gold`, operational copy, no marketing/AI-slop chrome).

## Agent timers / dual-path

- `SG_SCAN_INTERVAL` default `3s` — full list reconciliation (and Linux enforce path today).
- `SG_HEARTBEAT_INTERVAL` default `30s` — policy sync + heartbeat (not enforce).
- Windows `Watch`: WMI `__InstanceCreationEvent` first; on failure → Toolhelp PID-diff ~500ms; Linux → `ErrWatchUnsupported` (poll only).
- Linux process name: `/proc/comm` max 15 chars; agent upgrades truncated `comm` to cmdline basename (so `gnome-calculator` rules match).
- Shared `recentSet` dedupe (~45s) across watch + scan to avoid double kill/report.
- Win7 build: `./scripts/build-agent-win7.sh` (sed go 1.20 + `go mod tidy` with Go 1.20).
- Windows setup: `./scripts/build-agent-setup.sh` → `bin/sailguard-agent-setup.exe` (Go 1.20; required for Windows Download packages).
- Linux build: `./scripts/build-agent-linux.sh` → `bin/sailguard-agent-linux` (`SG_AGENT_LINUX_PATH`).

## When resuming

State the next concrete step from `handoff.md` (currently: magang demo/laporan, or prod TLS; email SMTP deferred). Prefer one complete method per guidance step. If handoff conflicts with the repo, trust the repo + checklist and update handoff.
Lab RBAC smoke: `python3 scripts/smoke-rbac-roles.py` (seed `002_lab_roles.sql`; password `ChangeMe!SailGuard1`).
Agent ingest limits: `SG_RATE_ENROLL_PER_MIN` / `SG_RATE_EVENTS_PER_MIN` / `SG_ENROLL_MAX_BODY` / `SG_EVENTS_MAX_BODY` / `SG_EVENTS_MAX_BATCH`.
Event retention: `SG_EVENTS_RETENTION_DAYS` (default 180; `0` = off) — worker only; not endpoint offline queue; not `admin_audit_logs`.

## Lab smoke

```bash
# Linux real enum (no fake) — expect watch unsupported, poll fallback:
unset SG_LAB_FAKE_PROCESS
export SG_DATA_DIR=$HOME/.sailguard-agent-lab
go run ./agent/cmd/sailguard-agent
# expect: process watch: unsupported — poll fallback only
# expect: audit scan: processes=<large N> …

# Linux packaged: ./scripts/build-agent-linux.sh → Download Linux ZIP → sudo ./install.sh
# enforce smoke: block process_name=gnome-calculator → journalctl -u sailguard-agent -f

# Win7 enforce: ./scripts/build-agent-win7.sh → SG_SERVER_URL=http://<lan-ip>:18080
# API LAN bind: SG_HTTP_ADDR=0.0.0.0:18080 (source .env)

# Optional lab fake match:
# export SG_LAB_FAKE_PROCESS=notepad.exe
```
