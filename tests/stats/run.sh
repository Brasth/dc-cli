#!/usr/bin/env bash
# dc-stats contract + lie-tests (fake Docker).
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
  stub_colima_stopped
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

stub_colima_stopped() {
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
echo "PROFILE    STATUS     ARCH       CPUS    MEMORY    DISK      RUNTIME    ADDRESS"
exit 0
EOF
  chmod +x "$STATE/bin/colima"
}

stub_colima_running() {
  cat >"$STATE/bin/colima" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  list)
    echo "PROFILE    STATUS     ARCH       CPUS    MEMORY    DISK      RUNTIME    ADDRESS"
    echo "default    Running    aarch64    4       8GiB      60GiB     docker"
    exit 0
    ;;
  ssh)
    shift
    [[ "${1:-}" == "--" ]] && shift
    if [[ "${1:-}" == "cat" && "${2:-}" == "/proc/meminfo" ]]; then
      printf '%s\n' "MemTotal:        8000000 kB" "MemAvailable:    2000000 kB"
      exit 0
    fi
    if [[ "${1:-}" == "nproc" ]]; then
      echo 4
      exit 0
    fi
    if [[ "${1:-}" == "awk" ]]; then
      echo 0.80
      exit 0
    fi
    exit 1
    ;;
  *) exit 0 ;;
esac
EOF
  chmod +x "$STATE/bin/colima"
}

seed_ws() {
  local ws="$1"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{}' >"$ws/.devcontainer/devcontainer.json"
}

seed_stack() {
  local ws="$1"
  seed_ws "$ws"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=app" \
    image=node:22 \
    'stats={"CPUPerc":"12.40%","ID":"app1","MemUsage":"410MiB / 0B","Name":"app-1","NetIO":"1.23kB / 456B"}'
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projst" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    'stats={"CPUPerc":"0.30%","ID":"db1","MemUsage":"128MiB / 512MiB","Name":"db-1","NetIO":"0B / 0B"}'
  fake_add_container fwd1 dc-fwd-3000 running \
    "com.docker.compose.project=projst" \
    "dc.forward.for=app1" \
    image=alpine/socat \
    command="TCP-LISTEN:3000,fork" \
    'stats={"CPUPerc":"99.00%","ID":"fwd1","MemUsage":"1MiB / 0B","Name":"dc-fwd-3000","NetIO":"9MB / 9MB"}'
}

case_help() {
  dc-stats --help | grep -q 'read-only'
}

case_bad_flag() {
  set +e
  dc-stats --nope >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_not_dir() {
  set +e
  dc-stats /not/a/dir >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 2
}

case_json_schema() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  out="$(dc-stats --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null
  printf '%s\n' "$out" | grep -q '"schemaVersion": 1\|"schemaVersion":1'
  printf '%s\n' "$out" | grep -q '"engine"'
  printf '%s\n' "$out" | grep -q '"guest"'
  printf '%s\n' "$out" | grep -q '"containers"'
}

case_desktop_no_live() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  printf '%s\n' "Docker Desktop" >"$STATE/info_os"
  printf '%s\n' "docker-desktop" >"$STATE/info_name"
  out="$(dc-stats --json "$ws")"
  python3 -c '
import json,sys
o=json.loads(sys.argv[1])
assert o["engine"]=="desktop", o["engine"]
g=o["guest"]
assert g["live"] is False
assert "cpuPct" not in g
assert "memoryUsedBytes" not in g
assert g.get("cpus") == 4
assert g.get("memoryBytes") == 8589934592
' "$out"
}

case_unlimited_mem() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  out="$(dc-stats --json "$ws")"
  python3 -c '
import json,sys
o=json.loads(sys.argv[1])
app=next(c for c in o["containers"] if c["service"]=="app")
assert app["memLimitBytes"]==0, app
assert app["memUsedBytes"]>0, app
' "$out"
  human="$(dc-stats "$ws")"
  printf '%s\n' "$human" | grep -q ' / —'
}

case_sidecar_omitted() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  out="$(dc-stats --json "$ws")"
  python3 -c '
import json,sys
o=json.loads(sys.argv[1])
ids=[c["id"] for c in o["containers"]]
assert "fwd1" not in ids, ids
assert "app1" in ids
assert "db1" in ids
assert not any("socat" in (c.get("name") or "") for c in o["containers"])
' "$out"
}

case_stats_timeout() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  : >"$STATE/fail/stats_hang"
  set +e
  DC_STATS_TIMEOUT=1 dc-stats --json "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_zero_mutation() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  dc-stats --json "$ws" >/dev/null
  ! grep -E '^(run|stop|rm|prune|compose)( |$)' "$FAKE_DOCKER_LOG"
}

case_colima_live() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  stub_colima_running
  out="$(dc-stats --json "$ws")"
  python3 -c '
import json,sys
o=json.loads(sys.argv[1])
assert o["engine"]=="colima", o["engine"]
g=o["guest"]
assert g["label"]=="colima"
assert g["live"] is True
assert "cpuPct" in g
assert g["memoryUsedBytes"]>0
assert g["cpus"]==4
' "$out"
}

case_linux_host() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_stack "$ws"
  printf '%s\n' "Alpine Linux" >"$STATE/info_os"
  printf '%s\n' "linuxkit" >"$STATE/info_name"
  cat >"$STATE/bin/uname" <<'EOF'
#!/usr/bin/env bash
echo Linux
EOF
  chmod +x "$STATE/bin/uname"
  cat >"$STATE/bin/nproc" <<'EOF'
#!/usr/bin/env bash
echo 2
EOF
  chmod +x "$STATE/bin/nproc"
  mkdir -p "$STATE/proc"
  printf '%s\n' "MemTotal:        4000000 kB" "MemAvailable:    1000000 kB" >"$STATE/proc/meminfo"
  printf '%s\n' "1.00 0.50 0.25 1/100 1" >"$STATE/proc/loadavg"
  out="$(DC_STATS_PROC_ROOT="$STATE/proc" dc-stats --json "$ws")"
  python3 -c '
import json,sys
o=json.loads(sys.argv[1])
assert o["engine"]=="linux", o["engine"]
g=o["guest"]
assert g["label"]=="host"
assert g["live"] is True
assert g["memoryUsedBytes"]>0
assert g["cpus"]==2
' "$out"
}

run_case help case_help
run_case bad-flag case_bad_flag
run_case not-dir case_not_dir
run_case json-schema case_json_schema
run_case desktop-no-live case_desktop_no_live
run_case unlimited-mem case_unlimited_mem
run_case sidecar-omitted case_sidecar_omitted
run_case stats-timeout case_stats_timeout
run_case zero-mutation case_zero_mutation
run_case colima-live case_colima_live
run_case linux-host case_linux_host

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "FAILED $FAILED / $ran" >&2
  exit 1
fi
echo "ok $ran / $ran"
