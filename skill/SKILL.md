---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers. ALWAYS use dc-exec / dc-exec --service NAME — NEVER docker exec. Disk full: dc-df then dc-prune --yes (engine-wide cache/images/nets). Port clash on dc-up: shows holder, prompt or --take-ports (labeled only; unlabeled report-only). dc-forward needs one app or --id. dc-doctor is read-only. NEVER docker system prune -af --volumes. Install: curl main/install.sh | bash -s -- --with-cli standalone (or brew tap Brasth/dc-cli && brew install dc-cli). No project .devcontainer edits."
version: 23
created: "2026-08-13"
updated: "2026-08-16"
---
## When to Use
Use automatically for official @devcontainers/cli without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-tui, dc-up, dc-down, dc-open, extra ports, override.json, Docker disk full / ENOSPC / no space left.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (read-only `dc-doctor` or `docker ps`). `dc-doctor` never mutates.
2. Never edit project `.devcontainer/devcontainer.json`.
3. Interactive: `dc-tui` (current folder; Go/release/brew click-board) or `dc-tui --all` (fleet). Logo splash first (any key skips; `DC_TUI_NO_SPLASH=1`). Primary: start/shell/stop. Meta: open/attach/ports/logs. `rm` asks y/n. `j`/`k` + enter on rows. Header shows compact disk (`dc-df`). Key `d` = full `dc-df`. After shell/logs the board resumes. open = host editor; attach = VS Code Remote. ports = sidecar. Install: `curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli` (standalone CLI) or `brew tap Brasth/dc-cli && brew install dc-cli`. Diagnose with read-only `dc-doctor` before destructive recovery.
4. Start: `dc-up` (project config) then auto `dc-forward`. Only `dc-up --ports` if user accepts REPLACE. On ENOSPC, `dc-up` prints `dc-df` / `dc-prune` hints. On **port already allocated**, `dc-up` lists the holder container (name/compose/folder); on a TTY it asks to stop and retry; agents use `dc-up --take-ports` / `--yes` / `DC_UP_TAKE_PORTS=1`. That stops **positively labeled** foreign stacks/sidecars only. Unlabeled, ambiguous, or inspect-unknown holders are **report-only** — they are not stopped. Do not invent `docker system prune`.
5. Exec (mandatory): never `docker exec`. Always `dc-exec` from the project folder. App: `dc-exec -- cmd`. Other services: `dc-exec --list` then `dc-exec --service NAME -- cmd` / `--id`. Stopped service is started first.
6. Stop: `dc-down` = **full compose stack** (app + db + mitm + … + sidecars). App-only: `dc-down --app`. Remove stack: `dc-down --rm`. Nuclear: `dc-down --all --yes`. Old `--compose` is an alias for the default full stack.
7. Open: `dc-open` host. `--attach` with `code` = VS Code Remote URI. Without `code`, Zed: prints `dc-up` → `dc-open` → Project: Open Remote → Connect Dev Container (exit 0). Zed does not publish `forwardPorts` (`dc-forward` does).
8. Ports: `dc-forward` reconciles owned sidecars (not host socat on Colima). Host+container pair identity (`9001:80` stays asymmetric). Requires exactly one labeled app or `--id`. Any mapping it cannot ensure fails (and fails `dc-up` after a successful start).
9. Disk: `dc-df` (report) → `dc-prune` dry-run → `dc-prune --yes` (**engine-wide** cache + dangling images + nets; **owned-only** orphan sidecars). Unused tagged images: `dc-prune --all --yes` (also engine-wide). One named volume: `dc-prune --volume NAME --yes` only when user names it (owned-only; mount inventory must succeed). Colima still full after prune: `dc-prune --colima-hint`.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- **Never** `docker system prune -af --volumes` — deletes named DB / node_modules volumes.
- **Never** `docker exec NAME` — use `dc-exec`.
- Named volumes are listed by `dc-df --volumes` but not auto-deleted.
- Colima disk cap is a VM size; prune frees inside the VM only.
- Host port clash (e.g. two stacks both bind 9001): `dc-up` shows the holder; `--take-ports` stops **labeled** foreign holders (full stack via `dc-down`) and retries once. Unlabeled / ambiguous / inspect-unknown = report-only. Never stop same-workspace containers.
- `dc-forward` with two labeled apps in one folder fails closed unless you pass `--id`.
- Default `dc-down` is the whole compose project — not only the labeled app. That is intentional so host ports free up.
- Sublime cannot attach. Zed attach is first-party; `dc-open --attach` only prints those steps. Native Windows unsupported.

## Verification
1. `dc-df --help`, `dc-prune --help`, `dc-up --help`, `dc-doctor --help` work. `dc-doctor` is read-only.
2. `dc-prune` without `--yes` is dry-run (exit 0, no delete).
3. `dc-prune --volume x` without `--yes` does not delete.
4. `dc-ls --json --all` with no containers prints `[]`.
