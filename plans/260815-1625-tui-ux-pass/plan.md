---
title: "dc-tui UX pass — fix the board"
description: "Make the existing Bubble Tea board scannable and safe. No desktop app."
status: completed
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
| 1 | [Board hierarchy + prettier dump](./phase-01-board-hierarchy.md) | completed | Groups, disable, confirm, status split, split files |
| 2 | [Cursor + leave/return](./phase-02-cursor-and-leave.md) | completed | j/k, Enter, fleet hover, leave banner, docs |

Phase 2 depends on 1. Ship-clean follow-up after 1–2 is also **done** (no extra phase).

## Success criteria

- [x] Primary row is start / shell / stop. Meta is quieter.
- [x] `rm` asks before `dc-down --rm`. Cancel is a no-op.
- [x] Fleet and “no `.devcontainer`” never swallow keys. logs tile is disabled with no container.
- [x] Exclusive status/err slots. start exit 1 is err and survives reload; shell/logs still swallow 1.
- [x] Visual-width clip + stack row ≤ `w` so click hitboxes match paint.
- [x] Keyboard can open a fleet row and exec a stack row without the mouse.
- [x] `go test ./cmd/dc-tui` 27+ PASS. Ship-clean review **9/10**.
- [x] README + `site/src/content/guides/tui.md` match the board. Screenshot is the 3-row board.
- [x] No desktop, no new deps, no `dc-*` contract changes.

## Out of scope

- Wails / Tauri / Electron / cask
- Embedded PTY or in-board log viewport
- Charm `bubbles` (fallback only if hitboxes break)
- `dc-prune` in the TUI
- Bash fallback rewrite
- Release tag / Homebrew bump (do after this ships if we want it on curl)

## Out of this pass (leftover)

Not blocking close. Do not treat as open work for this plan:

- While `leaving`, drop `reloadMsg` (50ms window can reorder stack before exec). Invalid `stack:N` already `refuse`s.

## Review (2026-08-15)

| Pass | Report | Score |
|------|--------|-------|
| UX pass | [reports/260815-tui-ux-review.md](./reports/260815-tui-ux-review.md) | 6/10 — request changes |
| Ship-clean | [reports/260815-tui-shipclean-review.md](./reports/260815-tui-shipclean-review.md) | **9/10 — ship** |
| QA | [reports/260815-tui-shipclean-qa.md](./reports/260815-tui-shipclean-qa.md) | 27/27 PASS |

Phases 1–2 implemented. Ship-clean landed: start exit 1 is err and survives reload; visual-width clip + stack row width so hitboxes match; exclusive status/err slots; new 3-row screenshot.

## Next steps

None for this plan. Plan **completed**.

Later (not this pass): `reloadMsg` while leaving — see leftover above. Release/Homebrew bump is a separate ship.

## Unresolved

- tea.Quit vs pending `leaveTickMsg` not executed.
- Visual tokens stay the current lipgloss set.
