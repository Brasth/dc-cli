# DC-CLI

Site: [dc.brasth.com](https://dc.brasth.com) · Guides: [dc.brasth.com/guide](https://dc.brasth.com/guide/) · Issues: [dc.brasth.com/issues](https://dc.brasth.com/issues/)

Host-global helpers around official [`@devcontainers/cli`](https://github.com/devcontainers/cli). **No VS Code required. Does not edit** project `.devcontainer`.

`.devcontainer` → official CLI. Else a root compose file → Compose. Else `dc up` refuses (use `dc try` for a sandbox).

**Two spellings, same command:** `dc up` = `dc-up`. `dc` with no args is the board.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

Then `source ~/.zshrc` or `~/.bashrc`. Homebrew: `brew tap Brasth/dc-cli && brew install dc-cli`.

`--with-cli` is the official standalone installer (not npm). Clone with no flags stays wrappers only. Other flags: `--with-cli-npm`, `--with-skill`, `--full`, `--ref`, `--no-yazi`. See the [install guide](https://dc.brasth.com/guide/install/).

## Daily

```bash
cd /path/to/your/project
dc              # board (same as dc-tui)
dc up           # start
dc exec         # shell in the app
dc down         # stop the stack
```

Stuck engine: `dc doctor` then `dc recover --yes`. Disk: `dc df` then `dc prune --yes`. **Never** `docker system prune -af --volumes`.

## Commands

`dc <verb>` and `dc-<verb>` are both installed. `dc --help` lists them. Flags: `dc <verb> --help`.

| Command | Does |
|---|---|
| `dc` / `dc-tui` | board (`--all` = fleet) |
| `dc up` / `dc-up` | start this folder |
| `dc exec` / `dc-exec` | app shell (`--service NAME` for siblings) |
| `dc down` / `dc-down` | stop the stack (`--app`, `--rm`) |
| `dc try` / `dc-try` | sandbox when there is no config |
| `dc doctor` / `dc-doctor` | read-only diagnose |
| `dc recover` / `dc-recover` | one next step (`--yes` applies) |
| `dc upgrade` / `dc-upgrade` | check / install latest release |
| `dc engine` / `dc-engine` | which engine (`--fix`) |
| `dc forward` / `dc-forward` | Colima-safe ports |
| `dc db` / `dc-db` | host DB client on a declared port |
| `dc files` / `dc-files` | yazi/nnn in the box |
| `dc open` / `dc-open` | host editor (`--attach` = VS Code) |
| `dc df` / `dc-df` | disk report |
| `dc stats` / `dc-stats` | CPU / RAM / net |
| `dc net` / `dc-net` | declared compose nets |
| `dc prune` / `dc-prune` | safe reclaim (`--yes`) |
| `dc ls` / `dc-ls` | labeled apps |
| `dc ps` / `dc-ps` | docker + labels |

Task guides: [board](https://dc.brasth.com/guide/tui/) · [kind](https://dc.brasth.com/guide/kind/) · [doctor](https://dc.brasth.com/guide/doctor/) · [ports](https://dc.brasth.com/guide/ports/) · [disk](https://dc.brasth.com/guide/disk/) · [stuck](https://dc.brasth.com/guide/stuck/).

## Platform

macOS and Linux: one live engine (Colima **or** Desktop). WSL2 best-effort. Native Windows not supported.

`--with-skill` copies into existing agent homes (`~/.cursor`, `~/.claude`, `~/.codex`, `~/.pi`, `~/.gemini`, `~/.opencode`, `~/.agents/skills`). Restart the agent after.

[Canvilled](https://github.com/Canvilled) (Huy Nguyen). MIT.
