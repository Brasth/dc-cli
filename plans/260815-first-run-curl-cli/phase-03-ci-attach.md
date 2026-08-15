---
phase: 3
title: CI attach + grep
status: completed
priority: P1
dependencies: [1]
effort: 1h
---

# Phase 03: CI attach + grep

Update README `grep -Fqx` to the new one-liner.

Fix `dc-open --attach` job: `HOME=no-editors` hides `~/.config/devcontainer/dc-common.sh`. Copy `lib/dc-common.sh` into that HOME after setting it. Do not change `dc-open` product behavior.
