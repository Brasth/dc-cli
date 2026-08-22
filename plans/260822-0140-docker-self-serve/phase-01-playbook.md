---
title: "Phase 1 — recover playbook contract"
status: proposed
parent: ./plan.md
---

# Phase 1 — recover playbook contract

No new user-facing command in this phase. Lock the table so `dc-recover` cannot invent actions later.

Source of truth today:

- Host codes: `lib/dc-host.sh`
- Engine split: `lib/dc-engine.sh`
- Workspace checks: `bin/dc-doctor`

Recover **ranks**. First matching row wins.

Direction: **C — this-machine engine manager.** Host rows apply. Folder rows call existing verbs.

If `dc-host` sees engine evidence (Desktop.app, Colima, or a docker socket), **install rows do not match**. Start / context / group / stop-extra win. Install is empty-machine only.

## Rank order

1. Host not ready (`dc-host` code ≠ `ready`)
2. Split-brain
3. Workspace cannot start (disk, grow, nets, ports)
4. Workspace confusing (kind=none, unpublished ports, stale sidecars)
5. Usable — `dc-tui` / `dc-up`

## Closed table

| Rank | When | `next.id` | Apply | Verify |
|---|---|---|---|---|
| 1 | `docker_cli_missing` / `docker_engine_missing` + default=colima + brew | `install_colima` | `brew_install_colima` | `dc-host` ready |
| 1 | same + default=desktop + brew | `install_desktop` | `brew_install_desktop` | `dc-host` ready |
| 1 | same + no brew / Linux | `install_engine_manual` | `open_guide` or `linux_install_docker` (Ubuntu/Debian confirm) | `dc-host` ready |
| 1 | `docker_engine_stopped` + hint=desktop | `start_desktop` | `launch_desktop` | `dc-host` ready |
| 1 | `docker_engine_stopped` + hint=colima | `start_colima` | `colima_start` | `dc-host` ready |
| 1 | `docker_engine_stopped` + hint=linux | `start_linux_docker` | `sudo_start_docker` | `dc-host` ready |
| 1 | `docker_permission_denied` | `fix_socket_group` | `sudo_docker_group` then stop (re-login) | `dc-host` ready after re-login |
| 1 | `docker_context_invalid` | `reset_context` | print `unset DOCKER_HOST`; `context_use` | one engine |
| 2 | `docker_split_brain` | `pick_engine` | `context_use` then `stop_extra_engine` | extra empty |
| 3 | ENOSPC | `reclaim_disk` | `prune_safe` | retry `dc-up` |
| 3 | Colima guest `/` ~100% after prune | `grow_colima_disk` | `colima_grow_disk` | guest not full |
| 3 | Desktop disk full after prune | `grow_desktop_disk` | `open_desktop_settings` (hint only) | user retry |
| 3 | missing declared external bridge | `ensure_nets` | `create_nets` | doctor nets ok |
| 3 | labeled port clash | `take_ports` | `take_ports` | `dc-up` |
| 3 | unlabeled port clash | `report_holders` | none | user retry |
| 4 | kind=none | `try_sandbox` | existing `dc-try` | labeled app |
| 4 | stale owned sidecars | `prune_orphans` | `prune_safe` | doctor ok |
| 4 | unpublished desired ports | `forward_ports` | none in v1 (`dc-forward` print) | doctor ports |
| 5 | usable | `ready` | none | — |

## Apply allowlist

| `apply` | Command | Confirm | Notes |
|---|---|---|---|
| `brew_install_colima` | `brew install docker colima && colima start` | yes | macOS + brew. Default if Colima is the chosen default. |
| `brew_install_desktop` | `brew install --cask docker` then `launch_desktop` | yes | Only if default=desktop or user picks Desktop. |
| `linux_install_docker` | one documented Ubuntu/Debian install line | yes | v1: that family only. Other distros = `open_guide`. |
| `launch_desktop` | `open -a Docker` / documented Linux Desktop | yes | Retry, not sleep-forever. |
| `colima_start` | `colima start` | yes | hint/engine is colima. |
| `sudo_start_docker` | `sudo systemctl start docker` | yes | Interactive sudo. No password stored. |
| `context_use` | `docker context use NAME` | yes / `--yes` | Same as today’s engine `--fix --yes`. |
| `stop_extra_engine` | `colima stop` **or** quit Docker.app **or** `sudo systemctl stop docker` | yes, after pick-one | Stops only the engine that is **not** chosen. Never both. Never `disable --now`. |
| `sudo_docker_group` | `sudo usermod -aG docker "$USER"` | yes | Then refuse further apply until re-login. |
| `colima_grow_disk` | `colima stop` then `colima start --disk N` | yes + size | After prune + guest still ~100%. |
| `open_desktop_settings` | open Desktop dashboard / docs for disk | no | We do not wrap Desktop VM resize. |
| `prune_safe` | `dc-prune --yes` | yes | Never `--all` or `--volume` from recover. |
| `create_nets` | `dc-up --create-nets` | yes | Overlay/IPAM still refuse. |
| `take_ports` | `dc-up --take-ports` | yes | Labeled / proven compose-kind only. |
| `open_guide` | `open` / `xdg-open` | no | Already TUI `d`. |
| `copy` | clipboard | no | Already TUI `c`. |

Forbidden (must remain print-only):

- `docker system prune -af --volumes`
- `systemctl disable --now`
- stop **both** engines
- unlabeled holder stops
- image/volume/container zoo commands
- writing `~/.zshrc` / `~/.bashrc` / Docker json by hand
- silent Desktop dmg without confirm
- growing Desktop VM via undocumented internals

## JSON shape (`dc-recover --json`)

```json
{
  "schemaVersion": 1,
  "command": "dc-recover",
  "host": { "code": "docker_engine_missing", "engineHint": "unknown" },
  "next": {
    "id": "install_colima",
    "summary": "No Docker engine. Install Colima via Homebrew.",
    "command": "brew install docker colima && colima start",
    "apply": "brew_install_colima",
    "applyAllowed": true,
    "verify": "dc-host",
    "escalate": "dc-recover --report"
  }
}
```

`next.id` for install follows the locked **default engine** (plan open question).

## Tests (contract only)

Table-driven, no daemon required.

| Fixture | Expect `next.id` | Expect `apply` |
|---|---|---|
| no engine, macOS, brew, default=colima | `install_colima` | `brew_install_colima` |
| no engine, macOS, brew, default=desktop | `install_desktop` | `brew_install_desktop` |
| no engine, no brew | `install_engine_manual` | `open_guide` |
| stopped, hint colima | `start_colima` | `colima_start` |
| stopped, hint desktop | `start_desktop` | `launch_desktop` |
| stopped, hint linux | `start_linux_docker` | `sudo_start_docker` |
| permission denied | `fix_socket_group` | `sudo_docker_group` |
| split-brain extra=colima | `pick_engine` | `context_use` + `stop_extra_engine` |
| Colima guest full after prune | `grow_colima_disk` | `colima_grow_disk` |
| host ready, usable | `ready` | none |

## Exit

- [ ] Default engine picked (Colima / Desktop / ask)
- [ ] Mapper tests pass
- [ ] No `dc-doctor --fix`
