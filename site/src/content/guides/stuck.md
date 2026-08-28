---
title: "Stuck: dc-recover"
description: "When Docker or this folder blocks dc-cli, run dc-recover. It prints one next step. --yes applies it if you already have an engine. Doctor stays read-only."
h1: "One next step. Then retry."
updated: 2026-08-22
howto: true
steps:
  - name: Diagnose
    text: dc recover or dc. The blocked board shows the next command. [f] applies it when safe.
  - name: Apply if you already have an engine
    text: dc-recover --yes starts Desktop/Colima/Linux docker, picks one engine, or fixes context/group. Confirm first.
  - name: Recheck
    text: dc doctor then dc. Still stuck? dc recover --report and attach that folder on GitHub. No .env.
faq:
  - q: Will this install Docker for me?
    a: Not in v1. Empty machine still uses the Desktop guide or the copied Colima brew line. Recover attaches to an engine you already installed.
  - q: Does dc-doctor start Docker?
    a: No. Doctor is read-only. dc-recover is the mutate door.
  - q: Two engines live?
    a: Recover asks you to keep this CLI engine, switches context, and stops only the extra (colima stop or quit Docker.app). It does not disable systemd.
---

`dc-recover` is the day-2 **mutate** door when the engine you already have is the wall.

```bash
dc-recover                 # this folder; print one next command
dc-recover --yes           # apply that command (allowlisted)
dc-recover --json          # agents
dc-recover --report ./out  # redacted bundle for a GitHub issue
```

| You already have | `--yes` may |
|---|---|
| Desktop or Colima, stopped | start that engine |
| Two live engines | `docker context use` + stop the extra |
| Bad `DOCKER_HOST` / context | point the CLI at the existing engine |
| Linux permission denied | `usermod -aG docker`, then re-login |
| Disk full | `dc-prune --yes`; Colima guest still full → `--grow-disk` |
| Host ready, folder has no config | `dc-try` (sandbox + default localhost ports; `--yes` applies `try_sandbox`) |

It does **not** run `docker system prune -af --volumes`, stop unlabeled port holders, `systemctl disable --now`, or edit `~/.zshrc`.

No engine at all? See [No Docker engine](/guide/no-docker/). Diagnose only? [dc-doctor](/guide/doctor/).
