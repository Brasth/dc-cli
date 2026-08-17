#!/usr/bin/env bash
# Phase 3 dc-doctor contract gates.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
DOC="$ROOT/bin/dc-doctor"
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
  "$DOC" --help >/dev/null
  assert_eq "$?" 0
}

case_bad_flag() {
  set +e
  "$DOC" --nope >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_not_dir() {
  set +e
  "$DOC" /not/a/dir >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_json_schema() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  out="$("$DOC" --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null || { echo "$out" >&2; exit 1; }
  printf '%s\n' "$out" | grep -q '"schemaVersion": 1\|"schemaVersion":1'
  for id in platform_bash common_library docker_cli docker_daemon docker_context colima \
    devcontainer_cli devcontainer_read_configuration workspace_config duplicate_labels \
    stack_identity desired_ports required_networks actual_ports stale_owned_sidecars disk dc_cli_version dc_cli_channel; do
    printf '%s\n' "$out" | grep -q "\"$id\""
  done
  harness_teardown
}

case_missing_common() {
  set +e
  out="$(DC_COMMON_PATH="/no/such/dc-common.sh" "$DOC" --json . 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q common_library
  python3 -m json.tool <<<"$out" >/dev/null
}

case_zero_mutation() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{"forwardPorts":[3000]}' >"$ws/.devcontainer/devcontainer.json"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "$@" >>"${FAKE_DOCKER_LOG}.devc"
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
if [[ "$1" == "read-configuration" ]]; then echo '{}'; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  "$DOC" --json "$ws" >/dev/null
  if [[ -f "$FAKE_DOCKER_LOG" ]]; then
    ! grep -Eq '(^| )(run|create|start|stop|rm|rmi|exec|commit)( |$)' "$FAKE_DOCKER_LOG"
    ! grep -Eq 'compose (up|down)' "$FAKE_DOCKER_LOG"
  fi
  harness_teardown
}

case_below_floor() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "0.1.0"
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(DC_DEVCONTAINER_MIN_VERSION=0.88.0 "$DOC" --json "$ws")"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q below_floor
  harness_teardown
}

case_missing_config() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "0.88.0"
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(DC_DEVCONTAINER_MIN_VERSION= "$DOC" --json "$ws")"
  rc=$?
  set -e
  # engine/CLI may still produce other blockers (daemon). Accept warning on workspace_config.
  printf '%s\n' "$out" | grep -q workspace_config
  printf '%s\n' "$out" | grep -q 'dependency_workspace_config\|"hasConfig": false\|"hasConfig":false'
  harness_teardown
}

case_context_engine_fields() {
  harness_setup
  unset DOCKER_HOST
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  printf '%s\n' "colima" >"$STATE/context_name"
  printf '%s\n' "unix://${DC_ENGINE_HOME}/.colima/default/docker.sock" >"$STATE/context_host"
  mkdir -p "${DC_ENGINE_HOME}/.colima/default"
  : >"${DC_ENGINE_HOME}/.colima/default/docker.sock"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
if [[ "$1" == "read-configuration" ]]; then echo '{}'; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  out="$("$DOC" --json "$ws")"
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
c=next(x for x in d["checks"] if x["id"]=="docker_context")
assert c["status"]=="ok", c
assert c["data"]["engine"]=="colima", c["data"]
assert c["data"]["name"]=="colima", c["data"]
assert ".colima/" in c["data"]["socket"], c["data"]
assert c["data"]["extraLive"]==[]
' "$out"
  harness_teardown
}

case_split_brain_blocker() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mkdir -p "${DC_ENGINE_HOME}/.colima/default" "${DC_ENGINE_HOME}/.docker/run"
  : >"${DC_ENGINE_HOME}/.colima/default/docker.sock"
  : >"${DC_ENGINE_HOME}/.docker/run/docker.sock"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
if [[ "$1" == "read-configuration" ]]; then echo '{}'; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$("$DOC" --json "$ws")"
  rc=$?
  set -e
  assert_eq "$rc" 1
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
c=next(x for x in d["checks"] if x["id"]=="docker_context")
assert c["status"]=="blocker", c
assert c["error"]["code"]=="split_brain", c
assert "desktop" in c["data"]["extraLive"], c["data"]
' "$out"
  if [[ -f "$FAKE_DOCKER_LOG" ]]; then
    ! grep -Eq '(^| )(run|create|start|stop|rm|rmi|exec|commit)( |$)' "$FAKE_DOCKER_LOG"
    ! grep -Eq 'compose (up|down)' "$FAKE_DOCKER_LOG"
  fi
  harness_teardown
}

case_linux_desktop_context_ok() {
  harness_setup
  unset DOCKER_HOST
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  printf '%s\n' "desktop-linux" >"$STATE/context_name"
  printf '%s\n' "unix://${DC_ENGINE_HOME}/.docker/desktop/docker.sock" >"$STATE/context_host"
  mkdir -p "${DC_ENGINE_HOME}/.docker/desktop"
  : >"${DC_ENGINE_HOME}/.docker/desktop/docker.sock"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
if [[ "$1" == "read-configuration" ]]; then echo '{}'; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  out="$("$DOC" --json "$ws")"
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
c=next(x for x in d["checks"] if x["id"]=="docker_context")
assert c["status"]=="ok", c
assert c["data"]["engine"]=="desktop", c["data"]
assert c["data"]["name"]=="desktop-linux", c["data"]
assert ".docker/desktop/" in c["data"]["socket"], c["data"]
assert c["data"]["extraLive"]==[]
assert c["data"].get("extraSockets")==[]
' "$out"
  human="$("$DOC" "$ws")"
  ! printf '%s\n' "$human" | grep -q split-brain
  harness_teardown
}

case_secret_noleak() {
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  cat >"$ws/.devcontainer/devcontainer.json" <<'EOF'
{
  "forwardPorts": [3000],
  "containerEnv": { "TOKEN": "supersecret-token-xyz" }
}
EOF
  echo 'PASSWORD=supersecret-token-xyz' >"$ws/.env"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
if [[ "$1" == "read-configuration" ]]; then echo '{"ok":true}'; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  out="$("$DOC" --json "$ws"; "$DOC" "$ws")"
  ! printf '%s\n' "$out" | grep -q 'supersecret-token-xyz'
  harness_teardown
}

echo "== doctor gates =="
run_case "help exit 0" case_help
run_case "unknown flag exit 2" case_bad_flag
run_case "non-dir exit 2" case_not_dir
run_case "json schema + 18 ids" case_json_schema
run_case "missing common valid json" case_missing_common
run_case "fake-docker zero mutation" case_zero_mutation
run_case "below-floor blocker" case_below_floor
run_case "missing config reported" case_missing_config
run_case "secrets not leaked" case_secret_noleak
run_case "docker_context engine fields" case_context_engine_fields
run_case "split-brain blocker" case_split_brain_blocker
run_case "linux desktop context ok" case_linux_desktop_context_ok

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
