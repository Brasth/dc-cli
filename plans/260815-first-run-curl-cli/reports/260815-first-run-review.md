# Code Review: first-run advertised curl includes `--with-cli`

Date: 2026-08-15
Reviewer: code-reviewer
Scope: uncommitted first-run curl contract only. Untracked github-pages plans ignored. SEO plan status rewrite is dirt, not this work.

**Score: 8/10**
**Ship-clean: yes** (contract files only)
**Locks held:** advertised `… | bash -s -- --with-cli` · `WITH_CLI=0` · no vendor · `dc-open` product untouched

---

## Code Review Summary

### Scope
- Files reviewed:
  - `README.md`, `assets/branding/copy.md`
  - `site/src/components/Install.astro`, `site/src/content/guides/install.md`
  - `skill/SKILL.md`
  - `install.sh`, `bin/dc-up`, `packaging/homebrew/dc-cli.rb`
  - `.github/workflows/ci.yml`
  - `bin/dc-open` (read; **no diff**)
  - `lib/dc-common.sh` (`dc_editor_bin` / source order) — read, not changed
  - plan + phase-01/02/03 + QA report
- Out of scope: `plans/260815-1041-github-pages-marketing-site/**`, `docs/journals/260815-1041-github-pages-plan.md`, SEO plan YAML/status edits
- Lines of code analyzed: ~60 product/docs/CI (+ ~300 read of install/ci/open)
- Review focus: advertised curl, default still wrappers-only, missing-CLI copy, CI attach HOME fix
- Updated plans: `plans/260815-first-run-curl-cli/plan.md`

### Overall Assessment

Does what the plan locked. One copy-paste string everywhere people copy. `install.sh` flag parse unchanged (`WITH_CLI=0`). `dc-up` / doctor / npm-missing / footer print that curl. No npm from `dc-up`. `dc-open` byte-untouched; CI only copies `lib/dc-common.sh` into the empty HOME so attach is not a false red on missing lib.

Small CI hole: grep pins README only. Recovery strings in `dc-up` / `install.sh` can drift. Not a ship blocker.

### Critical Issues

None.

### High Priority Findings

None.

### Medium Priority Improvements

1. **CI locks README, not the recovery path.** `grep -Fqx` on `README.md` only. `bin/dc-up` and `install.sh` `ADVERTISED_CURL` are what the four users see after a wrappers-only install. Drift there re-breaks first-run. QA already asked for `PATH=/usr/bin:/bin dc-up /tmp` exit 1 + curl grep. Add that; optional second grep on `install.sh`.

2. **Do not commit workspace dirt with this change.** Tracked SEO plan files (`plans/260815-1444-guides-issues-seo/{plan,phase-01,phase-02,phase-03}.md`) are status/frontmatter rewrites, not curl work. Untracked github-pages plan tree is leftover. `git add -A` would mix them. Stage the 9 contract files + this plan only.

### Low Priority Suggestions

1. **`skill/SKILL.md` frontmatter** is abbreviated (`curl main/install.sh | bash -s -- --with-cli`). Procedure line 3 has the exact advertised line. Expand description only if agents copy YAML.

2. **Brew caveats** print the curl after `npm i -g`. Correct for curl users; a brew user who pastes curl gets a second copy in `~/bin` that can shadow brew. Lead line is already `npm i -g`. Leave it — plan asked for the string.

3. **Landing subtitle** still “Safer: clone, then `bash install.sh`”. Footnote says no-flags = wrappers only. Fine.

4. **`ADVERTISED_CURL` not shared with `dc-up`.** Hardcoded twin. Sharing across install + wrapper is YAGNI. CI check in Medium #1 is enough.

5. **No CI assert `WITH_CLI=0`.** Static + help text (“does not install CLI”) is obvious. Prefix install in CI still no `--with-cli`.

6. **Live `dc.brasth.com` + raw `main/install.sh`** still old until push. Pages builds from `site/**`. Local `site/dist` already has `--with-cli` (gitignored).

### Positive Observations

- Locked advertised string is exact in README, Install.astro `data-cmd`, install guide howto + fence, branding copy, skill procedure, `dc-up` stderr, `install.sh` const, brew caveats, CI grep.
- Flag tables (README + guide) still say default = wrappers only. `install.sh` usage same. `--full` not advertised.
- `WITH_CLI=0` only flipped by `--with-cli` / `--full`. npm gated. Did not vendor CLI. Did not change `dc-open`.
- Missing-CLI `dc-up` is 4 lines, no auto-npm. TUI **start** is `tea.ExecProcess` (full stderr), not `stayCmd`/`compactLines`. Isolated `PATH=/usr/bin:/bin ./bin/dc-up /tmp` → exit 1, advertised curl on stderr.
- Attach CI: prefix install has no sibling `../lib`; `HOME=no-editors` hid `~/.config/devcontainer/dc-common.sh`. Copy from repo `lib/` into that HOME is the right fallback. Product source order unchanged (`../lib` then `$HOME/.config`).
- HowTo JSON-LD step text is the new curl (`guides/install.md` frontmatter).

### Recommended Actions

1. Ship contract files as-is. Do not add SEO / github-pages dirt.
2. Optional follow-up: CI `dc-up` missing-CLI grep (Medium #1).
3. After push: spot-check live copy button + raw `install.sh` on `main`.
4. Phase 0: text the 4. Not a code blocker.

### Metrics
- Type Coverage: n/a (bash + copy). `go.mod` unchanged.
- Test Coverage: `go test -count=1 ./cmd/dc-tui` **29/29 PASS** (0.220s). TUI cover 55.2% (QA; out of scope). No install unit tests.
- Linting Issues: `bash -n install.sh bin/dc-up bin/dc-open` PASS. `gofmt -l cmd/dc-tui` empty.
- Contract: advertised line exact in 7 required surfaces + brew + CI. Old `| bash` (no flags) gone from product files.

### Unresolved questions

- GitHub Actions not executed here. Attach + grep inspected + local repro only.
- Did not run `install.sh` end-to-end (would write `~/bin` + rc). Static + CI prefix path.
- Live site / raw `main/install.sh` not fetched.
