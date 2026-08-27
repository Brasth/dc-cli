---
title: "Wedge first-run — product + website"
description: "Deepen the existing wedge. First minute gets a shell. Homepage sells that win. Do not expand scope."
status: implemented
priority: P0
branch: feature/wedge-first-run-plan-955f
tags:
  - first-run
  - site
  - try
  - recover
  - launch
blockedBy: []
blocks: []
created: "2026-08-27T09:27:00.000Z"
createdBy: "cursor:cloud"
source: ask
---

# Wedge first-run — product + website

**Approach A.** Users try `dc-cli` and leave. The leak is first contact, not missing features.

- Design: [`docs/superpowers/specs/2026-08-27-wedge-first-run-design.md`](../../docs/superpowers/specs/2026-08-27-wedge-first-run-design.md)
- Task plan: [`docs/superpowers/plans/2026-08-27-wedge-first-run.md`](../../docs/superpowers/plans/2026-08-27-wedge-first-run.md)

## Locked decisions

| Decision | Value |
|---|---|
| Scope | Deepen this-folder wrappers. No desktop, no team host, no spec rewrite |
| Wow | Got a shell in this folder |
| Site | Copy + SEO only. Layout, videos, `/play`, curl stay |
| H1 | Dev containers from your terminal. |
| Trust | Wraps the official Dev Containers CLI. (below the win) |
| Advertised curl | Unchanged `--with-cli` |
| `dc-up` none | TTY offers `dc-try` (default N). `--yes` does not try |
| Recover none | Apply `dc-try --yes` (`try_sandbox`) |
| `dc-try` ports | Still not MVP |
| Empty machine | Still print/copy. Not this plan |
| Launch | After first-run + site are on `main` |

## Why not expand

DevPod (~15k stars) already owns desktop + SSH + any backend. crib / `dev` / brig own “one binary, reimplement the spec.” This kit’s wedge is: official CLI, never edit `.devcontainer`, board + doctor + recover + Colima ports + agent skill. More verbs will not fix a dead first minute.

## What already exists (do not rebuild)

| Surface | Gap this plan closes |
|---|---|
| Advertised `--with-cli` | Copy still says “helpers”; install Next dumps eight verbs |
| `dc-try` + TUI confirm | `dc-up` still feels like a refuse; recover `kind_none` does not apply |
| Recover v1 (existing engine) | Owned by `260822-0140-docker-self-serve`. This plan only adds try-apply |
| Site / `/play` / clips | H1 and pain sell wrappers, not a shell |
| Launch kits | Same H1 drift |

## Phases

| # | Phase | Status | Plan tasks |
|---|---|---|---|
| 1 | Copy contract + CI try | proposed | Task 1 |
| 2 | Site, branding, README, launch copy | proposed | Task 2 |
| 3 | Install Next + `dc-up` TTY try + TUI string | proposed | Tasks 3–5 |
| 4 | Recover `try_sandbox` + skill | proposed | Tasks 6–7 |
| 5 | Verify suites | proposed | Task 8 |
| 6 | Text the 4 users + Show HN / PH | proposed | Task 9 (ops) |

## Success

- [ ] Hero, `<title>`, `assets/branding/copy.md`, and README lead with **Dev containers from your terminal.**
- [ ] `tests/copy/run.sh` and `tests/try/run.sh` run in CI
- [ ] Install Next is `source` → `cd` → `dc` → `dc try` → `dc recover`
- [ ] Non-TTY `dc-up` on `kind=none` still exits 1 and mentions `dc-try`
- [ ] TTY / `DC_UP_TRY_PROMPT=1` + `y` runs `dc-try --yes`
- [ ] `dc-up --yes` on `kind=none` does not start a sandbox
- [ ] Recover `kind_none` has `applyAllowed=1` and `--yes` runs `dc-try --yes`
- [ ] `go test ./cmd/dc-tui` still green
- [ ] No desktop, no empty-machine brew install, no `.devcontainer` writes

## Cook

Implement from `docs/superpowers/plans/2026-08-27-wedge-first-run.md` (subagent-driven or inline).
