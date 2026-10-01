# SailGuard documentation

| Folder | In git? | Audience | Purpose |
|--------|---------|----------|---------|
| [public/](public/) | **Yes** | Operators, reviewers, stakeholders | Product documentation suitable for sharing |
| `internal/` | **No** (gitignored) | You / local machine only | Checklists, handoff, full API maps, security gaps, lab notes |

`docs/internal/` stays on your workstation (and any copy you sync yourself). It is intentionally **not** pushed to the remote so progress notes and lab details do not leak.

If you open the repo on another machine, copy `docs/internal/` privately (USB, encrypted sync, etc.) or recreate handoff from memory using `docs/public/` + code.

Do not copy internal threat-gap notes or lab passwords into `docs/public/` without review.
