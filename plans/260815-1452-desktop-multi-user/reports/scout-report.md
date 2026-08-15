---
type: scout
date: 2026-08-15
---

# Scout Report

## Summary

dc-cli is a per-login bash kit + optional Bubble Tea board. It shells out to local `docker` / `devcontainer`. No daemon, no auth, no prebuilt binaries, no Homebrew, no GUI. Multi-user and desktop are distribution/UI problems, not missing server features.

## Relevant Files

- `bin/dc-up` — `devcontainer up`; can stop foreign port holders
- `bin/dc-exec` — app via `devcontainer exec`; others via `docker exec` (`-it` if TTY)
- `bin/dc-forward` — Colima-safe socat sidecar + host `-p`
- `bin/dc-down` — stack stop; `--all --yes` is fleet-wide
- `bin/dc-ls` / `bin/dc-ps` / `bin/dc-df` / `bin/dc-prune` / `bin/dc-open`
- `bin/dc-tui` — bash fallback menu
- `lib/dc-common.sh` — labels, path match, holders, forward helpers
- `cmd/dc-tui/main.go` — Bubble Tea; ExecProcess vs CombinedOutput
- `install.sh` — copy to `~/bin`, config to `~/.config/devcontainer`, optional local `go build`
- `config/override.json` — replace-not-merge ports
- `.github/workflows/ci.yml` — linux smoke only, no artifacts
- `README.md` — platform matrix; native Windows unsupported

## Findings

### Single host, single login

Install prefix `$HOME/bin`. Runtime lib `$HOME/.config/devcontainer`. Docker context inherited. Label is host folder path. Fleet = every labeled container on that engine.

### TUI is not an API

`package main` only. JSON CLIs are the reuse surface. Shell/logs/`dc-up` prompt need a TTY.

### Distribution is source + curl

Latest GitHub release = source tarball, not binaries. CI does not upload artifacts. No GoReleaser, Formula, cask, dmg.

## Recommendations

1. Treat "multiple users" as N local installs.
2. Reuse `dc-* --json`, do not wrap Bubble Tea.
3. Do not add a listen port.

## Unresolved Questions

- Product shape (distribute vs GUI vs shared host) is a user decision.
