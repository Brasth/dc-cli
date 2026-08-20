---
title: "No Docker engine"
description: "dc-cli installs without Docker. Installer, doctor, up, and TUI guide recovery. Never auto-installs an engine."
order: 25
---

# No Docker engine

dc-cli needs a Docker engine. It does **not** install, start, or stop one for you.

## What you will see

| Surface | Behavior |
|---|---|
| `install.sh` | Helpers still install. Readiness block explains what is missing. |
| `dc-doctor` | Blocker on `docker_cli` / `docker_daemon` with a host code and remediation. |
| `dc-up` | Refuses early with the same recovery text. |
| `dc-tui` | Blocked setup screen: `[d]` Desktop guide, `[c]` copy Colima setup, `[r]` retry, `[q]` quit. |

## Codes

- `docker_cli_missing` — `docker` not on PATH
- `docker_engine_missing` — no engine evidence on the machine
- `docker_engine_stopped` — engine present but not reachable
- `docker_permission_denied` — socket permission / group issue
- `docker_context_invalid` — bad context / `DOCKER_HOST`
- `docker_split_brain` — more than one live engine (`dc-engine --fix`)

## Beginner path

1. Install [Docker Desktop](https://docs.docker.com/desktop/)
2. Wait until it is ready
3. `dc-doctor`
4. `dc-tui` or `dc-up`

## Lightweight macOS path

```bash
brew install docker colima
colima start
dc-doctor
dc-tui
```

One live engine only — Colima **or** Desktop, not both.
