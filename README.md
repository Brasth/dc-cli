# dc-cli

Host-global helpers for the official [`@devcontainers/cli`](https://github.com/devcontainers/cli). No VS Code. **Does not edit** project `.devcontainer/devcontainer.json`.

The official CLI has `up` and `exec`. **It has no `down`.** That is the main gap this repo fills.

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

Tagged one-liner (prefer a release tag, not `main`):

```bash
curl -fsSL https://raw.githubusercontent.com/Canvilled/dc-cli/v0.1.4/install.sh | bash
```

(`curl | bash` runs remote code. Clone + `bash install.sh` is safer.)

Needs: `bash` 4+, Docker. Node 18+ + npm only for `--with-cli` / `--full`. `socat` only for `dc-forward`.

## Platform support

| OS | Status | Notes |
|---|---|---|
| macOS | Supported | What we dogfood (Colima or Docker Desktop). |
| Linux | Supported | `bash install.sh` writes `~/bin` + `~/.bashrc`. Docker Engine or Desktop. `sudo apt-get install socat` for `dc-forward`. |
| Windows + **WSL2** | Best-effort | Run the same bash installer **inside WSL**. Docker Desktop WSL backend. |
| Windows native (cmd/PowerShell) | **Not supported** | No `.ps1`. Do not run Git Bash against Docker Desktop labels mixed with `C:\` vs `/mnt/c` — `dc-down` matches `devcontainer.local_folder` as a string. |

`dc-down` compares the Docker label to `pwd`. Create and stop from the **same** environment (all WSL, or all macOS/Linux), or pass `dc-down --id`.

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
| `dc-up [dir]` | `devcontainer up` using the **project** config (safe default) |
| `dc-up --ports` | **REPLACE** project config with `~/.config/devcontainer/override.json` |
| `dc-exec` / `dc-exec -- cmd` | exec in the workspace container |
| `dc-down` | **stop** the labeled container (keep it for next `dc-up`) |
| `dc-down --rm` | stop + remove |
| `dc-down --compose` | stop/down the compose project if labeled |
| `dc-down --all --yes` | every `devcontainer.local_folder` container |
| `dc-ps` | list docker + labeled containers |
| `dc-forward 9000` | extra host→container port via `socat` |

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

Matching uses Docker label `devcontainer.local_folder` (cwd, then git root).

## License

MIT
