#!/usr/bin/env python3
"""Render terminal-only product demo videos (no editor/Cursor chrome)."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "site" / "public" / "videos"
POSTERS = OUT / "posters"

BG = (7, 8, 6)
CORE = (12, 12, 10)
INK = (244, 241, 234)
MUTE = (138, 134, 128)
AMBER = (111, 207, 123)
LINE = (44, 42, 38)

W, H = 1280, 720
HERO_W, HERO_H = 960, 540
FPS = 24


def font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for path in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
        "/System/Library/Fonts/Menlo.ttc",
    ):
        if Path(path).exists():
            return ImageFont.truetype(path, size=size)
    return ImageFont.load_default()


FONT = font(28)
FONT_SM = font(22)
FONT_XS = font(18)
HERO_FONT = font(24)
HERO_FONT_XS = font(15)


def base_frame(margin: int = 48, width: int = W, height: int = H) -> Image.Image:
    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)
    draw.rounded_rectangle(
        (margin, margin, width - margin, height - margin),
        radius=12,
        fill=CORE,
        outline=LINE,
        width=2,
    )
    y = margin + 22
    for i, color in enumerate([(90, 88, 84), (90, 88, 84), AMBER]):
        x = margin + 28 + i * 22
        draw.ellipse((x, y - 6, x + 12, y + 6), fill=color)
    draw.text((margin + 110, y - 10), "dc-cli — ~/src/app", font=FONT_XS, fill=MUTE)
    draw.rectangle((margin + 2, margin + 44, width - margin - 2, height - margin - 2), fill=CORE)
    return img


def hero_frame() -> Image.Image:
    """Tight terminal fill — no outer letterboxing for the hero panel."""
    img = Image.new("RGB", (HERO_W, HERO_H), CORE)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, HERO_W - 1, HERO_H - 1), outline=LINE, width=1)
    draw.rectangle((0, 0, HERO_W, 40), fill=(16, 16, 14))
    for i, color in enumerate([(90, 88, 84), (90, 88, 84), AMBER]):
        x = 16 + i * 18
        draw.ellipse((x, 14, x + 10, 24), fill=color)
    draw.text((70, 11), "dc-cli — ~/src/app", font=HERO_FONT_XS, fill=MUTE)
    return img


def draw_lines(
    img: Image.Image,
    lines: list[tuple[str, tuple[int, int, int]]],
    typed: int | None = None,
    *,
    hero: bool = False,
) -> None:
    draw = ImageDraw.Draw(img)
    x0 = 28 if hero else 80
    y0 = 64 if hero else 140
    line_h = 34 if hero else 40
    fnt = HERO_FONT if hero else FONT
    shown = 0
    for text, color in lines:
        if typed is not None and shown >= typed:
            break
        draw.text((x0, y0), text, font=fnt, fill=color)
        y0 += line_h
        shown += 1
    if typed is not None and typed > 0 and typed <= len(lines):
        last = lines[typed - 1][0]
        bbox = draw.textbbox((x0, 0), last, font=fnt)
        cx = x0 + (bbox[2] - bbox[0]) + 4
        cy = (64 if hero else 140) + (typed - 1) * line_h
        if int(typed * 2) % 2 == 0:
            draw.rectangle((cx, cy + 4, cx + 12, cy + 26), fill=AMBER)


def typewriter_frames(
    lines: list[tuple[str, tuple[int, int, int]]],
    hold: int = 18,
    *,
    hero: bool = False,
) -> list[Image.Image]:
    frames: list[Image.Image] = []
    built: list[tuple[str, tuple[int, int, int]]] = []
    mk = hero_frame if hero else base_frame
    x0 = 28 if hero else 80
    y_base = 64 if hero else 140
    line_h = 34 if hero else 40
    fnt = HERO_FONT if hero else FONT

    for text, color in lines:
        if text.startswith("$ "):
            prefix, rest = "$ ", text[2:]
            for i in range(len(rest) + 1):
                partial = built + [(prefix + rest[:i], color)]
                img = mk()
                draw_lines(img, partial, hero=hero)
                draw = ImageDraw.Draw(img)
                y0 = y_base + len(built) * line_h
                bbox = draw.textbbox((x0, 0), prefix + rest[:i], font=fnt)
                cx = x0 + (bbox[2] - bbox[0]) + 4
                draw.rectangle((cx, y0 + 4, cx + 12, y0 + 26), fill=AMBER)
                frames.append(img)
            built.append((text, color))
            for _ in range(6):
                img = mk()
                draw_lines(img, built, hero=hero)
                frames.append(img)
        else:
            built.append((text, color))
            for _ in range(8):
                img = mk()
                draw_lines(img, built, hero=hero)
                frames.append(img)
    for _ in range(hold):
        img = mk()
        draw_lines(img, built, hero=hero)
        frames.append(img)
    return frames


def encode(frames: list[Image.Image], mp4: Path, poster: Path) -> None:
    tmp = Path(tempfile.mkdtemp(prefix="dcvid-"))
    try:
        for i, frame in enumerate(frames):
            frame.save(tmp / f"f{i:05d}.png")
        poster_frame = frames[min(len(frames) - 1, max(0, len(frames) // 2))]
        poster_frame.save(poster, "WEBP", quality=86)
        subprocess.run(
            [
                "ffmpeg",
                "-y",
                "-framerate",
                str(FPS),
                "-i",
                str(tmp / "f%05d.png"),
                "-c:v",
                "libx264",
                "-pix_fmt",
                "yuv420p",
                "-movflags",
                "+faststart",
                "-crf",
                "23",
                str(mp4),
            ],
            check=True,
            capture_output=True,
        )
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    POSTERS.mkdir(parents=True, exist_ok=True)

    hero_lines = [
        ("$ dc", INK),
        ("  up      start this folder", MUTE),
        ("  exec    shell into the app", MUTE),
        ("  doctor  read-only diagnose", MUTE),
        ("$ dc up", INK),
        ("→ kind=devcontainer", AMBER),
        ("→ started  ports forwarded", AMBER),
        ("$ dc", INK),
    ]
    up_lines = [
        ("$ dc up", INK),
        ("→ kind=compose", MUTE),
        ("→ docker compose up -d", MUTE),
        ("→ app  running", AMBER),
        ("→ db   running", AMBER),
    ]
    exec_lines = [
        ("$ dc exec", INK),
        ("→ never docker exec NAME", MUTE),
        ("→ compose exec app /bin/sh", MUTE),
        ("root@app:/workspaces/app#", AMBER),
    ]
    doctor_lines = [
        ("$ dc doctor", INK),
        ("engine     colima", MUTE),
        ("socket     ~/.colima/.../docker.sock", MUTE),
        ("extraLive  []", MUTE),
        ("exit  0", AMBER),
    ]

    jobs = [
        ("hero.mp4", "hero.webp", hero_lines, 24, True),
        ("clip-up.mp4", "clip-up.webp", up_lines, 20, False),
        ("clip-exec.mp4", "clip-exec.webp", exec_lines, 20, False),
        ("clip-doctor.mp4", "clip-doctor.webp", doctor_lines, 20, False),
    ]
    for mp4_name, poster_name, lines, hold, is_hero in jobs:
        frames = typewriter_frames(lines, hold=hold, hero=is_hero)
        if mp4_name == "hero.mp4":
            frames = frames + frames[-12:] * 2
        encode(frames, OUT / mp4_name, POSTERS / poster_name)
        print(f"wrote {mp4_name} ({len(frames)} frames)")


if __name__ == "__main__":
    main()
