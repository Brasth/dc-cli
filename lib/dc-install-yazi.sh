#!/usr/bin/env bash
# Host-only yazi bootstrap for dc-files. Never installs into a guest image.
# Sourced by install.sh. DC_SKIP_YAZI=1 / --no-yazi skips. Fail-soft.

DC_YAZI_VERSION="${DC_YAZI_VERSION:-26.8.15}"
DC_HOST_FM_NAMES=(yazi nnn lf mc ranger)

dc_host_fm_on_path() {
  local t
  for t in "${DC_HOST_FM_NAMES[@]}"; do
    if command -v "$t" >/dev/null 2>&1; then
      printf '%s\n' "$t"
      return 0
    fi
  done
  return 1
}

dc_yazi_home() {
  printf '%s\n' "${DC_YAZI_HOME:-${XDG_DATA_HOME:-$HOME/.local/share}/dc-cli/tools}"
}

dc_yazi_asset_stem() {
  local os arch
  case "$(uname -s)" in
    Darwin) os=apple-darwin ;;
    Linux) os=unknown-linux-musl ;;
    *) return 1 ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) arch=aarch64 ;;
    x86_64|amd64) arch=x86_64 ;;
    *) return 1 ;;
  esac
  printf 'yazi-%s-%s\n' "$arch" "$os"
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

dc_link_yazi() {
  local src="$1" name
  [[ -x "$src/yazi" ]] || return 1
  for name in yazi ya; do
    [[ -e "$src/$name" ]] || continue
    chmod +x "$src/$name" 2>/dev/null || true
    if [[ -n "${PREFIX:-}" ]]; then
      mkdir -p "$PREFIX"
      ln -sfn "$src/$name" "$PREFIX/$name"
    fi
    if [[ -n "${GEN_ROOT:-}" && -d "$GEN_ROOT/current/bin" ]]; then
      ln -sfn "$src/$name" "$GEN_ROOT/current/bin/$name"
    fi
  done
}

dc_install_yazi() {
  local dest stem url tmp zip found ya
  dest="$(dc_yazi_home)"
  mkdir -p "$dest"
  if [[ -x "$dest/yazi" && -f "$dest/VERSION" && "$(tr -d '[:space:]' <"$dest/VERSION")" == "$DC_YAZI_VERSION" ]]; then
    dc_link_yazi "$dest"
    echo "yazi $DC_YAZI_VERSION already in $dest"
    return 0
  fi
  stem="$(dc_yazi_asset_stem)" || {
    echo "dc-cli: no yazi build for $(uname -s)/$(uname -m)" >&2
    return 1
  }
  url="https://github.com/sxyazi/yazi/releases/download/v${DC_YAZI_VERSION}/${stem}.zip"
  tmp="$(mktemp -d)"
  zip="$tmp/yazi.zip"
  if [[ -n "${DC_YAZI_ZIP:-}" ]]; then
    cp "$DC_YAZI_ZIP" "$zip" || { rm -rf "$tmp"; return 1; }
  else
    echo "Fetching yazi $DC_YAZI_VERSION ($stem) ..."
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
  ya="$(dirname "$found")/ya"
  if [[ -f "$ya" ]]; then
    cp "$ya" "$dest/ya"
    chmod +x "$dest/ya"
  fi
  printf '%s\n' "$DC_YAZI_VERSION" >"$dest/VERSION"
  rm -rf "$tmp"
  dc_link_yazi "$dest"
  echo "Installed yazi $DC_YAZI_VERSION -> $dest/yazi"
}

ensure_host_file_manager() {
  local have
  if [[ "${DC_SKIP_YAZI:-}" == "1" || "${SKIP_YAZI:-0}" == "1" ]]; then
    echo "host file manager: skipped (DC_SKIP_YAZI / --no-yazi)"
    return 0
  fi
  if have="$(dc_host_fm_on_path)"; then
    echo "host file manager already on PATH: $have ($(command -v "$have"))"
    return 0
  fi
  echo "No host file manager (yazi/nnn/lf/mc/ranger); installing yazi ${DC_YAZI_VERSION} on the host"
  if ! dc_install_yazi; then
    echo "yazi install failed (non-fatal). dc-files still uses an in-box FM if the image has one."
    return 0
  fi
}
