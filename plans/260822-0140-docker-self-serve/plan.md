---
title: "Self-serve Docker recovery"
description: "dc-cli becomes this-machine's Docker host manager so users can finish engine walls without leaving the product. Not a container zoo. Not a shared team host."
status: proposed
priority: P1
branch: cursor/docker-self-serve-plan-2db4
tags:
  - architecture
  - doctor
  - host
  - engine
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

The kit still **runs on** a local Docker engine. When that engine is missing, stopped, split, permission-denied, or disk-full, every product verb dies. Today we diagnose, then send them to become Docker admins.

That bounce is the bug. Diagnosis already exists. Host lifecycle does not.

## What already exists (do not rebuild)

| Surface | What it does | Gap |
|---|---|---|
| `install.sh` doctor | Host readiness at install. Helpers still install. Never auto-installs an engine. | Day-0 only. Prints, then leaves. |
| `dc-host` codes | `docker_cli_missing`, `docker_engine_missing`, `docker_engine_stopped`, `docker_permission_denied`, `docker_context_invalid`, `docker_split_brain` | Actions are `open_guide` / `copy` / `retry` / `run_doctor`. No apply. |
| `dc-doctor` | 18 read-only checks. Human + `--json`. | No ranked next step. Never mutates (keep that). |
| `dc-engine --fix` | Prints pick-one. `--yes` is `docker context use` only. | Extra engine stays live. |
| TUI `hostView` | `[d]` Desktop guide, `[c]` copy Colima, `[r]` retry | First-run wall only. Does not start/stop/install. |
| `dc-df` / `dc-prune` | Disk report → safe reclaim | Grow-disk is hint-only. |
| Guides / `/issues/` | Task pages + GitHub door | No "I'm stuck" loop that applies a fix. |

Locked from `260817-1316-docker-workspace-kind` (this plan **relaxes** the host half):

> Product = stateless this-folder wrapper. **No daemon TUI, no Colima/Desktop start, no image/volume zoo.**

Folder verbs stay stateless. **Engine lifecycle moves into recover.** Image/volume zoo stays refused.

`260815-1452-desktop-multi-user` already said a shared team host is a different product. This plan is **one login, one machine**. That is not Coder.

## Failure classes

1. No engine / CLI
2. Engine present, not reachable (quit / stopped / permission)
3. Wrong or split engine
4. Engine up, workspace cannot start (ENOSPC, nets, ports)
5. Engine up, workspace is confusing (kind=none, …)

1–3 are host-manager work. 4–5 stay existing folder verbs, invoked by recover.

## Approaches (history)

| | Idea | Status |
|---|---|---|
| A | Playbook only (print / copy) | Too weak for the original ask |
| B | Start an already-installed engine, no sudo | User said more than B |
| B+ | B + brew Colima + stop extra + grow Colima disk | Still a half-tool |
| C | This-machine host manager | **Chosen 2026-08-22** |

C is not “rebuild Docker Desktop.” C is: **install, start, stop, grow, fix group** for the one engine this login uses, then get back to `dc-up`.

## Primary user: they already have Docker

Most dc-cli users already installed Colima, Docker Desktop, or Linux `dockerd`. Recover must treat that as the **main** path. Install-from-zero is only the empty-machine fallback.

If an engine is already present, recover **never installs another one**. It attaches to what is there:

| Already on the machine | Recover does |
|---|---|
| Desktop or Colima installed, just quit/stopped | Start that engine. Wait until `dc-host` is ready. |
| `docker` on PATH, socket permission denied | Linux: offer group add, then tell them to re-login. |
| CLI points at the wrong context / `DOCKER_HOST` | `unset` guidance + `docker context use` the existing engine. |
| Two engines live (Desktop + Colima, or Desktop + leftover `dockerd`) | Pick the one they want to keep. Switch context. Stop only the extra. |
| Engine up, disk/port/net blocked | Existing `dc-df` / `dc-prune` / `--create-nets` / `--take-ports`. Grow Colima disk only after prune. |
| One healthy engine | Ready. No install. No stop. `dc-tui` / `dc-up`. |

Do not export a Colima `DOCKER_HOST` onto a Desktop laptop. Do not replace Desktop with Colima (or the reverse) unless they confirm a pick-one.

## C — this-machine engine manager

Recover owns host lifecycle. Doctor stays read-only. Folder verbs stay folder verbs.

```
dc-tui / dc-up
    │
    ├─ engine already there, not ready ──► start | context | stop extra | group
    ├─ no engine at all ──► install default (empty machine only)
    │
    └─ host ready, folder blocked ──► prune | nets | take-ports | try
                              verify dc-host / dc-up → back to the board
```

### What we manage

| Verb | Colima | Docker Desktop | Linux dockerd |
|---|---|---|---|
| Install | `brew install docker colima` | `brew install --cask docker` **or** open the official installer | Distro docs + confirm one known command (Ubuntu/Debian first). No twenty-distro matrix in v1. |
| Start | `colima start` | `open -a Docker` / documented Linux Desktop launch | `sudo systemctl start docker` |
| Stop (extra only, after pick-one) | `colima stop` | quit Docker.app | `sudo systemctl stop docker` (never `disable --now` in v1) |
| Grow disk | `colima stop` + `colima start --disk N` after prune | Open Desktop settings + hint (no public grow CLI we will wrap) | Not a VM. `dc-prune` only. |
| Group / socket | n/a (user-owned socket) | n/a | `sudo usermod -aG docker $USER` then **stop**. Re-login required. |

