---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers: dc-tui (clickable if Go; this folder; --all fleet), dc-up (auto dc-forward), dc-exec, dc-down, dc-ls, dc-open (zed/code/subl; --attach VS Code only), dc-ps, dc-forward (Colima sidecar). Install: curl main/install.sh (latest release). No project .devcontainer edits."
version: 8
created: "2026-08-13"
updated: "2026-08-14"
---
## When to Use
Use automatically for official @devcontainers/cli without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-tui, dc-up, dc-down, dc-open, extra ports, override.json.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (`docker ps`).
2. Never edit project `.devcontainer/devcontainer.json`.
3. Interactive: `dc-tui` (current folder; click buttons if Go-built; `?` more) or `dc-tui --all` (fleet). Labels: start/shell/open host/attach vscode/ports. open = host editor; attach = VS Code Remote in the container. ports = sidecar publish. Do not invent a `dc` alias. Install latest: `curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/main/install.sh | bash`.
4. Start: `dc-up` (project config) then auto `dc-forward`. Only `dc-up --ports` if user accepts REPLACE via ~/.config/devcontainer/override.json. TUI `u` refuses folders with no `.devcontainer`. `DC_FORWARD=0` / `--no-forward` skips sidecars.
5. Exec: `dc-exec` / `dc-exec -- <cmd>`. List: `dc-ls --json` / `dc-ps`.
6. Stop with `dc-down` (same label matcher as `dc-ls`). Default stop only; `--rm`; `--compose`; `--all --yes`.
7. Open content: `dc-open` host folder (zed/code/subl on PATH **or** macOS .app). TUI **open** must not ExecProcess (that looked like a crash). `dc-open --attach` is VS Code only. Bind-mount is the files.
8. Extra ports: `dc-forward` (Docker sidecar on the compose network). Host socat to 172.x fails on Colima. TUI **ports** / `p`. Do not use override just for ports.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- `appPort` is create-time; later ports need dc-forward (sidecar, not host socat).
- Zed/Sublime cannot attach inside the container.
- Colima must be running. Native Windows unsupported.

## Verification
1. `dc-down --help`, `dc-up --help`, `dc-tui --help` work.
2. `dc-down --all` without `--yes` exits 2.
3. `dc-ls --json --all` with no containers prints `[]`.
4. Default `dc-tui` header is this folder, not a global list.
