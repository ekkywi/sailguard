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
# go run ./services/controlplane/cmd/worker
# redis-cli -p 16380 XADD sg:events '*' type heartbeat note hello
# Auth smoke:
# curl -s http://127.0.0.1:18080/v1/auth/login -H 'Content-Type: application/json' \
#   -d '{"email":"admin@sailguard.local","password":"ChangeMe!SailGuard1"}'
# Inventory smoke (Bearer token):
# curl -s http://127.0.0.1:18080/v1/groups -H "Authorization: Bearer $TOKEN"
# curl -s http://127.0.0.1:18080/v1/devices -H "Authorization: Bearer $TOKEN"
# curl -s http://127.0.0.1:18080/v1/enrollment-tokens -H "Authorization: Bearer $TOKEN" \
#   -H 'Content-Type: application/json' -d '{"label":"lab","max_uses":5}'
# Agent enroll:
# curl -s http://127.0.0.1:18080/v1/agent/enroll -H 'Content-Type: application/json' \
#   -d '{"schema_version":1,"enrollment_token":"…","hostname":"lab-pc","machine_guid":"g1","os_family":"windows","os_version":"10","agent_version":"0.0.1"}'
# Agent device auth + heartbeat:
# DEVICE_TOKEN=$(jq -r .device_token "$SG_DATA_DIR/device.json")
# curl -s http://127.0.0.1:18080/v1/agent/whoami -H "Authorization: Bearer $DEVICE_TOKEN"
# curl -s -X POST http://127.0.0.1:18080/v1/agent/events -H "Authorization: Bearer $DEVICE_TOKEN" \
#   -H 'Content-Type: application/json' -d '{"schema_version":1,"device_id":"…","sent_at":"…","events":[{"client_event_id":"hb-1","event_type":"heartbeat","occurred_at":"…"}]}'
# Web: http://127.0.0.1:15180 — Devices, Groups, Tokens
```

## Milestone orientation

- M0 done (scaffold).
- M1 foundation **done** (auth/RBAC, `/ready`, Streams skeleton, web login + theme).
- M2 **in progress**: inventory + membership + agent enroll client + device auth + **heartbeat ingest (server) done**; next **agent periodic heartbeat**, then MachineGuid/ACL or device PATCH.
- Later: policy sync, enforce Level 1, alerts, full admin UI.

## When resuming

State the next concrete step from `handoff.md` (currently: agent client heartbeat flush). Prefer one complete method per guidance step. If handoff conflicts with the repo, trust the repo + checklist and update handoff.
