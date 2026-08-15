---
title: "Phase 2: Local desktop GUI"
status: pending
---

# Phase 02: Local desktop GUI

## Context Links

- Plan: [plan.md](./plan.md)
- Phase 1 must ship first or in parallel only if a real non-CLI user exists
- TUI: `cmd/dc-tui/main.go` shells out to `dc-*`

## Overview

- **Priority:** low
- **Status:** cancelled
- **Description:** Optional local window that calls the same CLIs. Not a multi-user server.

## Key Insights

- TUI is already a click board. GUI value is: dock icon, folder picker, embedded PTY.
- Do not import `package main`. Reuse `dc-ls --json`, `dc-exec --list --json`, `dc-df --json`.
- Wails v2 fits (Go backend + WebView). Tauri adds Rust. Electron is too heavy. Fyne is native but a full rewrite of the board.
- Shell, logs, and `dc-up` still need a PTY (or `dc-up --yes`).

## Requirements

### Functional

- Board: start / stop / rm / ports / open / attach / disk
- Embedded terminal for shell + `docker logs -f`
- Folder picker for workspace; fleet list read-only then switch
- Inherit the logged-in user's `HOME`, `PATH`, Colima Docker context

### Non-functional

- No listen port. No remote control.
- macOS first. Linux later. Native Windows out.
- Do not reimplement Docker logic in the GUI.

## Architecture

```
dc.app (Wails v2)
  UI  ──invoke──► Go bindings
                    │
                    ├─ exec dc-ls/dc-df/dc-open/dc-forward/dc-down
                    └─ PTY: dc-up, dc-exec, docker logs -f
  still requires: docker, @devcontainers/cli, Colima or Desktop
```

## Related Code Files

- **Modify:** none of `bin/` contracts
- **Create:** `desktop/` (Wails) or `cmd/dc-app/` — isolated tree like `site/`
- **Delete:** none

## Implementation Steps

1. PoC: Wails window lists `dc-ls --json` and runs `dc-df --json`.
2. PoC: embed PTY for `dc-exec`. If this fails, kill the GUI idea.
3. Wire stay-in-board actions. Confirm button = `dc-up --yes` (no TTY prompt).
4. macOS `.app` + ad-hoc sign. Homebrew cask only after Developer ID + notarize.
5. Keep CLI as the product. App is a front.

## Todo

- [ ] PTY PoC works with Colima socket as a normal login user
- [ ] No Docker API in the GUI process
- [ ] Gatekeeper path documented

## Success Criteria

- Non-terminal user can start / shell / stop one workspace
- Two people still means two Macs (or two logins with two Docker contexts)
- `dc-prune` stays CLI-only

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| App launched from Finder has empty PATH / no Colima sock | Board empty or "docker missing" | Resolve `~/.colima/default/docker.sock`; ship PATH bootstrap |
| Recreate TUI in web UI | Drift | Thin wrapper only |
| Users expect shared team view | Wrong product | Refuse; point at Coder |
| Notarization cost/time | Cask blocked | Formula (CLI) first; cask later |

## Next Steps

- Do not start this phase without a named user who will not use `dc-tui`.
