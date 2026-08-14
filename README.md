# dc-cli

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
curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/main/install.sh | bash
```

Always tracks the **latest GitHub release**. Safer: clone, then `bash install.sh`.

| Flag | What you get |
|---|---|
| *(default)* | wrappers in `~/bin` only — **not** the official CLI |
| `--with-cli` | `npm i -g @devcontainers/cli` if missing |
| `--with-skill` | copy the agent skill into homes you already have |
| `--full` | `--with-cli` + `--with-skill` |
| `--ref v0.4.3` / `main` | pin a tag or use HEAD |

Needs: bash 4+, Docker (Colima or Desktop). Go 1.22+ builds the clickable TUI; otherwise you get the bash menu. `source ~/.zshrc` or `~/.bashrc` after install.

## Daily flow

Run these from **your** project folder (the one with `.devcontainer`):

```bash
cd /path/to/your/project
dc-tui                          # or the CLI below

dc-up                           # start the labeled app, then publish ports
dc-exec                         # bash in the app
dc-exec --list                  # labeled app + other services in that compose project
dc-exec --service NAME          # start that compose service if it is down, then bash
dc-exec --service NAME -- cmd   # same, run a command
dc-forward                      # if the host browser cannot reach the app port
dc-down                         # stop the app + drop sidecars
```

**App vs other services**

| Target | Command | How |
|---|---|---|
| labeled **app** | `dc-exec` | `devcontainer exec` (remoteUser, cwd, features) |
| any other compose **service** | `dc-exec --service NAME` | `docker start` if exited, then `docker exec` |

`NAME` is whatever `docker compose` calls the service (`dc-exec --list`). Click a **stack row** in the TUI for the same path. `e` / **shell** is always the app.

## TUI

```
dc-tui              # this folder (cwd, then git root)
dc-tui ~/src/app
dc-tui --all        # every labeled workspace
```

Equal-width button grid. Compact header. Stack rows: **up** / **down**, hover, click to exec (starts the box first).

| Button | Key | Does |
|---|---|---|
| **start** | `u` | `dc-up` + `dc-forward` |
| **shell** | `e` | `dc-exec` — **app** only |
| **open** | `o` | host editor on the bind-mount |
| **attach** | `a` | VS Code Remote **into** the app (`code` only) |
| **ports** | `p` | `dc-forward` (Colima sidecar) |
| **stop** / **rm** | `s` / `x` | `dc-down` / `dc-down --rm` |
| **logs** | `l` | `docker logs -f` on the app |
| **fleet** | `f` | other workspaces |
| **more** / **quit** | `?` / `q` | legend / exit |

After **shell** / **logs**, the TUI comes back. A normal `exit` is not a crash.

**open ≠ attach.** Open = files on the Mac/Linux host. Attach = VS Code terminal/debugger inside Linux. Zed and Sublime cannot attach.

## Commands (all of them)

| Command | Purpose |
|---|---|
| `dc-tui [dir]` | this folder |
| `dc-tui --all` | fleet |
| `dc-up [dir]` | project config, then `dc-forward` |
| `dc-up --no-forward` | skip sidecars (`DC_FORWARD=0`) |
| `dc-up --ports` | **REPLACE** project config (not a merge) |
| `dc-exec` | app shell |
| `dc-exec --list` | stack table |
| `dc-exec --service NAME` | other compose service (starts if down) |
| `dc-exec --id NAME` | that container (starts if down) |
| `dc-down` / `--rm` / `--compose` | stop / delete / whole compose project |
| `dc-down --all --yes` | every labeled box |
| `dc-ls [--json] [--all]` | labeled app list |
| `dc-open` / `--attach` | host editor / VS Code attach |
| `dc-forward` / `3000` / `--stop` | sidecar publish |
| `dc-ps` | docker + labels |

There is **no** `dc` meta-binary.

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

## Editors

| Editor | Host open | Attach into container |
|---|---|---|
| Zed | yes (`/Applications/Zed.app` is enough) | **no** |
| VS Code | yes | **yes** — `dc-open --attach` |
| Sublime | yes | **no** |

`--editor` / `$DC_EDITOR` / first of `zed`, `code`, `subl` on PATH **or** as a macOS `.app`.

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