Confirm every apply. `--yes` is agents/CI for the same allowlist, never a hidden sudo.

### What we still refuse (plain language)

These are **not** “we refuse to help.” They are actions recover will **not run for you**, because they destroy data, take over the OS, or turn dc-cli into Docker Desktop.

| Phrase | What it means | Why we do not auto-run it |
|---|---|---|
| Container / image / volume zoo | A UI to browse every box, image, and named volume on the daemon (Docker Desktop’s Containers / Images / Volumes screens). | That is a different product. We stay on **this folder** plus **this login’s engine**. `dc-ls` / `dc-df` already list what we own. |
| `docker system prune -af --volumes` | Docker’s nuclear reclaim: delete unused images **and named volumes**. | Named volumes are databases and `node_modules`. dc-cli exists so people do **not** run this. Recover uses `dc-prune --yes` (cache, dangling images, unused nets, owned orphan sidecars) only. |
| Unlabeled mass-stop | Stop every container holding a port, even ones we cannot prove belong to a dc-cli workspace. | A random `nginx` or company VPN box can hold `:443`. We stop **labeled** foreign stacks (`--take-ports`). Unknown holders: show the list, user stops them. |
| `systemctl disable --now` | Linux: stop Docker **and** prevent it from starting at boot. | `stop` is enough to clear split-brain. Disable is a lasting OS change. Easy to leave someone with no Docker after reboot. |
| Rewriting shell rc | Silently edit `~/.zshrc` / `~/.bashrc` to export `DOCKER_HOST` or add PATH. | That is how Colima vs Desktop split-brain starts. We print `unset DOCKER_HOST` for **this** shell. We do not rewrite their profile. |
| A macOS `.app` | Shipping `DcCli.app` as a desktop host manager. | Parked in `260815`. Recover is CLI + TUI on the machine they already use. |

We also still refuse: auto-fix with no confirm, editing project `.devcontainer`, a shared team/remote Docker host.

### Surfaces

| Surface | Role |
|---|---|
| `dc-doctor` / `dc-host` | Read-only. Unchanged contract. |
| `dc-engine` | Still “which engine + socket.” `--fix` can print. Mutate goes through recover. |
| `dc-recover` | **Only mutate door.** Rank → one next → confirm → apply → verify. `--json`. `--report`. |
| TUI blocked / recover board | Same playbook. Keys apply the next action, then retry. |
| `/guide/stuck` | Front door. Task guides stay. |

One next step on the primary UI. Full recipe behind `?`.

## Locked decisions

| Decision | Value |
|---|---|
| Product | This-folder verbs **plus** this-login engine lifecycle. Not a daemon UI. Not a team host. |
| Diagnose | `dc-doctor` / `dc-host` stay read-only. No `dc-doctor --fix`. |
| Mutate door | `dc-recover` only. |
| Apply | Closed allowlist. Confirm required. |
| Primary user | Engine **already installed**. Attach, start, pick-one, prune, grow. |
| Empty machine | Install only when `dc-host` says no engine evidence. Never install on top of a live engine. |
| Scope | Start/stop-extra/group/grow **plus** install-from-zero. |
| Refuse | Zoo, volume-nuke, unlabeled mass-stop, disable-now, rc edits, `.app`, no-confirm. |
| 260817 | Relaxed for recover host lifecycle only. |
| Support | `dc-recover --report` redacted bundle. |

## Playbook

See [phase-01-playbook.md](./phase-01-playbook.md). Host rows now **apply**. Folder rows still call existing verbs.

## Phases

| # | Phase | Status | File |
|---|---|---|---|
| 1 | [Playbook contract](./phase-01-playbook.md) | proposed | Closed code→action table including C applies. |
| 2 | CLI `dc-recover` | gated on phase 1 + default-engine | Diagnose → one next → `--yes` apply → recheck. `--report`. |
| 3 | TUI recover board | gated on phase 2 | Replace `hostView`. Day-2 disk/split/net open the same board. |
| 4 | Stuck guide + issues + skill | gated on phase 2 | `/guide/stuck`. Template asks for `--report`. |

## Success criteria

- [ ] Desktop or Colima already installed but stopped: recover starts **that** engine. No second install.
- [ ] Two live engines: user picks one; recover switches context **and** stops the extra. Does not install a third.
- [ ] Wrong `DOCKER_HOST` / context on a machine that already has a working engine: recover points the CLI at it.
- [ ] Linux permission denied on an existing socket: group add, then re-login. No reinstall.
- [ ] Engine up, folder blocked (disk / net / labeled ports): existing verbs. Colima grow only after prune.
- [ ] Empty machine (no engine evidence) + Homebrew: recover installs the **default** engine only then.
- [ ] `dc-doctor` tests stay green. No volume-nuke. No zoo.

## Open question (one)

**When there is no engine yet, what does recover install by default?**

- **Colima** (recommended for C) — we can install, start, stop, and grow. Desktop stays first-class if it is already present, and remains an explicit other path.
- **Desktop** — matches today’s beginner copy. We can install via cask/installer and launch/quit; we cannot own disk grow.
- **Ask every time** — safest, more bounce, weaker “tool.”

Do not implement until the default is picked.
