---
title: "No Docker engine"
description: "dc-cli installs without Docker. Installer, doctor, up, and TUI guide recovery. Never auto-installs an engine."
h1: "Helpers install. Docker is still your job."
updated: 2026-08-20
howto: true
steps:
  - name: Install dc-cli anyway
    text: bash install.sh still installs helpers when Docker is missing. Read the Host Docker readiness block.
  - name: Install or start an engine
    text: Beginners use Docker Desktop. Lightweight macOS path is brew install docker colima && colima start.
  - name: Recheck
    text: dc doctor then dc. On the blocked board press r after Desktop/Colima is ready.
faq:
  - q: Does install fail without Docker?
    a: No. Helpers install. Machine readiness is reported separately. Day-2 diagnosis is dc-doctor.
  - q: Will dc-cli start Docker for me?
    a: If an engine is already installed, dc-recover --yes / TUI [f] can start it. It still never auto-installs Desktop or Colima.
  - q: What does the TUI show?
    a: A blocked setup screen. Engine already there — [f] try fix. Empty machine — [d] Desktop guide, [c] copy Colima setup. [r] retry, [q] quit.
---

dc-cli needs a Docker engine. It does **not** install one for you. If Desktop or Colima is already installed, `dc-recover --yes` / TUI `[f]` can start it.

## What you will see

| Surface | Behavior |
|---|---|
| `install.sh` | Helpers still install. Readiness block explains what is missing. |
| `dc doctor` | Blocker on `docker_cli` / `docker_daemon` with a host code and remediation. |
| `dc up` | Refuses early with the same recovery text. |
| `dc` | Blocked setup screen: `[f]` try fix (existing engine), `[d]` Desktop guide, `[c]` copy Colima, `[r]` retry, `[q]` quit. |

## Codes

- `docker_cli_missing` — `docker` not on PATH
- `docker_engine_missing` — no engine evidence on the machine
- `docker_engine_stopped` — engine present but not reachable
- `docker_permission_denied` — socket permission / group issue
- `docker_context_invalid` — bad context / `DOCKER_HOST`
- `docker_split_brain` — more than one live engine (`dc-recover --yes` or `dc-engine --fix`)

## Beginner path

1. Install [Docker Desktop](https://docs.docker.com/desktop/)
2. Wait until it is ready
3. `dc-doctor`
4. `dc` or `dc up`

## Lightweight macOS path

```bash
brew install docker colima
colima start
dc doctor
dc
```

One live engine only — Colima **or** Desktop, not both.

See also [doctor](/guide/doctor/) and [install](/guide/install/).
