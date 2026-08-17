---
title: "Review — Phase 2 compose-kind identity"
status: concerns
score: 7/10
created: 2026-08-17
---

## Code Review Summary

### Scope
- Files reviewed: `lib/dc-common.sh`, `lib/dc-net.sh`, `bin/dc-ls`, `bin/dc-up`, `bin/dc-doctor`, `tests/kind/run.sh`, `README.md`, `skill/SKILL.md` v27
- Out of scope (verified untouched): `site/src/components/Manifesto.astro`, `bin/dc-down`, TUI start, `bin/dc-forward`
- Lines of code analyzed: ~2750 in touched sources; +190 / −32 vs HEAD. New `tests/kind/run.sh` 166 lines
- Review focus: Phase 2 compose-kind identity (detect / ls / classify / doctor / refuse-up). No start
- Updated plans: `phase-02-compose-kind-identity.md`, `plan.md`

### Overall Assessment

Score: **7/10**

Detect-only surface matches the lock: kind order DC → compose → none, never fake `devcontainer.local_folder`, `--all` labeled-only, stopped compose-only → `[]`, name-alone never owns, `dc-up` refuses before `command -v devcontainer`, skill/README detect-only, manifesto unchanged, Phase 3 still gated.

High hole: `dc_compose_claimants` now unions labeled folders **and** compose `working_dir` / `config_files` parents **per container**, without collapsing a labeled app's own compose project dir. Official-CLI layout `local_folder=$ws` + `working_dir=$ws/.devcontainer` → n=2 → `dc-down` refuse + take-ports `ambiguous-compose` (report-only). Reproduced. Fail-closed (no new stop). Breaks existing labeled compose-in-`.devcontainer` stacks. Safety fixtures omit `working_dir`, so 19/19 stayed green.

### Spec compliance

| Requirement | Status | Evidence |
|---|---|---|
| `dc_workspace_kind` → devcontainer \| compose \| none | PASS | DC first; else root compose.yaml/yml / docker-compose.yaml/yml; else none. Both-present = DC. `kind-detect` |
| Never fake `devcontainer.local_folder` | PASS | no label writes; `ls-compose-kind` `log_lacks commit` + no DC label in JSON |
| This-folder `dc-ls --json` `kind` on real containers | PASS | compose rows `kind=compose`; `--all` default `kind=devcontainer` |
| TTY `dc-ls` (table) shows compose-kind | MISS | `dc_ls_table` still `dc_ids_for_workspace` only. Reproduced: JSON has `c1`, table header-only |
| Stopped compose-only → `[]` | PASS | `ls-stopped-empty` |
| `dc-ls --all` labeled-only; no daemon working_dir walk | PASS | `ls-all-labeled-only`; `--all` still `dc_labeled_ids` |
| Name-alone never owns; missing working_dir+config_files → unknown | PASS (ls) | `ls-no-working-dir` → `[]`. Classify still `unlabeled` (report-only; same effect) |
| Two folders same compose name: fail-closed / not owned | PASS (ls) | `ls-foreign-working-dir` → `[]`. Claimants n>1 also fail-closed — **over-fires** (High) |
| Doctor `workspace.kind` + compose start-disabled warning | PASS | top-level `workspace.kind`; `workspace_config` warning `compose-kind; start disabled`. schemaVersion 1 additive. 18 ids still present |
| `dc-up` kind detect **before** `command -v devcontainer` | PASS | `bin/dc-up:102-115` then CLI check. `up-compose-kind-refuse`: rc=1, no `devc.log`, no `compose up` |
| kind=compose: no start, no `compose up` | PASS | refuse exit 1 |
| take-ports: no new stop paths; unlabeled report-only | PASS* | unlabeled case green. *labeled + `.devcontainer` working_dir now **fewer** stops (High) |
| Skill detect-only; manifesto unchanged | PASS | SKILL v27 step 1; Manifesto.astro not in diff |
| Phase 3 still gated | PASS | no dc-down mutate / TUI start / forward / manifesto rewrite |
| Claimants = labeled **or** compose working_dirs; n>1 fail closed | WARN | union implemented; same stack counted twice when working_dir ≠ local_folder |

### Critical Issues

None. No new mutate path. Unlabeled / name-alone still report-only. No label writes. `dc-up` cannot start compose-kind.

### High Priority Findings

