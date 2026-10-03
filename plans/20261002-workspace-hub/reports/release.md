# dc-cli 0.22.0

Daily workspace hub on the same board. `dc up` still starts this folder. No config still uses `dc try`. Nothing here edits `.devcontainer`.

## Workspaces

`w` lists favorites first, then the latest 30 folders this board opened, including stopped projects. Opening one does not start it. The list works while Docker is down. Ctrl+F favorites. Ctrl+D forgets the registry row only. The folder is untouched.

Recorded at `${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json` when the board opens a folder or you switch. No telemetry.

## Actions

`c` on the board, and `dc actions` (`list`, `trust`, `run`, `untrust`). Shared `.dc/actions.json` stays off until you review every command and approve that file's exact bytes. Any edit turns it off again. Personal commands live in `${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256 of the folder>.json`. A personal id overrides the shared one. dc-cli does not write the shared file.

Approval is stored in `${XDG_STATE_HOME:-~/.local/state}/dc-cli/actions-trust.json` (folder + sha256 of the raw shared file). Nothing is trusted on open.

A run goes through `dc exec --no-start`. The container must already be running. Stopped, missing, or forged targets exit 1 with no `docker start`, compose start, or `dc up`. Not valid with `--id` or `--restart`. argv is passed as-is, with no shell. **Actions run inside your containers and can change data there.**

`dc actions` needs the compiled binary (release kit, Homebrew, or a source install with Go). Without Go, the shell fallback answers `--help` and `--version`; other verbs report that the compiled binary is required. Every other command still works.

## Activity

`v` shows this folder's container events for the current session: latest 200, in memory only, nothing written to disk. Accepted events refresh the board. Switching folders, entering fleet, or clearing the view drops them. Fleet refuses `v`. Only containers proven to belong to this folder are kept. Port sidecars and other projects are left out. No env or log text is stored.

## Install

The advertised curl installs the latest GitHub release: compiled `dc-tui` and `dc-actions`. Homebrew is that same kit. A source install (`bash install.sh` in a clone) builds both when Go is on PATH. Without Go, `dc-tui` falls back to the shell menu. `dc actions` shell fallback answers `--help` and `--version`; other verbs report that the compiled binary is required.

After install, run `source ~/.zshrc` or `source ~/.bashrc`, then `dc`.

The curl line is unchanged:

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

## Website

Copy only. Same layout, media, and install command. Hero and the default page description say this is a daily workspace hub. The tour clip is labeled a core walkthrough (start, shell, logs, top, nets, more, rm). It does not claim to show every key. The homepage board and `/play` stay the existing core-workflow demo. They were not resimulated.

The bento is three static examples: `w`, `dc actions` / `dc exec --no-start`, and `v`. No invented command output.

The command list adds `dc actions` and `dc exec --no-start`. It no longer says TUI `[f]` applies recover. On the working board, `f` is fleet. The Docker-down screen still uses `[f]` to apply `dc-recover --yes`. That screen's guide body was left as-is.

## Checks

| Check | Result |
|---|---|
| PR 14 | Merged 2026-10-03. CI run 37087600382: linux success, macos success. |
| `npm ci` in `site/` | Exit 0. Lockfile unchanged. 236 packages added, 237 audited. npm reported 7 vulnerabilities (6 high, 1 critical). `npm audit fix` was not run. |
| `npm run build` in `site/` | Exit 0. 14 pages. Rebuilt after the install-guide shell lines. |
| `git diff --check` | Clean. |
| Homepage `site/dist/index.html` | `v0.22.0` ×6, `dc actions` ×4, `dc exec --no-start` ×4, `Core walkthrough` ×1, `source ~/.zshrc` ×1, `source ~/.bashrc` ×1, `compiled binary is required` ×1, `f is fleet` ×1. `wrappers only`, `TUI [f]`, `full feature tour`, and `Every key` absent. |
| Tag `v0.22.0`, GitHub release, Pages deploy | Pending. Release workflow runs on a `v*` tag. Pages runs on a push to `main` that touches `site/**`. |
| Homebrew formula | Still `0.21.0`. Left unchanged until release assets and checksums exist. |
| Live Linux smoke | Not performed. Prior Colima smoke (macOS host, Linux VM) is in `implementation.md` (27/27). Not re-run here. |
