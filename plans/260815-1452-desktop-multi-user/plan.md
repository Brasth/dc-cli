---
title: "dc-cli desktop + multi-user feasibility"
description: "Can we ship a desktop app so multiple people can use dc-cli? Verdict: distribute first; local GUI optional; shared multi-user is a different product."
status: in_progress
priority: P2
branch: main
tags:
  - architecture
  - desktop
  - distribution
blockedBy: []
blocks: []
created: '2026-08-15T14:52:00.000Z'
createdBy: 'ck:ask'
source: skill
---

# dc-cli desktop + multi-user

## Overview

**Yes** for many people installing a local tool. **No** for one shared desktop controlling one Docker host.

dc-cli is a per-login wrapper over local `docker` + `@devcontainers/cli`. No daemon, no auth, no tenant labels. Fleet / prune / `--take-ports` are engine-wide. A GUI does not create multi-user. It only replaces the TUI click-board.

## Verdict

| Ask | Answer |
|---|---|
| Ship so more people can use it | Yes. Homebrew tap + release binaries. |
| Local desktop GUI on each Mac | Possible. Optional. Still one user per Docker context. |
| Shared team desktop / remote board | No. That is Coder / Codespaces, not this repo. |

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 1 | [Clarify + distribute](./phase-01-clarify-and-distribute.md) | in_progress | Release kit + brew formula template |
| 2 | [Local desktop GUI](./phase-02-local-desktop-gui.md) | cancelled | Parked. No named non-CLI user. |

## Dependencies

- Decision: "more installers" vs "GUI" vs "shared team host"
- Apple Developer ID only if shipping a `.app` / official Homebrew cask
- Do not block on marketing site plans

## Success Criteria

- [x] User picks one of the three product shapes (A: distribute)
- [x] Multi-user = N local installs, not one shared session
- [x] Shared remote control explicitly out of scope
- [x] First `v*` tag publishes 4 tarballs (`v0.6.0`)
- [x] `Brasth/homebrew-dc-cli` exists and installs

## Research

- [Feasibility report](./research/researcher-01-desktop-multi-user.md)
- [Scout](./reports/scout-report.md)
