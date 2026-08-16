#!/usr/bin/env bash
# Cache a Linux yazi binary on the host, then dc-files docker-cps it into /tmp.
# Never apt-get. Never edit the image. Sourced by install.sh and dc-files.

DC_YAZI_VERSION="${DC_YAZI_VERSION:-26.8.15}"
DC_GUEST_YAZI_PATH="${DC_GUEST_YAZI_PATH:-/tmp/dc-cli-yazi}"

dc_yazi_home() {
  printf '%s\n' "${DC_YAZI_HOME:-${XDG_DATA_HOME:-$HOME/.local/share}/dc-cli/tools}"
}

dc_yazi_norm_arch() {
  case "${1:-}" in
    arm64|aarch64) printf 'arm64\n' ;;
    amd64|x86_64) printf 'amd64\n' ;;
    *) return 1 ;;
  esac
}

dc_yazi_guest_stem() {
  local arch
  arch="$(dc_yazi_norm_arch "$1")" || return 1
  case "$arch" in
    arm64) printf 'yazi-aarch64-unknown-linux-musl\n' ;;
    amd64) printf 'yazi-x86_64-unknown-linux-musl\n' ;;
  esac
}

dc_yazi_guest_home() {
  local arch
  arch="$(dc_yazi_norm_arch "$1")" || return 1
  printf '%s/guest/%s\n' "$(dc_yazi_home)" "$arch"
}

dc_yazi_unzip() {
  local zip="$1" dest="$2"
  mkdir -p "$dest"
  if command -v unzip >/dev/null 2>&1; then
    unzip -q -o "$zip" -d "$dest"
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$zip" "$dest" <<'PY'
import sys, zipfile
zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])
PY
    return
  fi
  echo "dc-cli: need unzip or python3 to unpack yazi" >&2
  return 1
}

dc_install_guest_yazi() {
  local arch dest stem url tmp zip found
  arch="$(dc_yazi_norm_arch "${1:-}")" || {
    echo "dc-cli: unknown container arch ${1:-}" >&2
    return 1
  }
  dest="$(dc_yazi_guest_home "$arch")"
  mkdir -p "$dest"
  if [[ -x "$dest/yazi" && -f "$dest/VERSION" && "$(tr -d '[:space:]' <"$dest/VERSION")" == "$DC_YAZI_VERSION" ]]; then
    printf '%s\n' "$dest/yazi"
    return 0
  fi
  stem="$(dc_yazi_guest_stem "$arch")" || return 1
  url="https://github.com/sxyazi/yazi/releases/download/v${DC_YAZI_VERSION}/${stem}.zip"
  tmp="$(mktemp -d)"
  zip="$tmp/yazi.zip"
  if [[ -n "${DC_YAZI_ZIP:-}" ]]; then
    cp "$DC_YAZI_ZIP" "$zip" || { rm -rf "$tmp"; return 1; }
  else
    echo "Fetching Linux yazi $DC_YAZI_VERSION ($stem) ..." >&2
    if ! curl -fsSL "$url" -o "$zip"; then
      rm -rf "$tmp"
      echo "dc-cli: could not download yazi from $url" >&2
      return 1
    fi
  fi
  if ! dc_yazi_unzip "$zip" "$tmp/out"; then
    rm -rf "$tmp"
    return 1
  fi
  found="$(find "$tmp/out" -type f -name yazi | head -1)"
  if [[ -z "$found" ]]; then
    echo "dc-cli: yazi binary missing from zip" >&2
    rm -rf "$tmp"
    return 1
  fi
  cp "$found" "$dest/yazi"
  chmod +x "$dest/yazi"
  printf '%s\n' "$DC_YAZI_VERSION" >"$dest/VERSION"
  rm -rf "$tmp"
  echo "Cached guest yazi $DC_YAZI_VERSION ($arch) -> $dest/yazi" >&2
  printf '%s\n' "$dest/yazi"
}

dc_container_arch() {
  local id="$1" raw
  raw="$(docker inspect -f '{{.Architecture}}' "$id" 2>/dev/null || true)"
  dc_yazi_norm_arch "$raw"
}

dc_inject_guest_yazi() {
  local id="$1" arch bin
  [[ -n "$id" ]] || return 1
  arch="$(dc_container_arch "$id")" || return 1
  bin="$(dc_install_guest_yazi "$arch")" || return 1
  if ! docker cp "$bin" "$id:${DC_GUEST_YAZI_PATH}"; then
    echo "dc-files: could not copy yazi into $id" >&2
    return 1
  fi
  docker exec "$id" chmod +x "$DC_GUEST_YAZI_PATH" >/dev/null 2>&1 || true
  printf '%s\n' "$DC_GUEST_YAZI_PATH"
}

ensure_guest_file_manager() {
  local arch
  if [[ "${DC_SKIP_YAZI:-}" == "1" || "${SKIP_YAZI:-0}" == "1" ]]; then
    echo "guest yazi: skipped (DC_SKIP_YAZI / --no-yazi)"
    return 0
  fi
  case "$(uname -m)" in
    arm64|aarch64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *)
      echo "guest yazi: skip prefetch (unknown host arch)"
      return 0
      ;;
  esac
  if dc_install_guest_yazi "$arch" >/dev/null; then
    echo "guest yazi ready for linux/$arch (dc-files copies it into /tmp)"
  else
    echo "guest yazi prefetch failed (non-fatal). dc-files will retry on first use."
  fi
}
