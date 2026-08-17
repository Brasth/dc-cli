---
phase: 1
title: "Owned stack verbs"
status: completed
priority: P2
effort: 1d
dependencies: []
---

# Phase 1: Owned stack verbs

## Context links

- Parent: [plan.md](./plan.md)
- Research: [researcher-01-owned-stack-verbs.md](./research/researcher-01-owned-stack-verbs.md)
- Scout: [scout-01-docker-scope.md](./scout/scout-01-docker-scope.md)

## Overview

Fix TUI logs so `l` follows the selected compose sibling, not `dc-ls` `rows[0]`. Add stack-membership restart (`R` / `dc-exec --service NAME --restart`). No compose-kind. No unlabeled mutate.

Date: 2026-08-17. Priority: P2. Implementation: done. Review: 7/10, 0 critical — [review-phase-01.md](./reports/review-phase-01.md). High id-prefix collision fixed + re-tested — [tester-phase-01.md](./reports/tester-phase-01.md). Closeout: [project-manager-phase-01.md](./reports/project-manager-phase-01.md).

## Key insights

- Two lists: `m.rows` = labeled apps; `m.stack` = `dc-exec --list` siblings. Logs ignore the cursor today.
- Restart does not exist. `r` is reload. `dc-exec --id` already starts **any** inspectable box — restart must not copy that.
- `dc_stack_rows` needs a labeled app + compose project. Empty stack → refuse restart; logs fall back to `rows[0]`.

## Requirements

- Functional: `l` logs `stack[cursor]` when a stack row is selected. Restart only IDs in `dc_stack_rows`. Labeled app row: refuse restart, tell user `u`/`s`. Fleet refuses `l` and `R`.
- Non-functional: stay-on-board for logs and restart. No new bin. No new Go module.

## Architecture

```text
TUI cursor → stack[i].ID
  l  → docker logs -f $id
  R  → dc-exec --service $svc --restart   # membership-checked

CLI: dc-exec --service NAME --restart
  dc_restart_in_stack $ws $ref
    match via dc_stack_resolve (three-pass: service, then name, then id prefix)
    miss → fail (no docker inspect of strangers)
    docker restart $id + poll running
```

`--id --restart` is invalid. `--id` without `--restart` unchanged (escape hatch).

## Related code files

- Modify: `cmd/dc-tui/logs.go`, `logs_test.go`, `actions.go`, `model.go`, `view.go`, `main_test.go`
- Modify: `lib/dc-common.sh` (`dc_restart_in_stack`)
- Modify: `bin/dc-exec` (flag + usage)
- Modify: `tests/exec/run.sh`
- Modify after ship: `README.md`, `skill/SKILL.md`, `site/src/content/guides/tui.md`
- Do not touch: unlabeled classify, `dc-ls`, prune, take-ports

## Implementation steps

1. Tests first: fake stack with two siblings; `openLogs` uses cursor id ≠ `rows[0]`; empty stack falls back to `rows[0]`; fleet key `l` still refuses.
2. Change `openLogs` to pick `m.stack[m.cursor]` when `!fleet && len(stack)>0 && ID != ""`. Enable the `l` button from that same id (`view.go` today disables `l` on empty `rows[0]` — `view.go:273`).
<!-- Updated: Validation Session 1 - logs enable must follow stack cursor -->
3. Add `dc_restart_in_stack` in `dc-common.sh`. Match like `dc-exec --service`. Poll like `dc_ensure_running`.
4. `dc-exec --restart` requires `--service` (or stack-resolved name). Reject `--id --restart`. Usage text.
5. Tests: restart matching sibling calls `docker restart`; unknown service no restart; `--id --restart` exits nonzero; no labeled app → refuse.
6. TUI `R`: stayCmd `dc-exec --service $stack[cursor].Service --restart`. App row = `stack[i].ID == rows[0].ID` (not service-name guess) → refuse, tell user `u`/`s`. Fleet refuse same copy as start/stop.
7. `view.go` / more-legend: `R` restart sibling. `r` stays reload.
8. Docs: README TUI table, tui guide, skill (restart is stack-only).

## Todo list

- [x] Lie-tests for per-row logs
- [x] `openLogs` binds cursor
- [x] `dc_restart_in_stack` + `dc-exec --service --restart`
- [x] TUI `R` + fleet/app refuse
- [x] exec tests: match / miss / `--id --restart` banned
- [x] README + skill + tui guide
- [x] Review High: three-pass match in `dc_restart_in_stack` (exact service, then name, then id prefix) + collision test (`db` vs app id `db12ffff`)

## Success criteria

- [x] `go test ./cmd/dc-tui` covers cursor ≠ `rows[0]`
- [x] `tests/exec/run.sh` covers restart membership + banned `--id --restart`
- [x] Fleet `R`/`l` do not mutate or follow logs
- [x] Unlabeled take-ports tests unchanged and still pass
- [x] `r` still reloads the board
- [x] Restart of `--service db` cannot hit another stack member whose id is a `db*` prefix

## Risk assessment

| Risk | Mitigation |
|---|---|
| `--id --restart` becomes unlabeled mutate | Flag combination refused |
| `docker restart` on labeled app skips official recreate | Refuse app row; `u`/`s` only |
| Silent `--list` parse fail → empty stack → logs `rows[0]` | Keep fallback; do not invent rows |
| `R` vs `r` confusion | Docs + more-legend only |

## Security considerations

Restart is mutate. Must run under existing patterns (no lock required for one `docker restart` of an already-owned stack member — same class as `dc_ensure_running`'s `docker start`). Do not inspect/restart IDs that failed stack match.

## Next steps

Phase 1 complete. Do **not** start Phase 2 from this file. Phase 3 stays gated.

1. Optional leftovers (not Phase 1 blockers): `openLogs` refuse fleet; TUI `R` refuse when `rows` empty; exec-path twin of id-prefix collision.
2. Do not unlock `--id --restart`.
3. Compose-only folders still have empty stack — expected until Phase 2.
