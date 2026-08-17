---
title: "Owned stack verbs + compose-kind workspace"
description: "Per-row logs, stack restart, and compose-kind identity shipped. Compose start stays gated."
status: in-progress
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

This plan grows **folder identity**, not a Docker UI. Phase 1 added verbs on stacks we already own. Phase 2 taught `kind=compose` (detect, list, classify) with **no start**. Phase 3 may start/stop/exec compose-only folders. Unlabeled stays report-only.

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
| Manifesto site | Unchanged until Phase 3 ships. Phase 2 skill: detect-only. |
| Phase 3 | Gated. Do not cook until Phase 2 tests land and user says start compose-kind. |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Owned stack verbs](./phase-01-owned-stack-verbs.md) | Done ([review](./reports/review-phase-01.md) 7/10, High fixed; [tester](./reports/tester-phase-01.md); [closeout](./reports/project-manager-phase-01.md)) |
| 2 | [Compose-kind identity](./phase-02-compose-kind-identity.md) | Done ([review](./reports/review-phase-02.md) 7/10, High fixed; [tester](./reports/tester-phase-02.md); [closeout](./reports/project-manager-phase-02.md)) |
| 3 | [Compose-kind lifecycle and docs](./phase-03-compose-kind-lifecycle-and-docs.md) | Pending (gated) |

Sequential. Phase 3 does not start until Phase 2 is completed **and** explicitly un-gated.

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
- [x] Phase 2 `dc-up` still cannot start a compose-only folder
- [ ] Phase 3 (if un-gated) starts that folder via `docker compose` and does not call official CLI
- [x] Unlabeled take-ports tests still refuse `stop`/`rm`

## Unresolved

- `--id` outside workspace: refuse vs hatch (open since safety roadmap)
- take-ports vs foreign compose-kind: locked to Phase 3 (predicate + `dc-down $working_dir`)
- Phase 2 High (closed): `dc_compose_claimants` skips labeled container `working_dir`/`config_files`. Same-stack `$ws` + `$ws/.devcontainer` n=1; labeled + foreign n=2. Review hole was [review-phase-02.md](./reports/review-phase-02.md); re-test [tester-phase-02.md](./reports/tester-phase-02.md).

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

