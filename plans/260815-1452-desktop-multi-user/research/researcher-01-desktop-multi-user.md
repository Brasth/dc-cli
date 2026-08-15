---
type: researcher
date: 2026-08-15
topic: desktop-app-and-multi-user-feasibility
conducted: 2026-08-15T14:52:00
---

# Research Report: Desktop version + multi-user for dc-cli

## Table of contents

1. [Executive Summary](#executive-summary)
2. [Research Methodology](#research-methodology)
3. [Key Findings](#key-findings)
4. [Comparative Analysis](#comparative-analysis)
5. [Implementation Recommendations](#implementation-recommendations)
6. [Resources & References](#resources--references)
7. [Appendices](#appendices)

## Executive Summary

dc-cli can be used by many people. It cannot be one shared desktop session.

The product is a per-login kit: bash wrappers + optional Go TUI, talking to the local Docker engine and `@devcontainers/cli`. Identity is the host path label `devcontainer.local_folder`. There is no auth, no dc daemon, no user/owner label. Fleet, prune, and `--take-ports` mutate the whole engine.

A desktop GUI is technically possible as a local front (Wails v2 is the least-wrong stack: Go + OS WebView). It still needs Docker/Colima on that machine and a PTY for shell/logs. It does **not** give multi-user.

"Multiple users" is solved by distribution: GitHub Release binaries + Homebrew tap. A shared team board is a different product (Coder, Codespaces, Gitpod).

**Do this:** Homebrew + prebuilt `dc-tui`. **Maybe later:** local `.app`. **Do not:** Electron remake, remote multi-tenant Docker control.

## Research Methodology

- Sources consulted: 12+ (official docs, GitHub, 2026 framework comparison, Homebrew, Coder, repo scout)
- Date range: 2022 (Docker Desktop multi-user issue) to 2026-08
- Key search terms: Wails vs Tauri vs Fyne 2026, Docker Desktop Colima multi-user, Homebrew tap Go CLI, Charm Bubble Tea desktop, Coder devcontainers
- Gemini CLI: **unavailable** (`command -v gemini` exit 127). Fell back to WebSearch.
- Recency: last 12 months preferred; Docker multi-user issue is older but still the constraint.

Evaluation criteria: YAGNI/KISS/DRY, fit to current bash+Go TUI, macOS-first, security of Docker socket, install friction.

## Key Findings

### 1. Technology Overview

dc-cli today:

```
user login
  ~/bin/dc-*          bash
  ~/.config/devcontainer/{dc-common.sh,override.json}
  cmd/dc-tui          Bubble Tea, shells out to dc-*
        │
        ▼
  docker CLI + @devcontainers/cli
        │
  Colima or Docker Desktop   (one engine per context)
```

No listen port. No session. Workspace = host abs path on a label.

Desktop wrap = new window that execs the same binaries. Shared multi-user = new server, auth, tenancy, remote PTY. That is not this repo.

### 2. Current State & Trends (2025–2026)

| Stack | State | Fit here |
|---|---|---|
| Wails v2.12 stable, v3 alpha | Go + OS WebView, ~15 MB vendor-stated | Best if GUI happens. Team already in Go. |
| Tauri v2 | 2026 default for greenfield; Rust backend; 3–15 MB | Overkill. No Rust in repo. |
| Electron 42 | Proven signing/auto-update; 50–150 MB+ | Wrong weight for a Docker click-board. |
| Fyne | Pure Go native; HN 2026 teams like it for simple UIs | Full board rewrite. No reuse of JSON CLI. |
| Deno desktop 2.9 | Experimental Jun 2026 | No. |
| Bubble Tea | TUI only. No official desktop host. | Keep as CLI board. Do not wrap the alt-screen in Electron. |

Homebrew: formula for CLI, cask for `.app`. Official casks moving to mandatory Apple codesign + notarize (policy called out for 2026). Private tap does not need that on day one.

Docker: Desktop on Mac is per-installing-user. Second macOS login often cannot talk to the daemon (docker/for-mac#6781). Colima socket lives under `~/.colima`. Two logins = two VMs, or a broken share.

True multi-user remote devcontainers already exist: Coder dashboard + `@devcontainers/cli` inside a workspace. Do not clone that.

### 3. Best Practices

1. **Distribute the existing tool first.** Prebuilt binary + brew. Most "desktop so others can use it" pain is install, not pixels.
2. **One human, one Docker context.** Document it. Do not pretend fleet is multi-tenant.
3. **GUI calls CLIs.** `dc-ls --json` / `dc-df --json` / `dc-exec --list --json` are the API. Do not import `package main`.
4. **Embed a PTY** for shell, logs, `dc-up`. Or force `dc-up --yes`.
5. **Launch as the login user.** Finder-launched apps drop Colima `DOCKER_HOST` and PATH. Resolve `~/.colima/default/docker.sock`.
6. **Keep prune CLI-only.** Daemon-wide. Too easy to nuke a sibling stack.
7. **Homebrew formula, not cask,** until there is a real `.app` and a Developer ID.

### 4. Security Considerations

- `dc-*` ≡ full Docker access. Do not expose over a network.
- No user label. Anyone with the socket can `dc-exec --id` / `dc-down --all`.
- `--take-ports` stops foreign compose projects on the same engine.
- `dc-prune --yes` is engine-global (cache, dangling images, unused nets). `--all --yes` drops unused tagged images.
- Host publish is `-p HOST:HOST` (typically all interfaces).
- Unsigned downloaded `dc-tui` will hit Gatekeeper.
- Electron default surface is larger than needed. Tauri/Wails capability model is tighter; still local root-equivalent via Docker.

### 5. Performance Insights

- TUI is cheap. Bottleneck is `devcontainer up` + image/VM disk, not UI.
- Electron idle RAM 100–300 MB to draw a board we already have in a terminal: waste.
- Wails/Tauri WebView: tens of MB. Fine if the PTY PoC works.
- Prebuilding `dc-tui` removes "need Go on the user machine" — the real multi-user install win.

## Comparative Analysis

### A. More people install CLI (recommended)

| | |
|---|---|
| What | Release tarball + Homebrew formula |
| Pros | Matches product. Small. No new runtime. |
| Cons | Still a terminal. Users must have Docker. |
| Effort | Days, not weeks |

### B. Local desktop GUI

| | |
|---|---|
| What | Wails `.app` calling `dc-*` + embedded PTY |
| Pros | Dock icon, folder picker, click without terminal literacy |
| Cons | PATH/socket hell; signing; still 1 user/1 engine; duplicates TUI |
| Effort | Weeks + Apple ID if you want a clean Gatekeeper path |

### C. Shared multi-user / team host

| | |
|---|---|
| What | Server + auth + remote PTY + tenant labels |
| Pros | "One board for the team" |
| Cons | New product. Fights Docker Desktop/Colima. Security cliff. Coder already exists. |
| Effort | Months. Do not do it in this repo. |

### D. Two macOS users, one Mac

Unsupported upstream. If they share a socket: fleet/prune/ports collide. If they do not: two Colimas. A desktop app cannot fix this.

## Implementation Recommendations

### Quick Start Guide

1. Ask: install more people, or GUI, or shared host?
2. If install: GoReleaser / Actions → tag assets → tap Formula.
3. If GUI: 1-day Wails PoC (list + PTY exec). Kill if Colima socket fails from the `.app`.
4. If shared host: stop. Point at Coder. Do not start a daemon.

### Code Examples

Release kit (shape, not final):

```text
dc-cli-0.x.y-darwin-universal/
  bin/dc-up dc-exec dc-down dc-ls dc-ps dc-forward dc-open dc-df dc-prune dc-tui
  lib/dc-common.sh
  config/override.json
```

Homebrew formula sketch:

```ruby
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  url "https://github.com/Brasth/dc-cli/releases/download/v0.x.y/dc-cli-0.x.y-darwin-universal.tar.gz"
  sha256 "…"
  license "MIT"

  def install
    bin.install Dir["bin/*"]
    (etc/"devcontainer").install "lib/dc-common.sh", "config/override.json"
  end

  test do
    system "#{bin}/dc-tui", "--help"
  end
end
```

GUI stay-in-board actions (already non-TTY):

```text
dc-ls --json
dc-exec --list --json
dc-df --json
dc-open / dc-open --attach
dc-forward / dc-down
dc-up --yes          # no TTY prompt
```

PTY still required: `dc-exec`, `dc-exec --id`, `docker logs -f`.

### Common Pitfalls

- Treating `dc-tui` as an embeddable widget. It is `package main` + alt-screen.
- Launching the app as a service user. Colima socket will be missing.
- Putting prune/down --all in a big red GUI button.
- Homebrew cask before an `.app` exists.
- Believing a GUI = multi-user.

## Resources & References

### Official Documentation

- [Wails introduction](https://wails.io/docs/introduction/)
- [Dev Containers CLI](https://github.com/devcontainers/cli)
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Coder Dev Containers](https://coder.com/docs/user-guides/devcontainers)
- [Homebrew Cask](https://formulae.brew.sh/cask/)

### Recommended Tutorials

- [Distributing a Go CLI via a Homebrew tap](https://www.yellowduck.be/posts/distributing-a-go-cli-via-a-homebrew-tap) (2026)
- [Desktop apps from web: Tauri vs Electron vs Deno vs Wails 2026](https://www.digitalapplied.com/blog/desktop-apps-web-stack-tauri-electron-deno-wails-2026)

### Community Resources

- [Docker Desktop for Mac does not work for multiple users](https://github.com/docker/for-mac/issues/6781)
- [Wails vs Tauri (Go shop)](https://www.reddit.com/r/golang/comments/17koicc/we_decided_to_use_golang_with_wails_instead_of/)
- [HN: Fyne vs Wails 2026](https://news.ycombinator.com/item?id=46030414)

### Further Reading

- Repo README platform table, ports, prune safety
- `lib/dc-common.sh` label + holder logic
- `cmd/dc-tui/main.go` ExecProcess vs CombinedOutput

## Appendices

### A. Glossary

| Term | Meaning |
|---|---|
| Fleet | `dc-ls --all` / `dc-tui --all` — every labeled container on this engine |
| Label | `devcontainer.local_folder` — host path, not a user id |
| Sidecar | `alpine/socat` publish from `dc-forward` |
| PTY | Real terminal needed for interactive exec/logs |
| Tap | Homebrew repo `homebrew-*` with a Formula |

### B. Version Compatibility Matrix

| Piece | Today | Desktop wrap | Shared multi-user |
|---|---|---|---|
| macOS + Colima/Desktop | yes | yes if login env | no |
| Linux | yes | later | only as a new server product |
| WSL2 | best-effort | no | no |
| Native Windows | no | no | no |
| Two macOS logins, one Docker | broken upstream | broken | broken |

### C. Raw Research Notes

- Gemini toggle on; binary missing; WebSearch used (5 queries).
- Scout: bin/lib host coupling; TUI process model; install/CI distribution. See `../reports/scout-report.md`.
- No `.goreleaser`, no tap, no `.app` in repo.
- Marketing site plans are unrelated and not blockers.

## Next steps

1. User picks A / B / C.
2. If A: cook phase 1 (release + brew).
3. If B: 1-day Wails PTY PoC before any plan cook.
4. If C: decline in this repo.

## Unresolved Questions

- Does "multiple users" mean teammates on their own Macs, or one shared machine / one shared Docker?
- Is there a named person who will not use a terminal?
- Apple Developer ID available if a `.app` is required?
- Accept a `dc` meta-binary later, or keep ten commands forever?
