#!/usr/bin/env bash
# Install host-global @devcontainers/cli helpers as one verified generation.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-$HOME/bin}"
WITH_CLI=0
WITH_CLI_NPM=0
WITH_SKILL=0
SKIP_YAZI=0
REF=""
REPO="${DC_REPO:-Brasth/dc-cli}"
ADVERTISED_CURL='curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli'
STANDALONE_URL="${DC_STANDALONE_INSTALLER_URL:-https://raw.githubusercontent.com/devcontainers/cli/main/scripts/install.sh}"
GEN_ROOT="${DC_GENERATION_ROOT:-${XDG_DATA_HOME:-$HOME/.local/share}/dc-cli/generations}"

if [[ -f "$ROOT/lib/dc-floor.sh" ]]; then
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-floor.sh"
fi

usage() {
  cat <<'EOF'
Usage: bash install.sh [options]

  (default)         helpers generation + override.json + PATH  (does not install CLI)
  --with-cli        official standalone @devcontainers/cli if missing (no npm)
  --with-cli-npm    explicit exact-pin npm install (never implied by --with-cli)
  --with-skill      copy SKILL.md into existing agent homes
  --full            --with-cli + --with-skill
  --prefix DIR      write stable shims here (default: ~/bin)
  --ref latest|TAG|main   fetch that GitHub release kit first
                    (prebuilt dc-tui; falls back to source tree)
                    (auto latest when this script is not next to bin/)
  --no-yazi         do not fetch host yazi (DC_SKIP_YAZI=1)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-cli) WITH_CLI=1 ;;
    --with-cli-npm) WITH_CLI_NPM=1 ;;
    --with-skill) WITH_SKILL=1 ;;
    --full) WITH_CLI=1; WITH_SKILL=1 ;;
    --no-yazi) SKIP_YAZI=1 ;;
    --prefix)
      shift
      PREFIX="$1"
      ;;
    --ref)
      shift
      [[ $# -ge 1 ]] || { echo "--ref needs latest|TAG|main" >&2; exit 2; }
      REF="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ "$WITH_CLI" -eq 1 && "$WITH_CLI_NPM" -eq 1 ]]; then
  echo "--with-cli and --with-cli-npm are mutually exclusive" >&2
  exit 2
fi

latest_tag() {
  local body
  body="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")" || return 1
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])' <<<"$body"
    return
  fi
  printf '%s\n' "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1
}

dc_os_arch() {
  local os arch
  case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) return 1 ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *) return 1 ;;
  esac
  printf '%s %s\n' "$os" "$arch"
}

use_extracted() {
  local tmp="$1" url="$2" extracted
  extracted="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -1)"
  if [[ -z "$extracted" || ! -f "$extracted/bin/dc-up" ]]; then
    return 1
  fi
  ROOT="$extracted"
  echo "Using $ROOT ($url)"
}

fetch_release_kit() {
  local ref="$1" os arch ver url tmp
  if [[ "$ref" == "latest" ]]; then
    ref="$(latest_tag)" || return 1
  fi
  case "$ref" in
    main|master) return 1 ;;
  esac
  ver="${ref#v}"
  read -r os arch < <(dc_os_arch) || return 1
  url="https://github.com/${REPO}/releases/download/v${ver}/dc-cli-${ver}-${os}-${arch}.tar.gz"
  tmp="$(mktemp -d)"
  echo "Fetching release kit $url ..."
  if ! curl -fsSL "$url" | tar -xz -C "$tmp"; then
    rm -rf "$tmp"
    return 1
  fi
  if ! use_extracted "$tmp" "$url"; then
    rm -rf "$tmp"
    return 1
  fi
}

fetch_tree() {
  local ref="$1" url tmp
  if [[ "$ref" == "latest" ]]; then
    ref="$(latest_tag)" || { echo "Could not resolve latest release of $REPO" >&2; exit 1; }
  fi
  if [[ "$ref" == "main" || "$ref" == "master" ]]; then
    url="https://github.com/${REPO}/archive/refs/heads/${ref}.tar.gz"
  else
    url="https://github.com/${REPO}/archive/refs/tags/${ref}.tar.gz"
  fi
  tmp="$(mktemp -d)"
  echo "Fetching $REPO@$ref ..."
  curl -fsSL "$url" | tar -xz -C "$tmp"
  if ! use_extracted "$tmp" "$url"; then
    echo "Fetched tree missing bin/dc-up ($url)" >&2
    exit 1
  fi
}

