# Workspace hub — implementation report

Status: phases 1–4 implemented and locally verified (Go race/vet, all 22 shell suites incl. the opt-in smoke skip, live Colima smoke 27/27). GitHub CI run of the new workflow: not yet observed. Live Linux smoke: NOT performed.

## Phase 1 — performance (done)

Changes
- `cmd/dc-tui/probes.go`, `process_unix.go`: read-only probes run in their own process group; cancel/timeout SIGKILLs the whole tree. Per-probe 5s, essential aggregate 10s. Probe session per context; each discovery generation cancels the previous one.
- `cmd/dc-tui/discovery.go`: essentials (`reloadMsg`: host, rows, stack, engine) paint first; disk / ports / nets are separate `optionalMsg`s tagged workspace+fleet+engine+generation (stale dropped). Section state loading / unavailable / stale. Disk cache 30s per engine; `r` and prune invalidate. One discovery in flight + one queued (`requestRefresh`). `quit`/`shutdown` reap logs, stats stream, probes.
- Async stay commands (`stayDoneMsg`, one at a time, no timeout). Pulse tagged workspace+engine.
- Stack row shell: `dc-exec --service <id> WS` → membership re-resolved fresh by the shell (was `--id`).
- `lib/dc-common.sh`: `dc_inspect_snapshot` (one `docker inspect` for N ids), `dc_labeled_folders` (labels via `docker ps`, no inspect), `dc_ports_snapshot`; `dc_stack_rows`, `dc_ids_for_workspace`, compose ids, `dc_ls_json`/`dc_ls_table` use them. Mutation paths keep fresh reads (each call re-reads; `dc_ensure_running`/restart inspect fresh). `dc_json_escape` Bash fast path for printable ASCII (byte-identical to `json.dumps`); python3 only for non-ASCII/control chars.
- dc-ls JSON rows now unit-separated internally, so empty `local_folder` (compose-kind) no longer shifts columns.
- `tests/lib/fake-docker`: multi-id inspect + batch template rendering, fork-free reads.

Measured (fake-docker, `bash tests/performance/run.sh`)

| scenario | n | inspects | baseline | docker calls | latency |
|---|---|---|---|---|---|
| devcontainer-stack | 1 | 1 | 17 | 3 | 181ms |
| compose-stack | 1 | 1 | 6 | 4 | 344ms |
| compose-ls | 1 | 1 | 5 | 5 | 474ms |
| devcontainer-stack | 10 | 1 | 52 | 3 | 530ms |
| compose-stack | 10 | 1 | 60 | 4 | 575ms |
| compose-ls | 10 | 1 | 50 | 5 | 574ms |
| devcontainer-stack | 50 | 1 | 252 | 3 | 1427ms |
| compose-stack | 50 | 1 | 300 | 4 | 1246ms |
| compose-ls | 50 | 1 | 250 | 5 | 1505ms |

Latency is fake-docker on macOS; before the escape fast path 50 containers took ~20s (python3 spawn per JSON field).

## Phase 2 — workspace registry + `w` picker (done)

- `internal/localstate/files.go`: `StateDir`/`ConfigDir` (absolute XDG only), `AtomicWrite` (tmp+fsync+rename+dir fsync), `WithLock` (`flock`, bounded wait, `ErrLockTimeout`).
- `internal/workspaces/registry.go`: `${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json`, schemaVersion 1. Canonical paths (Abs + EvalSymlinks). Latest 30 non-favorites, unlimited favorites. Sort favorites first, then lastOpen desc. Labels add parent path (~) on base-name clash. Malformed → preserved, nonfatal `MalformedError`, writes refused; newer schema → read-only. Forget is registry-only.
- `cmd/dc-tui/workspaces.go`, `workspace_view.go`: `w` picker (also on the Docker-down recover screen). Typing filters (q is a character), ↑/↓ (ctrl+p/n), enter opens, ctrl+f favorite, ctrl+d forget, esc back, ctrl+c quits. Missing folders listed + `missing`; enter shows an error and stays. Registry ops async (`wsListMsg`).
- Shared `switchContext` used by picker, fleet enter, `f` toggle: closes logs, stats stream, nets, events hook, picker; kills probes; clears context data; hard reload. `f` behaviour otherwise unchanged.
- Recording: board startup (non-fleet, any kind incl. no config) and each switch; never starts anything.

