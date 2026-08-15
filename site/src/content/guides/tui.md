---
title: "dc-tui board and keys"
description: "dc-tui is the clickable board for this folder. Keys for start, shell, open, attach, ports, stop, logs, fleet."
h1: "The board is the product."
updated: 2026-08-15
howto: false
faq:
  - q: What is the difference between open and attach?
    a: Open is the host editor on the bind-mount (Zed, VS Code, Sublime). Attach is VS Code Remote into the Linux app. Zed and Sublime cannot attach.
  - q: Does e / shell enter a compose sidecar?
    a: No. e and shell are always the labeled app. Click a stack row, or dc-exec --service NAME, for siblings.
  - q: Why does the TUI come back after shell or logs?
    a: A normal exit is not a crash. The board resumes so you can hit the next verb.
---

```bash
dc-tui              # this folder (cwd, then git root)
dc-tui ~/src/app
dc-tui --all        # every labeled workspace
```

Equal-width button grid. Compact header. Stack rows: **up** / **down**, hover, click to exec (starts the box first).

## Keys

| Button | Key | Does |
|---|---|---|
| **start** | `u` | `dc-up` + `dc-forward` |
| **shell** | `e` | `dc-exec` — **app** only |
| **open** | `o` | host editor on the bind-mount |
| **attach** | `a` | VS Code Remote **into** the app (`code` only) |
| **ports** | `p` | `dc-forward` (Colima sidecar) |
| **stop** / **rm** | `s` / `x` | full stack `dc-down` / `dc-down --rm` |
| **logs** | `l` | `docker logs -f` on the app |
| **fleet** | `f` | other workspaces |
| **more** / **quit** | `?` / `q` | legend / exit |
| **disk** | `d` | `dc-df` report (stays in TUI) |

Header shows a compact disk line from `dc-df`. Reclaim stays CLI-only (`dc-prune --yes`).

## App vs other services

| Target | Command | How |
|---|---|---|
| labeled **app** | `dc-exec` / key `e` | `devcontainer exec` |
| any other compose **service** | click the stack row, or `dc-exec --service NAME` | start if down, then exec |

There is **no** `dc` meta-binary.

See also [ports](/guide/ports/) and the [README TUI](https://github.com/Brasth/dc-cli#tui).
