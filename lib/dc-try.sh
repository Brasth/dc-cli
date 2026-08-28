# dc-try helpers — external override configs for kind=none folders.
# Sourced by bin/dc-try. Never writes into the workspace.
# shellcheck shell=bash

DC_TRY_IMAGE_GO="${DC_TRY_IMAGE_GO:-mcr.microsoft.com/devcontainers/go:1-1.24-bookworm}"
DC_TRY_IMAGE_PYTHON="${DC_TRY_IMAGE_PYTHON:-mcr.microsoft.com/devcontainers/python:1-3.12-bookworm}"
DC_TRY_IMAGE_NODE="${DC_TRY_IMAGE_NODE:-mcr.microsoft.com/devcontainers/javascript-node:1-22-bookworm}"
DC_TRY_IMAGE_GENERIC="${DC_TRY_IMAGE_GENERIC:-mcr.microsoft.com/devcontainers/base:1-ubuntu-24.04}"

dc_try_state_root() {
  printf '%s\n' "${DC_TRY_STATE_ROOT:-${XDG_STATE_HOME:-$HOME/.local/state}/dc-cli/try}"
}

# Stable short key for an absolute workspace path.
dc_try_key() {
  local abs="$1" digest
  if command -v sha256sum >/dev/null 2>&1; then
    digest="$(printf '%s' "$abs" | sha256sum | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    digest="$(printf '%s' "$abs" | shasum -a 256 | awk '{print $1}')"
  else
    digest="$(printf '%s' "$abs" | openssl dgst -sha256 | awk '{print $NF}')"
  fi
  printf '%s\n' "${digest:0:16}"
}

dc_try_dir() {
  local abs="$1" key root
  key="$(dc_try_key "$abs")"
  root="$(dc_try_state_root)"
  printf '%s\n' "$root/$key"
}

# Print profile name: go | python | node | generic
# Exactly one ecosystem signal → that profile. Zero or many → generic.
dc_try_detect_profile() {
  local dir="$1"
  local -a hits=()
  [[ -d "$dir" ]] || { printf '%s\n' generic; return 0; }

  if [[ -f "$dir/go.mod" ]]; then
    hits+=(go)
  fi
  if [[ -f "$dir/pyproject.toml" || -f "$dir/requirements.txt" || -f "$dir/Pipfile" ]]; then
    hits+=(python)
  fi
  if [[ -f "$dir/package.json" ]]; then
    hits+=(node)
  fi

  if [[ ${#hits[@]} -eq 1 ]]; then
    printf '%s\n' "${hits[0]}"
    return 0
  fi
  printf '%s\n' generic
}

dc_try_image_for() {
  case "$1" in
    go) printf '%s\n' "$DC_TRY_IMAGE_GO" ;;
    python) printf '%s\n' "$DC_TRY_IMAGE_PYTHON" ;;
    node) printf '%s\n' "$DC_TRY_IMAGE_NODE" ;;
    generic|*) printf '%s\n' "$DC_TRY_IMAGE_GENERIC" ;;
  esac
}

dc_try_valid_profile() {
  case "$1" in
    go|python|node|generic) return 0 ;;
    *) return 1 ;;
  esac
}

# Default localhost ports for a try sandbox (profile-aware).
# Printed as a JSON array fragment, e.g. 3000, 5173
dc_try_ports_for() {
  case "$1" in
    node) printf '%s' '3000, 5173' ;;
    python) printf '%s' '8000, 5000' ;;
    go) printf '%s' '8080' ;;
    generic|*) printf '%s' '3000, 8080' ;;
  esac
}

# Write complete override + metadata under state dir. Prints override path.
# Args: abs_workspace [profile]
dc_try_ensure_override() {
  local abs="$1"
  local profile="${2:-}"
  local tdir image name override meta tmpo tmpm now base ports

  [[ -d "$abs" ]] || return 1
  abs="$(cd "$abs" && pwd)"

  if [[ -z "$profile" ]]; then
    profile="$(dc_try_detect_profile "$abs")"
  fi
  if ! dc_try_valid_profile "$profile"; then
    echo "dc-try: unknown profile: $profile" >&2
    return 2
  fi

  image="$(dc_try_image_for "$profile")"
  tdir="$(dc_try_dir "$abs")"
  mkdir -p "$tdir"
  override="$tdir/override.json"
  meta="$tdir/metadata.json"
  base="$(basename "$abs")"
  name="dc-try-${profile}"
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  tmpo="$(mktemp "$tdir/override.XXXXXX")"
  tmpm="$(mktemp "$tdir/metadata.XXXXXX")"
  ports="$(dc_try_ports_for "$profile")"
  # Complete replacement config — no hooks, features, or project scripts.
  # forwardPorts so localhost works after dc-forward (Colima / Desktop).
  cat >"$tmpo" <<EOF
{
  "name": "$(dc_json_escape "$name")",
  "image": "$(dc_json_escape "$image")",
  "workspaceFolder": "/workspaces/$(dc_json_escape "$base")",
  "forwardPorts": [${ports}]
}
EOF
  cat >"$tmpm" <<EOF
{
  "workspace": "$(dc_json_escape "$abs")",
  "profile": "$(dc_json_escape "$profile")",
  "image": "$(dc_json_escape "$image")",
  "override": "$(dc_json_escape "$override")",
  "ports": [${ports}],
  "updatedAt": "$(dc_json_escape "$now")"
}
EOF
  mv -f "$tmpo" "$override"
  mv -f "$tmpm" "$meta"
  printf '%s\n' "$override"
}

dc_try_metadata_path() {
  local abs="$1"
  printf '%s\n' "$(dc_try_dir "$abs")/metadata.json"
}