fetch_ref() {
  local ref="$1"
  if [[ "$ref" == "main" || "$ref" == "master" ]]; then
    fetch_tree "$ref"
    return
  fi
  fetch_release_kit "$ref" || fetch_tree "$ref"
}

if [[ -n "$REF" ]]; then
  fetch_ref "$REF"
elif [[ ! -f "$ROOT/bin/dc-up" ]]; then
  fetch_ref latest
fi

HELPERS=(dc-up dc-exec dc-down dc-ps dc-forward dc-ls dc-open dc-df dc-prune dc-doctor dc-db dc-files)

if [[ -f "$ROOT/lib/dc-install-yazi.sh" ]]; then
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-install-yazi.sh"
fi

semver_ge() {
  local a="$1" b="$2"
  [[ -n "$a" && -n "$b" ]] || return 1
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$a" "$b" <<'PY'
import sys
def parts(s):
    out = []
    for p in s.split("."):
        try:
            out.append(int(p))
        except ValueError:
            out.append(0)
    while len(out) < 3:
        out.append(0)
    return out[:3]
a, b = parts(sys.argv[1]), parts(sys.argv[2])
sys.exit(0 if a >= b else 1)
PY
    return
  fi
  [[ "$a" == "$b" ]]
}

parse_cli_version() {
  local raw="$1"
  printf '%s' "$raw" | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -1
}

report_shadow() {
  local name="$1" intended="$2"
  local winner others
  winner="$(command -v "$name" 2>/dev/null || true)"
  [[ -n "$winner" ]] || return 0
  if [[ -n "$intended" && "$winner" != "$intended" ]]; then
    echo "shadow: $name winner is $winner (intended $intended)"
  fi
  others="$(type -a "$name" 2>/dev/null | awk '/is \// {print $3}' | awk 'NF && !seen[$0]++')"
  if [[ "$(printf '%s\n' "$others" | awk 'NF' | wc -l | tr -d ' ')" -gt 1 ]]; then
    echo "shadow: $name candidates:"
    printf '%s\n' "$others" | sed 's/^/  /'
  fi
}

cli_meets_floor() {
  local ver
  if ! command -v devcontainer >/dev/null 2>&1; then
    return 1
  fi
  ver="$(parse_cli_version "$(devcontainer --version 2>/dev/null || true)")"
  if [[ -z "${DC_DEVCONTAINER_MIN_VERSION:-}" ]]; then
    [[ -n "$ver" ]] && return 0
    devcontainer --version >/dev/null 2>&1
    return
  fi
  [[ -n "$ver" ]] || return 1
  semver_ge "$ver" "$DC_DEVCONTAINER_MIN_VERSION"
}

hard_reject_cli() {
  local why="$1"
  echo "devcontainer CLI $why" >&2
  if [[ -n "${DC_DEVCONTAINER_MIN_VERSION:-}" ]]; then
    echo "Required floor: ${DC_DEVCONTAINER_MIN_VERSION} (see docs/qualification/devcontainer-cli-floor.md)" >&2
  fi
  echo "Upgrade via official standalone installer or explicit --with-cli-npm (exact pin)." >&2
  echo "  ${ADVERTISED_CURL}" >&2
  exit 1
}

# --- wrapper generation (single root, one current pointer) ---
mkdir -p "$GEN_ROOT"
gid="$(date +%Y%m%d%H%M%S)-$$"
stage="$GEN_ROOT/$gid"
mkdir -p "$stage/bin" "$stage/lib"

for f in "${HELPERS[@]}"; do
  if [[ -f "$ROOT/bin/$f" ]]; then
    cp "$ROOT/bin/$f" "$stage/bin/$f"
    chmod +x "$stage/bin/$f"
  fi
done

if [[ -f "$ROOT/bin/dc-tui" && "$(head -c 2 "$ROOT/bin/dc-tui")" != "#!" ]]; then
  cp "$ROOT/bin/dc-tui" "$stage/bin/dc-tui"
  chmod +x "$stage/bin/dc-tui"
  echo "Staged prebuilt dc-tui"
elif [[ "${DC_SKIP_TUI_BUILD:-}" == "1" && -f "$ROOT/bin/dc-tui" ]]; then
  cp "$ROOT/bin/dc-tui" "$stage/bin/dc-tui"
  chmod +x "$stage/bin/dc-tui"
  echo "Staged bash dc-tui (DC_SKIP_TUI_BUILD=1)"
