# QA: dc-tui UX pass

Date: 2026-08-15 17:40 +0700
Host: macos Darwin arm64, Go 1.26.6 (CI pins 1.24)
Scope: focused `cmd/dc-tui` suite + help + gofmt + `bash -n bin/dc-tui`. No docker mutate. No file edits to source.

## Test Results Overview

| # | Check | Result |
|---|---|---|
| 1 | `gofmt -l cmd/dc-tui` | PASS (empty) |
| 2 | `go test -count=1 -v ./cmd/dc-tui` | PASS 15/15 |
| 3 | `go test -count=1 -covermode=atomic ./cmd/dc-tui` | PASS 35.1% stmts |
| 4 | `go build -o /tmp/dc-tui ./cmd/dc-tui` | PASS 5,150,882 B |
| 5 | `/tmp/dc-tui --help` primary / rm confirm / j/k | PASS |
| 6 | `/tmp/dc-tui --unknown-flag` | PASS exit 2 |
| 7 | `bash -n bin/dc-tui` | PASS |

Total: 15 unit tests run, 15 passed, 0 failed, 0 skipped. 7 checks above all PASS.

### Unit tests

| Test | Result | Time |
|---|---|---|
| TestRenderPrimaryEqualWidth | PASS | 0.00s |
| TestRenderPrimaryThenMeta | PASS | 0.00s |
| TestRenderNarrowWrapsMore | PASS | 0.00s |
| TestStartHitboxAfterHeader | PASS | 0.00s |
| TestBenignExecErr | PASS | 0.00s |
| TestDisabledStartReason | PASS | 0.00s |
| TestFleetKeyNotSilent | PASS | 0.00s |
| TestConfirmRmCancel | PASS | 0.00s |
| TestLeaveOnShell | PASS | 0.00s |
| TestLeaveIgnoresSecondShell | PASS | 0.00s |
| TestLeaveOnLogs | PASS | 0.00s |
| TestCursorClampAndEnterFleet | PASS | 0.00s |
| TestStayCmdSplitsStatusAndErr | PASS | 0.00s |
| TestDisabledStartTile | PASS | 0.00s |
| TestViewHasPrimaryHint | PASS | 0.00s |

`ok github.com/Canvilled/dc-cli/cmd/dc-tui 0.250s` (rerun 0.514s w/ cover)

## Coverage Metrics

`go test -covermode=atomic`: **35.1%** statements. Below typical 80% bar. Up from earlier 13.6% report.

### High (exercised by this pass)

| Func | Cover |
|---|---|
| `benignExecErr` | 100% |
| `refuse` | 100% |
| `kv` | 100% |
| `renderRow` | 94.4% |
| `renderGroups` | 92.9% |
| `moveCursor` | 83.3% |
| `compactLines` | 83.3% |
| `tileStyle` | 76.9% |
| `View` | 75.0% |
| `hasDevcontainer` | 75.0% |
| `startLeave` | 66.7% |
| `openRow` | 66.7% |
| `buttonGroups` | 66.7% |
| `runAction` | 56.2% |
| `layout` | 50.0% |
| `handleConfirmKey` | 44.4% |
| `activateRow` | 44.4% |
| `handleKey` | 41.4% |

### Zero / critical holes

| Func | Cover | Why it matters |
|---|---|---|
| `handleClick` / `clickKey` / `hitButton` / `hitRow` | 0% | click-board is the product |
| `runPending` / `execStack` / `stayCmd` | 0% | real start/shell/stop/open/rm-y |
| `leaveLine` / `formatStackRow` / `formatFleetRow` / `morePanel` | 0% | leave banner + row paint |
| `Update` / `Init` | 0% | tea loop, leaveTick, execDone |
| `main` / `resolveWorkspace` / `editorBin` / `pickEditor` | 0% | CLI entry |
| `shortID` / `trunc` | 0% | row formatting |

No branch coverage report beyond `-func`.

## Failed Tests

None.

## Performance Metrics

| Step | Time / size |
|---|---|
| `go test -count=1 -v ./cmd/dc-tui` | 0.250s |
| `go test -count=1 -covermode=atomic` | 0.514s |
| `go build -o /tmp/dc-tui` | instant; 5,150,882 B Mach-O |
| `dc-tui --help` | instant |
| slowest test | none (all 0.00s reported) |

No flaky rerun. `-count=1` used.

## Build Status

PASS. No compile warning. `gofmt -l cmd/dc-tui` empty.

CI (`.github/workflows/ci.yml`) already runs:

- `bash -n bin/dc-tui`
- `go test ./cmd/dc-tui`
- `go build -o /tmp/dc-tui ./cmd/dc-tui`
- installed `dc-tui --help`

Local equivalents of those four: PASS.

## Acceptance (this task)

| Criterion | Evidence | Result |
|---|---|---|
| all go tests pass | 15/15 PASS | PASS |
| help mentions primary start/shell/stop | `Primary: start (u)  shell (e)  stop (s)` | PASS |
| help mentions rm confirm | `Danger:  rm (x)    asks y/n before dc-down --rm` | PASS |
| help mentions j/k | `Rows:    j/k or arrows, enter (fleet = open folder, stack = exec)` | PASS |
| gofmt clean | `gofmt -l` empty | PASS |

View hint also asserted by `TestViewHasPrimaryHint` (`u start  e shell  s stop  j/k  enter`).

`TestConfirmRmCancel`: `x` sets `confirm=rm`, `confirmAction()=="dc-down --rm"`, `n` clears confirm, status `rm cancelled`, no exec.

## Critical Issues

None blocking ship of this pass.

## Recommendations

1. Add click hitbox tests (`handleClick` / `hitButton`) — board is mouse-first.
2. Cover `activateRow` stack path + `execStack` without exec (assert `leaving`/`pending` only).
3. Cover confirm `y` without calling real `dc-down --rm` (inject stayCmd / fake exec).
4. Cover `leaveTickMsg` → `runPending` with a stub command; do not run docker.
5. Paint tests for `leaveLine`, fleet/stack rows, `morePanel`.
6. Bash fallback `bin/dc-tui` help is stale vs Go (`j/k`, primary grouping). Plan said no bash rewrite this pass — ok, but fallback users see old menu.

## Next Steps

1. P0 none.
2. P1 click + stack-enter + confirm-y unit tests (no docker).
3. P2 leaveTick/runPending stub; more/leave paint.
4. P3 bash help parity after this ships, if fallback still used.

## Unresolved questions

- Confirm `y` never unit-tested; by design (would exec `dc-down --rm`). Need a seam?
- Local Go 1.26.6 vs CI 1.24. Any 1.24-only fail unknown here.
- README + `site/src/content/guides/tui.md` match not in this focused scope.
