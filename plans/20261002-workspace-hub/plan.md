---
title: "Workspace hub — fast board, workspace picker, project actions, activity timeline"
description: "Four phases on the board and wrappers. Faster first paint, recent workspaces, trusted per-project actions, live container timeline."
status: in-review
priority: P1
tags:
  - tui
  - performance
  - workspaces
  - actions
  - activity
blockedBy: []
blocks: []
created: "2026-10-02T23:10:00.000Z"
source: brief
---

# Workspace hub

Board stays a this-folder tool. No telemetry. No host actions. No automatic `dc up`.
Never edits project config. Ownership checks for mutations stay fresh (shell side).

## Phases

| # | Phase | File | Status |
|---|---|---|---|
| 1 | Performance: essential/optional split, cancel, deadlines, batch inspect | [phase-01-performance.md](phase-01-performance.md) | done |
| 2 | Workspace registry + `w` picker | [phase-02-workspaces.md](phase-02-workspaces.md) | done |
| 3 | Project actions: Go package, `dc-actions`, trust, `c` picker, `dc exec --no-start` | [phase-03-actions.md](phase-03-actions.md) | done |
| 4 | Activity timeline `v` from one `docker events` stream | [phase-04-activity.md](phase-04-activity.md) | done |

Order is strict: 1 → 2 → 3 → 4. Each phase builds on the shared switch/cancel path from phase 1.

## Shared decisions

| Decision | Value |
|---|---|
| Result tagging | every async result carries workspace + fleet + engine + generation; mismatches dropped |
| Read-only probes | own process group; cancelled (tree killed) on switch / quit |
| User actions | no timeout, never auto-cancelled |
| Context switch | one helper closes logs, stats stream, nets view, events stream, pickers; cancels probes |
| Local state | `internal/localstate`: atomic write (tmp + fsync + rename) + short `flock` lock |
| Registry path | `${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json` |
| Personal actions | `${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256(canonical path)>.json` |
| Shared actions | `<workspace>/.dc/actions.json`, disabled until trusted (hash of raw bytes) |
| Trust store | `${XDG_STATE_HOME:-~/.local/state}/dc-cli/actions-trust.json` |
| Action target | `dc-exec --no-start` only; never starts containers |

## Acceptance (all phases)

- `go vet ./...` and `go test -race ./...` pass.
- Every shell suite under `tests/*/run.sh` passes and runs in CI (Linux); macOS job runs state/wrapper/installer suites under Homebrew Bash.
- Stack discovery `docker inspect` count ≤ 50% of the recorded baseline at 1/10/50 containers.
- Release kit ships compiled `dc-actions`; source install without Go keeps the shell fallback (help works, run explains compiled binary is required).
- Live Linux + Colima smoke: run only if an engine is available; otherwise reported as NOT performed. (Colima on macOS: performed, `tests/workspace-hub-smoke/run.sh`, opt-in. Live Linux: NOT performed.)

## Reports

- [reports/implementation.md](reports/implementation.md)
