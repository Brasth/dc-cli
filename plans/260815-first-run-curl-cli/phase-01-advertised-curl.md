---
phase: 1
title: Advertised curl
status: completed
priority: P1
dependencies: []
effort: 1h
---

# Phase 01: Advertised curl

One string everywhere people copy:

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

Files: `README.md`, `site/src/components/Install.astro`, `site/src/content/guides/install.md`, `skill/SKILL.md`, `assets/branding/copy.md`, `.github/workflows/ci.yml` grep.

Do not change `install.sh` flag parsing.
