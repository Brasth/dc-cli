---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers. ALWAYS use dc-exec / dc-exec --service NAME — NEVER docker exec. Disk full: dc-df then dc-prune --yes. Port clash on dc-up: shows holder, prompt or --take-ports. NEVER docker system prune -af --volumes. Install: curl main/install.sh. No project .devcontainer edits."
version: 16
created: "2026-08-13"
updated: "2026-08-14"
---
## When to Use
Use automatically for official @devcontainers/cli without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-tui, dc-up, dc-down, dc-open, extra ports, override.json, Docker disk full / ENOSPC / no space left.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (`docker ps`).
2. Never edit project `.devcontainer/devcontainer.json`.
3. Interactive: `dc-tui` (current folder; padded clickable buttons if Go-built; hover; `?` more) or `dc-tui --all` (fleet). Labels: start/shell/open host/attach vscode/ports. Header shows compact disk (`dc-df`). Key `d` = full `dc-df`. After shell/logs the TUI resumes. open = host editor; attach = VS Code Remote. ports = sidecar. Install: `curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/main/install.sh | bash`.
4. Start: `dc-up` (project config) then auto `dc-forward`. Only `dc-up --ports` if user accepts REPLACE. On ENOSPC, `dc-up` prints `dc-df` / `dc-prune` hints. On **port already allocated**, `dc-up` lists the holder container (name/compose/folder); on a TTY it asks to stop and retry; agents use `dc-up --take-ports` / `--yes` / `DC_UP_TAKE_PORTS=1`. Do not invent `docker system prune`.
5. Exec (mandatory): never `docker exec`. Always `dc-exec` from the project folder. App: `dc-exec -- cmd`. Other services: `dc-exec --list` then `dc-exec --service NAME -- cmd` / `--id`. Stopped service is started first.
6. Stop: `dc-down` / `--rm` / `--compose` / `--all --yes`.
7. Open: `dc-open` host; `--attach` VS Code only.
8. Ports: `dc-forward` sidecar (not host socat on Colima).
9. Disk: `dc-df` (report) → `dc-prune` dry-run → `dc-prune --yes` (cache + dangling images + nets + orphan sidecars). Unused tagged images: `dc-prune --all --yes`. One named volume: `dc-prune --volume NAME --yes` only when user names it. Colima still full after prune: `dc-prune --colima-hint`.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- **Never** `docker system prune -af --volumes` — deletes named DB / node_modules volumes.
- **Never** `docker exec NAME` — use `dc-exec`.
- Named volumes are listed by `dc-df --volumes` but not auto-deleted.
- Colima disk cap is a VM size; prune frees inside the VM only.
- Host port clash (e.g. two stacks both bind 9001): `dc-up` shows the holder; `--take-ports` stops foreign holders and retries once. Never stop same-workspace containers.
- Zed/Sublime cannot attach. Native Windows unsupported.

## Verification
1. `dc-df --help`, `dc-prune --help`, `dc-up --help` work.
2. `dc-prune` without `--yes` is dry-run (exit 0, no delete).
3. `dc-prune --volume x` without `--yes` does not delete.
4. `dc-ls --json --all` with no containers prints `[]`.
