# Product Hunt — dc-cli

## Listing

| Field | Content |
|-------|---------|
| **Name** | dc-cli |
| **Tagline** | Dev containers from your terminal — no VS Code required |
| **Description** | `dc up`, `dc exec`, and `dc doctor` for any folder with `.devcontainer` or compose. A TUI board for ports, logs, disk, and fleet. Fixes Colima port pain and Docker confusion — without touching your config. MIT. macOS & Linux. |
| **Website** | https://dc.brasth.com |
| **Topics** | Developer Tools, Open Source, CLI, Docker, DevOps |

## Gallery (upload in order)

1. `site/public/videos/walkthrough-intro.mp4` — hero terminal demo (1280×720)
2. `site/public/videos/walkthrough-board.mp4` — TUI board
3. `site/public/videos/walkthrough-recover.mp4` — doctor & recover
4. `site/public/og.png` — social card (1200×630)

## Maker first comment (post at launch)

Hey PH — I built dc-cli because I kept bouncing between VS Code Remote, raw `docker exec`, and Colima port hacks.

dc-cli wraps the *official* Dev Containers CLI with host-global verbs: `dc up`, `dc exec`, `dc doctor`, and a TUI board. It never edits your `.devcontainer`.

When Docker blocks you, `dc recover` gives one next step.

Try:

```bash
curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

Feedback welcome on GitHub: https://github.com/Brasth/dc-cli

## Launch-day checklist

- [ ] Schedule 12:01 AM PT
- [ ] Rally 20–50 dev friends for first 4 hours
- [ ] Reply to every comment within 1 hour
- [ ] Pin install link + hero video in first comment
