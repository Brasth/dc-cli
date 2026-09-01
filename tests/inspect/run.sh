#!/usr/bin/env bash
# dc-inspect: read-only agent snapshot. No Docker required where possible.
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

INS="$ROOT/bin/dc-inspect"
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
  # Recover libs after harness_setup (inspect composes recover diagnose/plan).
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-recover.sh"
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

case_help() {
  "$INS" --help | grep -q 'dc-inspect'
  "$INS" --help | grep -q 'Never applies'
}

case_unknown_flag() {
  local rc
  set +e
  "$INS" --nope >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
  set +e
  "$INS" --yes >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_missing_dir() {
  local rc
  set +e
  "$INS" /not/a/dir >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_json_schema() {
  local ws out rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$("$INS" --json "$ws" 2>/dev/null)"
  rc=$?
  set -e
  [[ "$rc" -eq 0 || "$rc" -eq 1 ]]
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
assert d["schemaVersion"]==1
assert d["command"]=="dc-inspect"
assert "workspace" in d and "host" in d and "next" in d and "stack" in d
ws=d["workspace"]
for k in ("requested","resolved","kind","hasConfig"):
    assert k in ws, k
assert ws["kind"]=="none"
assert ws["hasConfig"] is False
assert isinstance(d["stack"], list)
host=d["host"]
for k in ("status","code","summary","detail","engineHint","guideUrl"):
    assert k in host, k
nxt=d["next"]
for k in ("id","summary","command","apply","applyAllowed","verify","escalate"):
    assert k in nxt, k
' "$out"
}

case_stack_array_no_secrets() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$("$INS" --json "$ws" 2>/dev/null)"
  set -e
  python3 -c '
import json,sys
d=json.loads(sys.argv[1])
assert isinstance(d["stack"], list)
def walk(o):
    if isinstance(o, dict):
        for k,v in o.items():
            kl=k.lower()
            assert "password" not in kl, k
            assert "secret" not in kl, k
            assert "token" not in kl, k
            walk(v)
    elif isinstance(o, list):
        for v in o:
            walk(v)
walk(d)
' "$out"
}

case_does_not_apply() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
echo ran >>"${DC_FAKE_STATE}/colima.log"
exit 0
EOF
  chmod +x "$STATE/bin/colima"
  cat >"$STATE/bin/dc-try" <<'EOF'
#!/usr/bin/env bash
echo "dc-try $*" >>"${DC_FAKE_STATE}/dc-try.log"
exit 0
EOF
  chmod +x "$STATE/bin/dc-try"
  hash -r 2>/dev/null || true
  set +e
  "$INS" --json "$ws" >/dev/null 2>&1
  set -e
  [[ ! -f "$STATE/colima.log" ]]
  [[ ! -f "$STATE/dc-try.log" ]]
  if [[ -f "$FAKE_DOCKER_LOG" ]]; then
    ! grep -Eq '(^| )(run|create|start|stop|rm|rmi|commit|install)( |$)' "$FAKE_DOCKER_LOG"
    ! grep -Eq 'compose (up|down)' "$FAKE_DOCKER_LOG"
  fi
}

case_dispatch() {
  "$ROOT/bin/dc" inspect --help | grep -q 'dc-inspect'
}

case_human() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  set +e
  out="$("$INS" "$ws" 2>/dev/null)"
  set -e
  printf '%s\n' "$out" | grep -q 'dc-inspect'
  printf '%s\n' "$out" | grep -q 'kind=none'
  printf '%s\n' "$out" | grep -q 'dc-exec'
  ! printf '%s\n' "$out" | grep -q 'apply with'
}

echo "== inspect gates =="
run_case "help exit 0" case_help
run_case "unknown flag exit 2" case_unknown_flag
run_case "non-dir exit 2" case_missing_dir
run_case "json schema workspace/host/next/stack" case_json_schema
run_case "stack is array; no secret fields" case_stack_array_no_secrets
run_case "does not apply" case_does_not_apply
run_case "dc inspect dispatches" case_dispatch
run_case "human points at dc-exec" case_human

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "inspect: $FAILED failed" >&2
  exit 1
fi
echo "$ran/$ran passed"
