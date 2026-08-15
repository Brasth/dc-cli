# DC-CLI

Site: [dc.brasth.com](https://dc.brasth.com) · Guides: [dc.brasth.com/guide](https://dc.brasth.com/guide/) · Issues: [dc.brasth.com/issues](https://dc.brasth.com/issues/)

Host-global helpers around official [`@devcontainers/cli`](https://github.com/devcontainers/cli). **No VS Code required. Does not edit** project `.devcontainer/devcontainer.json`.

Upstream only has `up` / `exec`. This repo adds **stop**, **list**, **open**, **port publish**, and a **TUI**.

```
this folder
   │
   ├─ dc-tui          clickable board (default)
   ├─ dc-up           start the labeled app + publish ports
   ├─ dc-exec         shell in the app
   ├─ dc-exec --service NAME   start (if needed) + shell in another service
   ├─ dc-forward      host localhost:3000 → app (Colima-safe)
   └─ dc-down         stop the app (sidecars go too)
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

Always tracks the **latest GitHub release** (prebuilt clickable TUI). The one-liner passes `--with-cli` so `dc-up` can start. Safer: clone, then `bash install.sh` (no flags = wrappers only; add `--with-cli` if you need official CLI).

macOS/Linux via Homebrew (same kit, no Go required):

```bash
brew tap Brasth/dc-cli
brew install dc-cli
```

Each person needs their **own** Docker/Colima. Do not share one engine — fleet, prune, and `--take-ports` see the whole daemon.

| Flag | What you get |
|---|---|
| *(default)* | wrappers in `~/bin` only — **not** the official CLI |
| `--with-cli` | `npm i -g @devcontainers/cli` if missing |
| `--with-skill` | copy the agent skill into homes you already have |
| `--full` | `--with-cli` + `--with-skill` |
| `--ref v0.4.3` / `main` | pin a tag or use HEAD |

Needs: bash 4+, Docker (Colima or Desktop). Release/brew installs include the clickable TUI. A source install builds it when Go is on PATH; otherwise you get the bash menu. `source ~/.zshrc` or `~/.bashrc` after a curl install (Homebrew already has PATH).

## Daily flow

Run these from **your** project folder (the one with `.devcontainer`):

```bash
cd /path/to/your/project
dc-tui                          # or the CLI below

dc-up                           # start the labeled app, then publish ports
dc-up --take-ports              # if host ports clash, stop holders and retry
dc-exec                         # bash in the app (starts it if it is down)
dc-exec --list                  # labeled app + other services in that compose project
dc-exec --service NAME          # start that compose service if it is down, then bash
dc-exec --service NAME -- cmd   # same, run a command (works without a TTY)
dc-forward                      # if the host browser cannot reach the app port
dc-down                         # stop FULL compose stack (app+db+mitm+…)
dc-down --app                   # labeled app only (+ its sidecars)
dc-down --rm                    # compose down (remove stack containers)

# out of disk (ENOSPC) — report, then safe reclaim
dc-df                           # Docker + Colima disk report
dc-prune                        # dry-run
dc-prune --yes                  # cache + dangling images + nets + orphan sidecars
```

**App vs other services**

| Target | Command | How |
|---|---|---|
| labeled **app** | `dc-exec` | `devcontainer exec` (remoteUser, cwd, features) |
| any other compose **service** | `dc-exec --service NAME` | `docker start` if exited, then `docker exec` |

`NAME` is whatever `docker compose` calls the service (`dc-exec --list`). Click a **stack row** in the TUI for the same path. `e` / **shell** is always the app.

## TUI

<img width="820" alt="dc-tui board: start, shell, stop, then meta actions" src="site/public/images/tui.png" />

```
dc-tui              # this folder (cwd, then git root)
dc-tui ~/src/app
dc-tui --all        # every labeled workspace
```

Primary row: **start** · **shell** · **stop**. Meta row is quieter. **rm** is danger and asks `y/n`. Stack / fleet rows: **up** / **down**, `j`/`k` or click, Enter to open or exec.

Startup draws the **dc-cli** mark (host frame + amber pip). Any key skips. `DC_TUI_NO_SPLASH=1` skips it. The compact mark stays in the header.

| Button | Key | Does |
|---|---|---|
| **start** | `u` | `dc-up` + `dc-forward` (needs `.devcontainer`) |
| **shell** | `e` | `dc-exec` — **app** only |
| **stop** | `s` | full stack `dc-down` |
| **rm** | `x` | `dc-down --rm` after `y` |
| **open** | `o` | host editor on the bind-mount |
| **attach** | `a` | VS Code Remote URI; Zed prints Connect Dev Container steps |
| **ports** | `p` | `dc-forward` (Colima sidecar) |
| **logs** | `l` | `docker logs -f` on the app |
| **fleet** | `f` | other workspaces |
| **more** / **quit** | `?` / `q` | legend / exit |
| **disk** | `d` | `dc-df` report (stays in TUI) |
| **rows** | `j`/`k`, Enter | move cursor; fleet opens a folder, stack execs |

Header shows a compact disk line from `dc-df`. Reclaim stays CLI-only (`dc-prune --yes`).

After **shell** / **logs** / **start**, the board comes back. A normal `exit` is not a crash.

**open ≠ attach.** Open = files on the Mac/Linux host. `a` / `dc-open --attach` is the VS Code Remote URI into Linux. Zed attaches itself. Sublime cannot.

## Commands (all of them)

| Command | Purpose |
|---|---|
| `dc-tui [dir]` | this folder |
| `dc-tui --all` | fleet |
| `dc-up [dir]` | project config, then `dc-forward` |
| `dc-up --no-forward` | skip sidecars (`DC_FORWARD=0`) |
| `dc-up --ports` | **REPLACE** project config (not a merge) |
| `dc-up --take-ports` / `--yes` | on host port clash: stop holders, retry once |
| `dc-exec` | app shell |
| `dc-exec --list` | stack table |
| `dc-exec --service NAME` | other compose service (starts if down) |
| `dc-exec --id NAME` | that container (starts if down) |
| `dc-down` | **full compose stack** (app + db + mitm + … + sidecars) |
| `dc-down --app` | labeled app only (+ sidecars) |
| `dc-down --rm` / `--volumes` | compose down / down -v |
| `dc-down --compose` | alias for default full stack (scripts) |
| `dc-down --all --yes` | every labeled workspace stack |
| `dc-ls [--json] [--all]` | labeled app list |
| `dc-open` / `--attach` | host editor / VS Code attach |
| `dc-forward` / `3000` / `--stop` | sidecar publish |
| `dc-ps` | docker + labels |
| `dc-df` / `--json` / `--volumes` | disk report (read-only) |
| `dc-prune` / `--yes` | safe reclaim (cache, dangling, nets, orphan sidecars) |
| `dc-prune --all --yes` | also unused tagged images |
| `dc-prune --volume NAME --yes` | delete **one** named volume |
| `dc-prune --colima-hint` | grow Colima VM disk guidance |

There is **no** `dc` meta-binary. **Never** `docker system prune -af --volumes` — use `dc-df` / `dc-prune`.

## Ports (Colima)

`dc-up` starts `.devcontainer/docker-compose.yml`. That file often has **no** `3000:3000`. The app listens **inside**; the Mac has nothing on `:3000`.

`dc-forward` starts `alpine/socat` **on the Docker network**. Host `socat` → `172.x` does **not** work on Colima.

```bash
dc-forward                 # detect app ports from compose `app.ports` + forwardPorts
dc-forward 3000
dc-forward --status
dc-forward --stop
```

`dc-up` runs this after a successful start. `dc-down` removes the sidecars.

`dc-up --ports` is **not** “add 3000”. It **replaces** the whole `devcontainer.json`.

## Disk full

When `dc-up` fails with **no space left** / ENOSPC:

```bash
dc-df                    # images / cache / volumes + Colima guest df
dc-prune                 # dry-run
dc-prune --yes           # safe set only
dc-up                    # retry
```

| Flag | Risk |
|---|---|
| `dc-prune --yes` | Low — build cache, dangling images, unused nets, orphan `dc-forward` sidecars |
| `dc-prune --all --yes` | Medium — unused **tagged** images; parked stacks rebuild on next `dc-up` |
| `dc-prune --volume NAME --yes` | **High** — named volume data (DBs). One name only; never bulk |
| `docker system prune -af --volumes` | **Do not** — not wrapped; destroys named volumes |

If Colima guest `/` is still ~100% after prune, the VM disk cap is full: `dc-prune --colima-hint`. Prune does **not** grow the qemu/VZ image.

## Host port conflict

Two projects can publish the same host port (e.g. both `wordpress-mitm` on `9001`). Compose then fails with **port is already allocated**.

```bash
dc-up
# dc-up: host port conflict
#   ports: 9001
#   holders:
#     d11b00f0880b  other-project-wordpress-mitm-1  compose=other  ...
# Stop the holder(s) above and retry dc-up for this folder? [y/N]
```

| Mode | Behavior |
|---|---|
| TTY (interactive) | lists holder → ask **y/N** → stop foreign holders → retry once |
| `dc-up --take-ports` / `--yes` / `DC_UP_TAKE_PORTS=1` | same without prompt (agents / CI) |
| non-TTY without flag | lists holder + how to re-run; does **not** stop |

Stops **other** workspaces only (same-folder containers are skipped). Prefer stopping a sibling stack over editing project compose ports.

## Editors

| Editor | Host open | Inside the container |
|---|---|---|
| Zed | yes (`/Applications/Zed.app` is enough) | yes — in Zed: **Project: Open Remote → Connect Dev Container** |
| VS Code | yes | yes — `dc-open --attach` |
| Sublime | yes | **no** |

`--editor` / `$DC_EDITOR` / first of `zed`, `code`, `subl` on PATH **or** as a macOS `.app`.

`dc-open` is always the host folder. `dc-open --attach` is VS Code only. Zed (v0.218+) attaches first-party. Start here so Colima ports exist (Zed does not publish `forwardPorts`):

```bash
dc-up            # compose + dc-forward
dc-open          # host folder
# then in Zed: Project: Open Remote → Connect Dev Container
```

dc-cli owns start / stop / ports / fleet. Zed owns the editor session. No Zed extension.

## Platform

| OS | Status |
|---|---|
| macOS | Supported (Colima or Docker Desktop) |
| Linux | Supported |
| WSL2 | Best-effort (run the installer **inside** WSL) |
| Native Windows | **Not supported** |

Create and stop from the same environment so `devcontainer.local_folder` matches.

## Skill

`--with-skill` copies into existing agent homes only (`~/.pi`, `~/.claude`, `~/.codex`, `~/.gemini`, `~/.cursor`, `~/.opencode`, plus `~/.agents/skills`). Restart the agent after install.

## Maintainer

[Canvilled](https://github.com/Canvilled) (Huy Nguyen). MIT.
