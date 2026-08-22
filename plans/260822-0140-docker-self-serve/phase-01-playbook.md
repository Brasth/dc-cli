---
title: "Phase 1 — recover playbook contract"
status: proposed
parent: ./plan.md
---

# Phase 1 — recover playbook contract

No new user-facing command in this phase. Lock the table so `dc-recover` and the TUI cannot invent actions later.

Source of truth today:

- Host codes: `lib/dc-host.sh` (`dc_host_set` / `dc_host_diagnose`)
- Engine split: `lib/dc-engine.sh`
- Workspace checks: `bin/dc-doctor` (`CHECK_IDS`)

Recover **ranks**. First matching row wins. Later warnings are not shown as the primary next step.

## Rank order

1. Host not ready (`dc-host` code ≠ `ready`)
2. Split-brain (`docker_context` blocker / `docker_split_brain`)
3. Workspace cannot start (disk ENOSPC, required networks missing, labeled port clash)
4. Workspace confusing (kind=none, unpublished ports, stale sidecars)
5. Usable — next is `dc-tui` / `dc-up`

## Closed table

| Rank | When | `next.id` | `next.command` (this machine) | `apply` | Verify |
|---|---|---|---|---|---|
| 1 | `docker_cli_missing` | `install_cli_or_engine` | macOS: copy `brew install docker colima && colima start && dc-doctor` **or** open Desktop guide. Linux: open Engine/Desktop guide. | `open_guide`, `copy` | `dc-host` → `ready` |
| 1 | `docker_engine_missing` | `install_engine` | same as above | `open_guide`, `copy` | `dc-host` → `ready` |
| 1 | `docker_engine_stopped` + hint=`desktop` | `start_desktop` | launch Docker Desktop; wait; retry | `launch_desktop` | `dc-host` → `ready` |
| 1 | `docker_engine_stopped` + hint=`colima` | `start_colima` | `colima start` | `colima_start` | `dc-host` → `ready` |
| 1 | `docker_engine_stopped` + hint=`linux` | `start_linux_docker` | print `sudo systemctl start docker` | none | `dc-host` → `ready` |
| 1 | `docker_engine_stopped` + hint=`unknown` | `start_unknown_engine` | print "Start Desktop / colima start / dockerd" | `open_guide` | `dc-host` → `ready` |
| 1 | `docker_permission_denied` | `fix_socket_group` | print Linux group recipe; no apply | none | `dc-host` → `ready` |
| 1 | `docker_context_invalid` | `reset_context` | print `unset DOCKER_HOST`; apply `docker context use default` if `--yes` | `context_use` | `dc-engine` extra empty |
| 2 | `docker_split_brain` | `pick_engine` | apply `docker context use <recommended>` only; print "stop the extra yourself" | `context_use` | `DC_ENGINE_LAST_EXTRA` empty |
| 3 | doctor `disk` + ENOSPC / `dc-up` no space | `reclaim_disk` | `dc-df` then `dc-prune --yes` | `prune_safe` | retry `dc-up` |
| 3 | doctor `required_networks` missing declared `external: true` bridge | `ensure_nets` | `dc-up --create-nets` | `create_nets` | doctor networks ok |
| 3 | host port allocated + holders labeled | `take_ports` | `dc-up --take-ports` | `take_ports` (existing rules) | `dc-up` |
| 3 | host port allocated + unlabeled / unknown | `report_holders` | print holder list + stop instructions | none | user retry |
| 4 | `workspace.kind=none` | `try_sandbox` | `dc-try` (TUI already confirms) | existing | labeled app |
| 4 | stale owned sidecars | `prune_orphans` | `dc-prune --yes` (owned-only orphans) | `prune_safe` | doctor sidecars ok |
| 4 | desired ports unpublished | `forward_ports` | `dc-forward` | none in v1 (keep current forward rules) | doctor `actual_ports` |
| 5 | usable | `ready` | `dc-tui` or `dc-up` | none | — |

`apply` values are the only strings phase 2 may execute. Anything else is print/copy.

## Apply allowlist (phase 2, gated)

| `apply` | Command | Confirm | Notes |
|---|---|---|---|
| `launch_desktop` | macOS `open -a Docker`. Linux: only if a documented Desktop binary exists. | yes | Wait loop is retry, not a hidden sleep-forever. |
| `colima_start` | `colima start` | yes | Only if `command -v colima` and hint/engine is colima. |
| `context_use` | `docker context use NAME` | `--yes` or TUI confirm | Same as today's `dc-engine --fix --yes`. |
| `prune_safe` | `dc-prune --yes` | yes | Safe set only. Never `--all` or `--volume` from recover. |
| `create_nets` | `dc-up --create-nets` | yes | This folder. Overlay/IPAM still refuse. |
| `take_ports` | `dc-up --take-ports` | yes | Labeled / proven compose-kind only. |
| `open_guide` | `open` / `xdg-open` URL | no (open is the action) | Already in TUI `d`. |
| `copy` | clipboard the one command | no | Already in TUI `c`. |

Forbidden applies (must remain print-only):

- any `sudo`
- `colima stop` / quit Desktop / `systemctl stop`
- `brew install` / package-manager install
- `usermod` / group changes
- `dc-prune --all` / `--volume`
- `colima start --disk` / qemu resize
- writing `~/.zshrc` / `~/.bashrc` / Docker context json by hand

## JSON shape (`dc-recover --json`, phase 2)

```json
{
  "schemaVersion": 1,
  "command": "dc-recover",
  "host": { "code": "docker_engine_stopped", "engineHint": "colima" },
  "doctor": { "summary": { "status": "blocker" } },
  "next": {
    "id": "start_colima",
    "summary": "Colima is installed but not running",
    "command": "colima start",
    "apply": "colima_start",
    "applyAllowed": true,
    "verify": "dc-host",
    "escalate": "dc-recover --report"
  }
}
```

`doctor` may be omitted when host is not ready (daemon checks would be skipped anyway).

`applyAllowed` is false on linux-stopped, permission-denied, unlabeled holders, install-missing.

## Tests to add in phase 1 (contract only)

Reuse `tests/host`, `tests/doctor`, `tests/engine` fixtures. Add a table-driven script that maps each known host code + hint to `next.id` / `apply`. No Docker daemon required.

| Fixture | Expect `next.id` | Expect `apply` |
|---|---|---|
| docker missing | `install_cli_or_engine` | `open_guide` or `copy` |
| engine stopped, hint colima | `start_colima` | `colima_start` |
| engine stopped, hint desktop | `start_desktop` | `launch_desktop` |
| engine stopped, hint linux | `start_linux_docker` | none |
| permission denied | `fix_socket_group` | none |
| split-brain extra=colima | `pick_engine` | `context_use` |
| host ready, kind=none | `try_sandbox` | existing |
| host ready, usable | `ready` | none |

Phase 1 ships the mapper + tests. Phase 2 wires `bin/dc-recover`.

## Docs in this phase

None user-facing. This file is the contract.

## Exit

Phase 1 is done when:

- [ ] Allowlist approved (plan open question)
- [ ] Mapper tests pass against current host/doctor codes
- [ ] No `dc-doctor --fix` was added
