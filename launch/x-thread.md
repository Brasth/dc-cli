# X (Twitter) — launch kit

## Pin tweet

```
dc-cli — dev containers from your terminal.

dc up · dc exec · dc doctor · TUI board

No VS Code. Never touches your .devcontainer.

https://dc.brasth.com
```

Attach: `docs/assets/walkthrough-intro.gif` or `site/public/videos/walkthrough-intro.mp4`

---

## Launch thread (7 tweets)

### 1/7 — Hook

```
I kept bouncing between VS Code Remote, raw docker exec, and Colima port hacks.

So I built dc-cli — dev containers from the terminal.

30s demo 👇
```

Attach GIF: `docs/assets/walkthrough-intro.gif`

### 2/7 — Problem

```
Without dc-cli:

— Dig through VS Code Remote UI
— Remember raw docker exec container names
— Hand-roll Colima port sidecars
— Guess which engine owns this folder
```

### 3/7 — Solution

```
dc-cli adds host-global verbs around the *official* Dev Containers CLI:

dc up      start this folder
dc exec    shell, never raw docker exec
dc doctor  read-only diagnose

Never edits your .devcontainer.
```

### 4/7 — Doctor demo

```
dc doctor is read-only — reports engine, socket, split-brain.

When something blocks you, dc recover prints one next step. --yes applies it.
```

Attach: `site/public/videos/walkthrough-recover.mp4` or poster

### 5/7 — Board

```
The board (dc / dc-tui) is the product:

— start / stop / shell
— logs, top, ports (keys 1–9)
— fleet view with dc --all
```

Attach: `site/public/videos/walkthrough-board.mp4`

### 6/7 — Trust

```
Works with @devcontainers/cli — not a fork.

MIT. macOS & Linux. Homebrew:

brew tap Brasth/dc-cli && brew install dc-cli
```

### 7/7 — CTA

```
Install:

curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli

Site: https://dc.brasth.com
GitHub: https://github.com/Brasth/dc-cli

RT if you're terminal-first 🙏
```

---

## Week 1 follow-ups

**Monday — ports**
```
Colima port pain? dc forward publishes Colima-safe sidecars.

dc up → dc forward → open http://127.0.0.1:3000 from the board.

https://dc.brasth.com/guide/ports/
```

**Wednesday — disk**
```
dc df shows where disk went.

dc prune --yes reclaims safely.

Never run docker system prune -af --volumes.

https://dc.brasth.com/guide/disk/
```

**Friday — agents**
```
dc-cli --with-skill copies into Cursor, Claude, Codex agent homes.

Your agent gets the same verbs you use in the terminal.

https://dc.brasth.com/guide/install/
```

## Hashtags (use sparingly)

`#devcontainers` `#docker` `#cli` `#opensource`
