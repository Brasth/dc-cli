---
title: "Install dc-cli"
description: "Install dc-cli with one curl. Tracks the latest GitHub release. Optional official CLI and agent skill."
h1: "One curl. Then source your shell."
updated: 2026-08-15
howto: true
steps:
  - name: Run the installer
    text: curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash
  - name: Reload the shell
    text: source ~/.zshrc or ~/.bashrc so ~/bin is on PATH.
  - name: Confirm
    text: dc-tui --help or dc-up --help should print usage.
---

Needs bash 4+ and Docker (Colima or Desktop). Go 1.22+ builds the clickable TUI; otherwise you get the bash menu.

Safer than piping curl: clone the repo, then `bash install.sh`.

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash
source ~/.zshrc   # or ~/.bashrc
```

## Flags

| Flag | What you get |
|---|---|
| *(default)* | wrappers in `~/bin` only — **not** the official CLI |
| `--with-cli` | `npm i -g @devcontainers/cli` if missing |
| `--with-skill` | copy the agent skill into homes you already have |
| `--full` | `--with-cli` + `--with-skill` |
| `--ref v0.4.3` / `main` | pin a tag or use HEAD |

Default install does **not** put `devcontainer` on PATH. Add `--with-cli` if you do not already have official `@devcontainers/cli`.

Confirm: `dc-tui --help` or `dc-up --help` should print usage.

## Daily start

Run from **your** project folder (the one with `.devcontainer`):

```bash
cd /path/to/your/project
dc-tui
```

Or skip the board: `dc-up`, then `dc-exec`.

Full flag tables live in the [README commands](https://github.com/Brasth/dc-cli#commands-all-of-them).
