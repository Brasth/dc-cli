---
title: "Owned stack verbs + compose-kind workspace"
description: "Per-row logs, stack restart, compose-kind identity, and compose start/stop/exec shipped. Review High on compose exec TTY/sh closed."
status: completed
priority: P2
effort: 3d
branch: "main"
tags: [architecture, tui, compose, safety]
blockedBy: []
blocks: []
created: "2026-08-17T06:28:23.427Z"
createdBy: "ck:plan"
source: skill
---

# Owned stack verbs + compose-kind workspace

## Overview

dc-cli already talks to Docker. It does not own the daemon.

This plan grows **folder identity**, not a Docker UI. Phase 1 added verbs on stacks we already own. Phase 2 taught `kind=compose` (detect, list, classify). Phase 3 starts/stops/execs compose-only folders. Unlabeled stays report-only.

Brainstorm: [../reports/260817-1316-docker-scope-expansion.md](../reports/260817-1316-docker-scope-expansion.md)

## Locked decisions

| Decision | Value |
|---|---|
| Product | Stateless this-folder wrapper. No daemon TUI, no Colima/Desktop start, no image/volume zoo. |
| Kind | `devcontainer` if project config exists. Else root compose file → `compose`. Else not a workspace. |
| Labels | Never fake `devcontainer.local_folder`. |
| Unlabeled | Report-only on unattended mutate. `--id` hatch stays named-present; restart must **not** copy that hole. |
| Restart | `docker restart` only if id ∈ `dc_stack_rows`. Labeled app row refused (use `u`/`s`). TUI `R`. CLI `dc-exec --service NAME --restart`. |
| Logs | TUI `l` follows selected stack row. Fleet still refuses. |
| Compose own | `compose config` `.name` + engine `working_dir`/`config_files` same-workspace. Name-alone is unknown. |
| Compose collide | Two folders, same project name → fail closed. Do not hash-rename. |
| Compose app | One service → that. Else service `app` then `web`. Else `dc-exec` lists and refuses. TUI `e` same refuse; Enter still execs the selected stack row. |
| Fleet | `dc-ls --all` stays **labeled-only**. Compose-kind is this-folder only. No daemon scan for `working_dir`. |
| `dc-ls` | Still a container list. Stopped compose-only folder → `[]`. Do not invent synthetic rows. Kind lives on doctor / this-folder detect. |
| Manifesto site | Phase 3: never raw `docker exec NAME`; always `dc-exec`. |
| Phase 3 | Un-gated 2026-08-17. Start/stop/exec compose-kind via `docker compose`. No auto `dc-forward`. |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Owned stack verbs](./phase-01-owned-stack-verbs.md) | Done ([review](./reports/review-phase-01.md) 7/10, High fixed; [tester](./reports/tester-phase-01.md); [closeout](./reports/project-manager-phase-01.md)) |
| 2 | [Compose-kind identity](./phase-02-compose-kind-identity.md) | Done ([review](./reports/review-phase-02.md) 7/10, High fixed; [tester](./reports/tester-phase-02.md); [closeout](./reports/project-manager-phase-02.md)) |
| 3 | [Compose-kind lifecycle and docs](./phase-03-compose-kind-lifecycle-and-docs.md) | Done ([review](./reports/review-phase-03.md) 7/10, High `-T`+bash/sh fixed; [tester](./reports/tester-phase-03.md); [closeout](./reports/project-manager-phase-03.md)) |

Sequential. Phase 3 un-gated 2026-08-17. All phases complete.

## Dependencies

- Safety-doctor locks (`260816-0949-safety-doctor-roadmap`) — reuse lock, tri-state, unlabeled report-only
- `dc-net` compose file list + `compose config` JSON (`260817-folder-nets`)
- Residual safety follow-ups (sidecar fingerprint, unpublished floor) are **not** blockers

## Out of scope

lazydocker/Portainer clone, unlabeled TUI kill list, engine start/stop, `docker system prune -af --volumes`, project `.devcontainer` edits, desktop GUI, shared remote Docker.

## Success criteria

- [x] TUI logs follow the selected sibling; fleet still refuses `l`
- [x] Restart only stack members; unlabeled/`--id` restart refused
- [x] This-folder `dc-ls --json` includes `kind` on any running/exited compose containers; `--all` stays labeled-only
- [x] Phase 3 starts that folder via `docker compose` and does not call official CLI (`up-compose-kind-starts`; CLI stub unused; compose-kind `dc-up` skips `command -v devcontainer`)
- [x] Unlabeled take-ports tests still refuse `stop`/`rm`

## Unresolved

- `--id` outside workspace: refuse vs hatch (open since safety roadmap; not this plan)
- take-ports vs foreign compose-kind: **closed in Phase 3** — proven `working_dir`/`config_files` → `compose|foreign` → `dc-down $folder`; unlabeled/name-only still report-only
- Phase 3 High: **closed** — compose-kind `dc-exec` uses `-T` when non-TTY; default shell bash then sh. `exec-compose-app` asserts `exec -T`.

## Validation Log

**Validated:** 2026-08-17  
**Questions asked:** 4 (user declined; recommended locks kept)

### Verification Results

- **Tier:** Standard (Fact Checker + Contract Verifier)
- **Claims checked:** 14
- **Verified:** 13 | **Failed:** 1 | **Unverified:** 0

Verified: `openLogs` uses `rows[0]` (`cmd/dc-tui/logs.go:50-61`); `r` is reload (`model.go:285`); fleet `l` refuses (`model.go:299-302`); `dc_stack_rows` / `dc_ensure_running` / `dc_labeled_ids` / `dc_has_devcontainer` / `dc_compose_claimants` / `dc_net_compose_json` exist; `dc-up` requires `devcontainer` on PATH first (`bin/dc-up:99`); no `docker restart` in repo; TUI sibling exec is `dc-exec --id` from stack (`actions.go:169`).

Failed: `view.go:273` disables `l` when `rows[0]` empty — Phase 1 must enable `l` from selected stack id, not only `dc-ls` rows[0].

### Confirmed decisions (defaults; interview declined)

- App-row `R`: refuse; use `u`/`s`
- Leave `--id` hatch; ban `--id --restart` only
- Compose app service: 1 / `app` / `web` / refuse
- Compose-kind: no auto `dc-forward`

### Action items

- [x] Phase 1: `view.go` logs button + `openLogs` both key off stack cursor (not only `rows[0]`)
- [x] Phase 1 review High: three-pass match in `dc_restart_in_stack` + collision test (`db` vs app id `db12ffff`)

### Whole-Plan Consistency Sweep

- Files reread: plan.md, phase-01, phase-02, phase-03
- Decision deltas checked: 4 (all already locked in plan/phases)
- Reconciled stale references: 1 (`view.go` logs-disable noted here; phase-01 files list already includes `view.go`)
- Unresolved contradictions: 0

