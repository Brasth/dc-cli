# Phase 4 — Activity timeline

Status: done (see [reports/implementation.md](reports/implementation.md)).

## Requirements

1. `v` opens a timeline for the selected workspace. Latest 200 entries, RAM only.
2. One `docker events --format '{{json .}}'` stream on the selected engine while the board is active on a workspace.
3. Events: create, start, stop, die, restart, destroy, oom, health_status.
4. Membership verified: known IDs from discovery (labeled app + stack siblings). Unknown IDs → one read-only inspect to verify workspace label / compose proof. `dc.forward.for` sidecars and unrelated containers excluded. Known IDs retained so `destroy` still attributes.
5. Entry = timestamp, name/service, description. Never env, never logs.
6. Relevant events trigger a soft discovery: debounce 300ms, one discovery in flight, one pending.
7. Stream loss: reconnect after 1s, 2s, 4s, 10s (cap 10s); gap marker entry; full rescan after reconnect.
8. Stream closed + reaped on context switch, engine change, exit, foreground leave. Resume + rescan on return.
9. Cleared on switch, entering fleet, and manual clear (`c` in the view). Fleet: no timeline.
10. Stream messages tagged with a stream generation; stale dropped.

## Files

- `cmd/dc-tui/activity.go` — session / stream lifecycle, ingest, debounce, reconnect, entries.
- `cmd/dc-tui/activity_stream.go` — pinned `docker events` process (own process group), bounded line reader, event parse.
- `cmd/dc-tui/activity_membership.go` — ownership contract, pinned read-only inspect, short-id normalization.
- `cmd/dc-tui/activity_view.go` — render + keys.
- `cmd/dc-tui/activity_test.go`, `activity_stream_test.go`.
- `cmd/dc-tui/model.go`, `view.go`, `actions.go`, `action_picker.go`, `main.go` — wire (`workspaces.go` / `discovery.go` already call the hooks).

## Tests

Ownership filter, sidecar exclusion, delete of known id, unknown id verify, reconnect backoff + gap marker + rescan, debounce coalescing, reap on switch/leave/quit, stale generation drop, 200 cap.