## Phase 3 — project actions (done)

- Shared package `internal/actions` used directly by the board (`c`) and `cmd/dc-actions`.
- Files: personal `${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256(canonical path)>.json`; shared `<ws>/.dc/actions.json` (read-only). schemaVersion 1; unique non-empty opaque id (no character rule; never used as a path), non-empty label, non-empty array of strings for argv with non-empty argv[0], optional non-empty string service. Rejected (whole file): unknown fields, trailing data, `null` argv elements or `argv: null`, `service: null` / non-string, NUL in argv or service.
- Precedence: personal id wins. Invalid or untrusted shared never blocks personal runs.
- Trust: canonical workspace + sha256 of raw shared bytes in `${XDG_STATE_HOME}/dc-cli/actions-trust.json` (localstate lock + atomic write). Never trusted on open. Review/preview lists all shared actions incl. overridden, exact argv + target. Approval re-reads and re-hashes the file (refuses if changed). Malformed trust file → nothing trusted, file preserved.
- Run: one read of both files; the validated + trust-checked entry is what executes (board re-checks approval for that hash at Enter). argv passed through `dc-exec --no-start [--service S] WS -- argv` — no shell. Child stdio inherited; our status lines on stderr.
- CLI exit: list 0 / 2 config-or-usage (still prints valid actions) ; 1 missing workspace / action, disabled shared, not running target; run returns child status (signal → 128+n). Trust non-TTY without `--yes` → preview + exit 2.
- `dc-exec --no-start`: service (fresh `dc_stack_resolve` membership + fresh status), compose-kind (resolve + status, no `compose start`), app (fresh labeled id + status, official `devcontainer exec`). Stopped / missing / forged → exit 1 with no start of any kind. `--id --no-start`, `--restart --no-start` → exit 2.
- Board: `c` picker (type filter, ↑/↓, enter run, ctrl+t review, esc). Review is a scrolled viewport of every shared action (overridden ones too): target + complete argv hard-wrapped, never truncated; ↑/↓ pgup/pgdn g/G; `y` is refused until the last line has been on screen. ids / labels / argv with control characters are shown escaped (board and CLI).
- Enter runs a pre-run check off the Update loop (async `actionReadyMsg`): both files are read again with current trust; the selection must still resolve to the same source and identical action, and for shared the same bytes that were shown (a changed file is refused even if re-approved). Revoked trust, edited personal action, source flip (personal appearing over shared) → refused with a warning; nothing runs. A broken shared file does not block a valid personal action. Only the freshly verified entry executes; no further file read before exec.
- Picker loads / approvals / pre-run checks carry workspace + picker generation (bumped on open / close / switch); stale results are dropped. The picker has no engine-dependent data, so engine is not part of its tag; the action result is tagged by workspace (different workspace or fleet → report without refresh).
- Foreground run: leaves for one action (Bubble Tea ignores SIGINT while the child has the terminal, so Ctrl+C stops only the action); status `action <label> · exit N` + refresh; result tagged with workspace (stale → no refresh). Switch closes the picker; stale loads dropped. Fleet refuses.
- Packaging: install.sh (prebuilt → Go build → shell fallback), pack-release.sh builds dc-actions; shims include dc-actions.
- Docs: README (command row, project actions section), guides tui.md (keys, FAQ, actions section) + install.md, skill/SKILL.md 9e — all state actions can modify container data.

## Tests run (this run, macOS arm64, Go 1.26.1, Bash 5.3)

