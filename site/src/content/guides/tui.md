---
title: "dc-tui board and keys"
description: "dc-tui is the clickable board for this folder. Primary keys: start, shell, stop. Click a published website URL or press 1-9 to open it. Meta: open, attach, ports, logs, top, db, files, fleet."
h1: "The board is the product."
updated: 2026-08-17
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
  - q: How do I move without the mouse?
    a: j/k or arrows move the cursor. Enter opens a fleet folder or execs the selected stack row.
  - q: How do I open the forwarded website?
    a: After start, website ports (80, 443, 3000, 5173, 8000, 8080, 9001, …) become clickable http://127.0.0.1:PORT tiles. Keys 1–9 open the first nine. Databases stay off that row — press b / dc-db.
  - q: How do I open TablePlus on the stack database?
    a: b or dc-db. Uses the host port you already set on the db service (compose ports or a well-known forwardPorts number). No declared map → refuse. Two DBs need dc-db --service NAME.
  - q: What is the difference between d and t?
    a: d dumps dc-df (disk document). t opens live CPU/RAM for this folder via dc-stats. Fleet refuses t. Desktop guest is cap only — it never invents a live percent. Reclaim stays dc-prune.
---

```bash
dc-tui              # this folder (cwd, then git root)
dc-tui ~/src/app
dc-tui --all        # every labeled workspace
```

Startup draws the **dc-cli** mark (host frame + phosphor pip). Any key skips. `DC_TUI_NO_SPLASH=1` skips it. The compact mark stays in the header.

![dc-tui board](/images/tui.png)

Primary row: **start** · **shell** · **stop**. Meta is quieter. **rm** asks `y/n`. Stack / fleet: **up** / **down**, `j`/`k` or click, Enter to open or exec.

## Keys

| Button | Key | Does |
|---|---|---|
| **start** | `u` | `dc-up` + `dc-forward` (needs `.devcontainer`) |
| **shell** | `e` | `dc-exec` — **app** only |
| **stop** | `s` | full stack `dc-down` |
| **rm** | `x` | `dc-down --rm` after `y` |
| **open** | `o` | host editor on the bind-mount |
| **attach** | `a` | VS Code Remote URI; Zed prints Connect Dev Container steps |
| **ports** | `p` | `dc-forward` (Colima sidecar) |
| **url** | `1`–`9` / click | open a published website in the host browser |
| **logs** | `l` | follow docker logs in the board (highlighted; q back) |
| **top** | `t` | CPU / RAM for this folder (stays in TUI). Fleet refuses. |
| **db** | `b` | `dc-db` — host TablePlus on a declared db port |
| **files** | `m` | `dc-files` — yazi/nnn in the box, else Linux yazi in /tmp |
| **fleet** | `f` | other workspaces |
| **more** / **quit** | `?` / `q` | legend / exit |
| **disk** | `d` | `dc-df` report (stays in TUI) |
| **rows** | `j`/`k`, Enter | cursor; fleet opens a folder, stack execs |

Header shows a compact disk line from `dc-df` and an app load pulse from `dc-stats`. `d` is still the disk document. `t` is live CPU/RAM for this folder (`q` back). Fleet refuses `t`. Desktop guest is cap only. Reclaim stays CLI-only (`dc-prune --yes`). CLI twin: `dc-stats` / `--json`.

## App vs other services

| Target | Command | How |
|---|---|---|
| labeled **app** | `dc-exec` / key `e` | `devcontainer exec` |
| any other compose **service** | click the stack row, or `dc-exec --service NAME` | start if down, then exec |

There is **no** `dc` meta-binary.

See also [ports](/guide/ports/) and the [README TUI](https://github.com/Brasth/dc-cli#tui).
