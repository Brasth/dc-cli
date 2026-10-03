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

# --- --no-start (used by dc-actions) ---

fake_devcontainer() {
  cat >"$STATE/bin/devcontainer" <<EOF
#!/usr/bin/env bash
printf 'devcontainer %s\n' "\$*" >>"$STATE/devcontainer.log"
exit "\${FAKE_DC_RC:-0}"
EOF
  chmod +x "$STATE/bin/devcontainer"
  hash -r 2>/dev/null || true
}

no_start_lacks() {
  log_lacks '^start '
  log_lacks '^restart '
  log_lacks '^compose .* start'
  [[ ! -f "$STATE/devcontainer.log" ]] || ! grep -q . "$STATE/devcontainer.log"
}

case_no_start_service_running() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  out="$(dc-exec --no-start --service db "$ws" -- echo 'a b' 2>/dev/null)"
  log_has '^exec .*db1 echo a b$'
  log_lacks '^start '
  [[ "$out" != *"exec  db"* ]]
}

case_no_start_service_stopped() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  echo exited >"$STATE/containers/db1/status"
  set +e
  dc-exec --no-start --service db "$ws" -- true >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  log_lacks '^exec '
  no_start_lacks
}

case_no_start_forged_service() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  # Same service name, other project: not this workspace's stack member.
  fake_add_container evil1 evil-db running \
    "com.docker.compose.project=other" \
    "com.docker.compose.service=evil"
  set +e
  dc-exec --no-start --service evil "$ws" -- true >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  log_lacks '^exec '
  no_start_lacks
}

case_no_start_app_stopped() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  fake_devcontainer
  echo exited >"$STATE/containers/app1/status"
  set +e
  dc-exec --no-start "$ws" -- true >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  no_start_lacks
}

case_no_start_app_missing() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_devcontainer
  set +e
  dc-exec --no-start "$ws" -- true >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  no_start_lacks
}

case_no_start_app_running_official() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  fake_devcontainer
  set +e
  FAKE_DC_RC=7 dc-exec --no-start "$ws" -- make 'x y' >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 7 ]]
  grep -q -- "devcontainer exec --workspace-folder $ws .* make x y" "$STATE/devcontainer.log"
  log_lacks '^start '
}

case_no_start_compose_kind_stopped() {
  local ws rc proj out
  ws="$(mktemp -d "$STATE/ck.XXXX")"
  printf 'services:\n  app:\n    image: alpine\n' >"$ws/compose.yaml"
  proj="$(basename "$ws")"
  printf '{"services":{"app":{}}}\n' >"$STATE/compose-config.json"
  fake_add_container c1 c-1 exited \
    "com.docker.compose.project=$proj" \
    "com.docker.compose.service=app" \
    "com.docker.compose.project.working_dir=$ws"
  set +e
  out="$(dc-exec --no-start "$ws" -- true 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q 'is exited — --no-start will not start it'
  no_start_lacks
  log_lacks '^compose .* exec'
}

case_no_start_compose_kind_running() {
  local ws proj
  ws="$(mktemp -d "$STATE/ck.XXXX")"
  printf 'services:\n  app:\n    image: alpine\n' >"$ws/compose.yaml"
  proj="$(basename "$ws")"
  printf '{"services":{"app":{}}}\n' >"$STATE/compose-config.json"
  fake_add_container c1 c-1 running \
    "com.docker.compose.project=$proj" \
    "com.docker.compose.service=app" \
    "com.docker.compose.project.working_dir=$ws"
  dc-exec --no-start "$ws" -- echo hi >/dev/null 2>&1
  log_has '^compose .* exec .*app echo hi$'
  no_start_lacks
}

case_no_start_refuses_id_and_restart() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  set +e
  dc-exec --no-start --id db1 -- true >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  set +e
  dc-exec --no-start --service db --restart "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  log_lacks '^exec '
  no_start_lacks
}

case_default_still_starts() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  echo exited >"$STATE/containers/db1/status"
  dc-exec --service db "$ws" -- true >/dev/null 2>&1
  log_has '^start db1$'
}

# A dc-try sandbox (no project config): the official CLI exec gets the
# sandbox's own override config; a configured folder never does.
case_sandbox_exec_uses_try_override() {
  local ws abs override
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  abs="$(cd "$ws" && pwd)"
  fake_add_container try1 try-1 running "devcontainer.local_folder=$abs"
  fake_devcontainer
  override="$(XDG_STATE_HOME="$STATE/xdg" bash -c 'source "$1/lib/dc-common.sh"; source "$1/lib/dc-try.sh"; dc_try_ensure_override "$2" generic' _ "$ROOT" "$abs")"
  [[ -f "$override" ]]
  XDG_STATE_HOME="$STATE/xdg" dc-exec --no-start "$ws" -- echo hi >/dev/null 2>&1
  grep -q -- "devcontainer exec --workspace-folder $abs --override-config $override .* echo hi" "$STATE/devcontainer.log"
  : >"$STATE/devcontainer.log"
  # Configured folder: no override even if a stale try dir exists.
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  XDG_STATE_HOME="$STATE/xdg" dc-exec --no-start "$ws" -- echo hi >/dev/null 2>&1
  ! grep -q -- "--override-config" "$STATE/devcontainer.log"
  log_lacks '^start '
}

run_case term-env case_term_env
run_case color-rc-hl case_color_rc_hl
run_case restart-sibling case_restart_sibling
run_case restart-unknown case_restart_unknown
run_case restart-id-banned case_restart_id_banned
run_case restart-no-app case_restart_no_app
run_case restart-id-prefix-collision case_restart_prefers_service_over_id_prefix
run_case restart-needs-service case_restart_needs_service
run_case no-start-service-running case_no_start_service_running
run_case no-start-service-stopped case_no_start_service_stopped
run_case no-start-forged-service case_no_start_forged_service
run_case no-start-app-stopped case_no_start_app_stopped
run_case no-start-app-missing case_no_start_app_missing
run_case no-start-app-official-exit case_no_start_app_running_official
run_case no-start-compose-kind-stopped case_no_start_compose_kind_stopped
run_case no-start-compose-kind-running case_no_start_compose_kind_running
run_case no-start-refuses-id-restart case_no_start_refuses_id_and_restart
run_case default-exec-still-starts case_default_still_starts
run_case sandbox-exec-uses-try-override case_sandbox_exec_uses_try_override

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
