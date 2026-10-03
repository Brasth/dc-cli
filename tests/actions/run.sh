#!/usr/bin/env bash
# dc-actions end to end: compiled binary → dc-exec --no-start → fake Docker.
# Never starts anything; trust gates shared actions; exit status propagates.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"

FAILED=0
ran=0
pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }

if ! command -v go >/dev/null 2>&1; then
  echo "  skip (no go: dc-actions is compiled)"
  exit 0
fi
BIN_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dc-actions-bin.XXXX")"
trap 'rm -rf "$BIN_DIR"' EXIT
(cd "$ROOT" && go build -o "$BIN_DIR/dc-actions" ./cmd/dc-actions)

run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  export XDG_CONFIG_HOME="$STATE/config" XDG_STATE_HOME="$STATE/state"
  cp "$BIN_DIR/dc-actions" "$STATE/bin/dc-actions"
  cat >"$STATE/bin/devcontainer" <<DC
#!/usr/bin/env bash
printf 'devcontainer %s\n' "\$*" >>"$STATE/devcontainer.log"
echo "dc-out"
exit "\${FAKE_DC_RC:-0}"
DC
  chmod +x "$STATE/bin/devcontainer"
  hash -r 2>/dev/null || true
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
    "com.docker.compose.project=projact" \
    "com.docker.compose.service=app"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projact" \
    "com.docker.compose.service=db"
}

canon() { (cd "$1" && pwd -P); }

personal() {
  local ws="$1" body="$2" h p
  if command -v sha256sum >/dev/null 2>&1; then
    h="$(printf '%s' "$(canon "$ws")" | sha256sum | awk '{print $1}')"
  else
    h="$(printf '%s' "$(canon "$ws")" | shasum -a 256 | awk '{print $1}')"
  fi
  p="$XDG_CONFIG_HOME/dc-cli/actions/$h.json"
  mkdir -p "$(dirname "$p")"
  printf '%s\n' "$body" >"$p"
}

shared() {
  mkdir -p "$1/.dc"
  printf '%s\n' "$2" >"$1/.dc/actions.json"
}

never_started() {
  log_lacks '^start '
  log_lacks '^restart '
  log_lacks '^compose .* (start|up)'
  ! grep -q 'dc-up' "$FAKE_DOCKER_LOG"
}

case_personal_app_exit_and_quoting() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  personal "$ws" '{"schemaVersion":1,"actions":[{"id":"t","label":"T","argv":["make","a b","$(x)"]}]}'
  set +e
  out="$(FAKE_DC_RC=9 dc-actions run t "$ws" 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -eq 9 ]]
  [[ "$out" == "dc-out" ]]
  grep -Fq -- 'exec --workspace-folder' "$STATE/devcontainer.log"
  grep -Fq -- 'make a b $(x)' "$STATE/devcontainer.log"
  never_started
}

case_service_via_docker_exec() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  personal "$ws" '{"schemaVersion":1,"actions":[{"id":"q","label":"Q","argv":["psql","-c","select 1"],"service":"db"}]}'
  dc-actions run q "$ws" >/dev/null 2>&1
  log_has '^exec .*db1 psql -c select 1$'
  never_started
}

case_stopped_target_refused() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  echo exited >"$STATE/containers/db1/status"
  echo exited >"$STATE/containers/app1/status"
  personal "$ws" '{"schemaVersion":1,"actions":[{"id":"q","label":"Q","argv":["true"],"service":"db"},{"id":"a","label":"A","argv":["true"]}]}'
  set +e
  dc-actions run q "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  set +e
  dc-actions run a "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  log_lacks '^exec '
  [[ ! -f "$STATE/devcontainer.log" ]]
  never_started
}

case_forged_service_refused() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  fake_add_container evil1 evil running \
    "com.docker.compose.project=elsewhere" \
    "com.docker.compose.service=evil"
  personal "$ws" '{"schemaVersion":1,"actions":[{"id":"x","label":"X","argv":["true"],"service":"evil"}]}'
  set +e
  dc-actions run x "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  log_lacks '^exec '
  never_started
}

case_shared_trust_gate() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  shared "$ws" '{"schemaVersion":1,"actions":[{"id":"lint","label":"Lint","argv":["make","lint"]}]}'
  set +e
  dc-actions run lint "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  [[ ! -f "$STATE/devcontainer.log" ]]
  # Non-TTY trust without --yes: preview on stdout, refuse with 2.
  set +e
  out="$(dc-actions trust "$ws" </dev/null 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  printf '%s\n' "$out" | grep -q 'argv:   make lint'
  dc-actions trust "$ws" --yes >/dev/null 2>&1
  dc-actions run lint "$ws" >/dev/null 2>&1
  grep -Fq 'make lint' "$STATE/devcontainer.log"
  # Any edit revokes.
  shared "$ws" '{"schemaVersion":1,"actions":[{"id":"lint","label":"Lint","argv":["make","lint2"]}]}'
  set +e
  dc-actions run lint "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  ! grep -Fq 'make lint2' "$STATE/devcontainer.log"
}

case_list_json_precedence() {
  local ws out rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed "$ws"
  personal "$ws" '{"schemaVersion":1,"actions":[{"id":"lint","label":"Mine","argv":["true"]}]}'
  shared "$ws" '{"schemaVersion":1,"actions":[{"id":"lint","label":"Team","argv":["false"]},{"id":"db","label":"DB","argv":["psql"],"service":"db"}]}'
  out="$(dc-actions list "$ws" --json 2>/dev/null)"
  python3 - "$out" "$(canon "$ws")" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
assert d["schemaVersion"] == 1 and d["workspace"] == sys.argv[2], d
a = d["actions"]
assert [x["id"] for x in a] == ["lint", "db"], a
assert a[0]["label"] == "Mine" and a[0]["source"] == "personal" and a[0]["enabled"] is True
assert a[1]["source"] == "shared" and a[1]["enabled"] is False and a[1]["service"] == "db"
PY
  shared "$ws" '{"schemaVersion":1,"actions":[{"id":"bad"}]}'
  set +e
  out="$(dc-actions list "$ws" --json 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -eq 2 ]]
  printf '%s\n' "$out" | grep -q '"Mine"'
  dc-actions run lint "$ws" >/dev/null 2>&1
}

case_dc_dispatch_fallback() {
  # Repo bin/ has the shell fallback: help works, run explains compiled is needed.
  local rc out
  "$ROOT/bin/dc" actions --help | grep -q 'dc-actions trust'
  set +e
  out="$("$ROOT/bin/dc" actions list 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -q 'compiled dc-actions is required'
}

run_case personal-app-exit-quoting case_personal_app_exit_and_quoting
run_case service-docker-exec case_service_via_docker_exec
run_case stopped-target-refused case_stopped_target_refused
run_case forged-service-refused case_forged_service_refused
run_case shared-trust-gate case_shared_trust_gate
run_case list-json-precedence case_list_json_precedence
run_case dc-dispatch-fallback case_dc_dispatch_fallback

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
