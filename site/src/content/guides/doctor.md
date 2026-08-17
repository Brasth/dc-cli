---
title: "dc-doctor read-only checks"
description: "dc-doctor diagnoses the engine and workspace without mutating Docker or project files. Human text and --json share one check list."
h1: "Diagnose first. Never autofix."
updated: 2026-08-17
howto: true
steps:
  - name: Run doctor
    text: dc-doctor or dc-doctor --json from the project folder.
  - name: Read the summary
    text: usable (exit 0, warnings allowed) or blocked (exit 1). Exit 2 is a bad flag or path only.
  - name: Then recover
    text: Use existing commands — dc-up, dc-up --create-nets, dc-net --ensure, dc-forward --status, dc-df, dc-prune. Doctor does not run them.
faq:
  - q: Does dc-doctor fix anything?
    a: No. It is read-only. install.sh Doctor lines are install-time presence only.
  - q: Missing .devcontainer — is that a blocker?
    a: No. Missing config is a warning. A compose-only folder is kind=compose (start via docker compose). Official CLI is required only for kind=devcontainer.
  - q: Should agents parse human text?
    a: No. Pass --json. schemaVersion 1. Same 18 check ids in fixed order.
---

`dc-doctor` is the day-2 diagnostic. It never creates, starts, stops, or removes containers. It never edits `.devcontainer` or shell rc.

```bash
dc-doctor                 # this folder
dc-doctor ~/src/app
dc-doctor --json          # one JSON document on stdout
```

| Exit | Meaning |
|---|---|
| `0` | usable (warnings allowed) |
| `1` | one or more blockers |
| `2` | invalid invocation only (unknown flag, not a directory) |

Checks are a closed list of 18 ids (bash, common library, Docker, Colima, official CLI, workspace path, duplicate labels, stack, desired/actual ports, required networks, stale owned sidecars, disk, dc-cli version/channel). Human and JSON are the same opinion. Doctor never creates a network.

When a **qualified floor** is recorded in `docs/qualification/devcontainer-cli-floor.md`, a present CLI below that floor is a **blocker**. The floor is unpublished in v0.8.0 — doctor reports the installed version and does not invent a minimum.

Run this **before** `dc-prune --yes` or `dc-up --take-ports`.

See [README doctor](https://github.com/Brasth/dc-cli#doctor).
