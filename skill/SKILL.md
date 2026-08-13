---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers: dc-up (safe default), dc-up --ports, dc-exec, dc-down, dc-ps, dc-forward. No project .devcontainer edits."
version: 3
created: "2026-08-13"
updated: "2026-08-13"
---
## When to Use
Use automatically for official @devcontainers/cli without VS Code and without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-up, dc-down, extra ports, override.json.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (`docker ps`).
2. Never edit project `.devcontainer/devcontainer.json`.
3. Start: `dc-up` (project config). Only `dc-up --ports` if user accepts REPLACE via ~/.config/devcontainer/override.json.
4. Exec: `dc-exec` / `dc-exec -- <cmd>`. List: `dc-ps`.
5. Stop with `dc-down` (this repo; upstream has no down). Default stop only; `--rm`; `--compose`; `--all --yes`.
6. Extra ports: `dc-forward` (socat), do not use override just for ports.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- `appPort` is create-time; later ports need dc-forward.
- Colima must be running.

## Verification
1. `dc-down --help` and `dc-up --help` work.
2. `dc-down --all` without `--yes` exits 2.
3. Default `dc-up --help` documents safe default.