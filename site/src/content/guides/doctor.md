---
title: "dc-doctor read-only checks"
description: "dc-doctor diagnoses the engine and workspace without mutating Docker or project files. Human text and --json share one check list."
h1: "Diagnose first. Never autofix."
updated: 2026-08-20
howto: true
steps:
  - name: Run doctor
    text: dc-doctor or dc-doctor --json from the project folder.
  - name: Read the summary
    text: usable (exit 0, warnings allowed) or blocked (exit 1). Exit 2 is a bad flag or path only.
  - name: Then recover
    text: Use dc-recover --yes for an existing engine (start, pick-one, context, group). Or dc-engine --fix, dc-up, dc-up --create-nets, dc-net --ensure, dc-forward --status, dc-df, dc-prune. Doctor does not run them.
faq:
  - q: Does dc-doctor fix anything?
    a: No. It is read-only. install.sh prints host Docker readiness but never auto-installs an engine. See the no-docker guide.
  - q: Missing .devcontainer — is that a blocker?
    a: No. Missing config is a warning. A compose-only folder is kind=compose (start via docker compose or docker-compose). Official CLI is required only for kind=devcontainer.
  - q: Should agents parse human text?
    a: No. Pass --json. schemaVersion 1. Same 18 check ids in fixed order.
  - q: Desktop shows containers but dc-up fails?
    a: Check docker_context. It reports the CLI engine and socket. Two live engines is a blocker (split_brain). Same daemon on two paths is not a split (Linux Desktop ~/.docker/desktop/docker.sock plus a proxy /var/run). Run dc-engine --fix for the exact recovery commands. Doctor stays read-only.
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

Checks are a closed list of 18 ids (bash, common library, Docker, Colima, official CLI, workspace path, duplicate labels, stack, desired/actual ports, required networks, stale owned sidecars, disk, dc-cli version/channel). `docker_context` is engine + socket + extra live engines — not the context name alone. Two live daemons is a blocker (`error.code=split_brain`). Two paths to the same daemon (inode, Colima default dual sock, or matching `docker info` ID) are one engine. Human output prints a Fix block on split. `dc-engine --fix` prints the same recipe; `--yes` only runs `docker context use`. Doctor never creates a network. Doctor never starts or stops an engine.

When a **qualified floor** is recorded in `docs/qualification/devcontainer-cli-floor.md`, a present CLI below that floor is a **blocker**. The floor is unpublished in v0.8.0 — doctor reports the installed version and does not invent a minimum.

Run this **before** `dc-prune --yes`, `dc-up --take-ports`, or `dc-up` when the machine also has Docker Desktop.

Stuck after this: [dc-recover](/guide/stuck/).
