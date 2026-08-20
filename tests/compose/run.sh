#!/usr/bin/env bash
# Compose frontend selection: docker compose vs docker-compose.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"

pass=0
fail=0
ran=0
pass() { echo "ok  $*"; pass=$((pass + 1)); }
fail() { echo "FAIL $*" >&2; fail=$((fail + 1)); }
run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  unset DC_COMPOSE_FRONTEND
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    fail "$name"
  fi
  harness_teardown
}

case_prefers_plugin() {
  unset DC_COMPOSE_FRONTEND
  assert_eq "$(dc_compose_frontend)" plugin
  dc_docker_compose version >/dev/null
}

case_standalone_fallback() {
  unset DC_COMPOSE_FRONTEND
  rm -f "$STATE/bin/docker"
  cat >"$STATE/bin/docker" <<'EOS'
#!/usr/bin/env bash
if [[ "${1:-}" == "compose" ]]; then
  echo "docker: 'compose' is not a docker command." >&2
  exit 1
fi
exit 0
EOS
  chmod +x "$STATE/bin/docker"
  cat >"$STATE/bin/docker-compose" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$STATE/compose.log"
if [[ "\$*" == *version* ]]; then
  echo "Docker Compose version standalone-fake"
  exit 0
fi
exit 0
EOF
  chmod +x "$STATE/bin/docker-compose"
  hash -r 2>/dev/null || true
  assert_eq "$(dc_compose_frontend)" standalone
  dc_docker_compose -p demo stop
  grep -q -- '-p demo stop' "$STATE/compose.log"
}

case_neither_errors() {
  unset DC_COMPOSE_FRONTEND
  rm -f "$STATE/bin/docker" "$STATE/bin/docker-compose"
  cat >"$STATE/bin/docker" <<'EOS'
#!/usr/bin/env bash
echo "no compose" >&2
exit 1
EOS
  chmod +x "$STATE/bin/docker"
  # Hide host docker-compose (Homebrew) for this case only.
  export PATH="$STATE/bin:/usr/bin:/bin"
  hash -r 2>/dev/null || true
  set +e
  out="$(dc_docker_compose -p x stop 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -qi 'docker compose or docker-compose'
}

run_case prefers_plugin case_prefers_plugin
run_case standalone_fallback case_standalone_fallback
run_case neither_errors case_neither_errors

echo "compose: $pass/$ran passed"
[[ "$fail" -eq 0 ]]
