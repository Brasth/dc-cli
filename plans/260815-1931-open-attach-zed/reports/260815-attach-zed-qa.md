# QA: editor-aware `dc-open --attach` (Zed path)

Date: 2026-08-15 19:34 +0700
Host: macos Darwin arm64 (25.2.0), Go 1.26.6 (CI pins 1.24)
Scope: `bin/dc-open` --attach message, TUI help/more, CI PATH cases. No product edits. No docker mutate.

Editors on this Mac:
- Zed.app: **present** (`/Applications/Zed.app/Contents/MacOS/cli` + `zed`)
- `zed` / `code` / `subl` on PATH: **missing**
- VS Code .app: **absent** — URI path would **not** have been taken
- Sublime .app: **absent**

## Test Results Overview

| # | Check | Result |
|---|---|---|
| 1 | `gofmt -l cmd/dc-tui` | PASS (empty) |
| 2 | `go test -count=1 ./cmd/dc-tui` | PASS **29/29** (expect 28+) |
| 3 | `bash -n bin/dc-open` | PASS |
| 4 | `./bin/dc-open --help` has Connect Dev Container; no `Zed/Sublime cannot attach` | PASS |
| 5a | `PATH=/usr/bin:/bin ./bin/dc-open --attach $tmpdir` (Zed.app) | PASS exit 0 + Connect Dev Container |
| 5b | fake `zed` on PATH, no `code` | PASS exit 0 + Connect Dev Container; never `Zed/Sublime cannot attach` |
| 5c | hide /Applications | N/A — cannot; Zed.app still found. Not a fail. |
| 6 | `go run ./cmd/dc-tui --help` + compiled `/tmp/dc-tui-qa --help` | PASS no “Zed and Sublime cannot attach” |
| 7 | TUI `a` still `dc-open --attach`; one attach tile | PASS |
| — | `go build -o /tmp/dc-tui-qa ./cmd/dc-tui` | PASS 5,185,058 B |

Total: 29 unit tests run, 29 passed, 0 failed, 0 skipped. Acceptance 1–7 all PASS. One leftover outside Go TUI (see Critical / Concerns).

### Unit tests

All 29 PASS, each 0.000s in `-json` (pkg 0.227s; first verbose run 0.327s; cover rerun 0.551s).

| Test | Result |
|---|---|
| TestRenderPrimaryEqualWidth | PASS |
| TestRenderPrimaryThenMeta | PASS |
| TestRenderNarrowWrapsMore | PASS |
| TestStartHitboxAfterHeader | PASS |
| TestStartHitboxNarrowLongPath | PASS |
| TestBenignExecErr | PASS |
| TestDisabledStartReason | PASS |
| TestFleetKeyNotSilent | PASS |
| TestConfirmRmCancel | PASS |
| TestConfirmRmYesUsesStayHook | PASS |
| TestLeaveOnShell | PASS |
| TestLeaveIgnoresSecondShell | PASS |
| TestLeaveOnLogs | PASS |
| TestCursorClampAndEnterFleet | PASS |
| TestStayCmdSplitsStatusAndErr | PASS |
| TestDisabledStartTile | PASS |
| TestClickDisabledStart | PASS |
| TestEnterStackLeaves | PASS |
| TestLogoMarkHasFrameAndPip | PASS |
| TestSplashSkipsOnKey | PASS |
| TestSplashTickEnds | PASS |
| TestViewHasPrimaryHint | PASS |
| TestExecDoneStartFailure | PASS |
| TestExecDoneShellExit1IsBack | PASS |
| TestStartHitboxLongPorts | PASS |
| TestHoveredStackRowFitsWidth | PASS |
| TestLayoutNoWrapAtNarrowWidth | PASS |
| TestOpenRowClearsStatusOnErr | PASS |
| TestHelpAndMoreTeachZedAttach | PASS |

`ok github.com/Canvilled/dc-cli/cmd/dc-tui 0.327s`

No flakes observed (2 sequential `-count=1` runs + 1 cover run, all 29/29).

## Coverage Metrics

`go test -count=1 -covermode=atomic`: **55.2%** statements. Below typical 80% bar. Expected — this pass is a message/help change, not a coverage push.

Attach-relevant:

| Func | Cover |
|---|---|
| `morePanel` | **100%** |
| `runAction` | 53.3% (`a` branch not hit by unit tests; wiring read from source) |
| `buttonGroups` | 66.7% |

`helpText` is a const; `TestHelpAndMoreTeachZedAttach` asserts it.

### Zero / not in scope this pass

`main`, `resolveWorkspace`, `editorBin`, `pickEditor`, `Init`, `leaveLine`, `formatFleetRow`, `hitButton` still 0%. Same holes as prior TUI QA. Not blocking for --attach copy.

## Isolated PATH attach

### 5a. Zed.app, PATH=/usr/bin:/bin

```
PATH=/usr/bin:/bin ./bin/dc-open --attach $tmpdir
```

- exit **0**
- stdout:

```
dc-open --attach is VS Code only (needs code on PATH).
Zed attaches first-party:
  1. dc-up     # compose + ports (Zed does not publish forwardPorts)
  2. dc-open   # host folder
  3. In Zed: Project: Open Remote → Connect Dev Container
```

- has `Connect Dev Container`
- never `Zed/Sublime cannot attach`
- never `cannot attach`

This is the product path on this Mac (Zed.app, no code).

### 5b. Fake `zed` on PATH

`PATH=/tmp/fake-zed...:/usr/bin:/bin` with empty `#!/bin/sh` `zed`.

- exit **0**
- same stdout as 5a (printed to stdout, not stderr)
- has `Connect Dev Container`
- never `Zed/Sublime cannot attach`
- never `cannot attach`

Note: `/Applications/Zed.app` is still visible, so this Mac would take the Zed branch even without the fake binary. Fake PATH `zed` is preferred by `dc_editor_bin` (`command -v` first). Result matches contract either way.

### 5c. Hide /Applications / no-editors (exit 2)

Not reproducible here. `HOME=/tmp/no-editors PATH=/usr/bin:/bin` still finds `/Applications/Zed.app` → exit 0 (same as 5a). Task said do not fail the job for that.

CI Linux step `dc-open --attach teaches Zed path` covers:
- no editors: exit 2, stderr has Connect Dev Container + `Sublime cannot attach`, never `Zed/Sublime cannot attach`
- fake zed: exit 0, Connect Dev Container, never `cannot attach`

### VS Code .app URI path

**Would not have taken the URI path.** `/Applications/Visual Studio Code.app` absent. `dc_editor_bin code` → empty. `--attach` never reached `vscode-remote://` / `No running/labeled container`.

## Help / copy

| Surface | Connect Dev Container | Forbidden “Zed cannot attach” lump |
|---|---|---|
| `./bin/dc-open --help` | yes | none |
| `go run ./cmd/dc-tui --help` | yes (“Zed attaches itself… Sublime cannot.”) | none |
| compiled `cmd/dc-tui --help` | yes | none |
| Go `helpText` / `morePanel` (unit test) | yes | none |
| `./bin/dc-tui --help` (bash fallback) | no | **still `Zed/Sublime cannot attach`** |

Go TUI more panel (source): `attach   VS Code Remote URI; Zed: Project → Open Remote → Connect Dev Container` + `open ≠ attach. Zed attaches itself. Sublime cannot.`

## TUI `a` wiring (no new tile)

Read-only:

```108:109:cmd/dc-tui/actions.go
	case "a":
		return m.stayCmd("dc-open", "--attach", m.workspace)
```

- one `{key: "a", label: "attach"}` in `buttonGroups` (meta row)
- bash fallback also still `dc-open --attach "$ws"` (no extra tile)
- no new Go deps, no new site route

## Failed Tests

None.

## Performance Metrics

| Run | Time |
|---|---|
| `go test -count=1 -v ./cmd/dc-tui` | 0.327s |
| `go test -count=1 -json` pkg | 0.227s |
| cover atomic | 0.551s |
| `go build ./cmd/dc-tui` | instant / 5.2 MB |
| isolated `dc-open --attach` | instant (print + exit, no editor launch) |

No slow tests. No resource issues.

## Build Status

- `gofmt -l cmd/dc-tui`: empty
- `go test`: PASS
- `go build -o /tmp/dc-tui-qa ./cmd/dc-tui`: PASS
- `bash -n bin/dc-open`: PASS
- CI workflow has isolated PATH cases (`.github/workflows/ci.yml` step `dc-open --attach teaches Zed path`). Not executed here (no GH runner). Linux no-.app isolation is the right place for exit-2.

No compile warnings.

## Critical Issues

None blocking the stated acceptance list.

**Non-blocking leftover:** `bin/dc-tui` usage still says `Zed/Sublime cannot attach` (line 45). Phase file list did not include this bash fallback. Acceptance #6 is Go `cmd/dc-tui` only — that PASS. Source-install users without the Go board still see the old lie if they `dc-tui --help`.

## Recommendations

1. Update `bin/dc-tui` usage (and the `[a] attach vscode` label) to match Go help: Zed first-party / Connect Dev Container; Sublime cannot. Out of this phase’s file list — follow-up, not a reject.
2. Keep CI Linux no-editor + fake-zed cases. They are the only reliable exit-2 proof.
3. Optional: unit-test `runAction("a")` args == `dc-open --attach` so wiring cannot drift.
4. Do not chase 80% coverage for this message-only change.

## Next Steps

1. Ship Go TUI + `dc-open` message as-is (acceptance 1–7 green on this host).
2. Decide whether bash `bin/dc-tui --help` leftover is same-PR or later (I would same-PR; it is the exact string the plan kills).
3. Let CI confirm the no-editor exit 2 path on Linux.

## Unresolved questions

- Should bash `bin/dc-tui --help` be in this phase? Plan “Modify” list omitted it; product lie remains for the fallback.
- No-editor exit 2 unverified on this Mac (Zed.app unhidable). Trust CI?
- Fake-zed case here is not isolated from `/Applications/Zed.app`. Accept as “same stdout, exit 0”?
