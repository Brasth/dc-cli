---
title: "Self-serve Docker recovery"
description: "Users come to dc-cli to escape Docker pain, then hit a Docker wall and cannot recover without becoming Docker admins. Close the diagnose→next-action→verify loop without becoming a daemon UI."
status: proposed
priority: P1
branch: cursor/docker-self-serve-plan-2db4
tags:
  - architecture
  - doctor
  - host
  - tui
  - support
blockedBy: []
blocks: []
created: "2026-08-22T01:40:00.000Z"
createdBy: "cursor:cloud"
source: ask
---

# Self-serve Docker recovery

## The paradox

dc-cli's job is to hide Docker for **this folder**. Users arrive because raw `docker exec NAME`, Colima ports, host clashes, and `docker system prune -af --volumes` hurt.

The kit still **runs on** a local Docker engine it does not own. When that engine is missing, stopped, split, permission-denied, disk-full, or on the wrong context, every product verb dies. The current honest answer is: diagnose (`dc-doctor` / `dc-host`), then go become a Docker admin (`Start Docker Desktop`, `colima start`, `docker context use`, `sudo systemctl …`).

That is the issue. Not "we lack diagnostics." We already diagnose. Users cannot **finish the loop without us** because recovery is scattered, Docker-flavored, and often refused by design.

## What already exists (do not rebuild)

| Surface | What it does | Gap |
|---|---|---|
| `install.sh` doctor | Host readiness at install. Helpers still install. Never auto-installs an engine. | Day-0 only. Prints, then leaves. |
| `dc-host` codes | `docker_cli_missing`, `docker_engine_missing`, `docker_engine_stopped`, `docker_permission_denied`, `docker_context_invalid`, `docker_split_brain` | Actions are `open_guide` / `copy_commands` / `retry` / `run_doctor`. No apply. |
| `dc-doctor` | 18 read-only checks. Human + `--json`. Exit 0/1/2. Motto: diagnose first, never autofix. | Remediation is a string. Only split-brain prints a Fix block. No ranked next step. |
| `dc-engine --fix` | Prints pick-one recipe. `--yes` is `docker context use` only. Never sudo, never stop an engine. | Extra engine still live → user must stop it themselves. |
| TUI `hostView` | Blocked setup: `[d]` Desktop guide, `[c]` copy Colima, `[r]` retry, `[q]` quit. | First-run engine wall only. Disk, ports, nets, split after a Desktop update do not land here. |
| `dc-df` / `dc-prune` | Disk report → dry-run → safe reclaim. | User must already know this path. TUI `d` reports; reclaim is CLI-only. |
| `dc-up --take-ports` | TTY ask / flag. Stops **labeled** foreign holders only. | Unlabeled = report-only. User still stuck. |
| `dc-net --ensure` | Missing declared external bridges. | Overlay / custom IPAM refused. |
| Guides | `/guide/doctor`, `/no-docker`, `/disk`, `/ports` | Task pages. No single "I'm stuck" door. |
| `/issues/` | GitHub templates. Asks for `dc-doctor --json`. | User must already know to run doctor and redact. |

Locked from `260817-1316-docker-workspace-kind`:

> Product = stateless this-folder wrapper. **No daemon TUI, no Colima/Desktop start, no image/volume zoo.**

That lock is why we are safe. It is also why beginners bounce to support when Docker, not dc-cli, is broken.

## Failure classes (user cannot use the product)

Ranked by "product is dead until this is handled":

1. **No engine / CLI** — `docker` missing, Desktop/Colima not installed.
2. **Engine present, not reachable** — Desktop quit, Colima stopped, Linux `dockerd` down, socket permission.
3. **Wrong or split engine** — Colima + Desktop, Desktop + leftover `dockerd`, exported `DOCKER_HOST`.
4. **Engine up, workspace cannot start** — ENOSPC, missing external net, host port clash, stale sidecar.
5. **Engine up, workspace is confusing** — kind=none, duplicate labels, unpublished ports, two DBs.

1–3 make **every** verb fail. 4 is day-2 "it worked yesterday." 5 is teachable, not a wall.

Today 1–2 get a blocked TUI. 3 gets `dc-engine --fix` printout. 4 is three different commands. 5 is doctor warnings.

## Approaches

### A — Guided playbook only (keep never-mutate-host)

New `dc-recover` reads host + doctor, prints **one** next command, copies it, rechecks. TUI shows the same card. `dc-recover --report` writes a sanitized bundle for GitHub.

- Keeps the 260817 lock.
- Users still leave the product to start Desktop / Colima / fix groups.
- Fastest, lowest footgun.

### B — Safe opt-in host recovery (recommended)

