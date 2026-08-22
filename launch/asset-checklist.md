# Launch asset checklist

All paths relative to repo root unless noted.

## Videos (site + docs)

| File | Size | Use |
|------|------|-----|
| `site/public/videos/walkthrough-intro.mp4` | 1280×720 | PH gallery #1, README hero, X/FB launch |
| `site/public/videos/walkthrough-board.mp4` | 1280×720 | PH gallery #2, board posts |
| `site/public/videos/walkthrough-recover.mp4` | 1280×720 | PH gallery #3, stuck-Docker posts |
| `site/public/videos/walkthrough-daily.mp4` | 1280×720 | README / guides |
| `site/public/videos/walkthrough-ports.mp4` | 1280×720 | Week 1 X post |
| `site/public/videos/walkthrough-disk.mp4` | 1280×720 | Week 1 X post |
| `site/public/videos/walkthrough-try.mp4` | 1280×720 | Sandbox demo |
| `site/public/videos/walkthrough-power.mp4` | 1280×720 | Power verbs demo |
| `site/public/videos/hero.mp4` | 960×540 | Site hero panel |
| `site/public/videos/clip-*.mp4` | 1280×720 | Site clip strip |

## GIFs (social / README)

| File | Use |
|------|-----|
| `docs/assets/walkthrough-intro.gif` | X thread hook, README fallback |
| `docs/assets/clip-up.gif` | README “See it run” |
| `docs/assets/clip-exec.gif` | README “See it run” |
| `docs/assets/clip-doctor.gif` | README “See it run” |

## Images

| File | Size | Use |
|------|------|-----|
| `site/public/og.png` | 1200×630 | PH gallery #4, FB cover base, link previews |
| `launch/assets/logo-mark.png` | 240×240 | PH logo, FB profile |
| `site/public/og.svg` | vector | Source for og.png |
| `assets/branding/logo-mark.svg` | vector | Source for logo PNG |

## Posters (site lazy-load)

| File | Use |
|------|-----|
| `site/public/videos/posters/*.webp` | Video poster frames |

## Regenerate assets

```bash
# Synthetic videos (no Docker required)
python3 scripts/generate-demo-videos.py

# Real terminal recordings (requires VHS + Docker + dc-cli)
cd scripts/demo-tapes && vhs walkthrough-intro.tape

# Export og.png from SVG (requires cairosvg or rsvg-convert)
python3 -c "import cairosvg; cairosvg.svg2png(url='site/public/og.svg', write_to='site/public/og.png', output_width=1200, output_height=630)"
```

## Hosted URLs (after deploy)

Replace local paths with:

- `https://dc.brasth.com/videos/walkthrough-intro.mp4`
- `https://dc.brasth.com/videos/clip-up.mp4`
- `https://dc.brasth.com/og.png`
