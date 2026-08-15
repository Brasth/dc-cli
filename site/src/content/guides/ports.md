---
title: "Colima ports and host clashes"
description: "dc-forward publishes app ports on Colima. dc-up shows the holder when a host port is already allocated."
h1: "The app is up. The Mac has nothing on :3000."
updated: 2026-08-15
howto: true
steps:
  - name: Start the project
    text: dc-up starts compose, then runs dc-forward.
  - name: If the browser cannot reach the app
    text: dc-forward. Host socat to 172.x does not work on Colima.
  - name: If compose says port is already allocated
    text: Read the holder list. Retry with dc-up --take-ports, or stop the other stack.
faq:
  - q: Does Zed publish app ports?
    a: "No. Zed Dev Containers do not support forwardPorts. dc-up already runs dc-forward. Attach in Zed after that (Project: Open Remote → Connect Dev Container)."
---

`dc-up` starts `.devcontainer/docker-compose.yml`. That file often has **no** `3000:3000`. The app listens **inside**; the Mac has nothing on `:3000`.

`dc-forward` starts `alpine/socat` **on the Docker network**. Host `socat` → `172.x` does **not** work on Colima.

```bash
dc-forward                 # compose app.ports + forwardPorts
dc-forward 3000
dc-forward --status
dc-forward --stop
```

`dc-up` runs this after a successful start. `dc-down` removes the sidecars.

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
| TTY | lists holder → ask y/N → stop foreign holders → retry once |
| `dc-up --take-ports` / `--yes` | same without prompt |
| non-TTY without flag | lists holder + how to re-run; does **not** stop |

Stops **other** workspaces only. Prefer stopping a sibling stack over editing project compose ports.

Full flag table: [README ports](https://github.com/Brasth/dc-cli#ports-colima) and [host port conflict](https://github.com/Brasth/dc-cli#host-port-conflict).