Same playbook, plus a **closed allowlist** of user-owned, no-sudo applies after confirm (`--yes` / TUI `f`):

| Allowed apply | Why it is safe |
|---|---|
| Open Docker Desktop (`open -a Docker` / documented Linux Desktop launch) | User-owned app. No install. No sudo. |
| `colima start` when Colima is installed **and** the chosen engine is Colima | User-owned CLI. No sudo. |
| `docker context use <recommended>` | Already shipped as `dc-engine --fix --yes`. |
| Print `unset DOCKER_HOST` (never rewrite rc files) | Shell-local. |
| `dc-prune --yes` after doctor says disk is the blocker | Existing safe set. |
| `dc-net --ensure` / `dc-up --create-nets` after doctor says missing declared bridge | Existing, this-folder only. |
| TTY `--take-ports` path when holders are labeled | Existing. |

Still **never**:

- `sudo`
- install Docker / Colima / Desktop
- stop or disable the other engine
- `docker system prune -af --volumes`
- edit project `.devcontainer`
- add the user to `docker` group
- grow the Colima VM disk (hint only)

This is the smallest change that lets a beginner **stay inside dc-cli** for the common "I opened the TUI and it said Docker setup required" cases.

### C — Become a Docker host manager

Install Desktop via packages, image/volume zoo, unlabeled mass-stop, auto-fix with no confirm, rewrite shell rc.

Still reject as the product shape. A recover tool can do **more host lifecycle** than B without becoming Docker Desktop.

## Decision (2026-08-22)

User chose **more than B**. Direction is a recover **tool** that finishes host walls, not a copy-paste board.

Recommended interpretation is **B+** (below), not full C. 260817 “no Colima/Desktop start” is relaxed **for `dc-recover` only**. Doctor stays read-only. No daemon UI. No image/volume zoo.

### B+ — recover owns the host lifecycle we already teach

Everything in B, plus three applies that close the walls B still leaves:

| Extra apply | Closes | Guard |
|---|---|---|
| `brew install docker colima` then `colima start` | No engine / no `docker` on PATH (macOS + Homebrew) | brew present; user confirms; never install Desktop via brew cask unless they pick Desktop path (open installer only) |
| Stop the extra engine after they pick which to keep | Split-brain (Desktop shows boxes, `dc-up` fails) | `colima stop` **or** quit Docker.app. Never `sudo systemctl disable`. Never stop both. |
| Grow Colima disk after prune | Guest `/` still ~100% (`dc-prune --colima-hint` today) | Only after `dc-df` says guest full **and** prune already ran. Confirm size. `colima stop` then start with larger `--disk`. |

Linux without sudo stays print-first unless the user also picks **B++**.

### B++ — B+ plus Linux host sudo (optional, not default)

| Extra apply | Guard |
|---|---|
| `sudo systemctl start docker` | Interactive sudo only. No password stored. `sudo -n` fail → print the command. |
| `sudo usermod -aG docker $USER` | Apply if they confirm. Session cannot finish until re-login — recover must say that and stop. |

B++ is the most we should ever do. It is still not C.

## Recommendation

Ship **B+**, same sequence: playbook contract → CLI recover → TUI recover board → support pack → one stuck guide.

Do not invent a second diagnosis engine. `dc-host` + `dc-doctor` stay the source of truth. Recover **ranks** their codes, applies the first allowed action, verifies.

## Locked decisions (proposed — extras gated on the B+ vs B++ question)

| Decision | Value |
|---|---|
| Product | Still this-folder verbs. Recover is a **host+workspace playbook**, not a Docker Desktop clone. |
| Diagnose | `dc-doctor` and `dc-host` stay read-only. Never add `--fix` to doctor. |
| Recover command | New `dc-recover` (not `dc-doctor --fix`). Human + `--json`. |
| Apply | Closed allowlist. Confirm required. `--yes` is agents/CI for allowlisted applies only. |
| More than B | **B+** extras: brew-install Colima, stop extra engine, grow Colima disk. B++ sudo is a separate yes. |
| Still refuse (C) | Desktop cask/apt install, `systemctl disable --now`, `docker system prune -af --volumes`, unlabeled mass-stop, rewrite rc files, image/volume zoo, apply with no confirm. |
| TUI | Blocked setup becomes the recover board. Day-2 blockers (disk / split / missing net) can open the same board. |
| One next step | Never dump a 12-line recipe as the primary UI. Show **this machine's** first action. Advanced recipe stays behind `?`. |
| Support | `dc-recover --report` writes a redacted bundle (doctor json, host json, engine json, df compact). Issues template points at it. |
| Docs | One `/guide/stuck` door. Existing doctor / no-docker / disk / ports stay task pages. |
| Agents | `--json` schema includes `next.action`, `next.command`, `next.applyAllowed`, `next.verify`. Skill learns recover before opening an issue. |

