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
  [[ -f "$rc" ]] || return 0
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

if [[ "$WITH_SKILL" -eq 1 ]]; then
  installed=0
  # Pi auto-loads these (Agent Skills spec).
  if [[ -d "$HOME/.pi/agent" ]]; then
    install_skill_dir "$HOME/.pi/agent/skills"
    installed=1
  fi
  if [[ -d "$HOME/.agents" || -d "$HOME/.agents/skills" ]]; then
    install_skill_dir "$HOME/.agents/skills"
    installed=1
  fi
  # Optional extra copies if those trees already exist.
  if [[ -d "$HOME/.claude/skills" ]]; then
    install_skill_dir "$HOME/.claude/skills"
    installed=1
  fi
  if [[ -d "$HOME/.pi/agent/pi-hermes-memory/skills" ]]; then
    install_skill_dir "$HOME/.pi/agent/pi-hermes-memory/skills"
    installed=1
  fi
  if [[ "$installed" -eq 0 ]]; then
    mkdir -p "$HOME/.pi/agent/skills"
    install_skill_dir "$HOME/.pi/agent/skills"
  fi
fi

echo
echo "Next:"
echo "  source ~/.zshrc   # or open a new terminal"
echo "  dc-up --help"
echo "  dc-down --help"
echo "Official CLI has no down — use dc-down."
echo "dc-up --ports REPLACES project devcontainer.json (not a merge)."
