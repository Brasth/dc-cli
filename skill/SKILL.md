---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers: dc-tui (this folder; --all fleet), dc-up, dc-exec, dc-down, dc-ls, dc-open (zed/code/subl; --attach VS Code only), dc-ps, dc-forward. No project .devcontainer edits."
version: 4
created: "2026-08-13"
updated: "2026-08-13"
---
## When to Use
Use automatically for official @devcontainers/cli without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-tui, dc-up, dc-down, dc-open, extra ports, override.json.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (`docker ps`).
2. Never edit project `.devcontainer/devcontainer.json`.
3. Interactive: `dc-tui` (current folder) or `dc-tui --all` (fleet). Do not invent a `dc` alias.
4. Start: `dc-up` (project config). Only `dc-up --ports` if user accepts REPLACE via ~/.config/devcontainer/override.json. TUI `u` refuses folders with no `.devcontainer`.
5. Exec: `dc-exec` / `dc-exec -- <cmd>`. List: `dc-ls --json` / `dc-ps`.
6. Stop with `dc-down` (same label matcher as `dc-ls`). Default stop only; `--rm`; `--compose`; `--all --yes`.
7. Open content: `dc-open` host folder (zed/code/subl). `dc-open --attach` is VS Code only. Bind-mount is the files.
8. Extra ports: `dc-forward` (socat), do not use override just for ports.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- `appPort` is create-time; later ports need dc-forward.
- Zed/Sublime cannot attach inside the container.
- Colima must be running. Native Windows unsupported.

## Verification
1. `dc-down --help`, `dc-up --help`, `dc-tui --help` work.
2. `dc-down --all` without `--yes` exits 2.
3. `dc-ls --json --all` with no containers prints `[]`.
4. Default `dc-tui` header is this folder, not a global list.
