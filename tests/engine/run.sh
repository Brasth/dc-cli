#!/usr/bin/env bash
# Engine identity helper + dc-up split-brain refuse.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-engine.sh"

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
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

mk_sock() {
  local p="$1"
  mkdir -p "$(dirname "$p")"
  : >"$p"
}

case_classify_matrix() {
  assert_eq "$(dc_engine_classify 'unix:///x/.colima/default/docker.sock')" colima
  assert_eq "$(dc_engine_classify 'unix:///x/.docker/run/docker.sock')" desktop
  assert_eq "$(dc_engine_classify 'unix:///x/.orbstack/run/docker.sock')" orbstack
  assert_eq "$(dc_engine_classify desktop-linux)" desktop
  local got
  got="$(dc_engine_classify 'unix:///var/run/docker.sock')"
  if [[ "$(uname -s)" == Linux ]]; then
    assert_eq "$got" linux
  else
    assert_eq "$got" desktop
  fi
  assert_eq "$(dc_engine_classify 'tcp://1.2.3.4:2375')" unknown
}

case_docker_host_wins() {
  printf '%s\n' "desktop-linux" >"$STATE/context_name"
  printf '%s\n' "unix:///x/.docker/run/docker.sock" >"$STATE/context_host"
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  assert_eq "$(dc_engine_cli_host)" "unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  assert_eq "$(dc_engine_classify "$(dc_engine_cli_host)")" colima
}

case_one_live_no_split() {
  printf '%s\n' "unix://${DC_ENGINE_HOME}/.colima/default/docker.sock" >"$STATE/context_host"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
  if dc_engine_split; then
    echo "unexpected split" >&2
    return 1
  fi
}

case_split_colima_plus_desktop() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" desktop
  dc_engine_split
}

case_dead_extra_not_split() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  printf '%s\n' "unix://${DC_ENGINE_HOME}/.docker/run/docker.sock" >"$STATE/info_dead"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
  if dc_engine_split; then
    echo "dead extra counted live" >&2
    return 1
  fi
}

case_symlink_alias_no_split() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/alias.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  ln -sf "${DC_ENGINE_HOME}/.docker/run/docker.sock" "${DC_ENGINE_HOME}/alias.sock"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
  assert_eq "$(dc_engine_classify "$DOCKER_HOST")" desktop
  if dc_engine_split; then
    echo "symlink alias counted as extra engine" >&2
    return 1
  fi
}

case_classify_symlink_to_colima() {
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  ln -sf "${DC_ENGINE_HOME}/.colima/default/docker.sock" "${DC_ENGINE_HOME}/alias.sock"
  assert_eq "$(dc_engine_classify "unix://${DC_ENGINE_HOME}/alias.sock")" colima
}

case_colima_dual_sock_no_split() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/docker.sock"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
  if dc_engine_split; then
    echo "colima default dual socket counted as split" >&2
    return 1
  fi
}

case_home_decoy_ignored() {
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mkdir -p "$STATE/real-home/.docker/run"
  : >"$STATE/real-home/.docker/run/docker.sock"
  export HOME="$STATE/real-home"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
}

case_probe_timeout_not_live() {
  export DC_ENGINE_PROBE_TIMEOUT=1
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  printf '%s\n' "unix://${DC_ENGINE_HOME}/.docker/run/docker.sock" >"$STATE/info_hang_hosts"
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  assert_eq "$extra" ""
}

case_up_refuses_split() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","services":{"web":{}},"networks":{}}' >"$STATE/compose-config.json"
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${DC_FAKE_STATE}/devc.log"
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  dc-up "$ws" >/dev/null 2>"$STATE/up.err"
  rc=$?
  set -e
  assert_eq "$rc" 1
  grep -q split-brain "$STATE/up.err"
  [[ ! -f "$STATE/devc.log" ]]
  log_lacks 'up -d'
}

case_up_hatch_proceeds() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","services":{"web":{}},"networks":{}}' >"$STATE/compose-config.json"
  export DOCKER_HOST="unix://${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.colima/default/docker.sock"
  mk_sock "${DC_ENGINE_HOME}/.docker/run/docker.sock"
  export DC_UP_ALLOW_SPLIT=1
  set +e
  out="$(dc-up "$ws" 2>"$STATE/up.err")"
  rc=$?
  set -e
  assert_eq "$rc" 0
  grep -q DC_UP_ALLOW_SPLIT "$STATE/up.err"
  log_has 'up -d'
  printf '%s\n' "$out" | grep -q 'engine colima'
}

echo "== engine identity =="
run_case classify-matrix case_classify_matrix
run_case docker-host-wins case_docker_host_wins
run_case one-live-no-split case_one_live_no_split
run_case symlink-alias-no-split case_symlink_alias_no_split
run_case classify-symlink-colima case_classify_symlink_to_colima
run_case colima-dual-sock-no-split case_colima_dual_sock_no_split
run_case home-decoy-ignored case_home_decoy_ignored
run_case split-colima-desktop case_split_colima_plus_desktop
run_case dead-extra-not-split case_dead_extra_not_split
run_case probe-timeout-not-live case_probe_timeout_not_live
run_case up-refuses-split case_up_refuses_split
run_case up-hatch-proceeds case_up_hatch_proceeds

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
