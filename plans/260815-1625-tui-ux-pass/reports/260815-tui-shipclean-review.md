# Code Review — dc-tui ship-clean

Date: 2026-08-15
Scope: uncommitted ship-clean follow-up (start fail, clip/hitbox, status xor err, screenshot)
Verdict: **Ship-clean** — 9/10 (was 7/10; see Re-review)
Plan: `plans/260815-1625-tui-ux-pass/plan.md`
Prior: `reports/260815-tui-ux-review.md`

## Code Review Summary

### Scope
- Files reviewed: `cmd/dc-tui/{model,view,actions,main,main_test}.go`, `go.mod`, `README.md` TUI, `site/src/content/guides/tui.md`, `site/src/pages/guide/[slug].astro`, `site/public/images/tui.png`
- Ignored: dirty leftover SEO/github-pages plan files
- Lines analyzed: ~1815 Go + tests/docs; PNG inspected
- Review focus: prior P0s + locked decisions (start exit 1, clip=hitbox, status xor err, 3-row screenshot, Update/stayCmd tests)
- Updated plans: none (report only)

### Overall Assessment
Ship-clean pass did the right pieces: `benignLeaveErr` splits start vs shell/logs, `withErr`/`withStatus` exist, header/status/more clip to width, screenshot is the 3-row board, tests now call `Update` + `stayCmd`. `go test ./cmd/dc-tui` 27/27. `gofmt`/`go vet`/build clean. `x/ansi` only promoted from indirect — no new module.

Not ship-clean. `execDone` start-fail sets `err`, then **always** `reload()`. Successful `reloadMsg` does `m.err=""` and wipes it. `TestExecDoneStartFailure` never applies that cmd. Selected stack row is `formatStackRow` w+2 then `rowHover.Width(w)` — lipgloss wraps one extra line; click Y hits the next service.

Locked decisions held: Go-only, no PTY/desktop/bubbles, screenshot honest, start exit 1 classified as fail (until reload undoes it).

### Critical Issues
None. No data-loss / `--rm` without confirm / contract break. Tests never call real `dc-down --rm` (`runStay` hook).

### High Priority Findings

1. **Start failure does not survive the tea loop.** Locked: start (u) exit 1 is a real failure. `execDoneMsg` does the right paint, then returns `m.reload()`. Success path of `reloadMsg` unconditionally clears `err`. Probe: after `execDone` `err="exit status 1"`; after `reloadMsg{rows:…}` `err=""`. User gets a flash then a quiet board. Shell/logs still fine (they want `back from …`, which reload keeps).

```108:116:cmd/dc-tui/model.go
	case execDoneMsg:
		m.leaving = ""
		m.pending = ""
		if msg.err != nil && !benignLeaveErr(msg.action, msg.err) {
			m = m.withErr(msg.err.Error())
		} else {
			m = m.withStatus(backStatus(msg.action))
		}
		return m, m.reload()
```

```93:105:cmd/dc-tui/model.go
	case reloadMsg:
		if msg.err != nil {
			m = m.withErr(msg.err.Error())
			...
		} else {
			m.err = ""
			m.rows = msg.rows
```

`TestExecDoneStartFailure` (`main_test.go:433`) asserts only the first `Update`. Fix: do not `m.err=""` on reload success if an action err is live, **or** skip `reload()` on start fail. Add a test that applies the returned cmd’s `reloadMsg`.

2. **Selected stack row wraps → hitbox Y drift.** `formatStackRow` visual width = `2+8+2+16+2+(w-28)` = **w+2**. Hover/cursor then `rowHover.Width(w).Render` wraps. Probe `width=40`, long name, `cursor=0`: app at L12, leftover wrap L13, db at L14. `hitRow` is `y-rowY0` one-line-per-row → click wrap line execs **db**. `TestLayoutNoWrapAtNarrowWidth` has no stack. `formatStackRow` cover 0%.

```164:178:cmd/dc-tui/view.go
	name := trunc(s.Name, max(8, width-28))
	return "  " + st + "  " + fmt.Sprintf("%-16s", trunc(svc, 16)) + "  " + mutedStyle.Render(name)
```

```136:141:cmd/dc-tui/view.go
			line := formatStackRow(s, w)
			if i == m.cursor || i == m.hoverStack {
				line = rowHover.Width(w).Render(line)
			}
```

Fix: trunc name to `width-30` (or `trunc(line,w)` **before** Width). Assert hovered stack line count == 1 and `ansi.StringWidth<=w`. Fleet rows are exactly `w` — OK.

### Medium Priority Improvements

1. **`x` / `f` / open-success leftover `err`.** `withErr`/`withStatus` are exclusive. `runAction("x")` and fleet toggle still do `m.status=""` and leave `err`. Probe: after `x`, view paints confirm **and** `start failed`. `openRow` success same (`actions.go:202`). Locked “write one clears the other” is only on the helper, not every write.

