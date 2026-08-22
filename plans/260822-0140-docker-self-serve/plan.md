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

Install engines, start/stop both, group membership, grow qemu disks, image/volume zoo.

Reject. Different product. Breaks the manifesto. High support surface. The 260815 desktop-multi-user plan already said a GUI does not create a host manager.

## Recommendation

Ship **B**, in this order: playbook contract → CLI recover → TUI recover board → support pack → one stuck guide.

Do not invent a second diagnosis engine. `dc-host` + `dc-doctor` stay the source of truth. Recover **ranks** their codes and offers the first safe next action.

## Locked decisions (proposed — confirm before implementation)

| Decision | Value |
|---|---|
| Product | Still this-folder verbs. Recover is a **host+workspace playbook**, not a Docker Desktop clone. |
| Diagnose | `dc-doctor` and `dc-host` stay read-only. Never add `--fix` to doctor. |
| Recover command | New `dc-recover` (not `dc-doctor --fix`). Human + `--json`. |
| Apply | Closed allowlist only. Confirm required. `--yes` is agents/CI for allowlisted applies. |
| Refuse | sudo, install engine, stop extra engine, bulk volumes, project edits. Print the exact manual command instead. |
| TUI | Blocked setup becomes the recover board. Day-2 blockers (disk / split / missing net) can open the same board. |
| One next step | Never dump a 12-line recipe as the primary UI. Show **this machine's** first action. Advanced recipe stays behind `?` or `dc-engine --fix`. |
| Support | `dc-recover --report` writes a redacted bundle (doctor json, host json, engine json, df compact). Issues template points at it. |
| Docs | One `/guide/stuck` door. Existing doctor / no-docker / disk / ports stay task pages. |
| Agents | `--json` schema includes `next.action`, `next.command`, `next.applyAllowed`, `next.verify`. Skill learns recover before opening an issue. |

## Playbook (code → next → verify)

See [phase-01-playbook.md](./phase-01-playbook.md) for the closed table. Summary:

| Code / check | Next (human) | Apply allowed? | Verify |
|---|---|---|---|
| `docker_cli_missing` / `docker_engine_missing` | Open Desktop guide **or** copy Colima install line | open guide / copy only | `dc-host` ready |
| `docker_engine_stopped` + hint=desktop | Start Docker Desktop | yes: launch app | `dc-host` ready |
| `docker_engine_stopped` + hint=colima | Start Colima | yes: `colima start` | `dc-host` ready |
| `docker_engine_stopped` + hint=linux | `systemctl start docker` | no (sudo) | `dc-host` ready |
| `docker_permission_denied` | add user to `docker` group, re-login | no | `dc-host` ready |
| `docker_context_invalid` | `unset DOCKER_HOST`; `docker context use default` | context use only | `dc-engine` one engine |
| `docker_split_brain` | pick this CLI engine; stop the extra **yourself** | context use only | no `extraLive` |
| disk / ENOSPC | `dc-df` then `dc-prune --yes` | yes after confirm | `dc-up` retry |
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
- [ ] A user with two live engines gets **one** next step (`docker context use` + "stop the other yourself"), not a dual recipe as the primary UI.
- [ ] A user with ENOSPC is offered `dc-df` / `dc-prune --yes`, not a GitHub issue.
- [ ] A user who still cannot recover has one redacted file to paste. No secrets, no `docker inspect`.
- [ ] `dc-doctor` remains read-only. Existing doctor tests stay green.
- [ ] No sudo path is added. Install-engine remains a guide, not an apply.

## Out of scope

- Installing Docker Desktop / Colima / Engine
- Stopping or disabling the extra engine
- Growing Colima / Desktop VM disks
- Linux `docker` group membership
- A desktop `.app` host manager (see `260815-1452-desktop-multi-user`)
- Daemon-wide image/volume browser
- Auto-fix without confirm
- Changing unlabeled-holder policy (still report-only)

## Open question (one)

**Confirm approach B's allowlist.** Specifically: may `dc-recover --yes` / TUI `f` run `colima start` and launch Docker Desktop when that engine is already installed?

- **Yes (B)** — recommended. Beginners stay in-product for the most common wall.
- **No, playbook only (A)** — we only copy/print. Safer, still a bounce.
- **More than B** — say what (not recommended).

Do not implement until this is answered.
