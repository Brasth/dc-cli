#!/usr/bin/env bash
# dc-db list + declared-port open gates (fake Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-common.sh"
# shellcheck source=/dev/null
source "$ROOT/lib/dc-db.sh"

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

seed_ws() {
  local ws="$1"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{"forwardPorts":[3000]}' >"$ws/.devcontainer/devcontainer.json"
}

seed_app() {
  local ws="$1" id="${2:-app1}"
  fake_add_container "$id" "$id" running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=app" \
    image=node:22 \
    network=netdb \
    ip=10.0.0.8
}

write_compose() {
  local ws="$1"
  cat >"$ws/.devcontainer/docker-compose.yml" <<'YAML'
services:
  app:
    ports:
      - "3000:3000"
  db:
    ports:
      - "5433:5432"
YAML
}

case_help() {
  dc-db --help | grep -q 'declared'
}

case_version() {
  out="$(DC_CLI_VERSION=0.10.0 dc-db --version)"
  printf '%s\n' "$out" | grep -q 'dc-db'
  printf '%s\n' "$out" | grep -q '0.10.0'
}

case_list_postgres_redact() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    network=netdb \
    ip=10.0.0.9 \
    env=POSTGRES_USER=postgres \
    env=POSTGRES_PASSWORD=s3cret \
    env=POSTGRES_DB=app \
    exposed=5432/tcp \
    "ports=0.0.0.0:5433->5432/tcp"
  out="$(dc-db --list "$ws")"
  printf '%s\n' "$out" | grep -q postgres
  printf '%s\n' "$out" | grep -q 5433
  ! printf '%s\n' "$out" | grep -q s3cret
  printf '%s\n' "$out" | grep -q postgres
}

case_json_no_password() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    env=POSTGRES_PASSWORD=s3cret \
    exposed=5432/tcp \
    "ports=0.0.0.0:5433->5432/tcp"
  out="$(dc-db --list --json "$ws")"
  python3 -m json.tool <<<"$out" >/dev/null
  ! printf '%s\n' "$out" | grep -qi password
  ! printf '%s\n' "$out" | grep -q s3cret
  out="$(dc-db --list --json --secrets "$ws")"
  printf '%s\n' "$out" | grep -q s3cret
}

case_no_db() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_app "$ws"
  set +e
  dc-db --list "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  assert_eq "$rc" 1
}

case_cache_not_redis() {
  local ws out rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_app "$ws"
  fake_add_container cache1 cache-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=cache" \
    image=memcached:latest
  set +e
  out="$(dc-db --list "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  ! printf '%s\n' "$out" | grep -qi redis
}

case_print_declared_live() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    env=POSTGRES_USER=postgres \
    env=POSTGRES_PASSWORD=s3cret \
    env=POSTGRES_DB=app \
    exposed=5432/tcp \
    "ports=0.0.0.0:5433->5432/tcp"
  out="$(dc-db --print "$ws")"
  printf '%s\n' "$out" | grep -q '127.0.0.1:5433'
  ! printf '%s\n' "$out" | grep -q ':5432/'
  ! printf '%s\n' "$out" | grep -q s3cret
  printf '%s\n' "$out" | grep -q ':\*\*\*\*@'
  log_lacks '^run '
}

case_print_secrets() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    env=POSTGRES_USER=postgres \
    env=POSTGRES_PASSWORD=s3cret \
    env=POSTGRES_DB=app \
    exposed=5432/tcp \
    "ports=0.0.0.0:5433->5432/tcp"
  out="$(dc-db --print --secrets "$ws")"
  printf '%s\n' "$out" | grep -q s3cret
}

case_unpublished_forwards_db() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    network=netdb \
    ip=10.0.0.9 \
    env=POSTGRES_USER=postgres \
    env=POSTGRES_DB=app \
    exposed=5432/tcp
  out="$(dc-db --print "$ws")"
  printf '%s\n' "$out" | grep -q '127.0.0.1:5433'
  log_has 'run '
  grep -q 'dc.forward.container=5432' "$STATE/containers/dc-fwd-db1-5433/labels"
  grep -q 'dc.forward.for=db1' "$STATE/containers/dc-fwd-db1-5433/labels"
}

