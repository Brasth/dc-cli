#!/usr/bin/env bash
# dc-recover playbook + apply gates (v1: existing engines).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-engine.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-host.sh"

FAILED=0
ran=0
pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }
run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  unset DOCKER_HOST
  # Recover lib is the unit under test; must exist after implementation.
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-recover.sh"
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

plan_from() {
  dc_host_set "$1" "${3:-summary}" "" "${2:-unknown}" "remediation" "retry"
  DC_RECOVER_FOLDER_HINT="${4:-}"
  DC_ENGINE_LAST_EXTRA="${5:-}"
  dc_recover_plan
}

case_stopped_colima() {
  plan_from docker_engine_stopped colima
  assert_eq "$DC_RECOVER_ID" start_colima
  assert_eq "$DC_RECOVER_APPLY" colima_start
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_stopped_desktop() {
  plan_from docker_engine_stopped desktop
  assert_eq "$DC_RECOVER_ID" start_desktop
  assert_eq "$DC_RECOVER_APPLY" launch_desktop
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_stopped_linux() {
  plan_from docker_engine_stopped linux
  assert_eq "$DC_RECOVER_ID" start_linux_docker
  assert_eq "$DC_RECOVER_APPLY" sudo_start_docker
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_permission() {
  plan_from docker_permission_denied linux
  assert_eq "$DC_RECOVER_ID" fix_socket_group
  assert_eq "$DC_RECOVER_APPLY" sudo_docker_group
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_permission_no_user() {
  unset USER
  plan_from docker_permission_denied linux
  printf '%s\n' "$DC_RECOVER_COMMAND" | grep -q 'usermod -aG docker'
  login="$(dc_recover_login_user)"
  [[ -n "$login" ]]
  printf '%s\n' "$DC_RECOVER_COMMAND" | grep -q "$login"
}

case_context() {
  plan_from docker_context_invalid unknown
  assert_eq "$DC_RECOVER_ID" reset_context
  assert_eq "$DC_RECOVER_APPLY" context_use
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_split() {
  plan_from docker_split_brain desktop "" "" colima
  assert_eq "$DC_RECOVER_ID" pick_engine
  assert_eq "$DC_RECOVER_APPLY" stop_extra_engine
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_missing_v1_no_install() {
  plan_from docker_engine_missing unknown
  assert_eq "$DC_RECOVER_ID" install_engine
  assert_eq "$DC_RECOVER_APPLY" open_guide
  [[ "$DC_RECOVER_APPLY" != brew_install_colima ]]
}

case_cli_missing_v1() {
  plan_from docker_cli_missing unknown
  assert_eq "$DC_RECOVER_ID" install_cli_or_engine
  assert_eq "$DC_RECOVER_APPLY" open_guide
}

case_ready() {
  plan_from ready colima
  assert_eq "$DC_RECOVER_ID" ready
  assert_eq "$DC_RECOVER_APPLY" none
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 0
}

case_ready_enospc() {
  plan_from ready colima "" enospc
  assert_eq "$DC_RECOVER_ID" reclaim_disk
  assert_eq "$DC_RECOVER_APPLY" prune_safe
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

case_ready_grow() {
  plan_from ready colima "" colima_full
  assert_eq "$DC_RECOVER_ID" grow_colima_disk
  assert_eq "$DC_RECOVER_APPLY" colima_grow_disk
}

case_ready_unlabeled_ports() {
  plan_from ready desktop "" port_clash_unlabeled
  assert_eq "$DC_RECOVER_ID" report_holders
  assert_eq "$DC_RECOVER_APPLY" none
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 0
}

case_kind_none_allows_try() {
  plan_from ready unknown "ready" kind_none
  assert_eq "$DC_RECOVER_ID" try_sandbox
  printf '%s\n' "$DC_RECOVER_COMMAND" | grep -q 'dc-try'
  assert_eq "$DC_RECOVER_APPLY" try_sandbox
  assert_eq "$DC_RECOVER_APPLY_ALLOWED" 1
}

# After dc-try, folder is still kind=none (external override). A labeled
# running/created app for this folder must not keep ranking try_sandbox.
case_kind_none_live_sandbox_ready() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  fake_add_container tryapp tryapp running "devcontainer.local_folder=$ws"
  dc_host_set ready "Docker engine reachable" "" linux "rem" "retry"
  dc_recover_folder_diagnose "$ws"
  assert_eq "$DC_RECOVER_FOLDER_HINT" ""
  dc_recover_plan
  assert_eq "$DC_RECOVER_ID" ready
}

case_apply_try_sandbox() {
  cat >"$STATE/bin/dc-try" <<'EOF'
#!/usr/bin/env bash
echo "dc-try $*" >>"${DC_FAKE_STATE}/dc-try.log"
exit 0
EOF
  chmod +x "$STATE/bin/dc-try"
  hash -r 2>/dev/null || true
  dc_recover_apply try_sandbox
  grep -q 'dc-try --yes \.' "$STATE/dc-try.log"
}

case_apply_colima_start() {
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
echo "colima $*" >>"${DC_FAKE_STATE}/colima.log"
exit 0
EOF
  chmod +x "$STATE/bin/colima"
  hash -r 2>/dev/null || true
  DC_HOST_ENGINE_HINT=colima
  dc_recover_apply colima_start
  grep -q 'colima start' "$STATE/colima.log"
}

case_apply_stop_extra_colima() {
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
echo "colima $*" >>"${DC_FAKE_STATE}/colima.log"
exit 0
EOF
  chmod +x "$STATE/bin/colima"
  hash -r 2>/dev/null || true
  DC_ENGINE_LAST_EXTRA=colima
  dc_recover_apply stop_extra_engine
  grep -q 'colima stop' "$STATE/colima.log"
}

case_apply_sudo_start() {
  cat >"$STATE/bin/sudo" <<'EOF'
#!/usr/bin/env bash
echo "sudo $*" >>"${DC_FAKE_STATE}/sudo.log"
exit 0
EOF
  chmod +x "$STATE/bin/sudo"
  hash -r 2>/dev/null || true
  dc_recover_apply sudo_start_docker
  grep -q 'systemctl start docker' "$STATE/sudo.log"
}

case_apply_context_fails() {
  dc_engine_apply_fix() { return 1; }
  if dc_recover_apply context_use; then
    echo "expected context_use failure" >&2
    return 1
  fi
}

case_grow_disk_rejects_desktop() {
  set +e
  out="$(DC_RECOVER_SKIP_DIAGNOSE=1 DC_HOST_CODE=ready DC_HOST_ENGINE_HINT=desktop \
    "$ROOT/bin/dc-recover" --grow-disk 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -qi 'Colima is the selected engine'
}

case_folder_missing_nets() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  dc_net_report_tsv() {
    printf '%s\n' 'dc_recover_ext|external|0|1|bridge|missing'
  }
  dc_host_set ready "Docker engine reachable" "" linux "rem" "retry"
  dc_recover_folder_diagnose "$ws"
  assert_eq "$DC_RECOVER_FOLDER_HINT" missing_nets
  dc_recover_plan
  assert_eq "$DC_RECOVER_ID" ensure_nets
  assert_eq "$DC_RECOVER_APPLY" create_nets
}

case_cli_json_yes_single() {
  export HOME="$DC_ENGINE_HOME"
  mkdir -p "$HOME/.colima/default"
  : >"$HOME/.colima/default/docker.sock"
  rm -f "$STATE/bin/docker" "$STATE/bin/colima"
  cat >"$STATE/bin/docker" <<EOF
#!/usr/bin/env bash
if [[ "\${1:-}" == "info" ]]; then
  echo "Error: Cannot connect to the Docker daemon" >&2
  exit 1
fi
exec "$ROOT/tests/lib/fake-docker" "\$@"
EOF
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$STATE/bin/docker" "$STATE/bin/colima"
  hash -r 2>/dev/null || true
  set +e
  out="$("$ROOT/bin/dc-recover" --json --yes 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  python3 -c '
import json,sys
raw=sys.argv[1].strip()
assert raw, "empty output"
json.loads(raw)
assert raw.count("\"schemaVersion\"") == 1, raw
' "$out"
}

case_cli_json_stopped() {
  export HOME="$DC_ENGINE_HOME"
  mk_sock() { mkdir -p "$(dirname "$1")"; : >"$1"; }
  mk_sock "${HOME}/.docker/run/docker.sock"
  export DOCKER_HOST="unix://${HOME}/.docker/run/docker.sock"
  rm -f "$STATE/bin/docker"
  cat >"$STATE/bin/docker" <<EOF
#!/usr/bin/env bash
if [[ "\${1:-}" == "info" ]]; then
  echo "Error: Cannot connect to the Docker daemon" >&2
  exit 1
fi
exec "$ROOT/tests/lib/fake-docker" "\$@"
EOF
  chmod +x "$STATE/bin/docker"
  # Desktop evidence so hint is desktop
  mkdir -p "$HOME/Applications"
  # Linux CI: socket under .docker/run is desktop classify
  hash -r 2>/dev/null || true
  out="$(dc-recover --json)"
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
assert d["schemaVersion"]==1
assert d["command"]=="dc-recover"
assert d["host"]["code"] in ("docker_engine_stopped","docker_cli_missing","docker_engine_missing")
assert "next" in d
assert d["next"]["id"]
' "$out"
}

case_cli_no_yes_does_not_start() {
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
echo ran >>"${DC_FAKE_STATE}/colima.log"
exit 0
EOF
  chmod +x "$STATE/bin/colima"
  DC_HOST_CODE=docker_engine_stopped DC_HOST_ENGINE_HINT=colima \
    DC_RECOVER_SKIP_DIAGNOSE=1 dc-recover >/dev/null || true
  [[ ! -f "$STATE/colima.log" ]]
}

case_cli_report() {
  dest="$STATE/report"
  dc-recover --report "$dest"
  [[ -f "$dest/host.json" ]]
  [[ -f "$dest/README.txt" ]]
  grep -q 'do not paste .env' "$dest/README.txt"
}

case_folder_system_df_not_enospc() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  printf '%s\n' "/no/such/docker-root" >"$STATE/info_root"
  dc_host_set ready "Docker engine reachable" "" linux "rem" "retry"
  dc_recover_folder_diagnose "$ws"
  assert_eq "$DC_RECOVER_FOLDER_HINT" ""
}

case_folder_host_df_enospc() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  cat >"$STATE/bin/df" <<'EOF'
#!/usr/bin/env bash
pct="${DC_FAKE_DF_PCT:-98}"
echo "Filesystem 1024-blocks Used Available Capacity Mounted on"
echo "/dev/fake 100000 98000 2000 ${pct}% /"
EOF
  chmod +x "$STATE/bin/df"
  hash -r 2>/dev/null || true
  dc_host_set ready "Docker engine reachable" "" linux "rem" "retry"
  dc_recover_folder_diagnose "$ws"
  assert_eq "$DC_RECOVER_FOLDER_HINT" enospc
}

case_grow_disk_requires_colima_full() {
  set +e
  out="$(DC_RECOVER_SKIP_DIAGNOSE=1 DC_HOST_CODE=ready DC_HOST_ENGINE_HINT=colima \
    "$ROOT/bin/dc-recover" --grow-disk 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -qi 'guest disk >=95%'
  ! printf '%s\n' "$out" | grep -qi 'applying'
}

case_apply_sudo_start_no_dockerd() {
  cat >"$STATE/bin/sudo" <<'EOF'
#!/usr/bin/env bash
echo "sudo $*" >>"${DC_FAKE_STATE}/sudo.log"
exit 1
EOF
  chmod +x "$STATE/bin/sudo"
  cat >"$STATE/bin/pgrep" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
  chmod +x "$STATE/bin/pgrep"
  hash -r 2>/dev/null || true
  if dc_recover_apply sudo_start_docker >"$STATE/apply.out" 2>"$STATE/apply.err"; then
    echo "expected sudo_start_docker failure" >&2
    return 1
  fi
  grep -q 'systemctl start docker' "$STATE/sudo.log"
  if grep -q 'dockerd' "$STATE/sudo.log" "$STATE/apply.out" "$STATE/apply.err"; then
    echo "dockerd should not be started" >&2
    return 1
  fi
}

run_case stopped-colima case_stopped_colima
run_case stopped-desktop case_stopped_desktop
run_case stopped-linux case_stopped_linux
run_case permission case_permission
run_case permission-no-user case_permission_no_user
run_case context case_context
run_case split case_split
run_case missing-v1-no-install case_missing_v1_no_install
run_case cli-missing-v1 case_cli_missing_v1
run_case ready case_ready
run_case ready-enospc case_ready_enospc
run_case ready-grow case_ready_grow
run_case ready-unlabeled case_ready_unlabeled_ports
run_case kind-none-try case_kind_none_allows_try
run_case kind-none-live-sandbox case_kind_none_live_sandbox_ready
run_case apply-try-sandbox case_apply_try_sandbox
run_case apply-colima-start case_apply_colima_start
run_case apply-stop-extra case_apply_stop_extra_colima
run_case apply-sudo-start case_apply_sudo_start
run_case apply-context-fails case_apply_context_fails
run_case grow-disk-rejects-desktop case_grow_disk_rejects_desktop
run_case folder-missing-nets case_folder_missing_nets
run_case cli-json-yes-single case_cli_json_yes_single
run_case cli-json-stopped case_cli_json_stopped
run_case cli-no-yes-no-start case_cli_no_yes_does_not_start
run_case cli-report case_cli_report
run_case folder-system-df-not-enospc case_folder_system_df_not_enospc
run_case folder-host-df-enospc case_folder_host_df_enospc
run_case grow-disk-requires-colima-full case_grow_disk_requires_colima_full
run_case apply-sudo-start-no-dockerd case_apply_sudo_start_no_dockerd

echo "recover: $((ran - FAILED))/$ran passed"
[[ "$FAILED" -eq 0 ]]
