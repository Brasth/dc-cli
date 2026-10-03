# Phase 1 — Performance

## Context

- Board discovery today: one goroutine runs host diagnose → `dc-ls` → `dc-exec --list` → `dc-df` → 2× `docker ps` → `dc-net`, then paints once (`cmd/dc-tui/actions.go` `reloadCmd`).
- Nothing is cancelled on switch/quit. No deadline. `stayCmd` blocks the Update loop.
- Shell discovery inspects each container several times (`lib/dc-common.sh` `dc_inspect_row`, `dc_stack_rows`, `dc_ids_for_workspace`).

### Baseline (fake-docker, this repo at fd01797)

| scenario | 1 | 10 | 50 |
|---|---|---|---|
| devcontainer stack `dc-exec --list --json` inspects | 17 | 52 | 252 |
| compose stack `dc-exec --list --json` inspects | 6 | 60 | 300 |
| compose `dc-ls --json` inspects | 5 | 50 | 250 |
| devcontainer `dc-ls --json` inspects (one app) | 5 | 5 | 5 |

## Requirements

1. Essentials = host check, rows (`dc-ls`), stack (`dc-exec --list`). Render as soon as they land.
2. Optionals = disk (`dc-df`), forwards (sidecar + published ports), nets (`dc-net`). Each its own async result.
3. Every result tagged workspace + fleet + engine + generation. Stale → dropped.
4. Read-only subprocess trees (own process group) killed on switch / quit.
5. Per-probe timeout 5s. Essential aggregate deadline 10s. User actions: no timeout.
6. Batch inspect: one `docker inspect` per discovery for N containers. Snapshot reused for reads only. Mutations (`--restart`, start, exec target) re-resolve membership fresh in the shell.
7. Disk cache 30s per engine. `r` invalidates. Prune invalidates.
8. Stay commands (`dc-db`, `dc-open`, `dc-down`, …) run async; the board keeps painting.
9. Optional section state: loading / unavailable / stale.
10. Tests + benchmarks at 1/10/50 containers record latency + call counts; ≥ 50% inspect reduction vs baseline.

## Files

- `cmd/dc-tui/probes.go` — `runProbe` seam, per-probe timeout, probe session (cancel on switch/quit).
- `cmd/dc-tui/process_unix.go` — process group + tree kill.
- `cmd/dc-tui/discovery.go` — essential/optional commands, tags, section states, disk cache, apply.
- `cmd/dc-tui/actions.go`, `model.go`, `view.go`, `host.go`, `urls.go`, `main.go` — wire in, async stay.
- `lib/dc-common.sh` — `dc_inspect_snapshot` batch; `dc_ls_json`, `dc_stack_rows`, `dc_ids_for_workspace`, compose ids use it.
- `tests/lib/fake-docker` — multi-id inspect + snapshot template.
- Tests: `cmd/dc-tui/discovery_test.go`, `probes_test.go`, `discovery_bench_test.go`, `tests/performance/run.sh`.

## Steps

1. Shell batch snapshot + fake-docker support; keep TSV/JSON contracts byte-compatible.
2. Probe runner + process group kill; session per context.
3. Split `reloadCmd` into essentials (`reloadMsg`, now with `engine`) + optional messages.
4. Section states + view lines; disk cache.
5. Async `stayCmd` (`stayDoneMsg`).
6. Stack exec: `dc-exec --service <id> WS` (fresh membership) instead of `--id`.
7. Tests, benchmarks, perf suite.

## Risks

- Template differences between real Docker and fake-docker: snapshot template uses only stable fields (`.Id .Name .State.Status .Config.Image .Config.Labels`).
- Killing a process group must never touch user actions: only probes use `Setpgid`.

## Status: done

- Inspects per stack discovery: 1 at 1/10/50 containers (baseline 17/52/252 devcontainer, 6/60/300 compose, 5/50/250 compose ls).
- `dc_json_escape` Bash fast path for printable ASCII (python3 only for non-ASCII/control): 50-container stack listing ~20s → ~1.4s on fake-docker.
- `tests/performance/run.sh` records inspects, docker calls, latency; fails above 50% of baseline.
- Go: `discovery_bench_test.go` (BenchmarkEssentials, BenchmarkShellStackDiscovery; TestShellDiscoveryInspectBudget gated by `DC_SHELL_PERF=1`).