## Playbook (code → next → verify)

See [phase-01-playbook.md](./phase-01-playbook.md) for the closed table. Summary:

| Code / check | Next (human) | Apply allowed? | Verify |
|---|---|---|---|
| `docker_cli_missing` / `docker_engine_missing` | macOS+brew: install Colima. Else Desktop guide. | B+: `brew_install_colima`. Else open/copy | `dc-host` ready |
| `docker_engine_stopped` + hint=desktop | Start Docker Desktop | yes: launch app | `dc-host` ready |
| `docker_engine_stopped` + hint=colima | Start Colima | yes: `colima start` | `dc-host` ready |
| `docker_engine_stopped` + hint=linux | `systemctl start docker` | B+: print. B++: `sudo_start_docker` | `dc-host` ready |
| `docker_permission_denied` | add user to `docker` group, re-login | B+: print. B++: `sudo_docker_group` then stop | `dc-host` ready after re-login |
| `docker_context_invalid` | `unset DOCKER_HOST`; `docker context use default` | context use only | `dc-engine` one engine |
| `docker_split_brain` | pick this CLI engine, then stop the extra | `context_use` + `stop_extra_engine` | no `extraLive` |
| disk / ENOSPC | `dc-df` then `dc-prune --yes` | `prune_safe` | `dc-up` retry |
| Colima guest `/` still ~100% after prune | grow VM disk | B+: `colima_grow_disk` | `dc-df` guest not full |
| missing declared external net | `dc-up --create-nets` | yes after confirm | doctor `required_networks` ok |
| labeled port clash | `dc-up --take-ports` | existing TTY / flag | `dc-up` |
| unlabeled port clash | report holders + how to stop them | no | user retry |
| kind=none | `dc-try` | existing TUI confirm | labeled exec |

## Phases

| # | Phase | Status | File |
|---|---|---|---|
| 1 | [Playbook contract](./phase-01-playbook.md) | proposed | Closed code→action table + JSON `next` shape. Tests against current host/doctor fixtures. No new verbs yet. |
| 2 | CLI `dc-recover` | gated on phase 1 | Diagnose → print one next → optional `--yes` apply → recheck. `--report` bundle. |
| 3 | TUI recover board | gated on phase 2 | Replace `hostView` keys with playbook actions. Route day-2 disk/split/net here. |
| 4 | Stuck guide + issues door | gated on phase 2 | `/guide/stuck`, issue template asks for `dc-recover --report`. Skill + README. |

Sequential. Do not start phase 2 until the allowlist in phase 1 is approved.

## Success criteria

- [ ] A user who has Colima installed but stopped can recover from `dc-tui` without leaving the board (confirm → start → retry).
- [ ] A user with Docker Desktop installed but quit can recover the same way.
- [ ] A user with two live engines picks one; recover switches context **and** stops the extra (`colima stop` or quit Docker.app).
- [ ] A macOS user with Homebrew and no engine can install Colima from recover after confirm.
- [ ] A user with ENOSPC is offered `dc-df` / `dc-prune --yes`; if Colima guest is still full, grow-disk after a second confirm.
- [ ] A user who still cannot recover has one redacted file to paste. No secrets, no `docker inspect`.
- [ ] `dc-doctor` remains read-only. Existing doctor tests stay green.
- [ ] Desktop cask/apt install, `systemctl disable`, and volume-nuke prune stay refused.

## Out of scope (C — still refused)

- Installing Docker Desktop via brew cask / apt / dmg silent install
- `systemctl disable --now` or stopping **both** engines
- Growing Docker Desktop VM disk
- Daemon-wide image/volume browser
- Auto-fix without confirm
- Changing unlabeled-holder policy (still report-only)
- A desktop `.app` host manager (see `260815-1452-desktop-multi-user`)
- Rewriting `~/.zshrc` / `~/.bashrc`

Linux sudo (`systemctl start`, `usermod`) is **out of B+**. It is in B++ only if chosen.

## Open question (one)

**Which “more than B” package?**

- **B+ (recommended)** — B plus brew-install Colima, stop the extra engine, grow Colima disk. No sudo.
- **B++** — B+ plus interactive `sudo systemctl start docker` and `usermod -aG docker` (then they must re-login).
- **Named extras** — reply with the exact applies you want beyond B+ (I will not add Desktop silent-install, volume-nuke, or a daemon UI).

Do not implement until this package is picked.
