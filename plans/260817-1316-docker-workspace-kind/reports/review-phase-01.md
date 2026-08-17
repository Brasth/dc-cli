---
title: "Review — Phase 1 owned stack verbs"
status: concerns
score: 7/10
created: 2026-08-17
---

## Code Review Summary

### Scope
- Files reviewed: `cmd/dc-tui/logs.go`, `logs_test.go`, `actions.go`, `model.go`, `view.go`, `main.go`, `main_test.go`, `lib/dc-common.sh`, `bin/dc-exec`, `tests/lib/fake-docker`, `tests/exec/run.sh`, `README.md`, `skill/SKILL.md`, `site/src/content/guides/tui.md`
- Lines of code analyzed: ~480 inserted / 11 deleted (14 files). No extra files.
- Review focus: Phase 1 owned-stack verbs (per-row TUI logs + stack-membership restart)
- Updated plans: `phase-01-owned-stack-verbs.md`, `plan.md`

### Overall Assessment

Score: **7/10**

Phase 1 ships the locked surface: `l` follows `stack[cursor]`, `R` / `dc-exec --service --restart` stay-on-board, fleet + labeled-app TUI row refuse, `--id --restart` banned, `--id` hatch unchanged, no new bin/module, docs updated. Tests green (70 TUI + 7 exec + 19 safety).

One High hole: match is OR-per-row (`service|name|id prefix`) in `docker ps -aq` order. `--service db --restart` can restart the labeled app when its hex id starts with `db`. Reproduced. Bypasses TUI app-row refuse. Still stack-bounded (not a stranger restart).

### Spec compliance

| Requirement | Status | Evidence |
|---|---|---|
| `l` logs `stack[cursor]` | PASS | `logTarget` + `TestOpenLogsUsesStackCursor` (`db1` not `app1`) |
| Empty stack logs fall back to `rows[0]` | PASS | `TestOpenLogsEmptyStackFallsBackToRows0` |
| Logs button from stack cursor, not only `rows[0]` | PASS | `canFollowLogs` / `view.go:274` / `TestLogsButtonEnabledFromStackCursor` |
| Restart only if id ∈ `dc_stack_rows` | PASS* | `dc_restart_in_stack` walks stack only; no inspect on miss. *wrong sibling inside stack possible (High) |
| Labeled app TUI row refuse (`ID==rows[0].ID`, use `u`/`s`) | PASS* | `restartSelected` + `TestRestartAppRowRefuse`. *CLI re-resolve can still hit the app (High) |
| `--id --restart` banned | PASS | `bin/dc-exec:96-98` exit 2; `restart-id-banned` |
| `--id` hatch without `--restart` unchanged | PASS | hatch block untouched; `term-env` still `--id` |
| `--restart` requires `--service` | PASS | exit 2 + `restart-needs-service` |
| Fleet refuses `l` and `R` | PASS | `handleKey` + `TestFleetLogsRefuse` / `TestRestartFleetRefuse` |
| `r` still reloads | PASS | `model.go:285-286` + `TestReloadStillLowercaseR` |
| No unlabeled mutate | PASS | no-app / unknown → no `docker restart`; take-ports unlabeled still green |
| No new bin / no new Go module | PASS | `git status`: 14 expected files |
| Stay-on-board logs + restart | PASS | `openLogs` overlay; `stayCmd` for `R` |
| Docs: README + skill + tui guide | PASS | TUI table, `--restart` row, skill pitfalls, FAQ R vs r |

### Critical Issues

None. Stranger / unlabeled restart is refused. `--id --restart` never inspects.

### High Priority Findings

1. **`dc_restart_in_stack` OR-per-row match can restart the wrong stack member (incl. labeled app)**
   - File: `lib/dc-common.sh:205`
   - `[[ "$svc" == "$ref" || "$name" == "$ref" || "$id" == "$ref"* ]]` on first `dc_stack_rows` hit.
   - Common service `db` is a valid hex prefix. If the labeled app id is `db12ffff` and it appears before service `db` in `docker ps -aq` order, `--service db --restart` restarts the app.
   - Reproduced (fake-docker):

     ```
     stack: app-1 id=db12ffff service=app
            db-1  id=ffff1111 service=db
     dc-exec --service db --restart
     → docker restart db12ffff     # app, not db
     stdout: restart  app  app-1  db12ffff
     rc=0
     ```

   - TUI `R` on the db row passes `--service db` after ID≠`rows[0].ID` check, then CLI re-resolves and hits the app. Bypasses the locked “app row → u/s” rule. `docker restart` on the labeled app skips official recreate (the risk Phase 1 refused in the TUI).
   - Still ∈ `dc_stack_rows` — not unlabeled, not a stranger.
   - Same OR exists on `dc-exec --service` exec (`bin/dc-exec:182`). Restart made it mutate.

   **Fix:** three-pass match in `dc_restart_in_stack`: exact service, then exact name, then id prefix. Add `tests/exec` case with app id `db12ffff` + service `db` → `restart ffff1111` only.

   Do **not** unlock `--id --restart` (locked). Three-pass keeps the same three keys.