1. **`dc_compose_claimants` double-counts a labeled app's own compose project dir → dc-down / take-ports fail-closed on a common official-CLI layout**
   - File: `lib/dc-common.sh:1076-1121`
   - First loop adds `devcontainer.local_folder`. Second loop adds `working_dir` (else dirname of first `config_files`) for **every** `com.docker.compose.project=$proj` container, including the same labeled app.
   - `dc_same_workspace` is exact/physical equality — `$ws` ≠ `$ws/.devcontainer`.
   - Reproduced (fake-docker):

     ```
     local_folder=$ws
     working_dir=$ws/.devcontainer
     dc_compose_claimants projA → n=2
     classify foreign holder → ambiguous|ambiguous-compose
     dc-down $ws → refuse compose-wide mutation for projA
     dc-up --take-ports → report-only reason=ambiguous-compose (no compose stop)
     ```

   - Same-dir `working_dir=$ws` still n=1 (dedupe works). Labeled + **foreign** compose-kind same name still n=2 (intended).
   - Safety suite does not set `working_dir` on labeled apps → 19/19 green, hole untested.
   - Impact: default `dc-down` (full stack) and `--take-ports` stop of a positively labeled foreign stack break when engine wrote working_dir to the compose-file directory (typical `.devcontainer/docker-compose.yml`). Fail-closed — will not stop the wrong thing — but a real regression vs Phase 1.

   **Fix:** only add working_dir / config_files claimants for containers **without** `devcontainer.local_folder`. Labeled identity stays the DC folder; compose-kind-only boxes claim via engine paths. Keep n>1 when a labeled folder and a *different* unlabeled working_dir share the project name.

   Add safety cases:
   - labeled `$ws` + `working_dir=$ws/.devcontainer` → `dc-down` still `compose -p … stop`; take-ports foreign still stops
   - labeled `$ws` + unlabeled compose-kind `working_dir=$other` same project → still refuse / ambiguous

### Medium Priority Improvements

1. **`dc_ls_table` ignores compose-kind** (`lib/dc-common.sh:484-488`, `bin/dc-ls:76-82`)
   - `--json` lists compose containers; TTY `dc-ls` is header-only. Success criterion / user demo say `dc-ls`.
   - **Fix:** share id selection with `dc_ls_json` (kind=compose → `dc_ids_for_compose_workspace`).

2. **Compose-kind JSON `local_folder` is `"<no value>"`** (`dc_inspect_row:235`)
   - `dc_label` strips `<no value>`; inspect_row does not. New first-class compose rows leak the docker template sentinel.
   - **Fix:** treat `<no value>` as empty (same as `dc_label` / `dc_holders_of_host_port`).

3. **Classify still maps compose-kind → `unlabeled`** (`dc_classify_port_holder:1173-1175`)
   - Correct for “no new stop paths”. Plan wording “unknown if working_dir missing” is only implemented in ls (`[]`), not as a classify reason.
   - Phase 3 cannot take a foreign compose-kind stack until classify grows a positive compose-kind class. Do not reuse `unlabeled` for that.

4. **Claimants `config_files` uses first path only**; `dc_compose_proves_workspace` walks all. Include / override files can disagree. Align or document.

5. **No `config_files`-only positive ls case.** Tester already flagged. Missing-working_dir covered (`[]`). Add one if that claim path is real.

### Low Priority Suggestions

1. `bin/dc-ls` usage still says “labeled `devcontainer.local_folder`”. README command table: “labeled app list”. This-folder compose-kind is now a second source.
2. Skill frontmatter still “NEVER docker exec”. Plan rewrite (“never raw `docker exec NAME`; compose-kind future = `compose exec` inside `dc-exec`”) can wait for Phase 3; detect-only sentence is enough now.
3. `dc-common.sh` is 1333 lines (pre-existing giant). New kind helpers are ~80 lines. Split later; do not block.
4. `dc_ids_for_compose_workspace` basename fallback if `dc_compose_declared_name` empty. Still prove working_dir (not name-alone). Compose-sanitized name vs raw basename can miss; fail-empty.
5. `kind-detect` only seeds `compose.yaml`. `compose.yml` / `docker-compose.yaml` paths untested (code lists all four).
6. Nested `_dc_claimants_add` + `unset -f` is fine in bash; a file-level helper is simpler after the High fix.

### Positive Observations

