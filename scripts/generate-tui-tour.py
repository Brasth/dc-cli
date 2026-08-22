#!/usr/bin/env python3
"""Render a chrome-free dc-tui board tour for the landing page."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
from dataclasses import dataclass, field
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "site" / "public" / "videos"
POSTERS = OUT / "posters"

BG = (7, 8, 6)
CORE = (28, 28, 28)
INK = (244, 241, 234)
MUTE = (138, 134, 128)
OK = (142, 207, 122)
BAD = (211, 107, 107)
AMBER = (215, 176, 108)
BTN = (126, 184, 168)
BTN_ON = (184, 255, 240)
META = (58, 58, 58)
META_ON = (92, 92, 92)
DANGER = (211, 107, 107)
LINE = (44, 42, 38)
INK_DARK = (26, 26, 26)

W, H = 960, 540
FPS = 16


def font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for path in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
    ):
        if Path(path).exists():
            return ImageFont.truetype(path, size=size)
    return ImageFont.load_default()


FONT = font(15)
FONT_SM = font(13)
FONT_XS = font(12)
FONT_LG = font(18)


@dataclass
class StackRow:
    name: str
    svc: str
    status: str
    img: str


@dataclass
class Board:
    view: str = "board"
    hover: str = ""
    status: str = ""
    leaving: str = ""
    confirm: str = ""
    running: bool = True
    folder: str = "running  ready"
    log_name: str = "app-1"
    log_follow: bool = True
    log_extra: list[str] = field(default_factory=list)
    nets_created: bool = False
    stack: list[StackRow] = field(default_factory=list)
    caption: str = ""


def initial_stack() -> list[StackRow]:
    return [
        StackRow("app-1", "app", "up", "node:22"),
        StackRow("db-1", "db", "up", "postgres:16"),
        StackRow("mitm-1", "mitm", "exited", "alpine/socat"),
    ]


def all_up() -> list[StackRow]:
    return [
        StackRow("app-1", "app", "up", "node:22"),
        StackRow("db-1", "db", "up", "postgres:16"),
        StackRow("mitm-1", "mitm", "up", "alpine/socat"),
    ]


def all_down() -> list[StackRow]:
    return [
        StackRow("app-1", "app", "exited", "node:22"),
        StackRow("db-1", "db", "exited", "postgres:16"),
        StackRow("mitm-1", "mitm", "exited", "alpine/socat"),
    ]


def no_mitm() -> list[StackRow]:
    return [
        StackRow("app-1", "app", "up", "node:22"),
        StackRow("db-1", "db", "up", "postgres:16"),
    ]


BUTTONS = [
    [("u", "start"), ("e", "shell"), ("s", "stop")],
    [("o", "open"), ("a", "attach"), ("p", "ports"), ("l", "logs"), ("t", "top"), ("n", "nets")],
    [("b", "db"), ("m", "files")],
    [("f", "fleet"), ("?", "more"), ("q", "quit"), ("x", "rm")],
    [("1", "1 :9001"), ("2", "2 :5173")],
]


def draw_btn(draw: ImageDraw.ImageDraw, xy: tuple[int, int], key: str, label: str, hover: str) -> int:
    x, y = xy
    on = hover == key
    if key == "x":
        fill, fg = DANGER, (255, 255, 255)
    elif key in {"u", "e", "s", "1", "2"}:
        fill, fg = (BTN_ON if on else BTN), INK_DARK
    else:
        fill, fg = (META_ON if on else META), INK
    pad_x = 10
    tw = draw.textlength(label, font=FONT_SM)
    w = int(tw) + pad_x * 2
    draw.rectangle((x, y, x + w, y + 24), fill=fill)
    draw.text((x + pad_x, y + 5), label, font=FONT_SM, fill=fg)
    return w + 6


def frame(state: Board) -> Image.Image:
    img = Image.new("RGB", (W, H), BG)
    draw = ImageDraw.Draw(img)
    m = 18
    draw.rounded_rectangle((m, m, W - m, H - m), radius=8, fill=CORE, outline=LINE, width=1)
    draw.rectangle((m, m, W - m, m + 36), fill=(22, 22, 20))
    for i, color in enumerate([(90, 88, 84), (90, 88, 84), OK]):
        x = m + 14 + i * 16
        draw.ellipse((x, m + 13, x + 10, m + 23), fill=color)
    draw.text((m + 70, m + 10), "dc-tui  —  ~/src/app", font=FONT_XS, fill=MUTE)
    if state.caption:
        draw.text((W - m - 8 - draw.textlength(state.caption, font=FONT_XS), m + 10), state.caption, font=FONT_XS, fill=AMBER)

    y = m + 52
    x = m + 20
    if state.view == "logs":
        draw_logs(draw, x, y, state)
    elif state.view == "top":
        draw_top(draw, x, y)
    elif state.view == "nets":
        draw_nets(draw, x, y, state)
    elif state.view == "more":
        draw_more(draw, x, y)
    else:
        draw_board(draw, x, y, state)
    return img


def draw_board(draw: ImageDraw.ImageDraw, x: int, y: int, state: Board) -> None:
    draw.text((x, y), "dc-cli", font=FONT_LG, fill=INK)
    draw.text((x + 72, y + 2), "app", font=FONT, fill=OK)
    draw.text((x, y + 24), "~/src/app", font=FONT_XS, fill=MUTE)
    draw.text((x, y + 42), "load  cpu 12.4%  mem 410M / —    t=top", font=FONT_XS, fill=MUTE)
    draw.text((x, y + 58), "disk  docker 42% · colima 61%    d=df", font=FONT_XS, fill=MUTE)
    nets = "shared-net ready" if state.nets_created else "missing shared-net"
    draw.text((x, y + 74), f"nets  {nets}    n=nets", font=FONT_XS, fill=MUTE)

    draw.text((x, y + 100), "this folder", font=FONT_XS, fill=MUTE)
    draw.text((x + 96, y + 100), "~/src/app", font=FONT_XS, fill=INK)
    draw.text((x, y + 118), "status", font=FONT_XS, fill=MUTE)
    draw.text((x + 96, y + 118), state.folder, font=FONT_XS, fill=OK if "running" in state.folder else AMBER)
    draw.text((x, y + 136), "editor", font=FONT_XS, fill=MUTE)
    draw.text((x + 96, y + 136), "zed", font=FONT_XS, fill=INK)

    by = y + 162
    for group in BUTTONS:
        bx = x
        for key, label in group:
            bx += draw_btn(draw, (bx, by), key, label, state.hover)
        by += 30

    msg_y = by + 4
    if state.leaving:
        draw.text((x, msg_y), state.leaving, font=FONT_SM, fill=AMBER)
        msg_y += 20
    if state.confirm == "rm":
        draw.text((x, msg_y), "remove stack containers? y/n", font=FONT_SM, fill=AMBER)
        msg_y += 20
    if state.status:
        draw.text((x, msg_y), state.status, font=FONT_SM, fill=OK)

    draw.text((x, H - 86), "stack    j/k · enter · e is app", font=FONT_XS, fill=MUTE)
    row_y = H - 68
    for i, row in enumerate(state.stack):
        if i == 0:
            draw.rectangle((x - 8, row_y - 2, W - 28, row_y + 16), fill=(40, 40, 38))
            draw.rectangle((x - 8, row_y - 2, x - 5, row_y + 16), fill=OK)
        st_color = OK if row.status == "up" else BAD
        draw.text((x, row_y), f"{row.name:<10} {row.svc:<6}", font=FONT_XS, fill=INK)
        draw.text((x + 148, row_y), f"{row.status:<8}", font=FONT_XS, fill=st_color)
        draw.text((x + 230, row_y), row.img, font=FONT_XS, fill=MUTE)
        row_y += 16


SAMPLE_LOGS = [
    "2026-08-22T05:00:01Z  GET   /health            200  12ms",
    "2026-08-22T05:00:04Z  GET   /api/projects      200  48ms",
    "2026-08-22T05:00:09Z  POST  /api/auth/login    200 112ms",
    "2026-08-22T05:00:11Z  GET   /api/workspaces    304   6ms",
    "2026-08-22T05:00:15Z  GET   /static/app.js     200   3ms",
    "2026-08-22T05:00:18Z  GET   /api/containers    200  31ms",
]


def draw_logs(draw: ImageDraw.ImageDraw, x: int, y: int, state: Board) -> None:
    draw.text((x, y), "dc-cli", font=FONT_LG, fill=INK)
    draw.text((x + 72, y + 2), "logs", font=FONT, fill=MUTE)
    draw.text((x + 122, y + 2), state.log_name, font=FONT, fill=OK)
    follow = "follow on" if state.log_follow else "follow off"
    draw.text((x, y + 28), f"q back · j/k scroll · f {follow}", font=FONT_XS, fill=MUTE)
    lines = SAMPLE_LOGS + state.log_extra
    ly = y + 58
    for line in lines[-6:]:
        color = (144, 196, 186) if "POST" in line else INK
        draw.text((x, ly), line, font=FONT_SM, fill=color)
        ly += 22


def draw_top(draw: ImageDraw.ImageDraw, x: int, y: int) -> None:
    draw.text((x, y), "dc-cli", font=FONT_LG, fill=INK)
    draw.text((x + 72, y + 2), "top", font=FONT, fill=MUTE)
    draw.text((x, y + 28), "q back · j/k select", font=FONT_XS, fill=MUTE)
    draw.text((x, y + 56), "SERVICE          CPU      MEM              NET", font=FONT_XS, fill=MUTE)
    rows = [
        ("app", "12.4%", "410M / 2G", "1.2M / 84K"),
        ("db", " 3.1%", "128M / 1G", "420K / 210K"),
        ("mitm", " 0.4%", "12M / 256M", "8K / 8K"),
    ]
    ly = y + 80
    for i, (svc, cpu, mem, net) in enumerate(rows):
        if i == 0:
            draw.rectangle((x - 8, ly - 2, W - 28, ly + 18), fill=(40, 40, 38))
        draw.text((x, ly), f"{svc:<16} {cpu:<8} {mem:<16} {net}", font=FONT_SM, fill=INK)
        ly += 24


def draw_nets(draw: ImageDraw.ImageDraw, x: int, y: int, state: Board) -> None:
    draw.text((x, y), "dc-cli", font=FONT_LG, fill=INK)
    draw.text((x + 72, y + 2), "nets", font=FONT, fill=MUTE)
    draw.text((x + 122, y + 2), "this folder", font=FONT, fill=OK)
    draw.text((x, y + 28), "q back · y create missing external bridge", font=FONT_XS, fill=MUTE)
    draw.text((x, y + 56), "NAME            KIND              STATE", font=FONT_XS, fill=MUTE)
    shared = "present" if state.nets_created else "missing"
    shared_c = OK if state.nets_created else AMBER
    draw.text((x, y + 80), "shared-net      external bridge", font=FONT_SM, fill=INK)
    draw.text((x + 360, y + 80), shared, font=FONT_SM, fill=shared_c)
    draw.text((x, y + 104), "default         compose          present", font=FONT_SM, fill=INK)
    if not state.nets_created:
        draw.text((x, y + 140), "create missing external nets and start? y/n", font=FONT_SM, fill=AMBER)
    elif state.status:
        draw.text((x, y + 140), state.status, font=FONT_SM, fill=OK)


def draw_more(draw: ImageDraw.ImageDraw, x: int, y: int) -> None:
    draw.text((x, y), "dc-cli", font=FONT_LG, fill=INK)
    draw.text((x + 72, y + 2), "more", font=FONT, fill=MUTE)
    lines = [
        (INK, "more — what each action does"),
        (MUTE, "  start   .devcontainer/compose → dc-up"),
        (MUTE, "  shell   bash in the labeled app"),
        (MUTE, "  logs    follow docker logs · q returns"),
        (MUTE, "  top     CPU / RAM for this folder (t)"),
        (MUTE, "  nets    declared compose nets (n) · y creates"),
        (MUTE, "  open    host editor on the bind-mount"),
        (MUTE, "  attach  VS Code Remote URI"),
        (MUTE, "  rm      compose down — asks y/n"),
        (MUTE, "  ? or q  return to the board"),
    ]
    ly = y + 36
    for color, line in lines:
        draw.text((x, ly), line, font=FONT_SM, fill=color)
        ly += 20


def hold(state: Board, frames: int) -> list[Image.Image]:
    img = frame(state)
    return [img] * frames


def encode(frames: list[Image.Image], mp4: Path, poster: Path) -> None:
    tmp = Path(tempfile.mkdtemp(prefix="dc-tui-"))
    try:
        for i, fr in enumerate(frames):
            fr.save(tmp / f"f{i:05d}.png")
        poster_frame = frames[min(len(frames) - 1, 48)]
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
                "26",
                str(mp4),
            ],
            check=True,
            capture_output=True,
        )
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def tour() -> list[Image.Image]:
    frames: list[Image.Image] = []
    stack = initial_stack()

    def add(st: Board, n: int) -> None:
        frames.extend(hold(st, n))

    add(Board(stack=stack, caption="board"), 20)
    add(Board(stack=stack, hover="u", caption="u  start"), 8)
    add(
        Board(
            stack=stack,
            hover="u",
            leaving="leaving to start — board returns when it finishes",
            folder="starting…",
            caption="u  start",
        ),
        12,
    )
    stack = all_up()
    add(
        Board(
            stack=stack,
            status="dc up — kind=devcontainer · started · ports forwarded",
            folder="running  ready",
            caption="u  start",
        ),
        16,
    )
    add(Board(stack=stack, hover="e", caption="e  shell"), 8)
    add(
        Board(
            stack=stack,
            hover="e",
            leaving="leaving to shell — exit to return",
            caption="e  shell",
        ),
        12,
    )
    add(Board(stack=stack, status="shell exited — board resumed", caption="e  shell"), 12)

    add(Board(stack=stack, hover="l", caption="l  logs"), 8)
    add(Board(view="logs", stack=stack, caption="l  logs"), 14)
    add(
        Board(
            view="logs",
            stack=stack,
            log_extra=["2026-08-22T05:00:22Z  GET   /api/health         200   8ms"],
            caption="f  follow",
        ),
        12,
    )
    add(Board(stack=stack, hover="t", caption="t  top"), 8)
    add(Board(view="top", stack=stack, caption="t  top"), 18)
    add(Board(stack=stack, hover="n", caption="n  nets"), 8)
    add(Board(view="nets", stack=stack, caption="n  nets"), 12)
    add(
        Board(
            view="nets",
            stack=stack,
            nets_created=True,
            status="created bridge shared-net — run start to attach",
            caption="y  create",
        ),
        14,
    )
    add(Board(stack=stack, nets_created=True, hover="?", caption="?  more"), 8)
    add(Board(view="more", stack=stack, nets_created=True, caption="?  more"), 18)

    flashes = [
        ("o", "open", "dc-open — host editor on bind-mount"),
        ("a", "attach", "dc-open --attach — VS Code Remote URI printed"),
        ("p", "ports", "dc-forward — reconciled owned sidecars"),
        ("b", "db", "dc-db — TablePlus on declared db port"),
        ("m", "files", "leaving to files — quit the manager to return"),
        ("f", "fleet", "fleet view needs install — try dc --all"),
        ("1", "url 1", "open http://127.0.0.1:9001 in host browser"),
    ]
    for key, label, status in flashes:
        leaving = status if key == "m" else ""
        stat = "" if key == "m" else status
        add(
            Board(
                stack=stack,
                nets_created=True,
                hover=key,
                leaving=leaving,
                status=stat,
                caption=f"{key}  {label}",
            ),
            11,
        )

    add(Board(stack=stack, nets_created=True, hover="x", confirm="rm", caption="x  rm"), 10)
    add(Board(stack=stack, nets_created=True, status="rm cancelled", caption="n  cancel"), 10)
    add(Board(stack=stack, nets_created=True, hover="s", caption="s  stop"), 8)
    add(
        Board(
            stack=stack,
            nets_created=True,
            hover="s",
            folder="stopping…",
            status="dc down — stopping compose stack…",
            caption="s  stop",
        ),
        10,
    )
    stack = all_down()
    add(
        Board(
            stack=stack,
            nets_created=True,
            running=False,
            folder="stopped",
            status="dc down — stack stopped",
            caption="s  stop",
        ),
        12,
    )
    add(Board(stack=stack, nets_created=True, hover="u", folder="starting…", caption="u  start"), 8)
    stack = all_up()
    add(
        Board(
            stack=stack,
            nets_created=True,
            status="dc up — kind=devcontainer · started · ports forwarded",
            caption="u  start",
        ),
        12,
    )
    add(Board(stack=stack, nets_created=True, hover="x", confirm="rm", caption="x  rm"), 10)
    stack = no_mitm()
    add(
        Board(
            stack=stack,
            nets_created=True,
            status="dc down --rm — mitm removed",
            caption="y  confirm",
        ),
        16,
    )
    return frames


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    POSTERS.mkdir(parents=True, exist_ok=True)
    frames = tour()
    encode(frames, OUT / "tui-tour.mp4", POSTERS / "tui-tour.webp")
    print(f"wrote tui-tour.mp4 ({len(frames)} frames, {len(frames)/FPS:.1f}s)")


if __name__ == "__main__":
    main()
