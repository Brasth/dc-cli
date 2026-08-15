---
title: First-run curl includes official CLI
description: The advertised curl must leave `devcontainer` on PATH. Four silent users all used wrappers-only.
status: completed
priority: P1
effort: 3h
branch: main
tags:
  - install
  - first-run
  - docs
blockedBy: []
blocks: []
created: '2026-08-15'
createdBy: 'ck:ask'
source: skill
---

# First-run curl includes official CLI

## Overview

Four users have Docker Desktop or Colima. All installed with the landing/README one-liner that is wrappers + TUI only. `dc-up` then dies without `devcontainer` on PATH. This plan makes the advertised install able to start a folder.

## Locked decisions

| Decision | Value |
|---|---|
| Advertised curl | `curl -fsSL …/install.sh \| bash -s -- --with-cli` |
| `install.sh` default (no flags) | Still wrappers only |
| Advertised is not `--full` | `--with-cli` only |
| Error / doctor string | Same advertised curl |
| Official CLI vendor / default flip | No |
| New `v*` | Optional, not required |

## Phases

| Phase | Name | Status | File |
|---|---|---|---|
| 0 | Text the 4 (ops) | pending | [plan](./plan.md#phase-0--text-the-4-ops) |
| 1 | [Advertised curl](./phase-01-advertised-curl.md) | completed | One string everywhere people copy |
| 2 | [Missing-CLI copy](./phase-02-missing-cli-copy.md) | completed | `dc-up` + `install.sh` doctor |
| 3 | [CI attach + grep](./phase-03-ci-attach.md) | completed | README grep + red attach job |

## Success

- [x] README, site, install guide, skill copy the `--with-cli` piped line
- [x] Flag table still says no-flags = wrappers only
- [x] `dc-up` / doctor print that curl
- [x] CI grep + attach HOME copy + isolated `dc-up` missing-CLI grep
- [x] No-flags install still does not run npm (`WITH_CLI=0`)

## Review

- 2026-08-15: code review **8/10**, ship-clean **yes** (contract files only). Report: [reports/260815-first-run-review.md](./reports/260815-first-run-review.md)
- No critical / high. Medium: CI locks README only (not `dc-up` / `install.sh`); do not commit SEO plan dirt or untracked github-pages plans with this change.

## Next

1. Commit only first-run contract files + this plan. Exclude `plans/260815-1444-guides-issues-seo/*` status rewrite and untracked github-pages plans.
2. Push so Pages rebuilds landing copy and raw `main/install.sh` is what curl users get.
3. Phase 0: text the 4 users the block below.
4. Optional after they retry: if start works and they stay quiet, freeze.

## Cook

`/ck:cook plans/260815-first-run-curl-cli/plan.md`

## Phase 0 — Text the 4 (ops)

```text
command -v devcontainer || echo MISSING
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
source ~/.zshrc
cd <their project>
dc-tui
```
