# dc-cli

Host-global helpers for the official [`@devcontainers/cli`](https://github.com/devcontainers/cli). No VS Code required. **Does not edit** project `.devcontainer/devcontainer.json`.

The upstream CLI only ships `up` / `exec`. This repo adds **`dc-down`**, **`dc-ls`**, **`dc-open`**, and **`dc-tui`** (this folder by default; `--all` is the fleet).

## Install

```bash
git clone https://github.com/Canvilled/dc-cli.git
cd dc-cli
bash install.sh
source ~/.bashrc   # Linux / WSL
source ~/.zshrc    # macOS zsh
```

Default install is **wrappers only**. It does **not** install `@devcontainers/cli`.

```bash
bash install.sh --with-cli      # npm i -g @devcontainers/cli if missing
bash install.sh --with-skill    # copy SKILL.md into each agent you already have
bash install.sh --full          # --with-cli + --with-skill
```

One-liner always installs the **latest GitHub release** (installer on `main` fetches `releases/latest`):

```bash
curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/main/install.sh | bash
```

Pin or use HEAD:

```bash
bash install.sh --ref v0.2.0    # that tag
bash install.sh --ref main      # default branch, not a release
```

(`curl | bash` runs remote code. Clone + `bash install.sh` is safer and uses the tree you cloned.)

Needs: `bash` 4+, Docker. Node 18+ + npm only for `--with-cli` / `--full`. `socat` only for `dc-forward`. **Go 1.22+** on PATH builds the clickable `dc-tui`; otherwise the bash menu is installed.

```bash
curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/main/install.sh | bash -s -- --full
```

## Platform support

| OS | Status | Notes |
|---|---|---|
| macOS | Supported | What we dogfood (Colima or Docker Desktop). |
| Linux | Supported | `bash install.sh` writes `~/bin` + `~/.bashrc`. Docker Engine or Desktop. `sudo apt-get install socat` for `dc-forward`. |
| Windows + **WSL2** | Best-effort | Run the same bash installer **inside WSL**. Docker Desktop WSL backend. |
| Windows native (cmd/PowerShell) | **Not supported** | No `.ps1`. Do not run Git Bash against Docker Desktop labels mixed with `C:\` vs `/mnt/c` — `dc-down` matches `devcontainer.local_folder` as a string. |

`dc-down` / `dc-ls` compare the Docker label to the resolved workspace. Create and stop from the **same** environment (all WSL, or all macOS/Linux), or pass `dc-down --id`.

### `--with-skill` (multi-agent)

If that product’s home dir exists, the same `skill/SKILL.md` is copied to:

| Agent | Path |
|---|---|
| Shared (Agent Skills) | `~/.agents/skills/devcontainer-cli-global/` |
| Pi | `~/.pi/agent/skills/` |
| Claude Code | `~/.claude/skills/` |
| Codex | `~/.codex/skills/` |
| Gemini CLI | `~/.gemini/skills/` |
| Cursor | `~/.cursor/skills/` |
| OpenCode | `~/.opencode/skills/` |

Does **not** create a harness home just to drop a skill. Restart the agent after install. Cursor/Gemini also read `~/.agents/skills` in many setups.

## Commands

| Command | What |
|---|---|
| `dc-tui [dir]` | TUI for **this folder** (cwd, then git root) |
| `dc-tui --all` | Fleet of labeled containers; Enter drills into that folder |
| `dc-up [dir]` | `devcontainer up` using the **project** config, then **`dc-forward`** |
| `dc-up --ports` | **REPLACE** project config with `~/.config/devcontainer/override.json` |
| `dc-up --no-forward` | skip sidecar publish (`DC_FORWARD=0` too) |
| `dc-exec` / `dc-exec -- cmd` | bash in the **labeled app** |
| `dc-exec --list` | compose siblings (db, mailpit, wordpress, …) |
| `dc-exec --service db` | `docker exec` into that sibling |
| `dc-down` | **stop** the labeled container (keep it for next `dc-up`) |
| `dc-down --rm` | stop + remove |
| `dc-down --compose` | stop/down the compose project if labeled |
| `dc-down --all --yes` | every `devcontainer.local_folder` container |
| `dc-ls [--json] [--all]` | machine-readable list (same labels as `dc-down`) |
| `dc-open [dir]` | open host folder in zed / code / subl |
| `dc-open --attach` | VS Code attach into the running container |
| `dc-ps` | list docker + labeled containers |
| `dc-forward [dir]` | sidecar-publish unpublished app ports (Colima-safe) |
| `dc-forward 3000` | same, explicit host port |
| `dc-forward --stop` | remove our sidecars |

There is **no** `dc` meta-binary. Scripts keep calling `dc-up` / `dc-down`.

## `dc-tui`

Default is the **current folder**, not every Docker container. With Go, install builds a **clickable** Bubble Tea TUI (mouse + keys). Without Go, you get the bash menu.

<img width="756" height="248" alt="Screenshot 2026-08-14 at 06 59 02" src="https://github.com/user-attachments/assets/cfd7abef-eddb-41e2-b93b-2f9b2d69c414" />

```
dc-tui              # this workspace
dc-tui ~/src/app
dc-tui --all        # fleet
```

Click a **padded** button (they wrap + highlight on hover), or press `?` / **more**. After **shell** / **logs**, the TUI comes back — a normal `exit` is not a crash.

| Button | Key | What it does |
|---|---|---|
| **start** | `u` | `dc-up` then auto **`dc-forward`**. Needs `.devcontainer`. Leaves TUI for pull logs. |
| **shell** | `e` | `dc-exec` — bash in the **labeled app**. Leaves TUI. Click a **stack row** (db / mailpit / …) for siblings. |
| **open host** | `o` | `dc-open` — host editor on the **bind-mount** (Zed / VS Code / Sublime). |
| **attach vscode** | `a` | `dc-open --attach` — VS Code **Remote into** the running container. `code` only. |
| **ports** | `p` | `dc-forward` — sidecar so **host** `localhost:3000` reaches the app (Colima). |
| **stop** | `s` | `dc-down` — stop, keep the container. Sidecars removed. |
| **rm** | `x` | `dc-down --rm` — stop and delete. |
| **logs** | `l` | `docker logs -f`. Ctrl-C returns to the TUI. |
| **fleet** | `f` | Every labeled workspace. Click a row to open that folder. |
| **more** | `?` | This legend in the TUI. |
| **quit** | `q` | Exit. |

**open ≠ attach.** Open edits files on the Mac/Linux host. Attach is VS Code’s in-container terminal/debugger. Zed and Sublime cannot attach.

`start` is refused if the folder has no `.devcontainer` (use CLI `dc-up --ports` only if you accept REPLACE).

## Editors (host only)

| Editor | `dc-open` | In-container attach |
|---|---|---|
| Zed (`zed`) | Host folder | **No** |
| VS Code (`code`) | Host folder | **Yes** (`dc-open --attach`) |
| Sublime (`subl`) | Host folder | **No** |

Bind-mount **is** the project content. Pick editor with `--editor`, `DC_EDITOR`, or first of `zed`, `code`, `subl` (PATH **or** macOS `/Applications/*.app` — Zed.app is enough; you do not need `zed` on PATH). Nothing is downloaded.

**open** in the TUI stays on screen (it does not tear down). If no editor is found you get an error line, not a crash.

## `dc-up --ports` is a replace, not a merge

`--override-config` **replaces** the project `devcontainer.json`. Image, features, and mounts from the project file are dropped.

Prefer `dc-forward` if you only need another port. That starts an `alpine/socat` sidecar on the **Docker network** (same trick as `benoy-next-proxy`). Host `socat` to `172.x` does **not** work on Colima.

```bash
dc-forward                 # detect 3000/5173/… from compose + forwardPorts
dc-forward 3000            # just that port
dc-forward --status
dc-forward --stop
```

`dc-up` runs this after a successful start. `dc-down` removes the sidecars.

## `dc-down`

```bash
dc-down                 # stop
dc-down --rm            # stop + rm
dc-down --volumes       # also anonymous volumes
dc-down --id NAME       # explicit container
dc-down --all --yes     # required confirmation
```

Matching uses Docker label `devcontainer.local_folder` (cwd, then git root) via `lib/dc-common.sh` (same as `dc-ls`).

## Maintainer

[Canvilled](https://github.com/Canvilled) (Huy Nguyen).

## License

MIT
