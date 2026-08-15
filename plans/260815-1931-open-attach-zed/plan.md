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

## Reports

| Pass | Report | Result |
|------|--------|--------|
| QA | [reports/260815-attach-zed-qa.md](./reports/260815-attach-zed-qa.md) | 29/29 PASS; Mac cannot hide Zed.app |
| Review | [reports/260815-attach-zed-review.md](./reports/260815-attach-zed-review.md) | **9/10, ship-clean yes** |

## Next steps

None for this plan. Plan **done**.

CI Linux no-editor exit 2 is the remaining proof (this Mac cannot hide Zed.app). Not a new phase.

## Out of scope

- Release tag / Homebrew (separate)
- TUI leftover (confirm-click, reload while leaving)
- Leftover SEO / github-pages plan files

## Close

Shipped 2026-08-15. `dc-open --attach` and TUI `a` now teach Zed first-party steps (print, exit 0) when `code` is missing; VS Code URI path unchanged; no-editor still exit 2 + Sublime-only cannot. QA 29/29; review 9/10 ship-clean. No new phases. Out of scope stays out (release/Homebrew, TUI leftover, leftover SEO / github-pages plans).
