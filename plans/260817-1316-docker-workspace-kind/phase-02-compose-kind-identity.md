---
phase: 2
title: "Compose-kind identity"
status: review-concerns
priority: P2
effort: 1d
dependencies: [1]
---

# Phase 2: Compose-kind identity

## Context links

- Parent: [plan.md](./plan.md)
- Depends on: [Phase 1](./phase-01-owned-stack-verbs.md)
- Research: [researcher-02-compose-kind.md](./research/researcher-02-compose-kind.md)
- Reuse: `lib/dc-net.sh` file list + `docker compose config --format json`

## Overview

Teach the kit a second this-folder identity: `kind=compose`. Detect, list, classify, doctor. **Do not start.** `dc-up` on a compose-only folder still fails closed with a kind-aware message.

Date: 2026-08-17. Priority: P2. Implementation: done. Review: 7/10, 0 critical — [review-phase-02.md](./reports/review-phase-02.md). High: `dc_compose_claimants` double-counts labeled app + own `.devcontainer` working_dir → `dc-down` refuse / take-ports `ambiguous-compose`. Tester: [tester-phase-02.md](./reports/tester-phase-02.md) 114 pass (kind 8 / safety 19 / doctor 9 / exec 8 / TUI 70).

## Key insights

- `dc_resolve_workspace` already returns folders without `.devcontainer`. `dc-up` is the gate (official CLI).
- `dc-ls` / fleet / take-ports all filter `devcontainer.local_folder`. Compose-only stacks are invisible and unlabeled to `--take-ports`.
- Compose JSON `.name` exists via `dc-net`. Runtime ownership needs engine `working_dir` / `config_files`, not name alone (basename collision).

## Requirements

- Functional: `dc_workspace_kind` → `devcontainer` | `compose` | `none`. This-folder `dc-ls --json` adds `kind` on real containers. Stopped compose-only folder → `[]` (not a synthetic workspace row). `dc-ls --all` **unchanged** (labeled only). Doctor reports kind. `dc-up` kind=compose: nonzero, no `devcontainer up`, no `compose up`.
- Non-functional: no daemon scan. Phase 2 skill says detect-only. Site manifesto waits for Phase 3.

## Architecture

Detect order:

1. `dc_has_devcontainer` → `devcontainer`
2. else root compose file (`compose.yaml|yml` / `docker-compose.yaml|yml`) and no project config → `compose`
3. else `none`

Ownership (engine-written only):

1. `com.docker.compose.project` == `docker compose config` `.name` (same `-f` set as `dc-net`)
2. `com.docker.compose.project.working_dir` or `config_files` `dc_same_workspace` as folder
3. Claimants = labeled folders **or** compose working_dirs. n>1 → fail closed
4. Missing both working_dir and config_files → unknown, report-only
5. Never infer from project name alone

`dc-ls`: still lists containers. This-folder compose-kind: if compose containers exist, tag `kind=compose`. If none exist, `[]`. Fleet `--all`: labeled only — do **not** walk the daemon for `working_dir`.

`--take-ports`: **no new stop paths in this phase.** Compose-kind holders stay report-only until Phase 3.

Rewrite skill/manifesto:

- Never raw `docker exec NAME`
- Always `dc-exec`
- kind=devcontainer app = official CLI
- kind=compose (future) = `docker compose exec` inside `dc-exec`

## Related code files

- Modify: `lib/dc-common.sh` (kind, claimants, ls, classify)
- Reuse: `lib/dc-net.sh` (do not fork YAML parse)
- Modify: `bin/dc-ls`, `bin/dc-up` (refuse compose-kind), `bin/dc-doctor`
- Modify: `tests/safety/run.sh`, new `tests/` cases for kind detect / ls / no fake labels
- Modify: `skill/SKILL.md`, `README.md` (detect-only sentence)
- Do not modify: `site/src/components/Manifesto.astro` (Phase 3)
- Do not modify: `dc-down` mutate path, TUI start, `dc-forward` (no sidecars yet)

## Implementation steps

1. Lie-tests: folder with compose.yml and no `.devcontainer` → kind=compose; folder with both → kind=devcontainer; empty → none. `dc-up` compose-kind does not invoke `devcontainer`. `dc-ls` JSON has `kind` and no invented DC label writes (`docker` command log).
2. Implement `dc_workspace_kind` + compose name from `dc_net_compose_json` `.name`.
3. Generalize claimants / classify: unknown if working_dir missing.
4. `dc-ls --json` schema: add `kind` (default `devcontainer` for old rows). schemaVersion bump if the doctor/ls contracts require it — keep compatible if field is additive.
5. Doctor: report `workspace.kind` + warning if compose and start disabled.
6. `dc-up`: kind-detect **before** `command -v devcontainer`. kind=compose → refuse (start not enabled), exit 1. No official CLI required for that path.
7. Skill: kind detect exists; do not `compose exec` yet. README one sentence. Leave site manifesto.
8. Safety tests: unlabeled still report-only; no `docker label` / commit of fake labels.

## Todo list

- [x] Kind detect + tests
- [x] `dc-ls` `kind` field; no fake labels (`--json` only; TTY table still labeled-only)
- [ ] Claimants / classify unknown-on-missing-working_dir — **High:** claimants over-fire when labeled `local_folder` ≠ own `working_dir` (e.g. `$ws/.devcontainer`). Classify still `unlabeled` for compose-kind (report-only; OK for no-new-stop)
- [x] Doctor kind field
- [x] `dc-up` compose-kind refuse
- [x] Skill + README detect-only (no manifesto edit)
- [x] Safety unlabeled tests still green (fixtures omit `working_dir`; High untested)

## Success criteria

- [x] Running compose-only containers in this folder appear on `dc-ls --json` with `kind=compose` and no DC labels (TTY `dc-ls` table still empty — review Medium)
- [x] Stopped compose-only folder: `dc-ls` is `[]`; doctor still reports kind=compose
- [x] `dc-ls --all` does not grow a daemon compose inventory
- [x] `dc-up` does not call `devcontainer` or `compose up` on that folder
- [x] Two folders same compose name: ls not owned; name-alone `[]` (claimants n>1 also fail-closed — over-fires on same-stack `.devcontainer` working_dir)
- [x] Unlabeled take-ports still report-only
- [x] Skill says detect-only; manifesto unchanged

## Risk assessment

| Risk | Mitigation |
|---|---|
| Basename collision two `app/` folders | Name-alone never owns |
| Old compose omits `working_dir` | unknown, not owned |
| Fleet `--all` scans whole daemon | Only emit compose-kind when working_dir proves a folder |
| Marketing identity blur | H1 unchanged; detect-only in docs |
| Phase 3 cooks early | `dc-up` refuse stays until Phase 3 un-gated |

## Security considerations

Listing compose-kind is read-only. Classify must not treat "same project name" as stop permission. Do not write labels onto running containers.

## Next steps

1. Fix High: `dc_compose_claimants` must not add working_dir/config_files for containers that already have `devcontainer.local_folder`. Safety case: labeled `$ws` + `working_dir=$ws/.devcontainer` → `dc-down` still stops; take-ports foreign still stops. Keep n>1 for labeled folder + *foreign* compose-kind working_dir.
2. Optional before demo polish: `dc_ls_table` same ids as JSON; strip inspect `<no value>` on compose-kind `local_folder`.
3. Show user `dc-ls --json` / `dc-doctor --json` on a compose-only folder. Do not demo `dc-down` on a labeled compose-in-`.devcontainer` workspace until (1) lands.
4. Phase 3 stays gated until user says start. Do not cook `compose up` / `compose exec` / manifesto / take-ports compose-kind stop.
