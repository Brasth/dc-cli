# Code Review: editor-aware `dc-open --attach` (Zed path)

Date: 2026-08-15
Reviewer: code-reviewer
Scope: uncommitted attach-message change only. SEO / github-pages plan dirt ignored.

**Score: 9/10**
**Ship-clean: yes**
**VS Code URI path untouched: yes**
**Old “Zed cannot attach” gone from bin + Go TUI: yes**

---

## Code Review Summary

### Scope
- Files reviewed:
  - `bin/dc-open`, `bin/dc-tui`
  - `cmd/dc-tui/main.go`, `cmd/dc-tui/view.go`, `cmd/dc-tui/main_test.go`, `cmd/dc-tui/actions.go`
  - `.github/workflows/ci.yml`
  - `README.md`, `skill/SKILL.md`, `site/src/content/guides/tui.md` (docs drift only)
  - `lib/dc-common.sh` (`dc_editor_bin`, resolve) — read, not changed
  - plan + phase + QA
- Lines of code analyzed: ~250 product + CI/docs
- Review focus: editor-aware `--attach` message vs product locks
- Updated plans: `plans/260815-1931-open-attach-zed/plan.md`, `phase-01-message.md`

### Overall Assessment

Change does what the plan says: stop the lie, teach Zed first-party steps, leave the VS Code URI alone. No extension, no remote-picker launch, no new tile / module / desktop / PTY. TUI `a` still `dc-open --attach`. Print-steps + exit 0/2 decision kept.

QA leftover (`bin/dc-tui` still “Zed/Sublime cannot attach”) is **fixed after QA**. Product surfaces now match.

### Critical Issues

None.

### High Priority Findings

None.

### Medium Priority Improvements

None that should block ship. Optional follow-ups under Low.

### Low Priority Suggestions

1. **TUI status is compacted.** `stayCmd` → `compactLines(..., 4)` on a 5-line success print. First line (`dc-open --attach is VS Code only`) dropped; remaining 4 joined with ` · `. Still has `Connect Dev Container`. Do not reverse print-steps / exit 0. Only change if someone hates the one-line status wrap.

2. **CI merges streams.** Step `dc-open --attach teaches Zed path` uses `2>&1`. Asserts rc + `Connect Dev Container` + no forbidden lump. Does not prove Zed→stdout vs none→stderr. Exit code is what TUI uses. Linux runner is the only reliable exit-2 proof (this Mac always finds Zed.app).

3. **No `runAction("a")` unit test.** Wiring read from source: `stayCmd("dc-open", "--attach", m.workspace)`. One attach tile. Optional.

4. **Site mock stale, not a lie.** `site/src/components/Board.astro` still `run: 'VS Code Remote into the app'`. Not “Zed cannot attach”. Out of phase file list. Skip unless a later docs pass.

5. **README command table** still `dc-open` / `--attach` → “host editor / VS Code attach”. Editors section already teaches Zed first-party. Fine.

6. **bash menu label** `[a] attach vscode` unchanged. Help text next to it is correct. YAGNI.

### Positive Observations

- VS Code URI block after `code_bin` empty-check is **byte-identical** to HEAD: same `mapfile` / inspect / `vscode-remote://attached-container+${hex}${inner}` / `exec "$code_bin" --folder-uri`.
- `dc-open` without `--attach` untouched (pick editor → `exec "$bin" "$ws"`).
- Zed branch is a static heredoc. No `zed` launch, no picker.
- Exit 0 on Zed so TUI `stayCmd` is status, not err. Exit 2 otherwise. Matches user lock.
- Forbidden lump gone from `bin/dc-open`, `bin/dc-tui`, `cmd/dc-tui` help/more. Only remaining “cannot attach” is `Sublime cannot attach` (required).
- CI PATH+HOME isolation covers the no-editor case this Mac cannot hide.
- Docs that drifted (`README` attach row, `skill/SKILL.md` v21, `site/.../tui.md`) stay aligned. No new site route.

### Recommended Actions

1. Ship as-is.
2. Let CI confirm Linux no-editor exit 2 (not reproducible here: `/Applications/Zed.app`).
3. Do not add a Zed extension, picker launch, extra tile, or stayCmd redesign.

