---
title: "dc-tui UX pass — fix the board"
description: "Make the existing Bubble Tea board scannable and safe. No desktop app."
status: review
priority: P1
effort: 4h
branch: main
tags:
  - tui
  - ux
  - dc-tui
blockedBy: []
blocks: []
created: '2026-08-15T16:25:00.000Z'
createdBy: 'ck:ask'
source: skill
---

# dc-tui UX pass

## Overview

The click-board is the product. It currently dumps the whole CLI as 11 equal tiles. This plan fixes **information architecture, safety, and a tight visual pass** in `dc-tui`. It does not add a `.app`.

**Good enough:** start, shell, and stop this folder without reading `?`.

## Verdict

| Ask | Answer |
|---|---|
| Ship a desktop GUI because TUI looks off | No |
| Restyle + regroup the Go board | Yes |
| Embed PTY / Wails / new Charm deps | No this pass |
| Rewrite bash `bin/dc-tui` gum menu | No. Keys stay valid |

## Locked decisions

| Decision | Value |
|---|---|
| Surface | Go `cmd/dc-tui` only |
| Primary verbs | **start** · **shell** · **stop** |
| Meta verbs | open, attach, ports, logs, fleet, more, quit |
| Danger | **rm** on its own, confirm y/n (Esc/n cancels) |
| Invalid actions | Disabled tile + one reason. Never silent no-op |
| Status vs error | Two slots. `dc-df` is not red |
| Keyboard | j/k or arrows + Enter = selected row |
| Shell / logs | Still `ExecProcess`. Paint leave line, then “back from …” |
| Logs viewport / PTY | Out. Reassess only if leave/return still feels like a crash |
| Prune / `--all` | Stay CLI |
| New Go modules | None (`bubbletea` + `lipgloss` only) |
| Desktop / Wails | Still cancelled |

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 1 | [Board hierarchy + prettier dump](./phase-01-board-hierarchy.md) | implemented | Groups, disable, confirm, status split, split files |
| 2 | [Cursor + leave/return](./phase-02-cursor-and-leave.md) | implemented | j/k, Enter, fleet hover, leave banner, docs |

Phase 2 depends on 1.

## Success criteria

- [x] Primary row is start / shell / stop. Meta is quieter.
- [x] `rm` asks before `dc-down --rm`. Cancel is a no-op.
- [x] Fleet and “no `.devcontainer`” never swallow keys. (logs tile still clickable; refuse is not silent)
- [ ] Last action and errors do not share one red line. (two slots exist; start exit 1 → “back from start”; leftover status+err)
- [x] Keyboard can open a fleet row and exec a stack row without the mouse.
- [x] `go test ./cmd/dc-tui` passes. CI already runs it. (15/15, 2026-08-15)
- [ ] README + `site/src/content/guides/tui.md` match the board. (copy yes; screenshots still old 11-tile row)
- [x] No desktop, no new deps, no `dc-*` contract changes.

## Out of scope

- Wails / Tauri / Electron / cask
- Embedded PTY or in-board log viewport
- Charm `bubbles` (fallback only if hitboxes break)
- `dc-prune` in the TUI
- Bash fallback rewrite
- Release tag / Homebrew bump (do after this ships if we want it on curl)

## Review (2026-08-15)

Report: [reports/260815-tui-ux-review.md](./reports/260815-tui-ux-review.md)

Phases 1–2 implemented. Locked decisions held (Go only, no PTY, no new deps). **Request changes** — not ship-clean.

## Next steps

1. `dc-up` exit 1 must not become teal `back from start`. Benign swallow is for shell/logs only.
2. Truncate header / disk / ports / more / status to `width` so hitboxes match visual rows. Test a long path + `more`.
3. Writing `err` clears `status` (and vice versa). Disable logs tile when no container.
4. While `leaving`, drop `reloadMsg` or `refuse` invalid `stack:N` instead of silent return.
5. Replace README + `site/public/images/tui.png`. Confirm-`y` needs a fake runner before a unit test.

## Unresolved

- tea.Quit vs pending `leaveTickMsg` not executed.
- Confirm `y` has no inject seam; a naive test would run `dc-down --rm`.
- Visual tokens stay the current lipgloss set.
