# Design: Wedge first-run (product + website)

Date: 2026-08-27  
Status: proposed  
Approach: **A — deepen the wedge** (do not expand into DevPod / desktop / spec-runner)

## Problem

`dc-cli` is a this-folder kit around the official Dev Containers CLI. Users try it and leave before they get a shell.

Evidence already in-repo:

- Four silent users installed wrappers-only; `dc-up` died with no `devcontainer` on `PATH` (`plans/260815-first-run-curl-cli`).
- Site H1 sells “Host-global helpers around the official Dev Containers CLI” (`site/src/components/Hero.astro`, `assets/branding/copy.md`).
- `dc-up` on `kind=none` refuses with a `dc-try` hint (`bin/dc-up`). Strangers who type `dc up` after the hero still hit a wall.
- Recover already ranks `kind_none` as `try_sandbox` but **does not apply** (`lib/dc-recover.sh`: `apply none`).
- Repo is 14 days old, 5 stars, 0 issues. Feature surface is wide; first contact is not.

This is a conversion problem, not a missing-category problem.

## Goal

A stranger who already has Docker can:

1. Copy the advertised install.
2. `source` their shell.
3. `cd` any folder and type `dc`.
4. Start a box (real `.devcontainer` / compose, or a `dc-try` sandbox).
5. Get a shell (`dc exec`).

If Docker is installed but quit / split, `dc recover` is the door — they do not become Docker admins.

Success is **got a shell**, not another verb.

## Non-goals (locked)

- Desktop / Wails / Electron / `.app`
- Shared team board, remote host, Coder clone
- Reimplementing `@devcontainers/cli` (crib / `dev` / brig path)
- Empty-machine engine install (recover v2 in `plans/260822-0140-docker-self-serve`)
- `dc-try` publishing ports (still not MVP)
- Changing the advertised curl (`--with-cli`)
- Editing project `.devcontainer`
- `dc-up --yes` on `kind=none` auto-running try (agents must call `dc-try --yes`)
- New clip videos, new site layout, new pages

## Product shape (unchanged)

```
this login
  dc / dc-<verb>     bash helpers
  dc-tui             Go board
        │
        ▼
  docker + official @devcontainers/cli
        │
  one engine (Colima or Desktop or dockerd)
```

Identity is still the host folder. No daemon. No auth.

## First-minute path

```
install (--with-cli)
    │
    ├─ engine not ready ──► dc recover (start / pick-one / context)
    │                       empty machine: still print/copy, not install
    │
    └─ engine ready
            │
            ├─ kind=devcontainer ──► dc up ──► dc exec
            ├─ kind=compose      ──► dc up ──► dc exec
            └─ kind=none         ──► dc try ──► dc exec
                                    TUI start (u) already confirms then dc-try --yes
                                    dc-up on a TTY offers the same confirm
                                    dc recover --yes may apply dc-try --yes
```

## Website (copy only)

The homepage must sell the same win as the product. Layout, videos, `/play`, and the install curl stay.

| Surface | Locked copy |
|---|---|
| Hero H1 | Dev containers from your terminal. |
| Hero sub | `dc up` starts this folder. No config? `dc try`. No VS Code. Never edits `.devcontainer`. |
| Trust (new, smaller, under sub) | Wraps the official Dev Containers CLI. |
| `<title>` | dc-cli — Dev containers from your terminal |
| Meta description | `dc up` starts this folder. `dc exec` is the shell. No config? `dc try`. No VS Code required. Never edits project `.devcontainer`. |
| Pain H2 | Same folder. A shell in one minute. |
| Install H2 | One curl. Then `dc`. |
| Install note | Official CLI included (`--with-cli`). Then `dc` in this folder. |
| `dc exec` titles | Shell in the app. (rule “never raw docker exec” stays in the body) |
| Manifesto | Unchanged. Stays below the win. |
| Footer | MIT. Canvilled / Brasth. Unchanged. |
| Install command | Unchanged. Must match README. |

