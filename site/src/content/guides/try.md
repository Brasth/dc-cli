---
title: "Sandbox start without config"
description: "dc-try starts a folder that has no .devcontainer and no root compose. External override. No project edits."
h1: "No config yet. Still get a shell."
updated: 2026-08-20
howto: true
steps:
  - name: Stand in a plain folder
    text: cd into a repo that has neither .devcontainer nor a root compose file.
  - name: Print the plan
    text: dc-try --print . shows profile, image, and the external override path under XDG state.
  - name: Start the sandbox
    text: dc-try . confirms, then runs the official CLI with --override-config. Agents and TUI use dc-try --yes.
  - name: Shell and stop
    text: dc-exec . then dc-down . Labels still identify the box. Nothing was written into the project.
faq:
  - q: Does dc-try edit my repo?
    a: No. Override JSON lives under $XDG_STATE_HOME/dc-cli/try/<hash>/. git status stays clean.
  - q: When should I use dc-up instead?
    a: When .devcontainer or a root compose file already exists. dc-try refuses those and points you at dc-up.
  - q: What profiles exist?
    a: go (go.mod), python (pyproject.toml / requirements.txt / Pipfile), node (package.json). Exactly one signal wins. Zero or many → generic base image.
  - q: Does it publish ports?
    a: Not in MVP. No forwardPorts in the generated config. Use a real .devcontainer when you need declared ports.
---

`dc-try` is the growth path for folders that would otherwise be `kind=none`.

| Command | Does |
|---|---|
| `dc-try --print .` | write override + print; do not start |
| `dc-try .` | confirm, then start |
| `dc-try --yes .` | start without prompt (TUI / agents) |
| `dc-try --profile node .` | force a profile |

```bash
cd /path/to/plain-repo
dc-try --print .
dc-try .
dc-exec .
dc-down .
```

TUI `u` / **start** on a configless folder asks `y/n`, then runs `dc-try --yes`.