### Medium Priority Improvements

1. **`openLogs` / `logTarget` have no fleet guard** (`logs.go:51-73`, `80-84`)
   - Fleet `l` is refused only in `handleKey`. `logTarget` skips stack when `fleet` then falls back to `rows[0]`. If routing changes, fleet would follow first labeled app logs.
   - `restartSelected` is safer: empty stack after fleet reload + key guard.
   - **Fix:** `if m.fleet { return m.refuse(...) }` at the top of `openLogs` (same copy as start/stop).

2. **App-row refuse skipped when `rows` is empty** (`actions.go:169`)
   - `len(m.rows) > 0 && s.ID == m.rows[0].ID`. `dc-ls` empty + `dc-exec --list` still populated (rare) → TUI can `docker restart` the labeled app on purpose.
   - **Fix:** if no `rows[0]`, refuse `R` (fail closed) or treat missing labeled-app id as “cannot prove this is not the app”.

3. **No exec test for hex-prefix collision**
   - Happy-path `restart-sibling` uses ids `app1` / `db1` (`app1` does not start with `db`). Hole untested.

### Low Priority Suggestions

1. `logName` prefers container Name then Service (`logs.go:62-65`). Research asked Service then Name. Cosmetic; test pins `db-1`.
2. `restart-unknown` / `restart-no-app` assert `rc≠0` not `2`. Pin if the CLI contract should be exact.
3. README / tui guide “Button” column lists **restart** but there is no clickable `R` (key + more-legend only). Fine by YAGNI; table wording is slightly off.
4. `dc-exec --service` exec still has the same OR match. Out of Phase 1; fix with a shared `dc_stack_match` when touching exec.

### Positive Observations

- Locked decisions implemented without scope creep. No unlabeled classify / `dc-ls` / prune / take-ports edits.
- `--id --restart` exits 2 *before* any docker call. `--id` hatch path untouched.
- TUI app refuse is ID equality (`stack[i].ID == rows[0].ID`), not a service-name guess (red-team #4).
- `r` / `R` split is clean; click `r` still reloads; fleet copy matches start/stop.
- `stayCmd` + `runStay` hook keeps restart testable without Docker.
- Docs (README TUI + commands, skill procedure/pitfalls, tui FAQ) match behavior.
- Fake-docker `restart` verb is small and consistent with `start`.

### Recommended Actions

1. **Before Phase 2:** three-pass match in `dc_restart_in_stack` + collision test (`app` id `db12ffff`, service `db` → restart `ffff1111` only).
2. Optional: `openLogs` refuse fleet; refuse `R` when `rows` is empty.
3. Optional later: share matcher with `dc-exec --service` exec.
4. Do not add `--id --restart` (even membership-checked) unless the lock is reopened.

### Locked decisions (verified)

| Decision | Verdict |
|---|---|
| Restart only if id ∈ `dc_stack_rows` | Hold (membership). Wrong-*member* possible (High). |
| Labeled app TUI row refused (`u`/`s`) | Hold on TUI row. CLI re-resolve can bypass (High). |
| `--id --restart` banned; `--id` hatch unchanged | Hold |
| Fleet refuses `l` and `R` | Hold |
| `r` still reloads | Hold |
| No unlabeled mutate | Hold |
| No new bin / no new Go module | Hold |

### Metrics
- Type Coverage: n/a (Go; `go vet ./cmd/dc-tui` clean, `gofmt -l` empty)
- Test Coverage: `go test -cover ./cmd/dc-tui` = **57.3%** statements. Phase 1 paths named in tester report are exercised.
- Linting Issues: 0 (`gofmt`, `go vet`)
- Fresh verification (this review):
  - `go test -count=1 ./cmd/dc-tui` — PASS (70)
  - `bash tests/exec/run.sh` — PASS (7)
  - `bash tests/safety/run.sh` — PASS (19/19), unlabeled take-ports still refuses stop/rm

### Unresolved questions

- Should `dc-exec --service` exec get the same three-pass match in this fix, or only restart?
- Fail closed on TUI `R` when `dc-ls` rows are empty?

---

Pre-Landing Review: 4 issues (0 critical, 1 high, 3 informational)

**CRITICAL** (blocking): none

**Issues:**
- [lib/dc-common.sh:205] OR-per-row id-prefix match restarts wrong sibling / labeled app when id starts with service name (`db`)
  Fix: exact service, then name, then id prefix; add collision test
- [cmd/dc-tui/logs.go:80] `openLogs` has no fleet guard
  Fix: refuse fleet inside `openLogs`
- [cmd/dc-tui/actions.go:169] empty `rows` skips app-row refuse
  Fix: refuse `R` without `rows[0]`
- [tests/exec/run.sh] no hex-prefix collision case
  Fix: add with the three-pass change