- `go vet ./...` — clean.
- `go test -race -count=1 ./...` — ok (`cmd/dc-tui`, `internal/localstate`, `internal/workspaces`).
- Covered: loading/essential-first, per-probe + aggregate deadlines, stale tag drop (workspace/engine/gen/fleet), cancel on switch/quit, real grandchild kill on cancel + timeout, disk cache TTL/engine/invalidate, async single stay, refresh coalescing; registry malformed/empty/bad schema/relative path/newer schema, unwritable dir, 20 goroutines + 6 processes concurrent writers (no lost update), retention 30 + favorites, sort, symlink dedup, missing folder, labels; picker filter/arrows/enter/favorite/forget/esc, Docker-down picker, malformed nonfatal, kind-none record, switch teardown, fleet toggle/enter.
- Shell: all 20 suites under `tests/*/run.sh` ok after Phase 1 (incl. new `tests/performance/run.sh`); after Phase 2: dc, exec, safety, kind, compose, stats, db, performance ok.
- `DC_SHELL_PERF=1 go test -run ShellDiscovery ./cmd/dc-tui` passed earlier (counts as above).
- Phase 3 correction run: `go vet ./...` clean; `go test -race -count=1 ./...` ok (new: opaque ids incl. spaces, NUL/null/service-null rejection, control-char escaping, review of 30 long wrapped commands at 60×20 all reachable + y refused before the end, shared changed-after-open refused even when re-approved, revoked trust refused, edited personal refused, precedence flip refused, personal runs with broken shared, stale pre-run/load generation dropped, CLI opaque-id run, CLI NUL refused before exec). Shell: actions 7, exec 18.
- Phase 3 first run: `go vet ./...` clean; `go test -race -count=1 ./...` ok (adds `internal/actions`, `cmd/dc-actions`, `action_picker_test.go`). Shell: dc 10/10, copy ok, exec 18 (10 new `--no-start` cases: running/stopped/forged service, app stopped/missing/running+exit, compose-kind stopped/running, `--id`/`--restart` refusal, default still starts), actions 7 (new e2e with compiled binary: quoting + exit propagation, service docker exec, stopped + forged refused with zero start/compose start/dc-up, trust gate + non-TTY + revoke on edit, list JSON precedence + config exit 2, `dc actions` fallback), install 18/18 (2 new: fallback, source build), safety 20/20. `pack-release.sh` linux/amd64 tarball contains bin/dc-actions + bin/dc-tui.

## Phase 4 — activity timeline (done)

- Files: `cmd/dc-tui/activity.go` (session + stream lifecycle, ingest, debounce, reconnect, entries), `activity_stream.go` (pinned `docker events`, own process group via `setProbeProcAttr`, bounded line reader, parse), `activity_membership.go` (ownership contract, pinned inspect, short-id normalization), `activity_view.go` (render + keys). Wired in `model.go` (msgs, `v`, view keys, mouse wheel), `actions.go` (`startLeave` pauses), `action_picker.go` + `execDoneMsg` (resume, then the reload is the rescan), `view.go` (overlay, hints, more panel), `main.go` (help). `discovery.go` / `workspaces.go` already called the hooks.
- Stream: one `docker [--context X | --host H] events --format '{{json .}}' --filter type=container --filter event=…` for create/start/stop/die/restart/destroy/oom/health_status. Engine identity from the discovery snapshot is pinned by flag; `DOCKER_HOST` / `DOCKER_CONTEXT` are removed from the child env. Unknown engine → no stream. Lines > 64KB dropped whole; malformed / other types / bad ids / no time ignored. Entry keeps time, service-or-name, short description (exit code digits, health keyword only) — no attributes, env or logs.
- Session id (global counter) + stream generation tag every msg (with workspace + engine); stale → dropped. Session ends (stream closed and reaped, bounded 2s; in-flight inspects cancelled) on switch, fleet, engine change (timeline cleared + note), Docker-down (pause, rescan on recovery), every foreground leave (`startLeave`: shell/start/files/upgrade/action/stack exec), quit/shutdown. Return from foreground resumes before the soft reload (= rescan), with a gap line.
- Membership: known ids = last successful discovery (stack rows; devcontainer app rows; compose-kind rows only give project names because they may include sidecars), retained for the session (cap 512, least recently verified dropped) so `destroy` attributes after the container is gone. Snapshot ids are short (`docker ps -q`), so they are normalized to full ids by one pinned inspect; matching is exact full-id only. Unknown id → one pinned read-only `docker inspect --type container` (≤16 ids per batch, probe budget 5s, killed with its tree on cancel). Accept: devcontainer `devcontainer.local_folder` is this folder (dc_same_workspace rules incl. `.devcontainer` trim + realpath) — that app's compose project becomes verified; sibling whose project is verified. Compose-kind: working_dir / config_files proof AND verified project. Sidecars (`dc.forward.for` in event attrs or inspect) never. Undecidable (project unverified) → waits for a discovery that started after the event (and for normalization); still unknown → rejected. Unknown destroy → dropped (nothing to inspect). Verdict cache bounded 512; pending bounded 100.
- Debounce: every accepted event (health too) arms one 300ms tick per window → `requestRefresh` (one in flight + one queued).
- Reconnect: 1s, 2s, 4s, then 10s; one loss gap line, "reconnected — rescanning" line + full rescan on reconnect; an event (or ≥10s lifetime) resets the backoff.
- View `v`: j/k ↑/↓ pgup/pgdn/space g/G, mouse wheel, `c` clears, esc/q/v back, ctrl+c quits. Fleet refuses `v`. Latest 200, time-sorted (engine timestamps).
- Parent findings applied: host engine inspect pinned by `--host` with env cleared (test with inherited `DOCKER_CONTEXT`/`DOCKER_HOST`), verified additions bounded, health goes through the coalesced refresh, `internal/actions` `--` end-of-options (+ usage text, shell fallback usage, docs), short-id normalization, test emitter deadlock (async writes; production channel stays bounded at 32).

