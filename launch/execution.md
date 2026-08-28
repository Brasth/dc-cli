# Launch execution — dc-cli growth

Run after first-run fixes are on `main` and Pages has deployed.

## Pre-flight

- [ ] `docs/growth/first-run-audit.md` reviewed — core paths PASS
- [ ] Site live at https://dc.brasth.com with updated copy
- [ ] `/play/` shows **no-config** demo (press u → y → e)
- [ ] Plausible (or analytics) receiving events on dc.brasth.com
- [ ] GitHub issue template **Activation feedback** available

## Phase 0 — Re-contact silent users (4)

These users installed wrappers-only or hit dead `dc-up`. Send individually (not a blast):

```text
Hey — thanks for trying dc-cli earlier. We fixed the first-run path.

Quick retry:
  command -v devcontainer || echo MISSING
  curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
  source ~/.zshrc   # or ~/.bashrc
  cd <your project>
  dc                # board
  dc try            # if no .devcontainer or compose

Did you get a shell with dc exec? Reply either way — that's the only metric we care about right now.

— Huy
```

Track responses in a private note (not in repo):

| User | Contacted | Got shell? | Stuck at |
|------|-----------|------------|----------|
| 1    |           |            |          |
| 2    |           |            |          |
| 3    |           |            |          |
| 4    |           |            |          |

## Phase 1 — Product Hunt

Use copy from [product-hunt.md](./product-hunt.md).

- [ ] Schedule **12:01 AM PT** on a weekday
- [ ] Upload gallery videos in order (intro → board → recover → og.png)
- [ ] Post maker first comment with install curl
- [ ] Rally 20–50 dev friends for first 4 hours
- [ ] Reply to every comment within 1 hour
- [ ] Ask: **"Did you get a shell?"** not "please star"

## Phase 2 — Hacker News

Use copy from [hacker-news.md](./hacker-news.md).

- [ ] Post **Show HN** weekday **8–10 AM US Eastern**
- [ ] Title: `Show HN: dc-cli – Dev containers from your terminal (no VS Code)`
- [ ] First comment: install block + link to `/play/` for browser demo
- [ ] Monitor for Docker / DevPod / VS Code comparisons — link to wedge FAQ in HN doc

## Phase 3 — Terminal-first communities

- [ ] r/commandline, r/devops — link `/play/` and install guide
- [ ] Dev Containers / Docker Discord threads (helpful, not spammy)
- [ ] Cursor / agent users — mention `--with-skill` in [skill/SKILL.md](../skill/SKILL.md)

## Success questions (every channel)

1. Did you copy the install command?
2. Did `dc` run after `source ~/.zshrc`?
3. Did you get a shell with `dc exec`?

If no to #3, ask them to file [Activation feedback](https://github.com/Brasth/dc-cli/issues/new?template=feedback.yml).

## Post-launch (week 1–2)

See [docs/growth/funnel-tracking.md](../docs/growth/funnel-tracking.md) for the 2-week measurement loop.
