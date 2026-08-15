# QA: release / install paths

Date: 2026-08-15 15:12 +0700
Host: macos darwin/arm64, Go 1.24
Scope: pack + install + brew layout. Did not cut a tag or touch Homebrew tap.

## Test Results Overview

| # | Check | Result |
|---|---|---|
| 1 | `bash -n` install.sh pack-release.sh print-release-shas.sh lib/dc-common.sh bin/dc-up | PASS |
| 2 | `go test ./cmd/dc-tui` | PASS 4/4 |
| 3 | `scripts/pack-release.sh 0.0.0-test darwin arm64 /tmp/dc-pack-test` | PASS |
| 4 | Extract tarball; `bin/dc-tui` not shebang | PASS (Mach-O arm64) |
| 5 | PREFIX prebuilt `install.sh`; no shebang; `--help` | PASS |
| 6 | PREFIX source `install.sh`; go-build; `--help` | PASS |
| 7 | Brew `bin/`+`lib/` ; `dc-up --help` ; `dc-ls --json --all` | PASS (`[]`) |
| 8 | README exact curl one-liner | PASS |

Total: 8 run, 8 passed, 0 failed, 0 skipped.

## Coverage Metrics

`go test -cover ./cmd/dc-tui`: **13.6%** statements.

| Func | Cover |
|---|---|
| `benignExecErr` | 100% |
| `renderButtons` | 87.2% |
| `buttonSpecs` | 66.7% |
| `main` / `Update` / `handleKey` / `handleClick` / `View` / resolve / editor | 0% |

No line/branch report beyond `-func`. Bash kit has no coverage harness.

Tests:

- `TestRenderButtonsEqualWidth` PASS
- `TestRenderButtonsTwoRows` PASS
- `TestRenderButtonsNarrowWrapsMore` PASS
- `TestBenignExecErr` PASS

## Failed Tests

None.

## Performance Metrics

| Step | Time / size |
|---|---|
| `go test ./cmd/dc-tui` | 0.22–0.56s |
| pack darwin/arm64 | tarball 1,390,308 B |
| packed `dc-tui` (`-s -w`) | 3,564,306 B Mach-O |
| source `go build` `dc-tui` | 5,117,186 B Mach-O |
| `dc-tui --help` / `dc-up --help` | instant |

No slow test. Size delta source vs pack expected (`-trimpath -ldflags -s -w` only in pack).

## Build Status

PASS. No compile warning. Pack output:

```
/tmp/dc-pack-test/dc-cli-0.0.0-test-darwin-arm64.tar.gz
```

Tarball contents (after follow-up): `install.sh`, `bin/` (10 cmds), `lib/dc-common.sh`, `config/override.json`, `skill/SKILL.md`. No `cmd/`.

## Step evidence

### 1. bash -n

Exit 0. No stderr.

### 2. go test

```
ok  github.com/Canvilled/dc-cli/cmd/dc-tui
coverage: 13.6% of statements
```

### 3–4. Pack + extract

`file` on extracted `bin/dc-tui`: `Mach-O 64-bit executable arm64`
`head -c 2` = `cf fa` (MH_MAGIC_64), not `#!`.

### 5. Prebuilt install

```
PREFIX=/tmp/dc-prebuilt-test bash extracted/install.sh --prefix /tmp/dc-prebuilt-test
Installed prebuilt dc-tui to /tmp/dc-prebuilt-test/dc-tui
```

Installed binary Mach-O, `--help` prints TUI legend. Doctor: docker/devcontainer/helpers OK.

### 6. Source install

Source `bin/dc-tui` is bash (`#!/usr/bin/env bash`). Install chose go-build:

```
Building clickable dc-tui (Go)...
Installed Go dc-tui to /tmp/dc-src-test/dc-tui
```

`--help` works.

### 7. Brew layout

Copied extracted `bin/*` → `/tmp/dc-brew/bin`, `lib/dc-common.sh` → `/tmp/dc-brew/lib`.

- `/tmp/dc-brew/bin/dc-up --help` exit 0
- `/tmp/dc-brew/bin/dc-ls --json --all` → `[]` exit 0
- Extra isolation (not requested): `HOME=/tmp/dc-brew-home` still works → uses `../lib`, not `~/.config`
- Extra negative: no `lib/` + isolated HOME → exit 1 `dc-common.sh not found`

### 8. README one-liner

Exact line present:

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash
```

Same string in `site/src/components/Install.astro`, `site/src/content/guides/install.md`, `skill/SKILL.md`.

## CI: would `.github/workflows/ci.yml` catch the same?

| This run | CI today | Catch? |
|---|---|---|
| bash -n install.sh / pack-release / dc-common / dc-up | yes | yes |
| bash -n print-release-shas.sh | **no** | **no** |
| bash -n bin/dc-df dc-prune | **no** | **no** |
| `go test ./cmd/dc-tui` | **no** (only `go build`) | **no** |
| pack darwin/arm64 | **no** (linux-amd64 only) | **no** (release.yml packs 4 GOOS/GOARCH, does not install/run) |
| extract + `dc-tui` not `#!` | yes | yes |
| prebuilt PREFIX install + `--help` | yes (`PATH` stripped) | yes |
| source PREFIX install + `--help` | yes | yes |
| brew `bin/`+`lib/` | partial: `$kit/bin/dc-up --help` is same relative layout, but HOME not isolated | **weak** — `~/.config` fallback can hide missing `lib/` after first install step |
| `dc-ls --json --all` | yes (`test "$out" = "[]"`) | yes on empty host |
| README curl one-liner | **no** | **no** |

`release.yml` packs all 4 tarballs on tag. Does not run install, shebang assert, or `--help`.

## Critical Issues

None blocking ship of these paths on this host.

## Unexpected output

- Install rewrote `$HOME/.config/devcontainer/dc-common.sh` (expected). Kept existing `override.json`. PATH block already in zshrc/bashrc — no extra append.
- `dc-ls --json --all` = `[]` (no labeled containers here). Matches CI empty-json gate.
- Packed vs source `dc-tui` sizes differ (strip flags). Not a bug.

## Recommendations

1. CI: add `go test ./cmd/dc-tui`.
2. CI: `bash -n scripts/print-release-shas.sh bin/dc-df bin/dc-prune`.
3. CI pack job: HOME-isolated brew-layout (`bin/`+`lib/` only) so `../lib` cannot hide behind `~/.config`.
4. CI: `grep -qx` the README curl one-liner.
5. TUI tests: `handleKey` / `handleClick` / `resolveWorkspace` / `benignExecErr` already there; rest of `main.go` uncovered.
6. Release kit has no `skill/` — `--with-skill` from tarball `install.sh` will fail. Document or pack the skill.

## Next Steps

1. Land CI gaps above before next `v*` tag.
2. Create `Brasth/homebrew-dc-cli` + fill SHAs (`scripts/print-release-shas.sh`).
3. Optional: assert packed `dc-tui` `file`/Mach-O in a macos runner if darwin pack regressions matter.

## Unresolved questions

- Should release tarball include `skill/SKILL.md` for `--with-skill`?
- Empty `dc-ls` here — no labeled fleet to exercise JSON non-empty path.
