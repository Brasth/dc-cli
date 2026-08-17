#!/usr/bin/env bash
# Compose-kind identity: detect, ls, doctor, dc-up refuse. No start.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-net.sh"

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

case_kind_detect() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  assert_eq "$(dc_workspace_kind "$ws")" none
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  assert_eq "$(dc_workspace_kind "$ws")" compose
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  assert_eq "$(dc_workspace_kind "$ws")" devcontainer
}

case_ls_compose_kind() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","networks":{}}' >"$STATE/compose-config.json"
  fake_add_container c1 web-1 running \
    "com.docker.compose.project=projk" \
    "com.docker.compose.project.working_dir=$ws" \
    "com.docker.compose.service=web"
  out="$(dc-ls --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null
  printf '%s\n' "$out" | grep -q '"kind":"compose"'
  printf '%s\n' "$out" | grep -q '"id":"c1"'
  printf '%s\n' "$out" | grep -qv 'devcontainer.local_folder'
  log_lacks 'commit'
}

case_ls_stopped_empty() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","networks":{}}' >"$STATE/compose-config.json"
  out="$(dc-ls --json "$ws")"
  assert_eq "$out" "[]"
}

case_ls_no_working_dir_unknown() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","networks":{}}' >"$STATE/compose-config.json"
  fake_add_container c1 web-1 running \
    "com.docker.compose.project=projk" \
    "com.docker.compose.service=web"
  out="$(dc-ls --json "$ws")"
  assert_eq "$out" "[]"
}

case_ls_all_labeled_only() {
  local ws other out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  other="$(mktemp -d "$STATE/other.XXXX")"
  mkdir -p "$other/.devcontainer"
  echo '{}' >"$other/.devcontainer/devcontainer.json"
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  printf '%s\n' '{"name":"projk","networks":{}}' >"$STATE/compose-config.json"
  fake_add_container c1 web-1 running \
    "com.docker.compose.project=projk" \
    "com.docker.compose.project.working_dir=$ws" \
    "com.docker.compose.service=web"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$other" \
    "com.docker.compose.project=lab"
  out="$(dc-ls --json --all)"
  printf '%s\n' "$out" | grep -q '"id":"app1"'
  printf '%s\n' "$out" | grep -q '"kind":"devcontainer"'
  printf '%s\n' "$out" | grep -qv '"id":"c1"'
}

case_ls_foreign_working_dir() {
  local ws other out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  other="$(mktemp -d "$STATE/other.XXXX")"
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  printf '%s\n' 'services: {}' >"$other/compose.yaml"
  printf '%s\n' '{"name":"same","networks":{}}' >"$STATE/compose-config.json"
  fake_add_container c1 web-1 running \
    "com.docker.compose.project=same" \
    "com.docker.compose.project.working_dir=$other" \
    "com.docker.compose.service=web"
  out="$(dc-ls --json "$ws")"
  assert_eq "$out" "[]"
}

case_up_compose_kind_refuse() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {web: {image: alpine}}' >"$ws/compose.yaml"
  cat >"$STATE/bin/devcontainer" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$STATE/devc.log"
exit 99
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q compose-kind
  [[ ! -f "$STATE/devc.log" ]]
  log_lacks 'compose .* up'
  log_lacks '^compose '
}

case_claimants_same_stack_not_double() {
  local ws n
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projA" \
    "com.docker.compose.project.working_dir=$ws/.devcontainer" \
    "com.docker.compose.service=app"
  n=0
  while IFS= read -r _; do
    [[ -n "${_:-}" ]] && n=$((n + 1))
  done < <(dc_compose_claimants projA)
  assert_eq "$n" 1
}

case_claimants_labeled_plus_foreign_compose() {
  local a other n
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  other="$(mktemp -d "$STATE/other.XXXX")"
  mkdir -p "$a/.devcontainer"
  echo '{}' >"$a/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=shared"
  fake_add_container web1 web-1 running \
    "com.docker.compose.project=shared" \
    "com.docker.compose.project.working_dir=$other" \
    "com.docker.compose.service=web"
  n=0
  while IFS= read -r _; do
    [[ -n "${_:-}" ]] && n=$((n + 1))
  done < <(dc_compose_claimants shared)
  assert_eq "$n" 2
}

case_doctor_compose_kind() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  printf '%s\n' 'services: {}' >"$ws/compose.yaml"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "--version" ]]; then echo "0.88.0"; exit 0; fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  out="$(dc-doctor --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null
  printf '%s\n' "$out" | grep -q '"kind": "compose"\|"kind":"compose"'
  printf '%s\n' "$out" | grep -q 'start disabled'
}

run_case kind-detect case_kind_detect
run_case ls-compose-kind case_ls_compose_kind
run_case ls-stopped-empty case_ls_stopped_empty
run_case ls-no-working-dir case_ls_no_working_dir_unknown
run_case ls-all-labeled-only case_ls_all_labeled_only
run_case ls-foreign-working-dir case_ls_foreign_working_dir
run_case up-compose-kind-refuse case_up_compose_kind_refuse
run_case claimants-same-stack case_claimants_same_stack_not_double
run_case claimants-labeled-plus-foreign case_claimants_labeled_plus_foreign_compose
run_case doctor-compose-kind case_doctor_compose_kind

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
