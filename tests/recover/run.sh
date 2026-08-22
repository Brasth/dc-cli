#!/usr/bin/env bash
# dc-recover playbook + apply gates (v1: existing engines).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
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

run_case stopped-colima case_stopped_colima
run_case stopped-desktop case_stopped_desktop
run_case stopped-linux case_stopped_linux
run_case permission case_permission
run_case context case_context
run_case split case_split
run_case missing-v1-no-install case_missing_v1_no_install
run_case cli-missing-v1 case_cli_missing_v1
run_case ready case_ready
run_case ready-enospc case_ready_enospc
run_case ready-grow case_ready_grow
run_case ready-unlabeled case_ready_unlabeled_ports
run_case apply-colima-start case_apply_colima_start
run_case apply-stop-extra case_apply_stop_extra_colima
run_case apply-sudo-start case_apply_sudo_start
run_case cli-json-stopped case_cli_json_stopped
run_case cli-no-yes-no-start case_cli_no_yes_does_not_start
run_case cli-report case_cli_report

echo "recover: $((ran - FAILED))/$ran passed"
[[ "$FAILED" -eq 0 ]]
