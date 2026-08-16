---
title: "Disk full: dc-df and dc-prune"
description: "When dc-up fails with ENOSPC, run dc-df then dc-prune --yes. Cache/images/nets are engine-wide. Never docker system prune -af --volumes."
h1: "No space left. Do not nuke volumes."
updated: 2026-08-16
howto: true
steps:
  - name: Report
    text: dc-df shows images, cache, volumes, and Colima guest df.
  - name: Dry-run
    text: dc-prune prints what it would delete. Exit 0. Nothing removed.
  - name: Reclaim
    text: dc-prune --yes. Then retry dc-up.
---

When `dc-up` fails with **no space left** / ENOSPC:

```bash
dc-df                    # images / cache / volumes + Colima guest df
dc-prune                 # dry-run
dc-prune --yes           # safe set only
dc-up                    # retry
```

| Flag | Risk |
|---|---|
| `dc-prune --yes` | **Engine-wide** build cache, dangling images, unused nets; **owned-only** orphan `dc-forward` sidecars (target proven absent) |
| `dc-prune --all --yes` | **Engine-wide** unused **tagged** images; parked stacks rebuild on next `dc-up` |
| `dc-prune --volume NAME --yes` | **High / owned-only** — named volume data (DBs). One name only; never bulk; mount inventory must succeed |
| `docker system prune -af --volumes` | **Do not** — not wrapped; destroys named volumes |

If Colima guest `/` is still ~100% after prune, the VM disk cap is full: `dc-prune --colima-hint`. Prune does **not** grow the qemu/VZ image.

Named volumes are listed by `dc-df --volumes` but not auto-deleted.

See [README disk](https://github.com/Brasth/dc-cli#disk-full).
