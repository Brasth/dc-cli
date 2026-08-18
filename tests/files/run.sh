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

fake_editor() {
  local name="$1"
  cat >"$STATE/bin/$name" <<EOF
#!/bin/sh
printf '%s\n' "\$*" >>"${STATE}/${name}.log"
exit 0
EOF
  chmod +x "$STATE/bin/$name"
}

case_host_open_sets_editor() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  unset DC_FILES_EDITOR
  dc-files --id app1 "$ws"
  grep -q 'EDITOR=/tmp/dc-cli-open' "$STATE/exec.log"
  grep -q 'DC_OPEN_QUEUE=' "$STATE/exec.log"
  grep -q 'nnn' "$STATE/exec.log"
  grep -q '/tmp/dc-cli-open' "$STATE/cp.log"
  grep -q 'chmod 666' "$STATE/exec.log"
}

case_hatch_vim() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  DC_FILES_EDITOR=vim dc-files --id app1 "$ws"
  grep -q 'nnn' "$STATE/exec.log"
  ! grep -q 'EDITOR=/tmp/dc-cli-open' "$STATE/exec.log"
  [[ ! -f "$STATE/cp.log" ]] || ! grep -q '/tmp/dc-cli-open' "$STATE/cp.log"
}

case_code_file_uri() {
  local ws uri
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_editor code
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1 DC_FILES_WS="$ws"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host app1 /workspaces/app/README.md "$ws"
  grep -q -- '--file-uri' "$STATE/code.log"
  grep -q -- '--folder-uri' "$STATE/code.log"
  grep -q 'attached-container+' "$STATE/code.log"
  grep -q '/workspaces/app/README.md' "$STATE/code.log"
  uri="$(dc_attached_container_uri app1 /workspaces/app/README.md)"
  printf '%s\n' "$uri" | grep -q "$(dc_hex '{"containerName":"app-1"}')"
}

case_compose_kind_file_uri() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  fake_add_container web1 web-1 running \
    "com.docker.compose.project=projf" \
    "com.docker.compose.project.working_dir=$ws" \
    "com.docker.compose.service=web" \
    bin=nnn
  fake_editor code
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1 DC_FILES_WS="$ws"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host web1 /app/main.go "$ws"
  grep -q 'attached-container+' "$STATE/code.log"
  grep -q '/app/main.go' "$STATE/code.log"
}

case_attach_compose_na() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  out="$(dc-open --attach "$ws")"
  printf '%s\n' "$out" | grep -qi 'N/A'
}

case_host_fallback_mount() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_add_container app2 app-2 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projf" \
    "com.docker.compose.service=app" \
    "mount=${ws}:/workspaces/app" \
    bin=nnn
  fake_editor zed
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host app2 /workspaces/app/src/a.ts "$ws"
  grep -q "${ws}/src/a.ts" "$STATE/zed.log"
}

case_refuse_dotdot() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  set +e
  dc_files_open_host app1 /workspaces/app/../etc/passwd "$ws"
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_cursor_dev_container() {
  local ws hex
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_editor cursor
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1 DC_FILES_WS="$ws"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host app1 /workspaces/app/README.md "$ws"
  grep -q -- '--file-uri' "$STATE/cursor.log"
  grep -q 'dev-container+' "$STATE/cursor.log"
  ! grep -q 'attached-container+' "$STATE/cursor.log"
  hex="$(dc_hex "$(printf '{"hostPath":"%s"}' "$ws")")"
  grep -q "$hex" "$STATE/cursor.log"
  grep -q '/workspaces/app/README.md' "$STATE/cursor.log"
}

case_cursor_forced_over_code() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_editor code
  fake_editor cursor
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1 DC_FILES_EDITOR=cursor DC_FILES_WS="$ws"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host app1 /workspaces/app/a.ts "$ws"
  grep -q 'dev-container+' "$STATE/cursor.log"
  [[ ! -f "$STATE/code.log" ]]
}

case_cursor_compose_host_fallback() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  fake_add_container web1 web-1 running \
    "com.docker.compose.project=projf" \
    "com.docker.compose.project.working_dir=$ws" \
    "com.docker.compose.service=web" \
    "mount=${ws}:/app" \
    bin=nnn
  fake_editor cursor
  fake_editor zed
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1 DC_FILES_WS="$ws"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  dc_files_open_host web1 /app/main.go "$ws"
  [[ ! -f "$STATE/cursor.log" ]]
  grep -q "${ws}/main.go" "$STATE/zed.log"
}

case_host_fallback_unmapped() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_editor zed
  export PATH="$STATE/bin:$ROOT/bin:/usr/bin:/bin" DC_EDITOR_PATH_ONLY=1
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-files-open.sh"
  set +e
  dc_files_open_host app1 /etc/os-release "$ws"
  rc=$?
  set -e
  assert_eq "$rc" 1
  [[ ! -f "$STATE/zed.log" ]]
}

run_case help case_help
run_case nnn-opens case_nnn_opens
run_case empty-refuses case_empty_refuses
run_case service-db-empty case_service_db
run_case service-app-nnn case_service_app_nnn
run_case inject-linux-yazi case_inject_yazi
run_case inject-docker29-arch case_inject_docker29_arch
run_case host-open-sets-editor case_host_open_sets_editor
run_case hatch-vim case_hatch_vim
run_case code-file-uri case_code_file_uri
run_case compose-kind-file-uri case_compose_kind_file_uri
run_case attach-compose-na case_attach_compose_na
run_case host-fallback-mount case_host_fallback_mount
run_case refuse-dotdot case_refuse_dotdot
run_case cursor-dev-container case_cursor_dev_container
run_case cursor-forced-over-code case_cursor_forced_over_code
run_case cursor-compose-host-fallback case_cursor_compose_host_fallback
run_case host-fallback-unmapped case_host_fallback_unmapped

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