## Final CI / docs / smoke

- `.github/workflows/ci.yml` linux: executable bits for every `bin/*` + `tests/*/run.sh`; `bash -n` over every shell file (install, bin, lib, scripts, test libs, suites); one loop runs every `tests/*/run.sh` (adds actions, performance, compose, kind, db, stats, workspace-hub-smoke [skips without opt-in]); `go vet ./...`; `go test -race -count=1 ./...`; build dc-tui + dc-actions; source install checks compiled `dc-actions` + `dc actions`; release kit contains compiled `bin/dc-actions`, prebuilt install runs it (`list --json`, whitespace-tolerant grep). New macos job: Homebrew Bash by explicit path (asserts Bash 5 there and 3.2 at /bin/bash), installer Bash 3 guard (re-exec + `DC_BASH_MAJOR=3` refusal), go vet + race, install/dc/actions/exec/upgrade/safety under `$BASH5`, source install (compiled dc-tui + dc-actions), darwin-arm64 kit + prebuilt install. Linux/macOS steps simulated locally in temp HOME/PREFIX (macOS ones); not yet observed on GitHub.
- Docs: README (board `v`, `run -- -id`), `site/.../tui.md` (description, FAQ, keys row, Activity section, `--` example, updated date), `skill/SKILL.md` (9e `run [--] ID`, board `v` read-only), `dc-tui --help`, `bin/dc-actions` fallback usage. install.md already accurate (no change). No telemetry, daemon, scanning, host actions, startup preview or sandbox profile changes.
- `bin/dc-exec`: official-CLI exec into a dc-try sandbox (kind=none) now passes the sandbox's own `--override-config` from dc-cli state (pre-existing bug, also in HEAD: `devcontainer exec` failed for configless folders; found by the live smoke). Regression case in `tests/exec/run.sh`.
- `tests/workspace-hub-smoke/run.sh`: opt-in `DC_LIVE_SMOKE=1`; local image only (never pulls; asserts no new image), temp root under `$HOME/.cache` (Colima shares only home), isolated XDG config/state/data, unique compose project + per-run label + devcontainer.local_folder under the temp root; cleanup removes only those (plus their forward sidecars / `vsc-<run>-*` images). Covers compose-kind, devcontainer, dc-try sandbox: actions running (exit propagates, stdout), `--` id, stopped → exit 1 with State/StartedAt unchanged; sandbox override isolated + real `~/.local/state/dc-cli/try` unchanged + membership exactly its container; live Go test (`TestLiveActivity`, skipped unless the script sets env) for compose-kind and sandbox: pinned stream, known restart attributed, verified new sibling accepted, forged (wrong working_dir) / sidecar / other-folder app excluded, switch reaps the real process.

## Tests run (phase 4 run, macOS arm64, Go 1.26.1, Bash 5.3, Colima Docker 29.5.2 linux/arm64)

