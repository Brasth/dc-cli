#!/usr/bin/env bash
# Host readiness diagnosis gates.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-engine.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-host.sh"

pass=0
fail=0
ran=0

pass() { echo "ok  $*"; pass=$((pass + 1)); }
fail() { echo "FAIL $*" >&2; fail=$((fail + 1)); }
mk_sock() {
  mkdir -p "$(dirname "$1")"
  : >"$1"
}
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

case_cli_missing() {
  harness_path_no_docker "$STATE/bin"
  unset DOCKER_HOST
  dc_host_diagnose
  assert_eq "$DC_HOST_CODE" docker_cli_missing
  out="$(dc_host_json)"
  printf '%s\n' "$out" | grep -q '"code":"docker_cli_missing"'
  printf '%s\n' "$out" | grep -q '"status":"blocker"'
}

case_daemon_stopped() {
  export HOME="$DC_ENGINE_HOME"
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
  hash -r 2>/dev/null || true
  dc_host_diagnose
  assert_eq "$DC_HOST_CODE" docker_engine_stopped
  human="$(dc_host_print_human)"
  printf '%s\n' "$human" | grep -qi 'Docker Desktop\|colima\|engine'
}

case_permission_denied() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.docker/run/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  rm -f "$STATE/bin/docker"
  cat >"$STATE/bin/docker" <<EOF
#!/usr/bin/env bash
if [[ "\${1:-}" == "info" ]]; then
  echo "permission denied while trying to connect to the Docker daemon socket" >&2
  exit 1
fi
exec "$ROOT/tests/lib/fake-docker" "\$@"
EOF
  chmod +x "$STATE/bin/docker"
  hash -r 2>/dev/null || true
  dc_host_diagnose
  assert_eq "$DC_HOST_CODE" docker_permission_denied
}

case_ready() {
  export HOME="$DC_ENGINE_HOME"
  export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
  mk_sock "${HOME}/.colima/default/docker.sock"
  dc_host_diagnose
  assert_eq "$DC_HOST_CODE" ready
  dc_host_is_ready
}

case_up_blocks_when_cli_missing() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  harness_path_no_docker "$STATE/bin:$ROOT/bin"
  set +e
  out="$(dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q 'dc-up:'
  printf '%s\n' "$out" | grep -Eqi 'docker|PATH|engine'
}

case_doctor_host_code() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  export HOME="$DC_ENGINE_HOME"
  export DOCKER_HOST="unix://${HOME}/.docker/run/docker.sock"
  mk_sock "${HOME}/.docker/run/docker.sock"
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
  hash -r 2>/dev/null || true
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  out="$(DC_DEVCONTAINER_MIN_VERSION= dc-doctor --json "$ws")"
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
c=next(x for x in d["checks"] if x["id"]=="docker_daemon")
assert c["status"]=="blocker", c
assert c["data"].get("hostCode")=="docker_engine_stopped", c
assert "Start" in (c.get("remediation") or "") or "colima" in (c.get("remediation") or "").lower() or "Desktop" in (c.get("remediation") or ""), c
' "$out"
}

run_case cli_missing case_cli_missing
run_case daemon_stopped case_daemon_stopped
run_case permission_denied case_permission_denied
run_case ready case_ready
run_case up_blocks_when_cli_missing case_up_blocks_when_cli_missing
run_case doctor_host_code case_doctor_host_code

echo "host: $pass/$ran passed"
[[ "$fail" -eq 0 ]]