2. **`runPending` still 0%.** Invalid `stack:N` now `refuse("stack row gone")` (`actions.go:147-151`) — good, prior silent return gone. No test calls `runPending` / `leaveTickMsg`. Default empty pending still silent (`actions.go:177-179`).

3. **`reloadMsg` still applied while `leaving`.** Plan left this on purpose. 50ms window: ls can reorder stack; valid index execs the wrong box. `refuse` only if index OOB.

4. **Confirm click still swallows the next verb.** Click start during y/n → cancel only. Keyboard `u` silent. Pre-existing; not this diff.

### Low Priority Suggestions

- `esc` still quits except confirm.
- `view.go` 376 lines (>200 guideline). Already 4 files.
- `hitButton`/`hitRow` 0% — `handleClick` inlines button hit; hover path untested.
- `q` during 50ms leave does not clear `pending`. tea.Quit vs `leaveTickMsg` still unverified.
- README image is repo-relative `site/public/images/tui.png` (fine on GitHub; not the old 11-tile attachment).

### Positive Observations

- `benignLeaveErr("u", exit 1)` false; shell/logs still swallow. Comment matches lock.
- `withErr` / `withStatus` used on execDone, stayCmd, refuse, openRow errs, fleet leftover keys, rm cancel.
- Header/path/ports/disk/status/more clipped; `clipBlock` + `ansi.Truncate`. Probe: start.y0=6 sits on the start/shell/stop line at width=40 + long path + more.
- Logs tile disabled when no container id (`view.go:245`).
- `runStay` hook: `TestConfirmRmYesUsesStayHook` proves `y` → `dc-down --rm <ws>` without Docker.
- `TestStayCmdSplitsStatusAndErr` now actually calls `stayCmd`.
- Screenshot is the 3-row board (start/shell/stop, open/attach/ports/logs, fleet/more/quit/rm). Guide embeds it; astro img CSS is fine.
- go.mod: `x/ansi` direct, still vendored. No bubbles/desktop.

### Recommended Actions

1. Keep start-fail `err` across the follow-up `reloadMsg`. Test the two-step Update.
2. Make `formatStackRow` ≤ `w` (name budget `w-30`). Test hover+long name: no extra `\n`, width ≤ `w`.
3. Route `x` / `f` / open-success through `withStatus`/`withErr`.
4. Unit `runPending` OOB + empty pending (no docker). Optional: drop `reloadMsg` while `leaving`.

### Metrics
- Type Coverage: Go. `go vet ./cmd/dc-tui` clean
- Test Coverage: **27/27 PASS** (`go test -count=1 -v ./cmd/dc-tui`). **50.3%** stmts (was 35.1%). `gofmt -l cmd/dc-tui` empty. `go build -o /tmp/dc-tui ./cmd/dc-tui` OK
- `runPending` / `formatStackRow` / `hitRow` / `leaveLine` = 0%
- Linting Issues: 0

### Spec / lock check

| Requirement | Status |
|---|---|
| start (u) exit 1 is err; shell/logs swallow 1 | PARTIAL (classified; wiped by reload) |
| status xor err | PARTIAL (helpers yes; x/f/reload-success no) |
| Clip so logical `\\n` = visual rows | PARTIAL (header/more yes; hovered stack no) |
| Screenshot 3-row board | PASS |
| Tests exercise Update / stayCmd | PARTIAL (yes; not Update+reload chain) |
| Go only; no PTY/desktop/bubbles; ansi already vendored | PASS |
| Primary start/shell/stop; rm y/n | PASS |

### Unresolved
- Does tea drain `leaveTickMsg` after `tea.Quit`? Still not executed.
- Live click on a hovered long stack name not run in a real terminal (probe only).
- Keep or drop `reloadMsg` while `leaving` — plan still lists it as leftover.

### Ship-clean?
**No.** Items 1–2 must land (or be explicitly waived) before this is clean.

## Re-review (same day)

Two Highs re-checked. `go test -count=1 ./cmd/dc-tui` PASS.

1. **Start-fail vs reload — gone.** `reloadMsg` success no longer does `m.err=""`. `TestExecDoneStartFailure` now applies `reloadMsg{rows:…}` and asserts err stays + rows refresh. `f`/`r`/`x`/open-success go through `withStatus("")` so they clear err on purpose.
2. **Hovered stack wrap — gone.** Name budget `width-30` (= exact `w`). Extra `trunc(..., w)` before `rowHover.Width(w)`. `TestHoveredStackRowFitsWidth`: both visual rows ≤40, second row is db, `hitRow(2, rowY0+1)==1`.

Leftover Mediums (`runPending` 0%, `reloadMsg` while leaving) unchanged — out of this re-review.

**Score: 9/10. Ship-clean: yes.**

