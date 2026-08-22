#!/usr/bin/env python3
"""Stitch walkthrough clips into one full launch demo MP4."""

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "site" / "public" / "videos"
POSTERS = OUT / "posters"
DOCS_ASSETS = ROOT / "docs" / "assets"

OUTPUT_NAME = "launch-full-demo.mp4"
POSTER_NAME = "launch-full-demo.webp"

# Product tour order (matches launch/asset-checklist.md)
WALKTHROUGH_ORDER = [
    "walkthrough-intro.mp4",
    "walkthrough-board.mp4",
    "walkthrough-daily.mp4",
    "walkthrough-recover.mp4",
    "walkthrough-ports.mp4",
    "walkthrough-disk.mp4",
    "walkthrough-try.mp4",
    "walkthrough-power.mp4",
]


def stitch_launch_demo(
    *,
    source_dir: Path = OUT,
    output_dir: Path = OUT,
    docs_dir: Path = DOCS_ASSETS,
) -> Path:
    missing = [name for name in WALKTHROUGH_ORDER if not (source_dir / name).is_file()]
    if missing:
        msg = "missing walkthrough files (run generate-demo-videos.py first): " + ", ".join(missing)
        raise FileNotFoundError(msg)

    output_dir.mkdir(parents=True, exist_ok=True)
    POSTERS.mkdir(parents=True, exist_ok=True)
    docs_dir.mkdir(parents=True, exist_ok=True)

    out_mp4 = output_dir / OUTPUT_NAME
    out_poster = POSTERS / POSTER_NAME

    with tempfile.TemporaryDirectory(prefix="dc-stitch-") as tmp:
        list_path = Path(tmp) / "concat.txt"
        list_path.write_text(
            "\n".join(f"file '{(source_dir / name).resolve()}'" for name in WALKTHROUGH_ORDER) + "\n",
            encoding="utf-8",
        )
        subprocess.run(
            [
                "ffmpeg",
                "-y",
                "-f",
                "concat",
                "-safe",
                "0",
                "-i",
                str(list_path),
                "-c",
                "copy",
                str(out_mp4),
            ],
            check=True,
            capture_output=True,
        )

    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-i",
            str(out_mp4),
            "-vf",
            "select=eq(n\\,0)",
            "-frames:v",
            "1",
            str(out_poster.with_suffix(".png")),
        ],
        check=True,
        capture_output=True,
    )
    from PIL import Image

    Image.open(out_poster.with_suffix(".png")).save(out_poster, "WEBP", quality=86)
    out_poster.with_suffix(".png").unlink(missing_ok=True)

    shutil.copy2(out_mp4, docs_dir / OUTPUT_NAME)

    probe = subprocess.run(
        [
            "ffprobe",
            "-v",
            "error",
            "-show_entries",
            "format=duration",
            "-of",
            "default=noprint_wrappers=1:nokey=1",
            str(out_mp4),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    duration = float(probe.stdout.strip())
    print(f"wrote {OUTPUT_NAME} ({duration:.1f}s, {len(WALKTHROUGH_ORDER)} clips)")
    return out_mp4


def main() -> int:
    try:
        stitch_launch_demo()
    except (FileNotFoundError, subprocess.CalledProcessError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