Guides stay how-tos. Reorder the index so **try** sits next to install. Update kind/stuck sentences that still say “`dc-up` refuses” as the whole story.

Launch kits (`launch/*.md`) lead with the same H1. Do not invent a second tagline.

## Product changes (narrow)

### 1. Install “Next” block

`install.sh` currently dumps eight verbs. Lead with the first minute:

```
Next:
  source ~/.zshrc   # or ~/.bashrc
  cd /path/to/your/project
  dc                # board — start this folder
  dc try            # no .devcontainer or compose
If Docker is quit or split: dc recover
```

Advertised curl and wrappers-only default stay.

### 2. `dc-up` on `kind=none`

Keep non-TTY refuse + `dc-try` hint (existing test `up-hint-none`).

On an interactive TTY, after the same two lines, ask:

```
Start a sandbox with dc-try? [y/N]
```

`y` runs `dc-try --yes "$workspace"`. Anything else exits 1.

`dc-up --yes` on `kind=none` does **not** start a sandbox.

### 3. TUI empty / confirm copy

Confirm string today: `no .devcontainer/compose — start sandbox via dc-try? y/n`

Change to: `No config — start a sandbox? y/n`

Behavior unchanged (confirm → `dc-try --yes`). Update the Go tests that assert the old string if any; most assert `confirm == "try"` only.

### 4. Recover `kind_none`

Playbook already says apply existing `dc-try`. Code sets `apply none`.

v1 of *this* plan: `next.id=try_sandbox`, command `dc-try --yes`, `applyAllowed=1`, `--yes` runs `dc-try --yes` on the resolved workspace. Confirm still required without `--yes`.

Do not implement empty-machine brew install here.

### 5. Agent skill

Lead the skill with the first-minute path. `kind=none` → `dc-try --yes`. Host wall → `dc-recover --yes`. Do not grow the skill into a host manager essay.

### 6. Copy contract test

Add `tests/copy/run.sh` that greps locked H1 / title / advertised curl so the site cannot drift back to “helpers” without a deliberate change.

Wire `tests/try/run.sh` into CI (it exists and is not in `.github/workflows/ci.yml`).

## Recover relationship

`plans/260822-0140-docker-self-serve` owns engine lifecycle. This plan **does not replace it**.

This plan only requires:

- Stopped Desktop/Colima: recover starts that engine (already the v1 cut).
- `kind=none`: recover can apply `dc-try --yes`.

Empty-machine install stays that plan’s v2.

## Launch (after first-run + site)

Not a code phase. Do not launch another feature.

1. Text the four silent users the retry block from `plans/260815-first-run-curl-cli` (Phase 0 still pending).
2. Show HN + Product Hunt using updated `launch/` copy, weekday morning US time.
3. Ask for “did you get a shell?” not star counts.

## Workstreams (one program, four shippable slices)

| # | Slice | Ships | Depends |
|---|---|---|---|
| 1 | Site + locked branding + README + launch copy | Immediately | — |
| 2 | First-minute CLI/TUI/install + copy tests + CI try | After 1 or parallel | — |
| 3 | Recover `try_sandbox` apply + skill + guide order | After 2 | recover lib already present |
| 4 | Listen + launch | After 1–2 live on `main` | Pages deploy |

## Risks

| Risk | Mitigation |
|---|---|
| Copy ships, first-run still dies | Do not call the program done after the site pass |
| TTY `dc-up` → try surprises power users | Default **N**; only TTY; `--yes` does not try |
| Recover `--yes` starts a sandbox they did not want | Print the command first; `--yes` is the existing mutate door |
| “Helpers” leftover in OG/schema | One grep contract covers Hero, Layout, `copy.md`, README |

## Open questions (defaults if unanswered)

- Hero H1 is the PH/README line, not a new brand.
- No `/play` in the primary nav this pass.
- No new try clip tile (walkthrough already linked from README).
