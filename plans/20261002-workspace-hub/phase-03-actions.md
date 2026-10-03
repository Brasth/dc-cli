# Phase 3 — Project actions

## Requirements

1. Shared Go package `internal/actions` + compiled `cmd/dc-actions`. `bin/dc-actions` shell fallback: help works, other verbs print "compiled dc-actions required" (exit 1). `dc actions` dispatches.
2. Personal: `${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256(canonical workspace)>.json`. Shared: `<workspace>/.dc/actions.json`.
3. Schema: `{"schemaVersion":1,"actions":[{"id","label","argv":[...],"service"?}]}`. Unique non-empty id, non-empty label, non-empty argv of strings (argv[0] non-empty), optional non-empty service. Unknown fields rejected. Whole file valid or whole file rejected.
4. Personal overrides shared by id. Malformed or untrusted shared never blocks valid personal actions.
5. argv runs directly; no implicit shell.
6. Trust = canonical workspace + sha256 of raw shared bytes. Any byte change revokes.
7. Trust preview lists ALL shared actions (argv + service), including ones a personal action overrides.
8. CLI:
   - `dc-actions list [workspace] [--json]` → JSON `{schemaVersion, workspace, actions:[{id,label,argv,service,source,enabled}]}`.
   - `dc-actions run ID [workspace]` → child status preserved; child stdout/stderr untouched; our status lines on stderr.
   - `dc-actions trust [workspace] [--yes]` → preview first; non-TTY refuses without `--yes`.
   - `dc-actions untrust [workspace]`.
   - Exit: 0 ok; 2 bad usage / bad config / non-TTY trust without `--yes`; 1 unavailable / missing / disabled.
9. Runs through `dc-exec --no-start` (new flag). `--no-start` in every branch: `--service` (docker exec), compose-kind (compose exec), default (official `devcontainer exec`). Target not running → exit 1. `--id` + `--no-start` refused (exit 2). Default `dc-exec` unchanged.
10. Execute the exact validated + approved bytes loaded once (no re-read between check and run).
11. TUI `c`: action picker. Disabled shared → review screen (all shared commands) → `y` enables. Enter runs one foreground action (leave → return). Ctrl+C reaches the child; board returns with label + exit status, soft refresh.
12. Packaging: `install.sh` builds/stages `dc-actions` like `dc-tui`; `scripts/pack-release.sh` ships the binary.
13. No host actions. No automatic up.

## Files

- `internal/actions/config.go`, `trust.go`, `runner.go`, `cli.go` (+ tests).
- `cmd/dc-actions/main.go`, `version.go` (+ test).
- `cmd/dc-tui/action_picker.go` (+ test), `model.go`, `view.go`, `actions.go`.
- `bin/dc-actions`, `bin/dc`, `bin/dc-exec`, `install.sh`, `scripts/pack-release.sh`.
- `tests/actions/run.sh`, `tests/exec/run.sh`, `tests/dc/run.sh`, `tests/install/run.sh`.

## Risks

- Actions can mutate container data (they are arbitrary commands). Docs say so.
- `devcontainer exec` semantics owned by the official CLI; `--no-start` only refuses when the labeled app is not running.

## Status: done

- `internal/actions`: config (strict whole-file validation), trust (`actions-trust.json`, flock + atomic write, `ApproveReviewed` re-hashes before approving), runner (`dc-exec --no-start` argv, exit-code mapping incl. 128+signal), cli (`Main`).
- `cmd/dc-actions` thin wrapper; `bin/dc-actions` shell fallback (help works, other verbs exit 1 "compiled dc-actions is required"); `dc actions` dispatches.
- `dc-exec --no-start`: fresh status read right before exec in service / compose-kind / app branches; stopped or missing → exit 1; `--id`/`--restart` with it → exit 2; status line to stderr. Default behaviour unchanged.
- Board `c`: searchable picker, off entries → review of all shared commands (overrides marked, exact argv + target, sha256) → `y` enables exactly that hash; enter re-checks trust then leaves for one foreground action; result `action <label> · exit N`; result for a different workspace reports without refreshing.
- Trust store: `${XDG_STATE_HOME:-~/.local/state}/dc-cli/actions-trust.json`.
- Packaging: `install.sh` stages prebuilt → Go build → shell fallback (`DC_SKIP_TUI_BUILD=1` skips the build); `pack-release.sh` builds `dc-actions`.

### Corrections (review feedback)

- Review screen: wrapped, untruncated, scrollable; `y` only after the end was shown; control characters escaped.
- Enter: async fresh re-read + exact-match verification (source, action, shared bytes) before leaving; only the verified entry executes.
- Picker results tagged with workspace + picker generation.
- ids are opaque (regex removed); NUL in argv/service, `null` argv elements, `service: null` / non-string rejected as config errors.
