# Code Review — dc-tui UX pass

Date: 2026-08-15
Scope: Go board + tests + README/guide + bash usage
Verdict: **Request changes** — 6/10
Plan: `plans/260815-1625-tui-ux-pass/plan.md`

## Code Review Summary

### Scope
- Files reviewed: `cmd/dc-tui/{main,model,view,actions,main_test}.go`, `bin/dc-tui` (usage), `README.md` TUI, `site/src/content/guides/tui.md`
- Lines of code analyzed: ~1336 Go + tests/docs
- Review focus: UX pass vs locked plan (hierarchy, confirm, leave, hitbox, silent no-op, status/err, docker-safe tests, docs)
- Updated plans: `plans/260815-1625-tui-ux-pass/plan.md` + phase-01/02

### Overall Assessment
Board regroup works. Primary/meta/danger split is real. `rm` confirms. Leave tick + `leaving` lock exist. `go test ./cmd/dc-tui` 15/15. No new deps, no desktop, tests do not run `dc-down --rm`.

Not ship-clean. `dc-up` exit 1 becomes green “back from start”. Hitboxes use logical `\n`, not visual wrap. Screenshots still the old 11-tile row.

Locked decisions held: Go-only, no PTY, no new modules, bash keys stay valid.

### Critical Issues
None (no data-loss / auth / contract break). `rm` y is the only `--rm` path; tests never call it.

### High Priority Findings

1. **start failure painted as success.** `start` now `ExecProcess`s `dc-up`. `execDoneMsg` + `benignExecErr` treat `exit status 1` as OK, then `backStatus("u")` → `status="back from start"`. `dc-up` port/disk/compose fail is usually exit 1. User sees teal “back from start”, `err` cleared. Shells need that swallow; start does not. Old board already used ExecProcess for `u`, but this pass added the success line.

```96:105:cmd/dc-tui/model.go
	case execDoneMsg:
		m.leaving = ""
		m.pending = ""
		if msg.err != nil && !benignExecErr(msg.err) {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.status = backStatus(msg.action)
		}
```

Fix: benign only for `e` / `l` / `exec-*`. `u` → `m.err = msg.err.Error()`, no back line.

2. **Hitbox Y drift on wrap.** `layout()` sets `y0` / `rowY0` via `strings.Count(..., "\n")`. Workspace path, disk, ports, status, `morePanel` are not truncated to `width`. Probe (`width=40`, 70-char path): `start.y0=6` but path line is 70 cells → terminal wraps, click sits on the wrong row. `more` + stack at `width=50`: `rowY0=27` while more lines (~70) wrap under the rows. Tile X vs lipgloss.Width matches (`7==7`). Tests use `/tmp/app` @ 80 — miss wrap. Plan already named bubbles as fallback if hitboxes break; cheaper fix is `trunc` header/status/more to `w`.

### Medium Priority Improvements

1. **status leftover on new err.** `stayCmd` / `reloadMsg` / `openRow` set `err` and leave old `status`. View paints both. `refuse()` also writes status, not err. `dc-df` success stays non-red (good). Clear the other slot when writing one.

2. **logs tile not disabled.** Plan: disabled tile + reason. No container: key `l` `refuse`s (not silent) but tile still active. Start is the only `disabled` spec.

3. **confirm click swallows the next verb.** Any click not on `rm` cancels and returns. Click start during y/n → `rm cancelled`, start does not run. Keyboard `u` during confirm is ignored (no reason). Mouse cannot confirm (y/n only — matches plan).

4. **leave vs in-flight reload.** `leaving` blocks keys/clicks (and second `e` — `TestLeaveIgnoresSecondShell`). `reloadMsg` still applies. `runPending` `stack:N` after a reload that emptied/reordered stack: `leaving="" ; return m, nil` — silent. Logs path at least `refuse`s.

5. **docs images stale.** README + `site/public/images/tui.png` still one equal-width row (`start shell open host …`). Text tables match the new board. `skill/SKILL.md` updated. Bash `bin/dc-tui` help still old grouping — allowed (no rewrite).

6. **`TestStayCmdSplitsStatusAndErr` does not call `stayCmd`.** Only `compactLines` + `backStatus`. `handleClick` / `runPending` / `stayCmd` still 0% (QA 35.1% cover).

### Low Priority Suggestions

- `esc` quits (except confirm). Easy to dump `more`.
- `h` = `?`, `r` reloads; neither on workspace tiles.
- Empty fleet `j`/`k` silent. Enter explains.
- `activateRow` with no stack runs shell. Undocumented, not silent.
- `view.go` 350 lines (over 200). Already split 4 files.
- `q` during 50ms leave does not clear `pending`. tea.Quit may drop the tick — unverified.

### Positive Observations

- `buttonGroups`: start/shell/stop primary; meta quieter; `rm` danger last.
- `x` only sets `confirm`; `y` is the sole `dc-down --rm`. `n`/`esc` → `rm cancelled`, no cmd.
- Fleet leftover `u`/`e`/… get a reason. No-config start: disabled + reason.
- Leave: paint 50ms, then ExecProcess. Second shell does not double-tick.
- `j`/`k` clamp; fleet Enter opens folder; stack Enter → `stack:N`.
- go.mod still bubbletea + lipgloss only.
- Tests never invoke `stayCmd` / `runPending` / `y`. `TestCursorClampAndEnterFleet` returns `reload` cmd, does not run it.

### Recommended Actions

1. Split benign: start failure → `err`, not `back from start`.
2. Truncate every header/status/more/ports line to `width` (or wrap-count into y). Add a long-path + `more` hitbox test.
3. On write: `status` xor `err`. Disable logs when `rows[0].ID==""`.
4. Ignore `reloadMsg` while `leaving`, or `refuse` if `stack:N` invalid.
5. Replace screenshots. Optional: click during confirm says why; `y` test with a fake runner.

### Metrics
- Type Coverage: Go; `go vet ./cmd/dc-tui` clean
- Test Coverage: 15/15 PASS (`go test -count=1 -v ./cmd/dc-tui`, 0.238s). QA earlier 35.1% stmts. `gofmt -l cmd/dc-tui` empty
- Linting Issues: 0

### Spec check

| Requirement | Status |
|---|---|
| Primary start / shell / stop | PASS |
| Meta quieter; rm danger + y/n | PASS |
| Invalid: disabled + reason, never silent | PARTIAL (start yes; logs no; confirm extra keys silent) |
| status vs err; dc-df not red | PARTIAL (two slots; start fail + leftover mix) |
| j/k + Enter fleet/stack | PASS |
| Leave tick then ExecProcess | PASS |
| No PTY / desktop / new deps | PASS |
| Tests must not exec `dc-down --rm` | PASS |
| README + tui.md match board | PARTIAL (copy yes; images old) |
| Bash rewrite | N/A (keys still valid; confirm already there) |

### Unresolved
- Does Bubble Tea drain `leaveTickMsg` after `tea.Quit`? Not executed.
- Confirm `y` has no seam; unit-testing it would run `dc-down --rm` today.
- Live click on a long `$PWD` not run (probe only).
