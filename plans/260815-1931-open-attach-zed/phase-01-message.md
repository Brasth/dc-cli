---
title: "Phase 1: editor-aware --attach message"
status: done
---

# Phase 01: Message + tests

## Context

- Plan: [plan.md](./plan.md)
- Contract: `bin/dc-open` `--attach` is VS Code only. Zed owns attach.

## Files

- **Modify:** `bin/dc-open`, `bin/dc-tui` (help only), `cmd/dc-tui/main.go`, `cmd/dc-tui/view.go`, `cmd/dc-tui/main_test.go`, `.github/workflows/ci.yml`
- **Maybe:** `README.md` attach row, `skill/SKILL.md`, `site/src/content/guides/tui.md` key table
- **Create:** none required (CI inline is enough)
- **Delete:** none

## Steps

1. Before the VS Code URI, if `dc_editor_bin code` is empty:
   - If `dc_editor_bin zed` works: print first-party steps, exit 0 (TUI status, not err).
   - Else: print cannot + Zed steps, exit 2.
2. Usage text: `--attach` is VS Code URI; without `code`, Zed users get the steps.
3. TUI `helpText` + `morePanel`: Zed attaches itself; Sublime cannot.
4. Tests: Go help/more; CI PATH-isolated `dc-open --attach`.

## Validation

- `bash -n bin/dc-open`
- `go test ./cmd/dc-tui`
- Isolated PATH cases in CI (no Docker)
