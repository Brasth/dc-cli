---
title: "Install dc-cli"
description: "Install dc-cli with one curl or Homebrew. Release kits include the clickable TUI. Optional official CLI and agent skill."
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

Needs bash 4+ and Docker (Colima or Desktop). Release kits and Homebrew include the clickable TUI. A source install builds it when Go is on PATH; otherwise you get the bash menu.

Safer than piping curl: clone the repo, then `bash install.sh`. Each person needs their own Docker/Colima.

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash
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
| *(default)* | wrappers in `~/bin` only — **not** the official CLI |
| `--with-cli` | `npm i -g @devcontainers/cli` if missing |
| `--with-skill` | copy the agent skill into homes you already have |
| `--full` | `--with-cli` + `--with-skill` |
| `--ref v0.4.3` / `main` | pin a tag or use HEAD |

Default install does **not** put `devcontainer` on PATH. Add `--with-cli` if you do not already have official `@devcontainers/cli`.

## Skill

`--with-skill` copies the `devcontainer-cli-global` skill into agent homes you **already** have. It does not install Claude / Codex / Cursor for you. Restart the agent after.

`--full` is `--with-cli` + `--with-skill`.

Homes and the exact copy list live in the [README Skill](https://github.com/Brasth/dc-cli#skill) section.

Confirm: `dc-tui --help` or `dc-up --help` should print usage.

## Daily start

Run from **your** project folder (the one with `.devcontainer`):

```bash
cd /path/to/your/project
dc-tui
```

Or skip the board: `dc-up`, then `dc-exec`.

Full flag tables live in the [README commands](https://github.com/Brasth/dc-cli#commands-all-of-them).
