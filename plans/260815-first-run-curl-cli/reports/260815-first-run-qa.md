# QA: first-run advertised curl includes `--with-cli`

Date: 2026-08-15 20:09 +0700
Host: macos Darwin arm64, Go 1.26.6 (CI pins 1.24)
Scope: advertised one-liner contract. No npm. No network. No source edits.

Advertised line (exact):

```
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

## Test Results Overview

| # | Check | Result |
|---|---|---|
| 1 | `bash -n install.sh` | PASS |
| 2 | `bash -n bin/dc-up` | PASS |
| 3 | `go test ./cmd/dc-tui` | PASS 29/29 |
| 4 | `grep -Fqx` advertised line in `README.md` | PASS |
| 5 | Same line in listed copy files | PASS (7/7) |
| 6 | `install.sh` no-flags `WITH_CLI` default 0 | PASS |
| 7 | `PATH=/usr/bin:/bin ./bin/dc-up /tmp` exit 1 + curl on stderr | PASS |
| 8 | CI `grep -Fqx` uses new line | PASS |
| 9 | CI attach job copies `lib/dc-common.sh` into `$HOME/.config/devcontainer` | PASS |

Total required checks: 9 run, 9 passed, 0 failed, 0 skipped.

`go test` is TUI suite only — does not exercise install/curl. 29 unit tests, 29 passed, 0 failed, 0 skipped.

### Copy locations (task list)

| File | Match |
|---|---|
| `README.md:23` | exact (also `grep -Fqx` green) |
| `site/src/components/Install.astro:2` | exact (`const install = '…'`) |
| `site/src/content/guides/install.md:9` + `:21` | exact (howto + fenced) |
| `skill/SKILL.md:14` | exact in procedure |
| `assets/branding/copy.md:14` | exact |
| `bin/dc-up:82` | exact (stderr, 2-space indent) |
| `install.sh:11` | exact (`ADVERTISED_CURL=`) |

Bonus (not in required list, same line):

- `.github/workflows/ci.yml:105` CI grep
- `packaging/homebrew/dc-cli.rb:49` caveats
- `site/dist/index.html` `data-cmd` + `site/dist/guide/install/index.html` (built copy already new)

README + install guide flag tables still say no-flags = wrappers only.

## Coverage Metrics

`go test -count=1 -covermode=atomic ./cmd/dc-tui`: **55.2%** statements. Below 80% bar. Out of scope for this contract (TUI, not install).

No Go coverage of `install.sh` / `bin/dc-up`. Contract proven by static grep + PATH-restricted run.

## Failed Tests

None.

## Performance Metrics

| Command | Time |
|---|---|
| `bash -n install.sh bin/dc-up` | <0.1s |
| `go test ./cmd/dc-tui` first (cache) | cached ok |
| `go test -count=1 ./cmd/dc-tui` | 0.224s |
| `go test -count=1 -covermode=atomic` | 0.453s |
| `PATH=/usr/bin:/bin ./bin/dc-up /tmp` | <0.1s |

No slow tests. No network.

## Build Status

No full product/site build (task: no npm). `bash -n` + `go test` green. `site/dist` already contains new curl — landing `data-cmd` matches advertised line.

CI workflow inspected only, not executed here.

## Check detail

### 6. `WITH_CLI` default 0

`install.sh`:

```
7:WITH_CLI=0
30:    --with-cli) WITH_CLI=1 ;;
32:    --full) WITH_CLI=1; WITH_SKILL=1 ;;
210:if [[ "$WITH_CLI" -eq 1 ]]; then
221:    npm install -g @devcontainers/cli
```

No other assignments. No-flags never sets `WITH_CLI=1`. npm gated on flag. Did not execute `install.sh` (would mutate `~/bin` + rc files).

CI `install to prefix` still `bash install.sh --prefix` with no `--with-cli`.

### 7. Missing-CLI `dc-up`

```
PATH=/usr/bin:/bin ./bin/dc-up /tmp
exit=1
```

stderr:

```
devcontainer CLI not found. Advertised install:
  curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
or: npm i -g @devcontainers/cli
or: bash install.sh --with-cli
```

`command -v devcontainer` false under `/usr/bin:/bin`. Sourced repo `lib/dc-common.sh` (relative), then failed at CLI check. Did not start a container.

`install.sh` doctor + npm-missing + footer also print `${ADVERTISED_CURL}`.

### 8. CI

`ci.yml:105`:

```
grep -Fqx 'curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli' README.md
```

Attach job (`dc-open --attach teaches Zed path`):

```
empty="$RUNNER_TEMP/no-editors"
mkdir -p "$empty/.config/devcontainer"
cp lib/dc-common.sh "$empty/.config/devcontainer/dc-common.sh"
PATH="/usr/bin:/bin" HOME="$empty" ... dc-open --attach
```

Matches phase-03: fake HOME no longer hides `dc-common.sh`. Product `dc-open` unchanged.

## Critical Issues

None.

## Recommendations

1. Add CI step for check 7 (`PATH=/usr/bin:/bin dc-up /tmp` exit 1 + grep advertised curl). Grep README only; first-run stderr can drift.
2. Optional: `skill/SKILL.md` frontmatter description is abbreviated (`curl main/install.sh | bash -s -- --with-cli`). Body has exact line. Expand if agents copy description only.
3. TUI stmt cover 55% — not this plan. No install/curl unit test.

## Next Steps

1. After pages deploy, spot-check live `dc.brasth.com` copy button (not done: no network).
2. After merge, raw `install.sh` on `main` is what curl users get — local tree verified only.
3. Phase 0 (text the 4 users) still pending in plan. Out of QA scope.

## Unresolved questions

- Live site + GitHub raw `install.sh` not fetched (task: no network). Local src + `site/dist` only.
- Did not run no-flags `install.sh` end-to-end (would write HOME). Static + CI prefix path used instead.
- `go test` does not assert the curl string. Contract is bash/docs.

Status: DONE
Summary: All 9 required contract checks pass. Advertised `--with-cli` curl is one string in README, site, skill, branding, `dc-up`, `install.sh`. Default still `WITH_CLI=0`. Restricted-PATH `dc-up /tmp` exits 1 and prints that curl. CI grep + attach `dc-common.sh` copy look correct.
Concerns/Blockers: none
