#!/usr/bin/env bash
# Install host-global @devcontainers/cli helpers into ~/bin.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-$HOME/bin}"
WITH_CLI=0
WITH_SKILL=0

usage() {
  cat <<'EOF'
Usage: bash install.sh [options]

  (default)         helpers + override.json + PATH  (does not install CLI)
  --with-cli        npm i -g @devcontainers/cli if missing
  --with-skill      copy SKILL.md into existing agent homes
  --full            --with-cli + --with-skill
  --prefix DIR      install helpers here (default: ~/bin)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-cli) WITH_CLI=1 ;;
    --with-skill) WITH_SKILL=1 ;;
    --full) WITH_CLI=1; WITH_SKILL=1 ;;
    --prefix)
      shift
      PREFIX="$1"
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

mkdir -p "$PREFIX"
for f in dc-up dc-exec dc-down dc-ps dc-forward; do
  cp "$ROOT/bin/$f" "$PREFIX/$f"
  chmod +x "$PREFIX/$f"
done
echo "Installed helpers to $PREFIX"

cfg="$HOME/.config/devcontainer"
mkdir -p "$cfg"
if [[ ! -f "$cfg/override.json" ]]; then
  cp "$ROOT/config/override.json" "$cfg/override.json"
  echo "Wrote $cfg/override.json"
else
  echo "Kept existing $cfg/override.json"
fi

marker="# dc-cli helpers"
append_rc() {
  local rc="$1"
  # Create bashrc on Linux if missing.
  if [[ ! -f "$rc" ]]; then
    case "$rc" in
      *bashrc) touch "$rc" ;;
      *) return 0 ;;
    esac
  fi
  if grep -q 'dc-cli helpers' "$rc" 2>/dev/null; then
    echo "PATH block already in $rc"
    return 0
  fi
  cat >> "$rc" <<EOF

$marker
case ":\$PATH:" in
  *":\$HOME/bin:"*) ;;
  *) export PATH="\$HOME/bin:\$PATH" ;;
esac
# dc-up is safe-by-default. Use: dc-up --ports
EOF
  echo "Appended PATH block to $rc"
}

append_rc "$HOME/.zshrc"
append_rc "$HOME/.bashrc"

if [[ "$WITH_CLI" -eq 1 ]]; then
  if command -v devcontainer >/dev/null 2>&1; then
    echo "devcontainer already on PATH"
  else
    if ! command -v npm >/dev/null 2>&1; then
      echo "npm not found. Install Node 18+ then rerun: bash install.sh --with-cli" >&2
      exit 1
    fi
    npm install -g @devcontainers/cli
  fi
fi

install_skill_dir() {
  local dest="$1/devcontainer-cli-global"
  mkdir -p "$dest"
  cp "$ROOT/skill/SKILL.md" "$dest/SKILL.md"
  echo "Installed agent skill to $dest"
}

# Copy into a harness skills dir only if that product is already on this machine.
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
  echo "Doctor:"
  if command -v docker >/dev/null 2>&1; then
    echo "  docker        OK  $(command -v docker)"
  else
    echo "  docker        MISSING  install Docker Engine or Desktop"
  fi
  if command -v devcontainer >/dev/null 2>&1; then
    echo "  devcontainer  OK  $(command -v devcontainer)"
  else
    echo "  devcontainer  MISSING  rerun: bash install.sh --with-cli"
  fi
  if command -v socat >/dev/null 2>&1; then
    echo "  socat         OK  (dc-forward)"
  else
    echo "  socat         optional  dc-forward only — brew/apt install socat"
  fi
  if [[ -x "$PREFIX/dc-up" ]]; then
    echo "  helpers       OK  $PREFIX"
  else
    echo "  helpers       MISSING  $PREFIX"
  fi
}

doctor

echo
echo "Next:"
echo "  source ~/.bashrc   # Linux / WSL"
echo "  source ~/.zshrc    # macOS zsh"
echo "  dc-up --help"
echo "  dc-down --help"
if ! command -v devcontainer >/dev/null 2>&1; then
  echo "Official CLI not installed. Need it: bash install.sh --with-cli"
fi
echo "Official CLI has no down — use dc-down."
echo "dc-up --ports REPLACES project devcontainer.json (not a merge)."
