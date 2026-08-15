---
date: 2026-08-15
session: distribute-cli
---

# Journal: 2026-08-15 — distribute CLI, not desktop

## Context

User wanted a desktop version so multiple people can use dc-cli. Research said no to a shared GUI. Picked Homebrew + release binaries.

## What Happened

- Locked option A (distribute). Parked GUI. Rejected shared team board.
- Added `scripts/pack-release.sh`, tag `release.yml`, install.sh prefers release kit + prebuilt TUI.
- Homebrew formula template uses `bin/` + `lib/dc-common.sh` — not libexec symlinks (`dirname $0` would miss `../lib`).
- Local pack + prebuilt install + brew-layout `dc-ls --json --all` = `[]`.

## Reflection

Desktop was the wrong ask. Install friction was the real one.

## Decisions Made

| Decision | Rationale | Impact |
|---|---|---|
| No Wails/Electron | TUI already clicks | Stay a CLI kit |
| Formula bin+lib | Scripts use `dirname $0/../lib` | Works without editing every bin |
| Tap is a second repo | Standard `brew tap Brasth/dc-cli` | User must create `homebrew-dc-cli` |

## Next Steps

- Cut a `v*` tag so assets exist
- Create `Brasth/homebrew-dc-cli` and fill SHAs
