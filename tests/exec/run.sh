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

seed_stack() {
  local ws="$1"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=app"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=db"
}

case_restart_sibling() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  dc-exec --service db --restart "$ws"
  log_has '^restart db1$'
}

case_restart_unknown() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  set +e
  dc-exec --service nope --restart "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  log_lacks '^restart '
}

case_restart_id_banned() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  set +e
  dc-exec --id db1 --restart "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  log_lacks '^restart '
}

case_restart_no_app() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=db"
  set +e
  dc-exec --service db --restart "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  log_lacks '^restart '
}

case_restart_prefers_service_over_id_prefix() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  # Hex id prefix "db" must not steal compose service db.
  fake_add_container db12ffff app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=app"
  fake_add_container ffff1111 db-1 running \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=db"
  dc-exec --service db --restart "$ws"
  log_has '^restart ffff1111$'
  log_lacks '^restart db12ffff$'
}

case_restart_needs_service() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  set +e
  dc-exec --restart "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  log_lacks '^restart '
}

run_case term-env case_term_env
run_case color-rc-hl case_color_rc_hl
run_case restart-sibling case_restart_sibling
run_case restart-unknown case_restart_unknown
run_case restart-id-banned case_restart_id_banned
run_case restart-no-app case_restart_no_app
run_case restart-id-prefix-collision case_restart_prefers_service_over_id_prefix
run_case restart-needs-service case_restart_needs_service

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
