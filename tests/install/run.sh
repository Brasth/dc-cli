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
    DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  [[ -L "$home/share/generations/current" ]]
  [[ -f "$home/share/generations/current/lib/dc-common.sh" ]]
  [[ -x "$prefix/dc-up" ]]
  [[ -x "$prefix/dc" ]]
  "$prefix/dc-up" --help >/dev/null
  "$prefix/dc" --help >/dev/null
  "$prefix/dc" up --help >/dev/null
  # Partial extra id dir is not current.
  mkdir -p "$home/share/generations/partial-id/bin"
  [[ "$(readlink "$home/share/generations/current")" != *partial-id ]]
  first="$(readlink "$home/share/generations/current")"
  HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 \
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
      DC_DEVCONTAINER_MIN_VERSION="0.88.0" DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 \
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
      DC_DEVCONTAINER_MIN_VERSION="0.88.0" DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 \
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
    DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 \
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
  out="$(PATH="$home/fake:/usr/bin:/bin" HOME="$home" \
    DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 DC_INSTALL_OFFER_CLI=0 \
    bash "$ROOT/install.sh" --prefix "$prefix" 2>&1)"
  printf '%s\n' "$out" | grep -q 'WARNING: wrappers-only'
  printf '%s\n' "$out" | grep -q -- '--with-cli'
  rm -rf "$home"
}

case_help_no_yazi() {
  bash "$ROOT/install.sh" --help | grep -q -- --no-yazi
}

case_yazi_from_zip() {
  local home prefix zipdir
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  zipdir="$home/yz"
  mkdir -p "$prefix" "$zipdir/yazi-aarch64-apple-darwin"
  cat >"$zipdir/yazi-aarch64-apple-darwin/yazi" <<'EOF'
#!/usr/bin/env bash
echo fake-yazi
EOF
  chmod +x "$zipdir/yazi-aarch64-apple-darwin/yazi"
  (cd "$zipdir" && zip -q -r "$home/yazi.zip" yazi-aarch64-apple-darwin)
  PATH="/usr/bin:/bin" HOME="$home" \
    DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 DC_YAZI_ZIP="$home/yazi.zip" \
    bash "$ROOT/install.sh" --prefix "$prefix" >/dev/null
  arch=amd64
  case "$(uname -m)" in arm64|aarch64) arch=arm64 ;; esac
  [[ -x "$home/.local/share/dc-cli/tools/guest/$arch/yazi" ]]
  "$home/.local/share/dc-cli/tools/guest/$arch/yazi" | grep -q fake-yazi
  rm -rf "$home"
}

case_bash3_failfast() {
  local out rc
  rc=0
  out="$(DC_BASH_MAJOR=3 bash "$ROOT/install.sh" --help 2>&1)" || rc=$?
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -qi 'Bash 4'
}

_install_os_arch() {
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

_sha256_file() {
  local f="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$f" | awk '{print $1}'
  else
    shasum -a 256 "$f" | awk '{print $1}'
  fi
}

_stage_tiny_release_kit() {
  local reldir="$1" os="$2" arch="$3" name kit f
  name="dc-cli-0.0.0-${os}-${arch}"
  kit="$reldir/$name"
  mkdir -p "$kit/bin" "$kit/lib"
  for f in dc dc-up dc-exec dc-down dc-ps dc-forward dc-ls dc-open dc-df dc-prune dc-doctor dc-engine dc-recover dc-upgrade dc-db dc-files dc-stats dc-net dc-try dc-inspect; do
    if [[ -f "$ROOT/bin/$f" ]]; then
      cp "$ROOT/bin/$f" "$kit/bin/$f"
      chmod +x "$kit/bin/$f"
    fi
  done
  cp "$ROOT/lib/"*.sh "$kit/lib/"
  tar -C "$reldir" -czf "$reldir/${name}.tar.gz" "$name"
  printf '%s\n' "$name"
}

case_checksum_match() {
  local home prefix reldir os arch name
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  reldir="$home/rel"
  mkdir -p "$prefix" "$reldir"
  read -r os arch < <(_install_os_arch)
  name="$(_stage_tiny_release_kit "$reldir" "$os" "$arch")"
  printf '%s  %s\n' "$(_sha256_file "$reldir/${name}.tar.gz")" "${name}.tar.gz" >"$reldir/SHA256SUMS"
  HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
    DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 DC_INSTALL_OFFER_CLI=0 \
    DC_RELEASE_BASE_URL="file://${reldir}" \
    bash "$ROOT/install.sh" --prefix "$prefix" --ref v0.0.0 >/dev/null
  [[ -L "$home/share/generations/current" ]]
  [[ -x "$home/share/generations/current/bin/dc-up" ]]
  rm -rf "$home"
}

case_checksum_mismatch() {
  local home prefix reldir os arch name out rc
  home="$(mktemp -d "${TMPDIR:-/tmp}/dc-inst.XXXX")"
  prefix="$home/bin"
  reldir="$home/rel"
  mkdir -p "$prefix" "$reldir"
  read -r os arch < <(_install_os_arch)
  name="$(_stage_tiny_release_kit "$reldir" "$os" "$arch")"
  printf '%s  %s\n' "0000000000000000000000000000000000000000000000000000000000000000" "${name}.tar.gz" >"$reldir/SHA256SUMS"
  rc=0
  out="$(
    HOME="$home" DC_GENERATION_ROOT="$home/share/generations" PREFIX="$prefix" \
      DC_SKIP_TUI_BUILD=1 DC_SKIP_YAZI=1 DC_INSTALL_OFFER_CLI=0 \
      DC_RELEASE_BASE_URL="file://${reldir}" \
      bash "$ROOT/install.sh" --prefix "$prefix" --ref v0.0.0 2>&1
  )" || rc=$?
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -qi 'checksum mismatch'
  [[ ! -e "$home/share/generations/current" ]]
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
run_case "help lists --no-yazi" case_help_no_yazi
run_case "installs guest yazi from zip" case_yazi_from_zip
run_case "bash 3 fail-fast requires Bash 4" case_bash3_failfast
run_case "release kit checksum match extracts" case_checksum_match
run_case "release kit checksum mismatch refuses" case_checksum_mismatch

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