elif [[ -f "$ROOT/cmd/dc-tui/main.go" ]] && command -v go >/dev/null 2>&1; then
  echo "Building clickable dc-tui (Go)..."
  ver=""
  if [[ -f "$ROOT/VERSION" ]]; then
    ver="$(tr -d '[:space:]' <"$ROOT/VERSION" || true)"
    ver="${ver#v}"
  fi
  ldflags="-s -w"
  [[ -n "$ver" ]] && ldflags="$ldflags -X main.version=${ver}"
  (cd "$ROOT" && go build -ldflags "$ldflags" -o "$stage/bin/dc-tui" ./cmd/dc-tui)
  chmod +x "$stage/bin/dc-tui"
  echo "Staged Go dc-tui"
elif [[ -f "$ROOT/bin/dc-tui" ]]; then
  cp "$ROOT/bin/dc-tui" "$stage/bin/dc-tui"
  chmod +x "$stage/bin/dc-tui"
  echo "Staged bash dc-tui (install Go for clickable buttons)"
fi

if [[ -d "$ROOT/lib" ]]; then
  cp "$ROOT/lib/"*.sh "$stage/lib/"
else
  echo "Missing $ROOT/lib" >&2
  rm -rf "$stage"
  exit 1
fi
if [[ -f "$ROOT/VERSION" ]]; then
  cp "$ROOT/VERSION" "$stage/VERSION"
fi

verify_stage() {
  local f
  for f in "$stage/bin"/dc-*; do
    [[ -e "$f" ]] || continue
    if [[ "$(head -c 2 "$f")" == "#!" ]]; then
      bash -n "$f"
    fi
    [[ -x "$f" ]] || return 1
  done
  bash -n "$stage/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$stage/lib/dc-common.sh"
  "$stage/bin/dc-up" --help >/dev/null
  "$stage/bin/dc-ls" --help >/dev/null
}

if ! verify_stage; then
  echo "Staged generation failed verify; leaving previous current untouched" >&2
  rm -rf "$stage"
  exit 1
fi

# Replace the current symlink in place. Do not `mv` onto it — on macOS
# `mv` follows a directory symlink and drops the new link inside the old kit.
ln -sfn "$stage" "$GEN_ROOT/current"
echo "Activated generation $gid -> $GEN_ROOT/current"

