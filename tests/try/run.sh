#!/usr/bin/env bash
# dc-try: profile detect, external override, refuse configured, up argv.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-try.sh"

FAILED=0
ran=0
pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }
run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  export DC_TRY_STATE_ROOT="$STATE/try-state"
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

case_detect_profiles() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  assert_eq "$(dc_try_detect_profile "$ws")" generic
  printf '%s\n' '{"name":"x"}' >"$ws/package.json"
  assert_eq "$(dc_try_detect_profile "$ws")" node
  rm -f "$ws/package.json"
  printf '%s\n' 'module example' >"$ws/go.mod"
  assert_eq "$(dc_try_detect_profile "$ws")" go
  printf '%s\n' 'flask==0' >"$ws/requirements.txt"
  # multi → generic
  assert_eq "$(dc_try_detect_profile "$ws")" generic
  rm -f "$ws/go.mod"
  assert_eq "$(dc_try_detect_profile "$ws")" python
}

case_print_writes_outside() {
  local ws out override before after
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' '{"name":"demo"}' >"$ws/package.json"
  before="$(find "$ws" -type f | sort | cksum)"
  out="$(dc-try --print "$ws")"
  printf '%s\n' "$out" | grep -q 'profile:   node'
  override="$(printf '%s\n' "$out" | awk '/override:/{print $2}')"
  [[ -f "$override" ]]
  [[ "$override" == "$STATE/try-state/"* ]]
  python3 -m json.tool <"$override" >/dev/null
  grep -q 'javascript-node' "$override"
  after="$(find "$ws" -type f | sort | cksum)"
  assert_eq "$after" "$before" "repo mutated"
}

case_refuse_devcontainer() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  set +e
  dc-try --yes "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_refuse_compose() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  set +e
  dc-try --yes "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_up_hint_on_none() {
  local ws out rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$(dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q 'dc-try'
}

case_up_yes_does_not_try() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  dc-up --yes "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_up_tty_no() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$(printf 'n\n' | DC_UP_TRY_PROMPT=1 dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -qi 'sandbox'
}

case_up_tty_yes() {
  local ws log
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'module x' >"$ws/go.mod"
  cat >"$STATE/bin/devcontainer" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$STATE/devc.log"
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  printf 'y\n' | DC_UP_TRY_PROMPT=1 dc-up "$ws" >/dev/null
  [[ -f "$STATE/devc.log" ]]
  log="$(cat "$STATE/devc.log")"
  printf '%s\n' "$log" | grep -q -- '--override-config'
}

case_try_up_argv() {
  local ws log
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'module x' >"$ws/go.mod"
  cat >"$STATE/bin/devcontainer" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$STATE/devc.log"
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  # engine guard needs docker info; fake-docker provides it
  dc-try --yes "$ws" >/dev/null
  [[ -f "$STATE/devc.log" ]]
  log="$(cat "$STATE/devc.log")"
  printf '%s\n' "$log" | grep -q 'up --workspace-folder'
  printf '%s\n' "$log" | grep -q -- '--override-config'
  printf '%s\n' "$log" | grep -q "$STATE/try-state/"
}

case_force_profile() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' '{"name":"x"}' >"$ws/package.json"
  out="$(dc-try --print --profile generic "$ws")"
  printf '%s\n' "$out" | grep -q 'profile:   generic'
  printf '%s\n' "$out" | grep -q 'devcontainers/base'
}

run_case detect-profiles case_detect_profiles
run_case print-outside case_print_writes_outside
run_case refuse-devcontainer case_refuse_devcontainer
run_case refuse-compose case_refuse_compose
run_case up-hint-none case_up_hint_on_none
run_case up-yes-does-not-try case_up_yes_does_not_try
run_case up-tty-no case_up_tty_no
run_case up-tty-yes case_up_tty_yes
run_case try-up-argv case_try_up_argv
run_case force-profile case_force_profile

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
