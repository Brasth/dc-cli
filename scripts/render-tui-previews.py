#!/usr/bin/env python3
"""Render exported Go TUI fixtures into documentation images.

DC_TUI_PREVIEW_DIR=/tmp/dc-previews go test ./cmd/dc-tui -run TestConsolePreviewFixtures -count=1
python scripts/render-tui-previews.py /tmp/dc-previews/fixtures.json

Requires Pillow. The Go renderer supplies all text, geometry and colors;
this script only rasterizes its SGR output. No Docker commands are executed.
"""
from __future__ import annotations

import argparse
import json
import re
import unicodedata
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
BG = (5, 5, 5)
FG = (244, 241, 234)
SGR = re.compile(r"\x1b\[([0-9;]*)m")
CELL_W, CELL_H, PAD = 10, 20, 24
FONT_PATH = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
FONT = ImageFont.truetype(FONT_PATH, 16)
BOLD = ImageFont.truetype(FONT_PATH.replace(".ttf", "-Bold.ttf"), 16)


def render(fixture: dict) -> Image.Image:
    image = Image.new("RGB", (fixture["Width"] * CELL_W + 2 * PAD,
                             fixture["Height"] * CELL_H + 2 * PAD), BG)
    draw = ImageDraw.Draw(image)
    for row, line in enumerate(fixture["View"].split("\n")):
        x, pos, foreground, background, bold = PAD, 0, FG, BG, False
        for match in list(SGR.finditer(line)) + [None]:
            end = match.start() if match else len(line)
            for char in line[pos:end]:
                cells = 0 if unicodedata.combining(char) else (2 if unicodedata.east_asian_width(char) in "WF" else 1)
                y = PAD + row * CELL_H
                if cells:
                    draw.rectangle((x, y, x + cells * CELL_W - 1, y + CELL_H - 1), fill=background)
                draw.text((x, y), char, fill=foreground, font=BOLD if bold else FONT)
                x += cells * CELL_W
            if match is None:
                break
            codes = [int(c or 0) for c in match.group(1).split(";")]
            i = 0
            while i < len(codes):
                code = codes[i]
                if code == 0:
                    foreground, background, bold = FG, BG, False
                elif code == 1:
                    bold = True
                elif code == 22:
                    bold = False
                elif code == 39:
                    foreground = FG
                elif code == 49:
                    background = BG
                elif code in (38, 48) and i + 4 < len(codes) and codes[i + 1] == 2:
                    color = tuple(codes[i + 2:i + 5])
                    if code == 38:
                        foreground = color
                    else:
                        background = color
                    i += 4
                i += 1
            pos = match.end()
    return image


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("fixtures", type=Path)
    parser.add_argument("--output", type=Path, default=ROOT / "docs/assets/tui")
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    fixtures = json.loads(args.fixtures.read_text())
    boards = []
    for fixture in fixtures:
        image = render(fixture)
        image.save(args.output / (fixture["Name"] + ".png"))
        if fixture["Name"].startswith("board-"):
            boards.append((fixture["Name"], image))
        if fixture["Name"] == "board-120x36":
            image.save(ROOT / "site/public/images/tui.png")
        if fixture["Name"] == "top":
            image.save(ROOT / "site/public/images/tui-top.png")
    width = max(image.width for _, image in boards)
    height = sum(image.height + 48 for _, image in boards)
    sheet = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(sheet)
    y = 0
    for name, image in boards:
        draw.text((PAD, y + 8), name, fill=(111, 207, 123), font=BOLD)
        sheet.paste(image, (0, y + 40))
        y += image.height + 48
    sheet.save(args.output / "responsive.png")
    print(f"Rendered {len(fixtures)} fixtures to {args.output}")


if __name__ == "__main__":
    main()
