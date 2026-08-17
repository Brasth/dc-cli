---
name: "devcontainer-cli-global"
description: "Host-global official @devcontainers/cli helpers. ALWAYS use dc-exec / dc-exec --service NAME — NEVER docker exec. Disk full: dc-df then dc-prune --yes (engine-wide cache/images/nets). Port clash on dc-up: shows holder, prompt or --take-ports (labeled only; unlabeled report-only). dc-forward needs one app or --id. dc-db opens a host DB client on a user-declared compose/forwardPorts port (never invents). dc-files leave-execs yazi/nnn in the box, else copies Linux yazi into /tmp. dc-doctor is read-only. NEVER docker system prune -af --volumes. Install: curl main/install.sh | bash -s -- --with-cli standalone (or brew tap Brasth/dc-cli && brew install dc-cli). No project .devcontainer edits."
version: 27
created: "2026-08-13"
updated: "2026-08-17"
---
## When to Use
Use automatically for official @devcontainers/cli without editing project .devcontainer files. Triggers: start/stop/exec a devcontainer, dc-tui, dc-up, dc-down, dc-open, extra ports, override.json, Docker disk full / ENOSPC / no space left.

## Procedure
1. Confirm `devcontainer` and Docker/Colima (read-only `dc-doctor` or `docker ps`). `dc-doctor` never mutates. This folder is `kind=devcontainer` if project config exists, else `kind=compose` if a root compose file exists, else not a workspace. Compose-kind is detect-only (`dc-ls` / `dc-doctor`). Do not `docker compose exec` / `compose up` yet. `dc-up` refuses compose-kind.
2. Never edit project `.devcontainer/devcontainer.json`.
3. Interactive: `dc-tui` (current folder; Go/release/brew click-board) or `dc-tui --all` (fleet). Logo splash first (any key skips; `DC_TUI_NO_SPLASH=1`). Primary: start/shell/stop. Meta: open/attach/ports/logs/top/nets/db/files. Published websites become clickable `http://127.0.0.1:PORT` tiles (`1`–`9`). Databases stay off that row — `b` / `dc-db` opens TablePlus on a **declared** compose/`forwardPorts` host port (never invents; two DBs need `--service`). `m` / `dc-files` leave-execs yazi/nnn if already in the image, else copies a Linux yazi into /tmp (never apt-get). `e` / `dc-exec` is a color shell (`hl` highlights access logs). `l` follows highlighted docker logs for the **selected stack row** (`q` back). `R` restarts that sibling via `dc-exec --service NAME --restart` (labeled app row refuses — `u`/`s`; `r` still reloads). `t` is live CPU/RAM for this folder via `dc-stats` (`q` back). `n` lists this folder's declared compose nets; `y` creates missing `external: true` bridge nets then `dc-up --create-nets`. Fleet refuses `t`, `n`, `l`, and `R`. `rm` asks y/n. `j`/`k` + enter on rows. Header shows compact disk (`dc-df`) and an app load pulse (`dc-stats`). Key `d` = full `dc-df`. After shell/files the board resumes. open = host editor; attach = VS Code Remote. ports = sidecar. Install: `curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli` (standalone CLI) or `brew tap Brasth/dc-cli && brew install dc-cli`. Diagnose with read-only `dc-doctor` before destructive recovery.
4. Start: `dc-up` (project config) then auto `dc-forward`. Only `dc-up --ports` if user accepts REPLACE. On ENOSPC, `dc-up` prints `dc-df` / `dc-prune` hints. Missing declared external nets: TTY prompt, or `dc-up --create-nets` / `--yes` / `DC_UP_CREATE_NETS=1`. `--create-nets` does not take ports. Overlay / custom IPAM / inspect-unknown = refuse. On **port already allocated**, `dc-up` lists the holder container (name/compose/folder); on a TTY it asks to stop and retry; agents use `dc-up --take-ports` / `--yes` / `DC_UP_TAKE_PORTS=1`. That stops **positively labeled** foreign stacks/sidecars only. Unlabeled, ambiguous, or inspect-unknown holders are **report-only** — they are not stopped. Do not invent `docker system prune`.
5. Exec (mandatory): never `docker exec`. Always `dc-exec` from the project folder. App: `dc-exec -- cmd`. Other services: `dc-exec --list` then `dc-exec --service NAME -- cmd` / `--id`. Stopped service is started first. Restart a stack sibling: `dc-exec --service NAME --restart` (membership-checked). `--id --restart` is refused. Never `docker restart` a stranger.
6. Stop: `dc-down` = **full compose stack** (app + db + mitm + … + sidecars). App-only: `dc-down --app`. Remove stack: `dc-down --rm`. Nuclear: `dc-down --all --yes`. Old `--compose` is an alias for the default full stack.
7. Open: `dc-open` host. `--attach` with `code` = VS Code Remote URI. Without `code`, Zed: prints `dc-up` → `dc-open` → Project: Open Remote → Connect Dev Container (exit 0). Zed does not publish `forwardPorts` (`dc-forward` does).
8. Ports: `dc-forward` reconciles owned sidecars (not host socat on Colima). Host+container pair identity (`9001:80` stays asymmetric). Requires exactly one labeled app or `--id`. Any mapping it cannot ensure fails (and fails `dc-up` after a successful start). Auto wanted set is still app|web only — `dc-db` may sidecar the **db** with an exact declared pair (`dc-forward --id $db HOST:CONTAINER`).
8b. Database: `dc-db --list` then `dc-db` / `--print` / `--service NAME`. Creds from official image env (first volume init). Stale volume → `--url`. Never put secrets in `dc-doctor --json`.
8c. Files: `dc-files` / `--service NAME`. Detects `$DC_FILES_TOOL`, yazi, nnn, lf, mc, ranger **in the box**. If none, copies a cached Linux yazi to `/tmp/dc-cli-yazi` and runs that. Never apt-get. Bind-mount editor stays `dc-open`.
9. Disk: `dc-df` (report) → `dc-prune` dry-run → `dc-prune --yes` (**engine-wide** cache + dangling images + nets; **owned-only** orphan sidecars). Unused tagged images: `dc-prune --all --yes` (also engine-wide). One named volume: `dc-prune --volume NAME --yes` only when user names it (owned-only; mount inventory must succeed). Colima still full after prune: `dc-prune --colima-hint`.
9b. Stats: `dc-stats` / `--json` is read-only CPU/RAM/net for this folder. Guest is Colima/Desktop VM (Desktop live guest is cap only). Never treat Mac host RAM as the box. TUI `t` consumes `--json`.
9c. Nets: `dc-net` / `--json` / `--ensure` is this folder only. Create missing declared `external: true` names as a default bridge. Compose-managed = list only. TUI `n` consumes `--json`; `y` runs `dc-up --create-nets`.