# Best-effort GC of old generation dirs (never touch current target).
cur_target="$(readlink "$GEN_ROOT/current" 2>/dev/null || true)"
for old in "$GEN_ROOT"/*; do
  [[ -d "$old" ]] || continue
  base="$(basename "$old")"
  [[ "$base" == "current" ]] && continue
  [[ "$old" == "$cur_target" ]] && continue
  [[ "$old" == "$stage" ]] && continue
  rm -rf "$old" 2>/dev/null || true
done

mkdir -p "$PREFIX"
write_shim() {
  local name="$1"
  cat >"$PREFIX/$name" <<EOF
#!/usr/bin/env bash
exec "${GEN_ROOT}/current/bin/${name}" "\$@"
EOF
  chmod +x "$PREFIX/$name"
}
for f in "${HELPERS[@]}" dc-tui; do
  if [[ -x "$GEN_ROOT/current/bin/$f" ]]; then
    write_shim "$f"
  fi
done
echo "Installed helper shims to $PREFIX (payload $GEN_ROOT/current)"

cfg="$HOME/.config/devcontainer"
mkdir -p "$cfg"
if [[ ! -f "$cfg/override.json" ]]; then
  if [[ -f "$ROOT/config/override.json" ]]; then
    cp "$ROOT/config/override.json" "$cfg/override.json"
    echo "Wrote $cfg/override.json"
  fi
else
  echo "Kept existing $cfg/override.json"
fi

append_path_block() {
  local rc="$1" marker="$2" dir="$3"
  if [[ ! -f "$rc" ]]; then
    case "$rc" in
      *bashrc) touch "$rc" ;;
      *) return 0 ;;
    esac
  fi
  if grep -Fq "$marker" "$rc" 2>/dev/null; then
    echo "PATH block already in $rc ($marker)"
    return 0
  fi
  cat >>"$rc" <<EOF

$marker
case ":\$PATH:" in
  *":$dir:"*) ;;
  *) export PATH="$dir:\$PATH" ;;
esac
EOF
  echo "Appended PATH block to $rc ($marker)"
}

append_path_block "$HOME/.zshrc" "# dc-cli generation" "$GEN_ROOT/current/bin"
append_path_block "$HOME/.bashrc" "# dc-cli generation" "$GEN_ROOT/current/bin"
# Keep a PREFIX shim dir on PATH for humans who already used ~/bin.
append_path_block "$HOME/.zshrc" "# dc-cli helpers" "$PREFIX"
append_path_block "$HOME/.bashrc" "# dc-cli helpers" "$PREFIX"

export PATH="$GEN_ROOT/current/bin:$PREFIX:$PATH"

if declare -F ensure_host_file_manager >/dev/null 2>&1; then
  ensure_host_file_manager
fi

# --- official CLI channels ---
install_standalone() {
  local script
  if [[ -n "${DC_STANDALONE_INSTALLER:-}" && -x "${DC_STANDALONE_INSTALLER}" ]]; then
    "${DC_STANDALONE_INSTALLER}"
    return
  fi
  script="$(mktemp)"
  if ! curl -fsSL "$STANDALONE_URL" -o "$script"; then
    rm -f "$script"
    echo "Failed to download official standalone installer" >&2
    return 1
  fi
  bash "$script"
  local rc=$?
  rm -f "$script"
  return "$rc"
}

persist_standalone_path() {
  local bin="$HOME/.devcontainers/bin"
  append_path_block "$HOME/.zshrc" "# dc-cli devcontainers bin" "$bin"
  append_path_block "$HOME/.bashrc" "# dc-cli devcontainers bin" "$bin"
  export PATH="$bin:$PATH"
}

npm_engines_ok() {
  local pin="$1" spec major
  if [[ -n "${DC_NPM_NODE_MIN_MAJOR:-}" ]]; then
    major="$DC_NPM_NODE_MIN_MAJOR"
  else
    spec="$(npm view "@devcontainers/cli@${pin}" engines.node 2>/dev/null || true)"
    major="$(printf '%s' "$spec" | grep -Eo '[0-9]+' | head -1)"
  fi
  [[ -n "$major" ]] || return 0
  local host
  host="$(node -p 'process.versions.node' 2>/dev/null || true)"
  host="${host%%.*}"
  [[ -n "$host" ]] || return 1
  [[ "$host" -ge "$major" ]]
}

if [[ "$WITH_CLI" -eq 1 || "$WITH_CLI_NPM" -eq 1 ]]; then
  if command -v devcontainer >/dev/null 2>&1; then
    echo "devcontainer already on PATH: $(command -v devcontainer)"
    echo "version: $(devcontainer --version 2>/dev/null || echo unparseable)"
    report_shadow devcontainer ""
    if ! cli_meets_floor; then
      hard_reject_cli "below floor or unparseable ($(command -v devcontainer))"
    fi
    echo "channel: pre-existing"
  elif [[ "$WITH_CLI" -eq 1 ]]; then
    if ! install_standalone; then
      echo "Standalone @devcontainers/cli install failed (no auto npm)." >&2
      echo "Retry: ${ADVERTISED_CURL}" >&2
      echo "Or explicit npm pin: bash install.sh --with-cli-npm" >&2
      if [[ -n "${DC_DEVCONTAINER_NPM_VERSION:-}" ]]; then
        echo "  npm i -g @devcontainers/cli@${DC_DEVCONTAINER_NPM_VERSION}" >&2
      else
        echo "  (npm pin unset until docs/qualification/devcontainer-cli-floor.md records DC_CLI_FLOOR_QUAL)" >&2
      fi
      exit 1
    fi
    persist_standalone_path
    if ! command -v devcontainer >/dev/null 2>&1 || ! cli_meets_floor; then
      hard_reject_cli "missing or below floor after standalone install"
    fi
    echo "channel: standalone  path=$(command -v devcontainer)  version=$(devcontainer --version 2>/dev/null || true)"
    report_shadow devcontainer "$HOME/.devcontainers/bin/devcontainer"
  else
    if [[ -z "${DC_DEVCONTAINER_NPM_VERSION:-}" ]]; then
      echo "--with-cli-npm requires a recorded exact pin (DC_DEVCONTAINER_NPM_VERSION)." >&2
      echo "Floor is unpublished until docs/qualification/devcontainer-cli-floor.md is signed." >&2
      echo "Candidate evidence only: npm i -g @devcontainers/cli@${DC_DEVCONTAINER_CANDIDATE_VERSION}" >&2
      exit 1
    fi
    if ! command -v npm >/dev/null 2>&1; then
      echo "npm not found. Host Node must satisfy that package's engines.node." >&2
      exit 1
    fi
    if ! npm_engines_ok "$DC_DEVCONTAINER_NPM_VERSION"; then
      echo "Host Node does not satisfy engines.node for @devcontainers/cli@${DC_DEVCONTAINER_NPM_VERSION}" >&2
      exit 1
    fi
    npm install -g "@devcontainers/cli@${DC_DEVCONTAINER_NPM_VERSION}"
    if ! command -v devcontainer >/dev/null 2>&1 || ! cli_meets_floor; then
      hard_reject_cli "missing or below floor after npm install"
    fi
    echo "channel: npm  path=$(command -v devcontainer)  version=$(devcontainer --version 2>/dev/null || true)"
    report_shadow devcontainer ""
  fi
fi

install_skill_dir() {
  local dest="$1/devcontainer-cli-global"
  mkdir -p "$dest"
  cp "$ROOT/skill/SKILL.md" "$dest/SKILL.md"
  echo "Installed agent skill to $dest"
}

maybe_skill() {
  local marker_dir="$1"
  local skills_dir="$2"
  if [[ -d "$marker_dir" ]]; then
    install_skill_dir "$skills_dir"
    return 0
  fi
  return 1
}

if [[ "$WITH_SKILL" -eq 1 ]]; then
  maybe_skill "$HOME/.pi/agent" "$HOME/.pi/agent/skills" || true
  maybe_skill "$HOME/.claude" "$HOME/.claude/skills" || true
  maybe_skill "$HOME/.codex" "$HOME/.codex/skills" || true
  maybe_skill "$HOME/.gemini" "$HOME/.gemini/skills" || true
  maybe_skill "$HOME/.cursor" "$HOME/.cursor/skills" || true
  maybe_skill "$HOME/.opencode" "$HOME/.opencode/skills" || true
  install_skill_dir "$HOME/.agents/skills"
  if [[ -d "$HOME/.pi/agent/pi-hermes-memory/skills" ]]; then
    install_skill_dir "$HOME/.pi/agent/pi-hermes-memory/skills"
  fi
fi

doctor() {
  echo
  echo "Doctor (install-time presence only; day-2: dc-doctor):"
  if command -v docker >/dev/null 2>&1; then
    echo "  docker        OK  $(command -v docker)"
  else
    echo "  docker        MISSING  install Docker Engine or Desktop"
  fi
  if command -v devcontainer >/dev/null 2>&1; then
    echo "  devcontainer  OK  $(command -v devcontainer)  $(devcontainer --version 2>/dev/null || true)"
  else
    echo "  devcontainer  MISSING  ${ADVERTISED_CURL}"
  fi
  if command -v socat >/dev/null 2>&1; then
    echo "  socat         OK  (optional; dc-forward uses a Docker sidecar)"
  else
    echo "  socat         optional  dc-forward only — brew/apt install socat"
  fi
  if command -v yazi >/dev/null 2>&1 || command -v nnn >/dev/null 2>&1; then
    echo "  files         OK  host $(command -v yazi 2>/dev/null || command -v nnn) (dc-files fallback)"
  else
    echo "  files         optional  no host yazi/nnn — dc-files uses an in-box FM if present"
  fi
  if [[ -x "$PREFIX/dc-up" && -x "$PREFIX/dc-tui" ]]; then
    echo "  helpers       OK  $PREFIX (generation $GEN_ROOT/current)"
  else
    echo "  helpers       MISSING  $PREFIX"
  fi
  for e in zed code subl; do
    if command -v "$e" >/dev/null 2>&1; then
      echo "  editor $e     OK  $(command -v "$e")"
    else
      echo "  editor $e     optional"
    fi
  done
}

doctor

echo
echo "Next:"
echo "  open a new terminal (or: source ~/.bashrc / ~/.zshrc)"
echo "  dc-up --help"
echo "  dc-tui --help     # this folder"
echo "  dc-tui --all      # fleet"
echo "  dc-doctor         # read-only diagnostics"
if ! command -v devcontainer >/dev/null 2>&1; then
  echo "Official CLI not installed. Need it (standalone):"
  echo "  ${ADVERTISED_CURL}"
  echo "or explicit npm pin: bash install.sh --with-cli-npm"
fi
echo "Stop/remove: dc-down (upstream CLI has no down)."
echo "dc-up --ports REPLACES project devcontainer.json (not a merge)."
echo "Stock standalone --node-version pins a Node major only (mutable patch)."