### Metrics
- Type Coverage: n/a (bash + string consts). `go.mod` unchanged.
- Test Coverage: `go test -count=1 ./cmd/dc-tui` **29/29 PASS** (0.572s). New: `TestHelpAndMoreTeachZedAttach`. Cover not re-run (QA 55.2%; message-only).
- Linting Issues: `gofmt -l cmd/dc-tui` empty. `bash -n bin/dc-open bin/dc-tui` PASS. `go build ./cmd/dc-tui` PASS.

---

## Product locks

| Lock | Verdict |
|---|---|
| No Zed extension | yes |
| Do not launch Zed remote picker | yes — print only |
| `dc-open` without `--attach` unchanged | yes |
| VS Code `code` path: still vscode-remote URI, still needs running labeled container | **untouched** |
| No new Go modules / tile / desktop / PTY | yes (`go.mod` clean; one `{key:"a"}`; `case "a"` same) |
| TUI `a` still `dc-open --attach` | yes — Go `actions.go:108-109`, bash `bin/dc-tui:188-189` |
| Do not flag SEO/github-pages dirt | ignored |
| Do not reverse print-steps (exit 0 Zed, exit 2 else) | kept |

## VS Code URI path

HEAD vs worktree after `if [[ -z "$code_bin" ]]`: identical.

Still:
- `dc_editor_bin code` first
- no container → `No running/labeled container…` exit 1
- not running → `Container $id is $status` exit 1
- `uri="vscode-remote://attached-container+${hex}${inner}"`
- `exec "$code_bin" --folder-uri "$uri"`

Both `code` + Zed → URI path (lock). This host has no VS Code .app; URI not executed here. Source identity is the proof.

## Old “Zed cannot attach” string

| Surface | Forbidden lump | Now |
|---|---|---|
| `bin/dc-open` usage + runtime | gone | steps; Sublime-only on exit 2 |
| `bin/dc-tui` usage | **gone** (QA was stale) | “Zed attaches itself… Sublime cannot.” |
| Go `helpText` | gone | same |
| Go `morePanel` | gone | “Zed: Project → Open Remote → Connect Dev Container” |
| Tests | assert absence | `TestHelpAndMoreTeachZedAttach` |

`rg` on `bin/` + `cmd/dc-tui`: no `Zed/Sublime cannot attach`, no `Zed and Sublime cannot attach`, no `Zed/Sublime = open only` except test needles.

## Plan completeness

| Criterion | Status | Evidence |
|---|---|---|
| `code` path unchanged | done | source diff, URI block identical |
| Fake `zed`, no `code`: exit 0, stdout has Connect Dev Container, never `cannot attach` | done | this host + CI step |
| No editors: exit 2, stderr has steps, never `Zed/Sublime cannot attach` | done in CI | this Mac cannot hide Zed.app (same as QA 5c) |
| TUI help + more do not say Zed cannot attach | done | unit test + `--help` |
| No new Go deps, no site route, no desktop | done | `go.mod` clean; no new routes |

Phase 01 file list done. `bin/dc-tui` help also updated (beyond phase “help only” — good).

## Fresh verification (this review)

```
gofmt -l cmd/dc-tui          # empty
bash -n bin/dc-open bin/dc-tui
go test -count=1 ./cmd/dc-tui   # 29 pass, 0 fail
go build -o /tmp/dc-tui-review ./cmd/dc-tui
PATH=/usr/bin:/bin ./bin/dc-open --attach $tmpdir
  rc=0 stdout=steps stderr=empty  no "cannot attach"
PATH=$fakezed:/usr/bin:/bin …   same (Zed.app also visible)
./bin/dc-open --help / ./bin/dc-tui --help / compiled --help
  have Connect Dev Container; no forbidden lump
```

## Unresolved questions

- Trust CI for no-editor exit 2 on Linux? Yes — `/Applications` unhidable here.
- Fake-zed on this Mac not isolated from Zed.app. Same stdout + exit 0. Accept.

QA questions about `bin/dc-tui` help: closed — string is gone.
