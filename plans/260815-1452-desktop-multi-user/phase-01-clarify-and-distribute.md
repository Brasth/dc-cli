---
title: "Phase 1: Clarify + distribute"
status: in_progress
---

# Phase 01: Clarify + distribute

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [researcher-01-desktop-multi-user.md](./research/researcher-01-desktop-multi-user.md)
- Current install: `install.sh`, README platform table

## Overview

- **Priority:** high
- **Status:** pending
- **Description:** Lock the product shape. Then make install boring for many users. No GUI yet.

## Key Insights

- "Multiple users" today = curl + local `go build` of TUI. That is the friction.
- Docker Desktop itself fails for two macOS logins on one machine (docker/for-mac#6781).
- Homebrew formula is the macOS-native path. Cask needs a real `.app`. Official casks need notarization (Homebrew policy tightening through 2026).

## Requirements

### Functional

- One-command install on macOS/Linux that does not require Go
- Prebuilt `dc-tui` binary in GitHub Releases
- `brew install` via a private tap (or formula in-repo later)

### Non-functional

- Keep `bin/` + `lib/` as the contract. Do not invent a `dc` meta-binary unless we accept that scope.
- Do not mutate `~/.zshrc` from Homebrew
- Native Windows stays unsupported

## Architecture

```
GitHub Release
  ├── dc-cli-<ver>-darwin-universal.tar.gz   # bash kit + lipo dc-tui
  └── dc-cli-<ver>-linux-amd64.tar.gz
        │
        ├─ install.sh --ref vX.Y.Z     (keep curl path)
        └─ homebrew-dc-cli Formula     (new tap)
```

Each user still needs their own Docker/Colima. Labels stay `devcontainer.local_folder`.

## Related Code Files

- **Modify:** `install.sh` (consume release artifacts, not only source tarball), `.github/workflows/ci.yml` (release job)
- **Create:** `.goreleaser.yaml` or Actions matrix; `Brasth/homebrew-dc-cli` Formula
- **Delete:** none

## Implementation Steps

1. Confirm with user: distribution vs GUI vs shared host.
2. Add tagged release that uploads the bash kit + prebuilt `dc-tui` (darwin universal via `lipo`, linux amd64).
3. Point `install.sh` at those assets when `--ref` is a tag.
4. Add Homebrew tap formula that installs scripts + binary to prefix; config still `~/.config/devcontainer`.
5. Document `brew tap Brasth/dc-cli && brew install dc-cli` next to the curl line. Do not replace curl.

## Todo

- [x] Product-shape decision recorded (distribute CLI; GUI parked)
- [x] Pack script + `release.yml` + install.sh prefers kit/prebuilt TUI
- [x] Formula template at `packaging/homebrew/dc-cli.rb` (`bin/` + `lib/`, not libexec)
- [ ] Release artifacts exist for a new `v*` tag
- [ ] `Brasth/homebrew-dc-cli` created and SHA-filled
- [x] Curl installer still works (source + kit)

## Success Criteria

- Second user on another Mac installs without cloning or having Go
- No new daemon, no auth, no shared Docker

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Formula vs 10 binaries | Homebrew expects one keg | Install all `dc-*` from one formula |
| Gatekeeper on unsigned Go binary | `dc-tui` killed on download | Ad-hoc sign; Developer ID later |
| Users share one Mac login Docker | Cross-stop via fleet/prune | Docs: one Docker context per human |

## Next Steps

- If install is still "too CLI" after this, do phase 2.
- If they want one board for a team, stop. That is a new product.
