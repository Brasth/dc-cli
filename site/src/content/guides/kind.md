---
title: "Dev Container vs compose"
description: "dc-cli detects this folder. .devcontainer is official CLI. Compose-only uses docker compose or docker-compose. Same keys. No daemon UI."
h1: "This folder decides. You type the same verbs."
updated: 2026-08-20
howto: true
steps:
  - name: Stand in the project
    text: cd into the folder you want to start. Not the Docker daemon. Not a random container name.
  - name: Ask doctor
    text: dc-doctor --json . and read workspace.kind. devcontainer, compose, or none.
  - name: Use the same verbs
    text: dc or dc up / dc exec / dc down. The kit forks. Do not type docker exec NAME.
faq:
  - q: Do I pick a mode?
    a: No. .devcontainer present → kind=devcontainer even if compose files exist. Else a root compose file → kind=compose.
  - q: What if none?
    a: dc try is the start path; on a TTY dc up offers that sandbox. Agents use dc-try --yes.
  - q: Does compose-kind publish Colima ports?
    a: Not automatically. dc-forward stays opt-in and still wants a labeled app or --id. Folder attach (a) is N/A. dc-files Enter still opens VS Code on that running box. Cursor needs kind=devcontainer (dev-container+); compose-kind falls back to a bind-mount host path.
  - q: What if two folders share a compose project name?
    a: Fail closed. We do not hash-rename. working_dir or config_files must prove this folder. project and project/.devcontainer are the same workspace. /Users vs /Volumes of the same dir too.
  - q: Why did dc-down --all refuse claimants?
    a: It used to count the labeled app folder and a sibling working_dir under .devcontainer as two owners. That is one stack now. Retry dc-down --all --yes.
  - q: Can fleet start a compose-only project?
    a: No. dc-ls --all and the fleet picker stay labeled Dev Container workspaces.
---

`kind` is this-folder identity. It is not a Docker UI.

| This folder | kind | Start | Exec |
|---|---|---|---|
| `.devcontainer` present | `devcontainer` | official CLI, then `dc-forward` | official `devcontainer exec` |
| else root compose file | `compose` | Compose `-p NAME -f FILES up -d` (`docker compose` or `docker-compose`) | Compose exec (1 / `app` / `web` / refuse) |
| else | none | `dc-up` refuses; `dc-try` sandbox | after try, labeled `dc-exec` |

```bash
cd /path/to/your/project
dc-doctor --json .     # workspace.kind
dc                     # u start  e shell  s stop
```

Same board. Same keys. Compose-kind does not invent VS Code attach and does not auto-publish ports.

See also the [TUI keys](/guide/tui/) and [install](/guide/install/).
