---
title: "Install dc-cli"
description: "Install dc-cli with one curl (--with-cli standalone) or Homebrew. Release kits include the clickable TUI. Clone with no flags is wrappers only."
h1: "One curl. Then source your shell."
updated: 2026-08-18
howto: true
steps:
  - name: Run the installer
    text: curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
  - name: Reload the shell
    text: Open a new terminal, or source ~/.zshrc / ~/.bashrc so generation current/bin is on PATH.
  - name: Confirm
    text: dc-tui --help, dc-up --help, dc-doctor --help, dc-engine --help, dc-stats --help, or dc-net --help should print usage.
---

Needs bash 4+ and Docker (Colima **or** Desktop — one live engine). Official `@devcontainers/cli` is required only for `kind=devcontainer` folders. Compose-only folders start via `dc-up` → `docker compose` (no official CLI). Release kits and Homebrew include the clickable TUI (logo splash, start/shell/stop first). A source install builds it when Go is on PATH; otherwise you get the bash menu.

Safer than piping curl: clone the repo, then `bash install.sh`. Each person needs their own Docker/Colima. Do not copy `export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock` onto a laptop that uses Docker Desktop. `dc-doctor` reports the CLI engine and socket; two live engines block `dc-up`. `dc-engine --fix` prints how to pick one.

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
source ~/.zshrc   # or ~/.bashrc
```

Or Homebrew (same kit, no Go):

```bash
brew tap Brasth/dc-cli
brew install dc-cli
```

## Flags

| Flag | What you get |
|---|---|
| *(default)* | wrappers generation only — **not** the official CLI |
| `--with-cli` | official **standalone** CLI if missing (never npm) |
| `--with-cli-npm` | explicit exact-pin npm fallback (host Node must meet that package's `engines.node`) |
| `--with-skill` | copy the agent skill into homes you already have |
| `--full` | `--with-cli` + `--with-skill` |
| `--ref v0.4.3` / `main` | pin a tag or use HEAD |
| `--no-yazi` | do not prefetch Linux yazi (`DC_SKIP_YAZI=1`) |

The advertised curl passes `--with-cli` so `devcontainer` is on PATH via the **upstream standalone** installer (bundled Node major, mutable patch). npm is **only** `--with-cli-npm` with an exact qualified pin. The compatibility floor is unpublished (`DC_DEVCONTAINER_MIN_VERSION` / `DC_DEVCONTAINER_NPM_VERSION` empty) until `docs/qualification/devcontainer-cli-floor.md` has four platform `result: pass` rows — `--with-cli-npm` hard-rejects until then. Current registry `0.88.0` / Node 20 is candidate evidence, not a pre-approved floor. Clone or `bash install.sh` with no flags stays wrappers only.

## Skill

`--with-skill` copies the `devcontainer-cli-global` skill into agent homes you **already** have. It does not install Claude / Codex / Cursor for you. Restart the agent after.

`--full` is `--with-cli` + `--with-skill`.

Homes and the exact copy list live in the [README Skill](https://github.com/Brasth/dc-cli#skill) section.

Confirm: `dc-tui --help`, `dc-up --help`, `dc-doctor --help`, `dc-engine --help`, `dc-stats --help`, or `dc-net --help` should print usage. Homebrew users on an older tap: `brew update && brew upgrade dc-cli` (v0.15.0).

## Daily start

Run from **this folder**. `.devcontainer` → official CLI. Compose-only → `docker compose`. Same verbs.

```bash
cd /path/to/your/project
dc-tui
```

Or skip the board: `dc-up`, then `dc-exec`.

Full flag tables live in the [README commands](https://github.com/Brasth/dc-cli#commands-all-of-them).
