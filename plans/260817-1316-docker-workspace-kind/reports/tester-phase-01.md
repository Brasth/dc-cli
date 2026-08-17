---
title: "Tester — Phase 1 owned stack verbs"
status: pass
created: 2026-08-17
updated: 2026-08-17
---

# Tester Phase 1 — owned stack verbs (re-run after three-pass matcher)

Scope: TUI cursor logs + `R` restart, `dc-exec --service --restart` membership, unlabeled take-ports unchanged. Re-run after review High: three-pass stack match (`dc_stack_resolve`) + `restart-id-prefix-collision`.

No product source changed by tester. No assertions weakened.

## Test Results Overview

| Suite | Command | Result | Pass | Fail | Skip | Time |
|---|---|---|---:|---:|---:|---|
| TUI unit | `go test -count=1 -v ./cmd/dc-tui` | PASS | 70 | 0 | 0 | 0.345s |
| exec | `bash tests/exec/run.sh` | PASS | 8 | 0 | 0 | <5s |
| safety | `bash tests/safety/run.sh` | PASS | 19 | 0 | 0 | ~few s |

**Total: 97 pass / 0 fail / 0 skip.**

Prior run was 96 (7 exec). New case `restart-id-prefix-collision` is the 8th exec test.

Required command was `go test ./cmd/dc-tui`. Ran uncached `-count=1 -v` so timing/count is not a cache hit.

## Phase 1 gates

### `go test ./cmd/dc-tui`

| Required | Test | Result |
|---|---|---|
| cursor ≠ `rows[0]` logs | `TestOpenLogsUsesStackCursor` — cursor=1 → follow `db1`, not `app1` | PASS |
| empty-stack fallback | `TestOpenLogsEmptyStackFallsBackToRows0` — cursor=3, no stack → `rows[0]` `abc123` | PASS |
| fleet `l` refuse | `TestFleetLogsRefuse` — no follow, no `logOpen` | PASS |
| `R` sibling stay | `TestRestartSiblingStay` — stay `dc-exec --service db --restart /tmp/app`, `leaving=""` | PASS |
| `R` app-row refuse | `TestRestartAppRowRefuse` — no stay, status has `u/s` | PASS |
| `R` fleet refuse | `TestRestartFleetRefuse` — no stay | PASS |
| lowercase `r` still reloads | `TestReloadStillLowercaseR` — no stay, `cmd != nil` | PASS |

Also PASS (not in required list): `TestLogsButtonEnabledFromStackCursor`, `TestRestartEmptyStackRefuse`, plus 61 other TUI cases. 70/70.

### `bash tests/exec/run.sh`

```
  ok  term-env
  ok  color-rc-hl
restart  db  db-1  db1
  ok  restart-sibling
  ok  restart-unknown
  ok  restart-id-banned
  ok  restart-no-app
restart  db  db-1  ffff1111
  ok  restart-id-prefix-collision
  ok  restart-needs-service

ok  8
```

| Required | Case | Assert | Result |
|---|---|---|---|
| matching sibling → `docker restart` | `restart-sibling` | `log_has '^restart db1$'` | PASS |
| unknown service no restart | `restart-unknown` | rc≠0, `log_lacks '^restart '` | PASS |
| `--id --restart` exits 2 | `restart-id-banned` | `rc==2`, no restart | PASS |
| no labeled app refuse | `restart-no-app` | rc≠0, no restart | PASS |
| `--restart` without `--service` exits 2 | `restart-needs-service` | `rc==2`, no restart | PASS |
| **three-pass: service `db` ≠ app id `db*`** | `restart-id-prefix-collision` | app `db12ffff` + svc `db` → `restart ffff1111` only, no `restart db12ffff` | PASS |

Collision stdout was `restart  db  db-1  ffff1111`. Fake-docker log must contain `^restart ffff1111$` and must not contain `^restart db12ffff$`. Both held. Review High (OR-per-row first `db*` hex prefix steals compose service `db`) is closed for restart.

`dc-exec --service` (non-restart) now also calls `dc_stack_resolve`. Same three-pass. No extra exec-collision case in this suite; restart case is the required gate.

### `bash tests/safety/run.sh`

```
19/19 passed
```

Unlabeled take-ports still present and green: `ok  port takeover unlabeled`. Other take-ports cases unchanged (foreign, ambiguous compose, same workspace, sidecar, sidecar inspect-unknown). No unlabeled mutate introduced by the matcher change.

## Coverage Metrics

Not generated. Required commands did not include `go test -cover`. Phase 1 paths + collision case exercised by named tests.

Matcher under test: `dc_stack_resolve` in `lib/dc-common.sh` — pass 1 exact service, pass 2 exact name, pass 3 id prefix. `dc_restart_in_stack` uses that helper. Miss still fail-closed (no inspect of strangers).

## Failed Tests

None.

## Performance Metrics

- `go test ./cmd/dc-tui`: 0.345s uncached. All cases ~0.00s. No slow tests.
- `tests/exec/run.sh`: 8 fake-docker cases, fast. Collision case added one extra `dc-exec --restart`.
- `tests/safety/run.sh`: 19 harness cases, fast.

## Build Status

No production build run (not required). Go tests compiled and passed. Bash harnesses sourced `dc-up` / `dc-exec` without syntax fail.

## Critical Issues

None.

## Recommendations

- Review High (three-pass + collision test) now empirically green. Phase 1 success criterion “`--service db` cannot hit another stack member whose id is a `db*` prefix” met.
- Optional: add an exec (non-restart) twin of `restart-id-prefix-collision` if you want the shared resolver covered on the `docker exec` path too. Not required for this gate.
- `restart-unknown` / `restart-no-app` still assert `rc≠0`, not exact 2. Fine.
- Review Medium leftover (fleet guard in `openLogs`, empty-`rows` app-row refuse) not re-tested as new cases; existing TUI refuse tests still PASS.

## Next Steps

1. Phase 1 success criteria met after matcher fix. Ready for re-review / Phase 2.
2. Do not unlock `--id --restart`.

## Unresolved questions

None.