case_forwardports_unpublished() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{"forwardPorts":[5432]}' >"$ws/.devcontainer/devcontainer.json"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    network=netdb \
    ip=10.0.0.9 \
    exposed=5432/tcp
  out="$(dc-db --print "$ws")"
  printf '%s\n' "$out" | grep -q '127.0.0.1:5432'
  grep -q 'dc.forward.for=db1' "$STATE/containers/dc-fwd-db1-5432/labels"
}

case_missing_declared() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    exposed=5432/tcp
  set +e
  out="$(dc-db --print "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -qi 'declared'
  log_lacks '^run '
}

case_two_need_service() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  cat >"$ws/.devcontainer/docker-compose.yml" <<'YAML'
services:
  db:
    ports: ["5433:5432"]
  mysql:
    ports: ["3307:3306"]
YAML
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    "ports=0.0.0.0:5433->5432/tcp"
  fake_add_container my1 my-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=mysql" \
    image=mysql:8 \
    "ports=0.0.0.0:3307->3306/tcp"
  set +e
  out="$(dc-db --print "$ws" 2>&1)"
  rc=$?
  set -e
  assert_eq "$rc" 1
  printf '%s\n' "$out" | grep -q -- '--service'
  out="$(dc-db --print --service mysql "$ws")"
  printf '%s\n' "$out" | grep -q '127.0.0.1:3307'
}

case_auto_forward_ignores_db() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    network=netdb \
    ip=10.0.0.9
  dc-forward "$ws" >/dev/null
  [[ -d "$STATE/containers/dc-fwd-app1-3000" ]]
  [[ ! -d "$STATE/containers/dc-fwd-app1-5433" ]]
  [[ ! -d "$STATE/containers/dc-fwd-db1-5433" ]]
}

case_inspect_fail_closed() {
  local ws rc
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  write_compose "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16
  fake_fail_inspect db1
  set +e
  dc-db --list "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
}

case_inline_compose_ports() {
  local ws out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  cat >"$ws/.devcontainer/docker-compose.yml" <<'YAML'
services:
  db:
    ports: ["5433:5432"]
YAML
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    network=netdb \
    ip=10.0.0.9 \
    exposed=5432/tcp
  out="$(dc-db --print "$ws")"
  printf '%s\n' "$out" | grep -q '127.0.0.1:5433'
}

case_live_not_declared() {
  local ws rc out
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_ws "$ws"
  seed_app "$ws"
  fake_add_container db1 db-1 running \
    "com.docker.compose.project=projdb" \
    "com.docker.compose.service=db" \
    image=postgres:16 \
    exposed=5432/tcp \
    "ports=0.0.0.0:32768->5432/tcp"
  set +e
  out="$(dc-db --print "$ws" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -eq 1 ]]
  printf '%s\n' "$out" | grep -qi 'declared'
  ! printf '%s\n' "$out" | grep -q 32768
}

case_classify_unit() {
  assert_eq "$(dc_db_classify postgres:16 db '')" postgres
  assert_eq "$(dc_db_classify mysql:8 db '')" mysql
  assert_eq "$(dc_db_classify alpine cache '')" ""
  assert_eq "$(dc_db_classify alpine cache 6379/tcp)" redis
}

run_case help case_help
run_case version case_version
run_case list-postgres-redact case_list_postgres_redact
run_case json-no-password case_json_no_password
run_case no-db case_no_db
run_case cache-not-redis case_cache_not_redis
run_case print-declared-live case_print_declared_live
run_case print-secrets case_print_secrets
run_case unpublished-forwards-db case_unpublished_forwards_db
run_case forwardports-unpublished case_forwardports_unpublished
run_case missing-declared case_missing_declared
run_case two-need-service case_two_need_service
run_case auto-forward-ignores-db case_auto_forward_ignores_db
run_case inspect-fail-closed case_inspect_fail_closed
run_case inline-compose-ports case_inline_compose_ports
run_case live-not-declared case_live_not_declared
run_case classify-unit case_classify_unit

echo
if [[ "$FAILED" -gt 0 ]]; then
  echo "FAILED $FAILED / $ran"
  exit 1
fi
echo "ok  $ran"
