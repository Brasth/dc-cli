#!/usr/bin/env bash
# dc-upgrade --check contract (no network; env overrides).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
UP="$ROOT/bin/dc-upgrade"
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

case_help() {
  "$UP" --help | grep -q 'dc-upgrade'
}

case_check_available() {
  local out rc cache
  cache="$(mktemp -d "${TMPDIR:-/tmp}/dc-up-cache.XXXX")"
  set +e
  out="$(
    DC_CLI_VERSION=0.1.0 \
    DC_CLI_LATEST_TAG=0.2.0 \
    DC_CLI_CHANNEL=source \
    DC_CLI_CACHE_DIR="$cache" \
    "$UP" --check 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q 'state:     available'
  printf '%s\n' "$out" | grep -q 'installed: 0.1.0'
  printf '%s\n' "$out" | grep -q 'latest:    0.2.0'
  rm -rf "$cache"
}

case_check_current() {
  local out rc
  set +e
  out="$(
    DC_CLI_VERSION=0.2.0 \
    DC_CLI_LATEST_TAG=0.2.0 \
    DC_CLI_CHANNEL=source \
    "$UP" --check 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 0 ]]
  printf '%s\n' "$out" | grep -q 'state:     current'
}

case_check_dev() {
  local out rc
  set +e
  out="$(
    DC_CLI_VERSION=dev \
    DC_CLI_LATEST_TAG=0.2.0 \
    DC_CLI_CHANNEL=unknown \
    "$UP" --check 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 0 ]]
  printf '%s\n' "$out" | grep -q 'state:     dev'
}

case_check_unknown() {
  local out rc cache
  cache="$(mktemp -d "${TMPDIR:-/tmp}/dc-up-cache.XXXX")"
  set +e
  # Force a failed fetch: empty override is not used; point cache empty and block network via bad repo.
  out="$(
    DC_CLI_VERSION=0.1.0 \
    DC_CLI_CHANNEL=source \
    DC_CLI_CACHE_DIR="$cache" \
    DC_REPO="Brasth/this-repo-does-not-exist-dc-cli-xyz" \
    "$UP" --check 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  printf '%s\n' "$out" | grep -q 'state:     unknown'
  rm -rf "$cache"
}

case_refuse_dev_apply() {
  local out rc
  set +e
  out="$(
    DC_CLI_VERSION=dev \
    DC_CLI_LATEST_TAG=0.9.0 \
    DC_CLI_CHANNEL=source \
    "$UP" --yes 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  printf '%s\n' "$out" | grep -qi 'refusing'
}

case_semver_helpers() {
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  dc_cli_semver_lt 0.1.0 0.2.0
  ! dc_cli_semver_lt 0.2.0 0.2.0
  ! dc_cli_semver_lt 0.3.0 0.2.0
  dc_cli_semver_le 0.2.0 0.2.0
  local state
  state="$(DC_CLI_VERSION=0.1.0 DC_CLI_LATEST_TAG=9.0.0 DC_CLI_CHANNEL=source dc_cli_update_state)"
  [[ "$state" == "available" ]]
  state="$(DC_CLI_VERSION=9.0.0 DC_CLI_LATEST_TAG=9.0.0 DC_CLI_CHANNEL=source dc_cli_update_state)"
  [[ "$state" == "current" ]]
}

case_yes_source_with_skill() {
  local bindir log out rc
  bindir="$(mktemp -d "${TMPDIR:-/tmp}/dc-up-skill.XXXX")"
  log="$bindir/args.log"
  cat > "$bindir/curl" <<EOF
#!/usr/bin/env bash
cat <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "\$*" > '$log'
STUB
EOF
  chmod +x "$bindir/curl"
  set +e
  out="$(
    PATH="$bindir:$PATH" \
    DC_CLI_VERSION=0.1.0 \
    DC_CLI_LATEST_TAG=0.2.0 \
    DC_CLI_CHANNEL=source \
    "$UP" --yes 2>&1
  )"
  rc=$?
  set -e
  [[ "$rc" -eq 0 ]]
  grep -q -- '--with-skill' "$log"
  grep -q -- '--with-cli' "$log"
  grep -q -- '--ref latest' "$log"
  rm -rf "$bindir"
}

echo "== upgrade gates =="
run_case "help" case_help
run_case "check available exit 1" case_check_available
run_case "check current exit 0" case_check_current
run_case "check dev exit 0" case_check_dev
run_case "check unknown exit 2" case_check_unknown
run_case "refuse --yes on dev" case_refuse_dev_apply
run_case "semver + update_state helpers" case_semver_helpers
run_case "yes source includes --with-skill" case_yes_source_with_skill

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
