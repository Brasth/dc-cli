#!/usr/bin/env bash
# dc-files detect + refuse gates (fake Docker).
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

seed() {
  local ws="$1"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projf" \
    "com.docker.compose.service=app" \
    image=alpine \
    bin=nnn
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projf" \
    "com.docker.compose.service=db" \
    image=postgres:16
}

case_help() {
  dc-files --help | grep -q 'file manager'
}

case_nnn_opens() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  dc-files --id app1 "$ws"
  grep -q 'nnn' "$STATE/exec.log"
}

case_empty_refuses() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    image=alpine
  set +e
  out="$(dc-files --id app1 "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -qi 'no file manager'
  [[ ! -f "$STATE/exec.log" ]] || ! grep -q . "$STATE/exec.log"
}

case_service_db() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  set +e
  out="$(dc-files --service db "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -qi 'no file manager'
  [[ ! -f "$STATE/containers/db1/exec.log" ]]
}

case_service_app_nnn() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  dc-files --service app "$ws"
  grep -q 'nnn' "$STATE/exec.log"
}

case_inject_yazi() {
  local ws zipdir
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    image=alpine \
    arch=amd64
  unset DC_FILES_NO_INJECT
  zipdir="$STATE/yz"
  mkdir -p "$zipdir/yazi-x86_64-unknown-linux-musl"
  cat >"$zipdir/yazi-x86_64-unknown-linux-musl/yazi" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$zipdir/yazi-x86_64-unknown-linux-musl/yazi"
  (cd "$zipdir" && zip -q -r "$STATE/yazi.zip" yazi-x86_64-unknown-linux-musl)
  export DC_YAZI_HOME="$STATE/tools" DC_YAZI_ZIP="$STATE/yazi.zip"
  dc-files --id app1 "$ws"
  grep -q 'cp .* /tmp/dc-cli-yazi' "$STATE/cp.log"
  grep -q '/tmp/dc-cli-yazi' "$STATE/exec.log"
  [[ -x "$STATE/tools/guest/amd64/yazi" ]]
}

case_inject_docker29_arch() {
  local ws zipdir
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    image=alpine \
    arch=arm64 \
    notoparch
  unset DC_FILES_NO_INJECT
  zipdir="$STATE/yz"
  mkdir -p "$zipdir/yazi-aarch64-unknown-linux-musl"
  cat >"$zipdir/yazi-aarch64-unknown-linux-musl/yazi" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$zipdir/yazi-aarch64-unknown-linux-musl/yazi"
  (cd "$zipdir" && zip -q -r "$STATE/yazi.zip" yazi-aarch64-unknown-linux-musl)
  export DC_YAZI_HOME="$STATE/tools" DC_YAZI_ZIP="$STATE/yazi.zip"
  dc-files --id app1 "$ws"
  grep -q 'cp .* /tmp/dc-cli-yazi' "$STATE/cp.log"
  [[ -x "$STATE/tools/guest/arm64/yazi" ]]
}

run_case help case_help
run_case nnn-opens case_nnn_opens
run_case empty-refuses case_empty_refuses
run_case service-db-empty case_service_db
run_case service-app-nnn case_service_app_nnn
run_case inject-linux-yazi case_inject_yazi
run_case inject-docker29-arch case_inject_docker29_arch

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