## Pitfalls
- `--override-config` / `dc-up --ports` replaces project config entirely.
- `dc-down --all` is nuclear; requires `--yes`.
- **Never** `docker system prune -af --volumes` — deletes named DB / node_modules volumes.
- **Never** `docker exec NAME` — use `dc-exec`.
- Restart is stack-only: `dc-exec --service NAME --restart`. `--id --restart` is refused. TUI `R` on the labeled app row is refused (`u`/`s`).
- Named volumes are listed by `dc-df --volumes` but not auto-deleted.
- Colima disk cap is a VM size; prune frees inside the VM only.
- Host port clash (e.g. two stacks both bind 9001): `dc-up` shows the holder; `--take-ports` stops **labeled** foreign holders (full stack via `dc-down`) and retries once. Unlabeled / ambiguous / inspect-unknown = report-only. Never stop same-workspace containers.
- `dc-forward` with two labeled apps in one folder fails closed unless you pass `--id`.
- Default `dc-down` is the whole compose project — not only the labeled app. That is intentional so host ports free up.
- Sublime cannot attach. Zed attach is first-party; `dc-open --attach` only prints those steps. Native Windows unsupported.

## Verification
1. `dc-df --help`, `dc-prune --help`, `dc-up --help`, `dc-doctor --help`, `dc-stats --help`, `dc-net --help` work. `dc-doctor` and `dc-stats` are read-only. `dc-net` without `--ensure` is read-only.
2. `dc-prune` without `--yes` is dry-run (exit 0, no delete).
3. `dc-prune --volume x` without `--yes` does not delete.
4. `dc-ls --json --all` with no containers prints `[]`.
