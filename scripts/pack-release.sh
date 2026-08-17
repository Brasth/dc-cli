#!/usr/bin/env bash
# Pack bash kit + prebuilt dc-tui for one GOOS/GOARCH.
# Usage: scripts/pack-release.sh VERSION GOOS GOARCH [OUT_DIR]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:?version (0.5.3 or v0.5.3)}"
GOOS="${2:?goos}"
GOARCH="${3:?goarch}"
OUT="${4:-$ROOT/dist}"

VERSION="${VERSION#v}"
case "$GOOS" in darwin|linux) ;; *) echo "unsupported GOOS: $GOOS" >&2; exit 2 ;; esac
case "$GOARCH" in amd64|arm64) ;; *) echo "unsupported GOARCH: $GOARCH" >&2; exit 2 ;; esac

name="dc-cli-${VERSION}-${GOOS}-${GOARCH}"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
dest="$stage/$name"
mkdir -p "$dest/bin" "$dest/lib" "$dest/config" "$dest/skill"

for f in dc-up dc-exec dc-down dc-ps dc-forward dc-ls dc-open dc-df dc-prune dc-doctor dc-engine dc-db dc-files dc-stats dc-net; do
  if [[ -f "$ROOT/bin/$f" ]]; then
    cp "$ROOT/bin/$f" "$dest/bin/$f"
    chmod +x "$dest/bin/$f"
  fi
done

printf '%s\n' "$VERSION" >"$dest/VERSION"
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o "$dest/bin/dc-tui" "$ROOT/cmd/dc-tui"
chmod +x "$dest/bin/dc-tui"

if [[ "$(head -c 2 "$dest/bin/dc-tui")" == "#!" ]]; then
  echo "dc-tui looks like a script, not a binary" >&2
  exit 1
fi

cp "$ROOT/lib/"*.sh "$dest/lib/"
cp "$ROOT/config/override.json" "$dest/config/override.json"
cp "$ROOT/skill/SKILL.md" "$dest/skill/SKILL.md"
cp "$ROOT/install.sh" "$dest/install.sh"

mkdir -p "$OUT"
tar -C "$stage" -czf "$OUT/${name}.tar.gz" "$name"
echo "$OUT/${name}.tar.gz"
