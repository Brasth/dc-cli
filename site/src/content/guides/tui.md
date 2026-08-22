---
title: "dc-tui board and keys"
description: "dc-tui is the clickable board for this folder. Primary keys: start, shell, stop. Click a published website URL or press 1-9 to open it. Meta: open, attach, ports, logs, top, nets, db, files, fleet."
h1: "The board is the product."
updated: 2026-08-19
howto: false
faq:
  - q: What is the difference between open and attach?
    a: Open is the host editor on the bind-mount (Zed, VS Code, Sublime). a / dc-open --attach is the VS Code Remote URI into Linux. Zed attaches itself. Sublime cannot.
  - q: How do I work inside the container in Zed?
    a: "dc-up then dc-open. In Zed use Project: Open Remote → Connect Dev Container. dc-cli owns ports, stop, and fleet. Zed does not publish forwardPorts. No extension."
  - q: Does e / shell enter a compose sidecar?
    a: No. e and shell are always the labeled app. Click a stack row, or dc-exec --service NAME, for siblings.
  - q: Why does the TUI come back after shell or logs?
    a: A normal exit is not a crash. The board announces leave, then resumes so you can hit the next verb.
  - q: What does R do versus r?
    a: r reloads the board. R restarts the selected compose sibling (dc-exec --service NAME --restart). The labeled app row refuses — use u/s. Fleet refuses R. --id --restart is invalid.
  - q: Why does the header say checking instead of stopped?
    a: checking… means discovery is still running. stopped means discovery finished and no container is there. unknown means discovery failed — press r. Manual reload keeps the last snapshot and shows refreshing….
  - q: How do I move without the mouse?
    a: j/k or arrows move the cursor. Enter opens a fleet folder or execs the selected stack row.
  - q: How do I open the forwarded website?
    a: After start, website ports (80, 443, 3000, 5173, 8000, 8080, 9001, …) become clickable http://127.0.0.1:PORT tiles. Keys 1–9 open the first nine. Databases stay off that row — press b / dc-db.
  - q: How do I open TablePlus on the stack database?
    a: b or dc-db. Uses the host port you already set on the db service (compose ports or a well-known forwardPorts number). No declared map → refuse. Two DBs need dc-db --service NAME.
  - q: What is the difference between d and t?
    a: d dumps dc-df (disk document). t opens live CPU/RAM for this folder via dc-stats. Fleet refuses t. Desktop guest is cap only — it never invents a live percent. Reclaim stays dc-prune.
  - q: What does n / nets do?
    a: n lists this folder's declared compose networks. Missing external:true names can be created as a default bridge (y then dc-up --create-nets). Compose-managed nets are shown, not created. Overlay, custom IPAM, and inspect-unknown are refused. Fleet refuses n.
---

```bash
dc                  # this folder (cwd, then git root) — same as dc-tui
dc ~/src/app
dc --all            # every labeled workspace
```

Startup draws the **dc-cli** mark (host frame + phosphor pip). Any key skips. `DC_TUI_NO_SPLASH=1` skips it. The compact mark stays in the header.

While containers are discovered the header shows **checking…** — not **stopped**. **stopped** is only after a successful empty result. Discovery failure shows **unknown** (`r` retries). Manual reload keeps the last snapshot and shows **refreshing…**. Fleet / folder switches clear old context first.

![dc-tui board with top next to logs](/images/tui.png)

Header **load** line is `dc-stats`. Press **t** or click **top**.

![dc-tui top overlay: CPU, RAM, net, q/t back](/images/tui-top.png)

Primary row: **start** · **shell** · **stop**. Meta is quieter. **rm** asks `y/n`. Stack / fleet: **up** / **down**, `j`/`k` or click, Enter to open or exec.

## Keys

| Button | Key | Does |
|---|---|---|
| **start** | `u` | `dc up` — `.devcontainer` uses official CLI + forward; compose-kind uses `docker compose` (no forward) |
| **shell** | `e` | `dc exec` — **app** only |
| **stop** | `s` | full stack `dc down` |
| **rm** | `x` | `dc-down --rm` after `y` |
| **open** | `o` | host editor on the bind-mount |
| **attach** | `a` | VS Code Remote URI; Zed prints Connect Dev Container steps. N/A for compose-kind. |
| **ports** | `p` | `dc-forward` (Colima sidecar) |
| **url** | `1`–`9` / click | open a published website in the host browser |
| **logs** | `l` | follow docker logs for the selected stack row (highlighted; q back). Fleet refuses. |
| **restart** | `R` | restart the selected stack sibling. Labeled app row refuses (`u`/`s`). Fleet refuses. `r` still reloads. |
| **top** | `t` | CPU / RAM for this folder (stays in TUI). Fleet refuses. |
| **nets** | `n` | this folder's declared compose nets. `y` creates missing externals then start. Fleet refuses. |
| **db** | `b` | `dc-db` — host TablePlus on a declared db port |
| **files** | `m` | `dc-files` — yazi/nnn in the box; Enter opens code/cursor on this container (`DC_FILES_EDITOR=vim` keeps vim) |
| **fleet** | `f` | other workspaces |
| **upgrade** | `U` | when a newer release is available — confirms then `dc-upgrade --yes` |
| **more** / **quit** | `?` / `q` | legend / exit |
| **disk** | `d` | `dc-df` report (stays in TUI) |
| **rows** | `j`/`k`, Enter | cursor; fleet opens a folder, stack execs |

Header shows a compact disk line from `dc-df`, an app load pulse from `dc-stats`, and declared compose nets when present. When a newer GitHub release exists, a banner points at `U` / `dc upgrade`. `d` is still the disk document. `t` is live CPU/RAM for this folder (`q` back). `n` lists this folder's required nets (`y` creates missing `external: true` bridge nets, then `dc-up --create-nets`). Fleet refuses `t` and `n`. Desktop guest is cap only. Reclaim stays CLI-only (`dc-prune --yes`). CLI twins: `dc-stats` / `dc-net`.

## App vs other services

| Target | Command | How |
|---|---|---|
| labeled **app** | `dc-exec` / key `e` | `devcontainer exec` |
| any other compose **service** | click the stack row, or `dc-exec --service NAME` | start if down, then exec |

`dc` with no args (or `dc tui`) is the board. Hyphenated `dc-tui` stays.

See also [ports](/guide/ports/).