- `go vet ./...` clean; `go test -race -count=1 ./...` ok (all 5 packages). Activity tests also `-race -count=3`.
- New Go tests: event parse (no attribute retention, health keyword whitelist, malformed / other type / bad id / no time), overlong + malformed lines, engine pin args, real fake-docker stream pinned (`--context` argv, env cleared) and reaped incl. grandchild, stream error end, pinned inspect ignores inherited env (host + context), board lifecycle reaps real process tree on switch / fleet / leave / quit / engine change; model: known accepted without inspect, destroy of known id after it is gone, unknown verified by inspect (sibling ok; event-labelled sidecar not inspected; forged rejected; verdict cache), inspect-labelled sidecar rejected, new app verifies its project for a sibling in the same batch, await resolved by later discovery (accepted / rejected), short ids normalized not prefix-matched, compose-kind working_dir + project, cap 200 + time order, debounce (20 events / 10 health → 1–2 discoveries), reconnect backoff 10/20/40/100ms scaled + one gap + reconnect marker + rescan + reset, real-scale 1/2/4/10/10/10s, stale msgs (session / gen / engine), switch clears + reaps, fleet no timeline (toggle, manual refresh, `v` refused), engine change restarts pinned + clears, foreground leave reaps + resumes + gap, action leave/return, quit, Docker-down pause + recovery, view keys, inspect bounded by probe timeout, known cap keeps newest. The test pump waits for every cmd goroutine after shutdown (no leaks).
- Shell: all 22 `tests/*/run.sh` ok earlier this run (21 + smoke skip); after the dc-exec change: exec 19, try 11, actions 7. `bash -n` over every shell file ok. `tests/exec/run.sh` was mode 644 in HEAD (every other suite 755); made executable so the new "every suite executable" CI gate holds.
- Smoke isolation fix (after parent check): with `DC_SMOKE_DIR` inside a Git checkout (parent ran it under `plans/…/reports` in this repo), `dc_resolve_workspace` walked up to the dc-cli git root and the compose step refused ("not a workspace") — no project mutation. The smoke now exports `GIT_CEILING_DIRECTORIES=<canonical temp root>`, unsets inherited `GIT_DIR` / `GIT_WORK_TREE` / `GIT_COMMON_DIR` / `GIT_INDEX_FILE` / `GIT_OBJECT_DIRECTORY` / `GIT_NAMESPACE` (smoke process only) and, before any container step, refuses to run unless each smoke folder resolves to itself and is not inside a Git work tree. Public workspace resolution unchanged. Re-run of the exact parent command `DC_LIVE_SMOKE=1 DC_SMOKE_DIR=…/plans/20261002-workspace-hub/reports bash tests/workspace-hub-smoke/run.sh`: 27/27; after it 0 containers, no new images, no smoke networks, temp dir removed, `git status` entry count unchanged (47). Verified separately: a subfolder of a temp dir inside the repo resolves to the repo root without the ceiling and to itself with it.
- Live Colima smoke `DC_LIVE_SMOKE=1 bash tests/workspace-hub-smoke/run.sh`: 27/27; after it 0 containers, image count unchanged (4), no smoke networks or temp dirs. An earlier attempt pulled `alpine/socat:latest` via dc-up's forward step (it was not present before); that image was removed and the smoke now uses `--no-forward` and fails on any new image.

## Not done / concerns

- Live smoke on a Linux host: NOT performed. The Colima engine is Linux (linux/arm64 VM), but the host, CLI and wrappers ran on macOS. CI Linux covers deterministic fixtures only.
- New CI workflow (linux loop + macos job) not yet observed on GitHub; steps were simulated locally only. `actions/checkout@v7` / `setup-go@v7` kept as before.
- Timeline order follows engine timestamps: Docker 29.5.2 stamped one `create` ~79ms after its `start` (seen in `docker events`), so "started" can list before "created".
- While logs / top / nets are open the stream keeps running and accepted events still trigger a soft refresh (board views, not a foreground leave).
- `dc_stack_json` still reads TSV with tab IFS (pre-existing): an empty service label shifts `image` into `service`. Not changed.
- Picker tile not added to the button grid (key + hint + help only) to keep existing hitbox tests/layout stable.

## Unresolved questions

- None blocking. Trust store placed in state dir (`actions-trust.json`), per brief.
