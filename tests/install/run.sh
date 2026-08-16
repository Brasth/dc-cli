#!/usr/bin/env bash
# Phase 2 installer gates (no network CLI install required).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FAILED=0
ran=0

pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }

run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
}

case_help_npm_flag() {
  bash "$ROOT/install.sh" --help | grep -q -- --with-cli-npm
}

case_with_cli_no_npm() {
  grep -n 'WITH_CLI' "$ROOT/install.sh" | grep -q .
  # The --with-cli branch must not call npm install.
  ! awk '/elif \[\[ "\$WITH_CLI" -eq 1 \]\]/,/else/' "$ROOT/install.sh" | grep -q 'npm install'
}

case_npm_exact_pin_only() {
  grep -q 'npm install -g "@devcontainers/cli@${DC_DEVCONTAINER_NPM_VERSION}"' "$ROOT/install.sh"
  ! grep -qE 'npm install -g @devcontainers/cli[[:space:]]*$' "$ROOT/install.sh"
}

case_no_node18() {
  ! grep -qi 'Node 18' "$ROOT/install.sh" "$ROOT/README.md" "$ROOT/site/src/content/guides/install.md"
}

case_floor_script() {
  bash "$ROOT/scripts/check-floor-qual.sh"
}

case_atomic_generation() {
  local home prefix
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  mkdir -p "$prefix"
  HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  [[ -L "$home/share/generations/current" ]]
  [[ -f "$home/share/generations/current/lib/dc-common.sh" ]]
  [[ -x "$prefix/dc-up" ]]
  "$prefix/dc-up" --help >/dev/null
  # Partial extra id dir is not current.
  mkdir -p "$home/share/generations/partial-id/bin"
  [[ "$(readlink "$home/share/generations/current")" != *partial-id ]]
  first="$(readlink "$home/share/generations/current")"
  HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  [[ ! -d "$home/share/generations/partial-id" ]]
  second="$(readlink "$home/share/generations/current")"
  [[ -n "$second" && "$second" != "$first" ]]
  [[ -x "$home/share/generations/current/bin/dc-up" ]]
  [[ ! -e "$first/current.tmp" ]]
  rm -rf "$home"
}

case_below_floor() {
  local home prefix
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  mkdir -p "$prefix" "$home/fake"
  cat >"$home/fake/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "0.1.0"
EOF
  chmod +x "$home/fake/devcontainer"
  set +e
  out="$(
    PATH="$home/fake:/usr/bin:/bin" HOME="$home" \
      DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
      DC_DEVCONTAINER_MIN_VERSION="0.88.0" DC_SKIP_TUI_BUILD=1 \
      bash "$ROOT/install.sh" --prefix "$prefix" --with-cli 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -qi 'floor'
  rm -rf "$home"
}

case_shadow_below_floor() {
  local home prefix
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  mkdir -p "$prefix" "$home/bad" "$home/good"
  cat >"$home/bad/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "0.1.0"
EOF
  cat >"$home/good/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "0.88.0"
EOF
  chmod +x "$home/bad/devcontainer" "$home/good/devcontainer"
  set +e
  out="$(
    PATH="$home/bad:$home/good:/usr/bin:/bin" HOME="$home" \
      DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
      DC_DEVCONTAINER_MIN_VERSION="0.88.0" DC_SKIP_TUI_BUILD=1 \
      bash "$ROOT/install.sh" --prefix "$prefix" --with-cli 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -qiE 'shadow|floor'
  rm -rf "$home"
}

case_fresh_login_path() {
  local home prefix
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  mkdir -p "$prefix"
  HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  grep -q 'dc-cli generation' "$home/.bashrc"
  out="$(
    HOME="$home" bash -lc "source '$home/.bashrc'; command -v dc-up"
  )"
  [[ "$out" == *"/current/bin/dc-up"* || "$out" == "$prefix/dc-up" ]]
  rm -rf "$home"
}

case_noflags_no_cli() {
  local home prefix
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  mkdir -p "$prefix" "$home/fake"
  cat >"$home/fake/npm" <<'EOF'
#!/usr/bin/env bash
echo "npm should not run" >&2
exit 99
EOF
  chmod +x "$home/fake/npm"
  PATH="$home/fake:/usr/bin:/bin" HOME="$home" \
    DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  rm -rf "$home"
}

echo "== installer gates =="
run_case "help lists --with-cli-npm" case_help_npm_flag
run_case "--with-cli does not npm install" case_with_cli_no_npm
run_case "npm path uses exact pin" case_npm_exact_pin_only
run_case "no Node 18 language" case_no_node18
run_case "qualification artifact gate" case_floor_script
run_case "atomic generation + interrupt leftover" case_atomic_generation
run_case "below-floor --with-cli reject" case_below_floor
run_case "shadowed below-floor winner fail-closed" case_shadow_below_floor
run_case "fresh-login PATH resolves helpers" case_fresh_login_path
run_case "no-flags does not run npm" case_noflags_no_cli

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
