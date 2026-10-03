# Phase 2 — Workspace registry + `w` picker

## Requirements

1. `w` opens a picker of recent workspaces. `f` (fleet) unchanged.
2. Switching (picker, fleet enter) goes through one helper: closes logs, stats stream, nets, events, pickers; cancels probes; hard reload.
3. Paths canonical absolute (`filepath.Abs` + `EvalSymlinks`); symlinked spellings dedup to one entry.
4. Record every folder the board successfully opened, config or not.
5. File: `${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json`, `schemaVersion: 1`.
6. Retention: latest 30 non-favorites by `lastOpen`; favorites unlimited.
7. Writes: read-modify-write under a short cross-process `flock` (bounded wait), atomic tmp+fsync+rename.
8. Malformed file: left untouched, nonfatal warning, writes skipped. Newer schema: read-only warning.
9. Order: favorites first, then `lastOpen` descending. Same base name → show parent path to disambiguate.
10. Missing folders stay listed (marked); opening one shows an error, board stays.
11. Keys: typing filters, ↑/↓ move, Enter open, Ctrl+F favorite, Ctrl+D forget, Esc back.
12. Picker works when Docker is unavailable (host recover screen).
13. No disk scanning, no start, no project edits.

## Files

- `internal/localstate/files.go` (+ test) — `StateDir`, `ConfigDir`, `AtomicWrite`, `WithLock`.
- `internal/workspaces/registry.go` (+ test) — `Canonical`, `Exists`, `Load`, `Record`, `SetFavorite`, `Forget`, `Sorted`, `Labels`.
- `cmd/dc-tui/workspaces.go` (+ test) — picker state + keys + switch helper.
- `cmd/dc-tui/workspace_view.go` — picker render.
- `cmd/dc-tui/model.go`, `view.go`, `main.go`, `actions.go` — wire.

## Tests

Concurrency (parallel writers, no lost update), corrupt file preserved, retention 30 + favorites, dedup through symlink, sort order, missing folder visible + open error, picker keys, picker under host block.

## Status: done

- Lock = `flock` on `workspaces.json.lock` (2s wait), atomic tmp+fsync+rename, mode 0600.
- Malformed / empty / bad schemaVersion / relative path → file untouched, warning, writes refused. schemaVersion > 1 → read-only.
- Picker rows: ★ favorite, label (parent path when base names clash), dim path, `missing` / `current` tags.
- `f` fleet toggle and fleet enter now use the shared `switchContext` (closes logs/top/nets/events, kills probes, clears data).
- Recording: board startup (non-fleet) and every switch, async, nonfatal.
