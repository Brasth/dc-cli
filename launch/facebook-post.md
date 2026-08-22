# Facebook fan page — launch kit

## Cover photo

- Size: 820×312 px
- Source: `site/public/og.png` cropped or extended with “dc-cli · terminal dev containers”
- Upload as Page cover in Meta Business Suite

## Profile picture

- Source: `launch/assets/logo-mark.png` (240×240)

---

## Launch post (pin this)

**Text:**

```
Tired of digging through VS Code Remote just to start your dev container?

dc-cli gives you the missing verbs from the terminal:

▸ dc up — start this folder
▸ dc exec — shell without raw docker exec
▸ dc doctor — read-only diagnose
▸ TUI board for ports, logs, disk

Works with the official Dev Containers CLI. Never edits your .devcontainer.
Free & open source (MIT).
```

**Video:** Upload `site/public/videos/walkthrough-intro.mp4` natively (FB favors native video)

**First comment (links):**
```
Install: https://dc.brasth.com
GitHub: https://github.com/Brasth/dc-cli

curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
```

**Format notes:**
- Add captions (many watch muted)
- Pin this post after publishing

---

## Follow-up post — board demo

**Text:**

```
The dc-cli board is the product.

Type dc in any project folder:
— Start and stop with one key
— Open forwarded ports (1–9)
— Logs, top, fleet view

No VS Code required.

https://dc.brasth.com
```

**Video:** `site/public/videos/walkthrough-board.mp4`

---

## Follow-up post — stuck Docker

**Text:**

```
Docker blocking your work?

dc doctor diagnoses read-only.
dc recover gives one next step — start Colima, fix split-brain, safe prune.

One command. Then back to coding.

https://dc.brasth.com/guide/stuck/
```

**Video:** `site/public/videos/walkthrough-recover.mp4`

---

## Story / Reel idea (15–30s)

1. Screen record or use `walkthrough-intro.mp4`
2. Text overlay: “dev containers from your terminal”
3. End card: dc.brasth.com + install one-liner
