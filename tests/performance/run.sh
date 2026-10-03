#!/usr/bin/env bash
# Stack discovery cost at 1 / 10 / 50 containers (fake-docker).
# Records docker inspect count, total docker calls, and latency. Fails when
# inspects exceed 50% of the pre-batch baseline (measured at fd01797).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"

FAILED=0
ran=0
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }

# Baseline inspects per scenario and size: "scenario:n=count".
baseline() {
  case "$1:$2" in
    devcontainer-stack:1) echo 17 ;;
    devcontainer-stack:10) echo 52 ;;
    devcontainer-stack:50) echo 252 ;;
    compose-stack:1) echo 6 ;;
    compose-stack:10) echo 60 ;;
    compose-stack:50) echo 300 ;;
    compose-ls:1) echo 5 ;;
    compose-ls:10) echo 50 ;;
    compose-ls:50) echo 250 ;;
    *) echo 0 ;;
  esac
}

now_ms() {
  python3 -c 'import time; print(int(time.time() * 1000))'
}

seed_devcontainer() {
  local ws="$1" n="$2" i
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app-1 running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=perf" \
    "com.docker.compose.service=app"
  for ((i = 2; i <= n; i++)); do
    fake_add_container "svc$i" "svc-$i" running \
      "com.docker.compose.project=perf" \
      "com.docker.compose.service=svc$i"
  done
  # A forward sidecar in the same project must stay excluded.
  fake_add_container fwd1 dc-fwd-perf running \
    "com.docker.compose.project=perf" \
    "dc.forward.for=app1"
}

seed_compose() {
  local ws="$1" n="$2" i proj
  printf 'services:\n  app:\n    image: alpine\n' >"$ws/compose.yaml"
  proj="$(basename "$ws")"
  for ((i = 1; i <= n; i++)); do
    fake_add_container "c$i" "c-$i" running \
      "com.docker.compose.project=$proj" \
      "com.docker.compose.service=s$i" \
      "com.docker.compose.project.working_dir=$ws"
  done
}

# measure SCENARIO N EXPECTED_ROWS -- cmd...
measure() {
  local scenario="$1" n="$2" want_rows="$3"
  shift 4
  local t0 t1 out inspects calls rows base
  : >"$FAKE_DOCKER_LOG"
  t0="$(now_ms)"
  out="$("$@")"
  t1="$(now_ms)"
  inspects="$(grep -c '^inspect' "$FAKE_DOCKER_LOG" || true)"
  calls="$(grep -c . "$FAKE_DOCKER_LOG" || true)"
  rows="$(grep -o '"id":' <<<"$out" | wc -l | tr -d ' ')"
  base="$(baseline "$scenario" "$n")"
  printf '  %-19s n=%-3s inspects=%-3s baseline=%-4s docker-calls=%-3s latency=%sms\n' \
    "$scenario" "$n" "$inspects" "$base" "$calls" "$((t1 - t0))"
  ran=$((ran + 1))
  if [[ "$rows" -ne "$want_rows" ]]; then
    fail "$scenario n=$n: rows=$rows want $want_rows"
  fi
  if [[ $((inspects * 2)) -gt "$base" ]]; then
    fail "$scenario n=$n: $inspects inspects > 50% of baseline $base"
  fi
}

for n in 1 10 50; do
  harness_setup
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_devcontainer "$ws" "$n"
  measure devcontainer-stack "$n" "$n" -- dc-exec --list --json "$ws"
  harness_teardown

  harness_setup
  ws="$(mktemp -d "$STATE/perfc.XXXX")"
  seed_compose "$ws" "$n"
  measure compose-stack "$n" "$n" -- dc-exec --list --json "$ws"
  measure compose-ls "$n" "$n" -- dc-ls --json "$ws"
  harness_teardown
done

if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran" >&2
  exit 1
fi
echo "ok  $ran"
