#!/usr/bin/env bash
# dc-net list / ensure lie-tests (fake Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
NET="$ROOT/bin/dc-net"
FAILED=0
ran=0

pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }
run_case() {
  local name="$1" rc
  shift
  ran=$((ran + 1))
  harness_setup
  (
    set -euo pipefail
    "$@"
  )
  rc=$?
  harness_teardown
  if [[ "$rc" -eq 0 ]]; then
    pass "$name"
  else
    fail "$name"
  fi
}

seed_ws() {
  local ws="$1"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{"dockerComposeFile":"../docker-compose.yml"}' >"$ws/.devcontainer/devcontainer.json"
  printf '%s\n' 'services: {}' >"$ws/docker-compose.yml"
}

seed_compose() {
  local json="$1"
  printf '%s\n' "$json" >"$STATE/compose-config.json"
}

case_help() {
  "$NET" --help >/dev/null
}

case_bad_flag() {
  set +e
  "$NET" --nope >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_empty() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{}}'
  out="$("$NET" --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null
  printf '%s\n' "$out" | grep -q '"schemaVersion":1'
  printf '%s\n' "$out" | grep -q '"networks":\[\]'
}

case_missing_external_create() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"shared":{"name":"shared-net","external":true}}}'
  out="$("$NET" --json "$ws")"
  printf '%s\n' "$out" | grep -q '"name":"shared-net"'
  printf '%s\n' "$out" | grep -q '"creatable":true'
  printf '%s\n' "$out" | grep -q '"missingCreatable":\["shared-net"\]'
  set +e
  "$NET" --ensure "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1 "non-tty ensure refuses"
  log_lacks 'network create'
  "$NET" --ensure --yes "$ws" >/dev/null
  [[ -d "$STATE/networks/shared-net" ]]
  log_has 'network create'
  out="$("$NET" --json "$ws")"
  printf '%s\n' "$out" | grep -q '"present":true'
  printf '%s\n' "$out" | grep -q '"missingCreatable":\[\]'
}

case_present_noop() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"shared":{"name":"already","external":true}}}'
  fake_add_network already
  "$NET" --ensure --yes "$ws" >/dev/null
  log_lacks 'network create'
}

case_compose_managed() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"frontend":{"name":"proj_frontend","driver":"bridge"}}}'
  out="$("$NET" --json "$ws")"
  printf '%s\n' "$out" | grep -q '"kind":"compose"'
  printf '%s\n' "$out" | grep -q '"reason":"compose-managed"'
  printf '%s\n' "$out" | grep -q '"creatable":false'
  "$NET" --ensure --yes "$ws" >/dev/null
  log_lacks 'network create'
  [[ ! -d "$STATE/networks/proj_frontend" ]]
}

case_unsupported_driver() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"ov":{"name":"ov-net","external":true,"driver":"overlay"}}}'
  set +e
  out="$("$NET" --ensure --yes "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q unsupported-driver
  log_lacks 'network create'
}

case_custom_ipam() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"n":{"name":"ipam-net","external":true,"ipam":{"config":[{"subnet":"172.28.0.0/16"}]}}}}'
  set +e
  out="$("$NET" --ensure --yes "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q custom-ipam
  log_lacks 'network create'
}

case_inspect_unknown() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"n":{"name":"ghost","external":true}}}'
  mkdir -p "$STATE/fail/network_inspect"
  : >"$STATE/fail/network_inspect/ghost"
  set +e
  out="$("$NET" --ensure --yes "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q inspect-unknown
  log_lacks 'network create'
}

case_external_name_object() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"shared":{"external":{"name":"actual-net"}}}}'
  out="$("$NET" --json "$ws")"
  printf '%s\n' "$out" | grep -q '"name":"actual-net"'
  "$NET" --ensure --yes "$ws" >/dev/null
  [[ -d "$STATE/networks/actual-net" ]]
}

stub_devcontainer_ok() {
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
}

case_up_create_nets() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"shared":{"name":"up-net","external":true}}}'
  stub_devcontainer_ok
  set +e
  dc-up --no-forward "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  log_lacks 'network create'
  dc-up --create-nets --no-forward "$ws" >/dev/null
  [[ -d "$STATE/networks/up-net" ]]
  log_has 'network create'
}

case_up_yes_creates() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"shared":{"name":"yes-net","external":true}}}'
  stub_devcontainer_ok
  dc-up --yes --no-forward "$ws" >/dev/null
  [[ -d "$STATE/networks/yes-net" ]]
}

case_up_blocked() {
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_compose '{"networks":{"ov":{"name":"ov-up","external":true,"driver":"overlay"}}}'
  stub_devcontainer_ok
  set +e
  out="$(dc-up --create-nets --no-forward "$ws" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q unsupported-driver
  log_lacks 'network create'
}

case_create_nets_not_take_ports() {
  grep -n 'create_nets' "$ROOT/bin/dc-up" | grep -q .
  ! awk '/--create-nets\)/,/;;/' "$ROOT/bin/dc-up" | grep -q take_ports
}

run_case help case_help
run_case bad-flag case_bad_flag
run_case empty-json case_empty
run_case missing-create case_missing_external_create
run_case present-noop case_present_noop
run_case compose-managed case_compose_managed
run_case overlay-refuse case_unsupported_driver
run_case custom-ipam case_custom_ipam
run_case inspect-unknown case_inspect_unknown
run_case external-name case_external_name_object
run_case up-create-nets case_up_create_nets
run_case up-yes-creates case_up_yes_creates
run_case up-blocked case_up_blocked
run_case create-nets-not-take-ports case_create_nets_not_take_ports

echo
echo "net: $ran cases, $FAILED failed"
[[ "$FAILED" -eq 0 ]]
