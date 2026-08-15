#!/usr/bin/env bash
# Print Homebrew sha256 lines for a tag's four tarballs.
# Usage: scripts/print-release-shas.sh v0.5.3
#    or: scripts/print-release-shas.sh /path/to/dist
set -euo pipefail

src="${1:?tag (v0.5.3) or directory of tarballs}"
if [[ -d "$src" ]]; then
  for f in "$src"/dc-cli-*.tar.gz; do
    [[ -f "$f" ]] || continue
    printf '%s  %s\n' "$(shasum -a 256 "$f" | awk '{print $1}')" "$(basename "$f")"
  done
  exit 0
fi

ver="${src#v}"
base="https://github.com/Brasth/dc-cli/releases/download/v${ver}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
for pair in darwin-arm64 darwin-amd64 linux-amd64 linux-arm64; do
  name="dc-cli-${ver}-${pair}.tar.gz"
  curl -fsSL "${base}/${name}" -o "$tmp/$name"
  printf '%s  %s\n' "$(shasum -a 256 "$tmp/$name" | awk '{print $1}')" "$name"
done
