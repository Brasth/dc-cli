---
title: "dc-open --attach tells Zed users the first-party path"
description: "Stop saying Zed cannot attach. VS Code URI unchanged. No extension."
status: done
priority: P2
effort: 1h
branch: main
tags:
  - dc-open
  - zed
  - docs
blockedBy: []
blocks: []
created: '2026-08-15T12:31:00.000Z'
createdBy: 'ck:ask'
source: skill
---

# dc-open --attach Zed path

## Overview

Docs already teach Zed first-party attach. `dc-open --attach` and TUI `a` still print “Zed cannot attach.” Close the lie. No new tile, no Zed extension, no Remote URI for Zed.

## Verdict

| Ask | Answer |
|---|---|
| Write a Zed extension | No |
| Launch Zed’s remote picker | No |
| Change `dc-open` without `--attach` | No |
| VS Code `--attach` URI | Unchanged |

## Behavior

| Editor | `--attach` |
|---|---|
| `code` on PATH / .app | Current `vscode-remote://` URI |
| Zed, no `code` | Print `dc-up` → `dc-open` → Project: Open Remote → Connect Dev Container. Exit 0 |
| Else (Sublime / none) | Still cannot. Mention Zed path. Exit 2 |

TUI `a` stays `dc-open --attach`. Help / more must not say “Zed cannot attach.”

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 1 | [Message + tests](./phase-01-message.md) | done | `bin/dc-open`, TUI help/more, CI |

## Success criteria

- [x] `code` path unchanged (still needs a running labeled container)
- [x] Fake `zed` on PATH, no `code`: exit 0, stdout has `Connect Dev Container`, never `cannot attach`
- [x] No editors: exit 2, stderr has the Zed steps, never `Zed/Sublime cannot attach` (CI Linux; this Mac cannot hide Zed.app)
- [x] TUI help + more panel do not say Zed cannot attach
- [x] No new Go deps, no site route, no desktop

## Next steps

- Ship. Review: [reports/260815-attach-zed-review.md](./reports/260815-attach-zed-review.md) — **9/10, ship-clean yes**.
- Let CI confirm Linux no-editor exit 2.
- Out of scope stays out: no release tag / Homebrew, no TUI leftover, no SEO plan dirt.

## Out of scope

- Release tag / Homebrew (separate)
- TUI leftover (confirm-click, reload while leaving)
- Leftover SEO / github-pages plan files
