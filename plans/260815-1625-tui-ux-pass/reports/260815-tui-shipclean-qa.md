# QA: dc-tui ship-clean pass

Date: 2026-08-15 19:13 +0700
Host: macos Darwin 25.2.0 arm64, Go 1.26.6 (CI pins 1.24)
Scope: focused `cmd/dc-tui` suite + help + gofmt + `bash -n bin/dc-tui`. No docker/dc-up/dc-down/dc-prune. No network. No source edits.

## Test Results Overview

| # | Check | Result |
|---|---|---|
| 1 | `gofmt -l cmd/dc-tui` | PASS (empty) |
| 2 | `go test -count=1 -v ./cmd/dc-tui` | PASS 27/27 |
| 3 | `go test -count=1 -covermode=atomic ./cmd/dc-tui` | PASS 50.2% stmts |
| 4 | `go build -o /tmp/dc-tui ./cmd/dc-tui` | PASS 5,185,058 B Mach-O arm64 |
| 5 | `/tmp/dc-tui --help` start/shell/stop | PASS |
| 6 | `bash -n bin/dc-tui` | PASS |

Total: 27 unit tests run, 27 passed, 0 failed, 0 skipped. 6 checks all PASS.

### Unit tests

| Test | Result | Time |
|---|---|---|
| TestRenderPrimaryEqualWidth | PASS | 0.00s |
| TestRenderPrimaryThenMeta | PASS | 0.00s |
| TestRenderNarrowWrapsMore | PASS | 0.00s |
| TestStartHitboxAfterHeader | PASS | 0.00s |
| TestStartHitboxNarrowLongPath | PASS | 0.00s |
| TestBenignExecErr | PASS | 0.00s |
| TestDisabledStartReason | PASS | 0.00s |
| TestFleetKeyNotSilent | PASS | 0.00s |
| TestConfirmRmCancel | PASS | 0.00s |
| TestConfirmRmYesUsesStayHook | PASS | 0.00s |
| TestLeaveOnShell | PASS | 0.00s |
| TestLeaveIgnoresSecondShell | PASS | 0.00s |
| TestLeaveOnLogs | PASS | 0.00s |
| TestCursorClampAndEnterFleet | PASS | 0.00s |
| TestStayCmdSplitsStatusAndErr | PASS | 0.00s |
| TestDisabledStartTile | PASS | 0.00s |
| TestClickDisabledStart | PASS | 0.00s |
| TestEnterStackLeaves | PASS | 0.00s |
| TestLogoMarkHasFrameAndPip | PASS | 0.00s |
| TestSplashSkipsOnKey | PASS | 0.00s |
| TestSplashTickEnds | PASS | 0.00s |
| TestViewHasPrimaryHint | PASS | 0.00s |
| TestExecDoneStartFailure | PASS | 0.00s |
| TestExecDoneShellExit1IsBack | PASS | 0.00s |
| TestStartHitboxLongPorts | PASS | 0.00s |
| TestLayoutNoWrapAtNarrowWidth | PASS | 0.00s |
| TestOpenRowClearsStatusOnErr | PASS | 0.00s |

`ok github.com/Canvilled/dc-cli/cmd/dc-tui 0.631s` (cover rerun 0.957s)

Vs prior UX-pass QA (15 tests / 35.1%): +12 tests, +15.1 pts cover.

## Coverage Metrics

`go test -covermode=atomic`: **50.2%** statements (`go tool cover -func` total **50.3%**). Below 80% bar. Up from 35.1%.

Line/branch % not available beyond `-func`. No `-covermode=atomic` branch split.

### High (exercised by this pass)

| Func | Cover |
|---|---|
| `benignExecErr` | 100% |
| `skipSplash` / `rowCount` / `refuse` / `withErr` / `withStatus` / `kv` / `colorMark` / `morePanel` | 100% |
| `renderRow` | 94.4% |
| `renderGroups` | 92.9% |
| `logoSplash` | 89.5% |
| `stayCmd` | 85.7% |
| `openRow` | 84.6% |
| `moveCursor` / `compactLines` / `clipBlock` | 83.3% |
| `logoCompact` | 81.8% |
| `trunc` | 80.0% |
| `tileStyle` | 76.9% |
| `hasDevcontainer` / `activateRow` / `benignLeaveErr` / `joinLogo` | 75.0% |
| `View` | 66.7% |
| `layout` | 63.4% |
| `handleConfirmKey` | 62.5% |
| `runAction` | 56.2% |
| `handleClick` | 47.4% |
| `handleKey` | 42.4% |
| `Update` | 30.8% |
| `clickKey` | 21.4% |
| `reload` | 13.0% |

### Zero / critical holes

| Func | Cover | Why it matters |
|---|---|---|
| `runPending` | 0% | real start/shell/stop leave path |
| `hitButton` / `hitRow` | 0% | click hit-test helpers (handleClick covered via layout coords) |
| `leaveLine` / `formatStackRow` / `formatFleetRow` | 0% | leave banner + row paint |
| `Init` / `splashView` / `colorFrame` | 0% | tea start + splash paint |
| `main` / `resolveWorkspace` / `editorBin` / `pickEditor` | 0% | CLI entry |

## Failed Tests

None.

## Performance Metrics

| Step | Time / size |
|---|---|
| `go test -count=1 -v ./cmd/dc-tui` | 0.631s |
| `go test -count=1 -covermode=atomic` | 0.957s |
| `go build -o /tmp/dc-tui` | instant; 5,185,058 B Mach-O arm64 |
| `dc-tui --help` | instant, exit 0 |
| slowest test | none (all 0.00s reported) |

No flaky rerun. `-count=1` used. No memory/resource issues observed.

## Build Status

PASS. No compile warning. `gofmt -l cmd/dc-tui` empty. `bash -n bin/dc-tui` clean.

## Help check

`/tmp/dc-tui --help` contains:

```
Primary: start (u)  shell (e)  stop (s)
```

Also mentions splash skip, meta keys, rm confirm, j/k rows, disk/prune CLI-only. Exit 0.

## Critical Issues

None blocking this ship-clean pass.

## Recommendations

1. Cover `runPending` with stub exec (no docker) — last real leave/return hole.
2. Direct tests for `hitButton`/`hitRow` + click-row, not only disabled-start click.
3. Paint tests for `leaveLine`, fleet/stack rows.
4. Coverage still 50% vs 80% bar; next tests should target `Update`/`handleKey`/`clickKey`.
5. Local Go 1.26.6 vs CI 1.24 — re-run on 1.24 if a release gate.

## Next Steps

1. P0 none.
2. P1 stub `runPending` + hit-test units (no docker).
3. P2 leave/fleet/stack paint tests.
4. P3 CI Go version pin vs local 1.26.6.

## Unresolved questions

- `runPending` still 0%. Intentional (would exec process) or missing seam?
- Local Go 1.26.6 vs CI 1.24. Any 1.24-only fail unknown here.
- `hitButton`/`hitRow` 0% even though `handleClick` at 47.4% — click path uses layout coords, never the helpers?
