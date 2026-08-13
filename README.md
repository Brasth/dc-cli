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
| `dc-up [dir]` | `devcontainer up` using the **project** config (safe default) |
| `dc-up --ports` | **REPLACE** project config with `~/.config/devcontainer/override.json` |
| `dc-exec` / `dc-exec -- cmd` | exec in the workspace container |
| `dc-down` | **stop** the labeled container (keep it for next `dc-up`) |
| `dc-down --rm` | stop + remove |
| `dc-down --compose` | stop/down the compose project if labeled |
| `dc-down --all --yes` | every `devcontainer.local_folder` container |
| `dc-ls [--json] [--all]` | machine-readable list (same labels as `dc-down`) |
| `dc-open [dir]` | open host folder in zed / code / subl |
| `dc-open --attach` | VS Code attach into the running container |
| `dc-ps` | list docker + labeled containers |
| `dc-forward 9000` | extra host→container port via `socat` |

There is **no** `dc` meta-binary. Scripts keep calling `dc-up` / `dc-down`.

## `dc-tui`

Default is the **current folder**, not every Docker container. With Go, install builds a **clickable** Bubble Tea TUI (mouse + keys). Without Go, you get the bash menu.

```
dc-tui              # this workspace
dc-tui ~/src/app
dc-tui --all        # fleet
```

Click the buttons (`up` `exec` `open` `attach` `stop` `rm` `logs` `fleet` `quit`). Fleet: click a row to open that folder. Keys still work: `u` `e` `o` `a` `s` `x` `l` `f` `q` `r`.

`u` / `e` / `l` **leave** the TUI so you see official CLI / docker output. `u` is refused if the folder has no `.devcontainer` (use CLI `dc-up --ports` if you really want REPLACE).

## Editors (host only)

| Editor | `dc-open` | In-container attach |
|---|---|---|
| Zed (`zed`) | Host folder | **No** |
| VS Code (`code`) | Host folder | **Yes** (`dc-open --attach`) |
| Sublime (`subl`) | Host folder | **No** |

Bind-mount **is** the project content. Pick editor with `--editor`, `DC_EDITOR`, or first of `zed`, `code`, `subl` on PATH. Nothing is downloaded.

## `dc-up --ports` is a replace, not a merge

`--override-config` **replaces** the project `devcontainer.json`. Image, features, and mounts from the project file are dropped.

Prefer `dc-forward` if you only need another port.

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
