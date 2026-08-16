#!/usr/bin/env bash
# dc-exec passes TERM and injects the color rc.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"

FAILED=0
ran=0
pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }
run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

case_term_env() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running "devcontainer.local_folder=$ws"
  dc-exec --id app1 -- true
  grep -q -- '-e TERM=' "$FAKE_DOCKER_LOG"
  grep -q -- '-e COLORTERM=' "$FAKE_DOCKER_LOG"
}

case_color_rc_hl() {
  bash -n "$ROOT/lib/dc-exec-color.sh"
  bash -c '
    # shellcheck source=/dev/null
    source "$1"
    type hl >/dev/null
  ' _ "$ROOT/lib/dc-exec-color.sh"
}

run_case term-env case_term_env
run_case color-rc-hl case_color_rc_hl

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
