---
title: "Colima ports and host clashes"
description: "dc-forward reconciles owned sidecars (one app or --id). --take-ports stops labeled holders only; unlabeled is report-only."
h1: "The app is up. The Mac has nothing on :3000."
updated: 2026-08-16
howto: true
steps:
  - name: Start the project
    text: dc-up starts compose, then reconciles dc-forward (needs one labeled app or --id).
  - name: If the browser cannot reach the app
    text: dc-forward. Host socat to 172.x does not work on Colima.
  - name: If compose says port is already allocated
    text: Read the holder list. --take-ports stops labeled foreign stacks only. Unlabeled holders stay report-only.
faq:
  - q: Does Zed publish app ports?
    a: "No. Zed Dev Containers do not support forwardPorts. dc-up already runs dc-forward. Attach in Zed after that (Project: Open Remote → Connect Dev Container)."
---

`dc-up` starts `.devcontainer/docker-compose.yml`. That file often has **no** `3000:3000`. The app listens **inside**; the Mac has nothing on `:3000`.

`dc-forward` **reconciles** owned `alpine/socat` sidecars **on the Docker network**. Host `socat` → `172.x` does **not** work on Colima. Identity is the **host+container pair** (so `9001:80` stays asymmetric). Requires exactly one labeled app or `--id`. Any requested mapping that cannot be ensured fails the command (and `dc-up` after a successful start).

```bash
dc-forward                 # reconcile compose `app.ports` + forwardPorts
dc-forward 3000
dc-forward 9001:80
dc-forward --status
dc-forward --stop
```

`dc-up` runs this after a successful start (`--no-forward` / `DC_FORWARD=0` skips). `dc-down` removes workspace-owned sidecars.

`dc-up --ports` is **not** “add 3000”. It **replaces** the whole `devcontainer.json`.

## Host port already allocated

Two projects can publish the same host port (for example both `wordpress-mitm` on `9001`). Compose then fails.

```text
dc-up
# dc-up: host port conflict
#   ports: 9001
#   holders:
#     d11b00f0880b  other-project-wordpress-mitm-1  compose=other
# Stop the holder(s) above and retry dc-up for this folder? [y/N]
```

| Mode | Behavior |
|---|---|
| TTY | lists holder → ask y/N → stop **labeled** foreign holders → retry once |
| `dc-up --take-ports` / `--yes` | same without prompt; only **positively labeled** foreign stacks/sidecars |
| non-TTY without flag | lists holder + how to re-run; does **not** stop |

Stops **other** labeled workspaces only. Unlabeled, ambiguous, or inspect-unknown holders are **report-only** — they are not stopped. Prefer stopping a sibling stack over editing project compose ports.

Full flag table: [README ports](https://github.com/Brasth/dc-cli#ports-colima) and [host port conflict](https://github.com/Brasth/dc-cli#host-port-conflict).