- Kind order matches the lock. Both-present = devcontainer. No invented DC labels.
- `dc_ids_for_compose_workspace` filters by declared `.name` then proves engine paths. Not a daemon-wide working_dir inventory. `--all` untouched.
- `dc-up` refuse is before official CLI require; no `devcontainer` / `compose up`. Message greppable `compose-kind`.
- Doctor: additive `workspace.kind`, schemaVersion stays 1, 18 check ids preserved, compose warning is start-disabled not “add .devcontainer”.
- Skill v27 + README one sentence are detect-only. Manifesto / TUI start / dc-down mutate / dc-forward not cooked.
- New `tests/kind/run.sh` covers the Phase 2 gates that were specified (detect, ls JSON, stopped, name-alone, `--all`, foreign dir, up refuse, doctor).
- `dc_compose_declared_name` reuses `dc_net_compose_json` (no YAML fork).
- TUI start still `hasConfig` + `dc-up` refuse — compose-kind cannot start from the board.

### Recommended Actions

1. **Before calling Phase 2 done:** fix `dc_compose_claimants` (skip working_dir/config_files on containers that already have a DC folder label) + safety cases above.
2. Wire `dc_ls_table` to the same id source as `dc_ls_json`; strip `<no value>` in `dc_inspect_row`.
3. Optional: `config_files`-only ls case; `compose.yml` detect; dc-ls help/README wording.
4. Do **not** enable compose start / `compose exec` / manifesto / take-ports compose-kind stop. Phase 3 stays gated until the user says start.
5. User demo can still show `dc-ls --json` + `dc-doctor --json` on a compose-only folder. Do not demo `dc-down` on a labeled compose-in-`.devcontainer` workspace until #1 lands.

### Locked decisions (verified)

| Decision | Verdict |
|---|---|
| kind: DC config → devcontainer; else root compose → compose; else none | Hold |
| Never fake `devcontainer.local_folder` | Hold |
| `dc-ls --all` labeled-only | Hold |
| Stopped compose-only folder → `[]` | Hold |
| Name-alone never owns | Hold (ls). Classify = unlabeled, report-only |
| `dc-up` compose-kind: no start | Hold |
| take-ports: no new stop paths | Hold (unlabeled). Labeled compose-in-subdir now **lost** a stop path (High) |
| Phase 3 still gated | Hold |

### Metrics
- Type Coverage: n/a (bash). Go TUI unchanged; `gofmt -l` empty, `go vet ./cmd/dc-tui` clean
- Test Coverage: bash suites no coverage %. Kind 8/8. TUI statements unchanged (~57%)
- Linting Issues: 0 (`bash -n` on all touched bash)
- Fresh verification (this review):
  - `bash -n` `lib/dc-common.sh` `lib/dc-net.sh` `bin/dc-ls` `bin/dc-up` `bin/dc-doctor` `tests/kind/run.sh` — ok
  - `bash tests/kind/run.sh` — PASS (8)
  - `bash tests/safety/run.sh` — PASS (19/19), unlabeled take-ports still refuses stop/rm
  - `bash tests/doctor/run.sh` — PASS (9/9), schemaVersion 1, 18 ids
  - `bash tests/exec/run.sh` — PASS (8), Phase 1 restart + collision still green
  - `go test -count=1 ./cmd/dc-tui` — PASS
  - Claimants / take-ports / table gaps reproduced in harness (not in committed tests)

### Unresolved questions

- Confirm official CLI's engine `working_dir` for `dockerComposeFile` inside `.devcontainer/` (cwd vs compose-file dir). Either value is legal; current claimants must tolerate both.
- Should compose-kind holders get a classify reason now (`compose-kind` / `unknown`) or stay `unlabeled` until Phase 3?

---

Pre-Landing Review: 5 issues (0 critical, 1 high, 4 informational)

**CRITICAL** (blocking): none

**Issues:**
- [lib/dc-common.sh:1100-1116] labeled app + own `.devcontainer` working_dir → n=2 claimants; dc-down refuse; take-ports ambiguous
  Fix: skip compose-path claim when container already has `devcontainer.local_folder`; add safety cases
- [lib/dc-common.sh:484] `dc_ls_table` does not list compose-kind
  Fix: same id selection as `dc_ls_json`
- [lib/dc-common.sh:235] inspect_row emits `local_folder=<no value>` on compose-kind rows
  Fix: normalize like `dc_label`
- [lib/dc-common.sh:1173] classify has no compose-kind / unknown-on-missing-working_dir reason
  Fix: leave report-only for Phase 2; add a positive class in Phase 3 before any stop
- [tests/kind/run.sh] no `config_files`-only ownership case
  Fix: optional positive case if that path is kept
