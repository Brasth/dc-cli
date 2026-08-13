#!/usr/bin/env bash
# Install host-global @devcontainers/cli helpers into ~/bin.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-$HOME/bin}"
WITH_CLI=0
WITH_SKILL=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-cli) WITH_CLI=1 ;;
    --with-skill) WITH_SKILL=1 ;;
    --prefix)
      shift
      PREFIX="$1"
      ;;
    -h|--help)
      echo "Usage: bash install.sh [--with-cli] [--with-skill] [--prefix DIR]"
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
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
  # Create bashrc on Linux if it is the login shell rc and missing.
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
  installed=0
  maybe_skill "$HOME/.pi/agent" "$HOME/.pi/agent/skills" && installed=1
  maybe_skill "$HOME/.claude" "$HOME/.claude/skills" && installed=1
  maybe_skill "$HOME/.codex" "$HOME/.codex/skills" && installed=1
  maybe_skill "$HOME/.gemini" "$HOME/.gemini/skills" && installed=1
  maybe_skill "$HOME/.cursor" "$HOME/.cursor/skills" && installed=1
  maybe_skill "$HOME/.opencode" "$HOME/.opencode/skills" && installed=1
  # Shared Agent Skills location (Pi, Gemini, Cursor often honor this).
  install_skill_dir "$HOME/.agents/skills"
  installed=1
  if [[ -d "$HOME/.pi/agent/pi-hermes-memory/skills" ]]; then
    install_skill_dir "$HOME/.pi/agent/pi-hermes-memory/skills"
  fi
  if [[ "$installed" -eq 0 ]]; then
    echo "--with-skill: no known agent home found; wrote ~/.agents/skills only" >&2
  fi
fi

echo
echo "Next:"
echo "  source ~/.zshrc   # or open a new terminal"
echo "  dc-up --help"
echo "  dc-down --help"
echo "Official CLI has no down — use dc-down."
echo "dc-up --ports REPLACES project devcontainer.json (not a merge)."
