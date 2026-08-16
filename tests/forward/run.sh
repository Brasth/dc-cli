#!/usr/bin/env bash
# Phase 4 forwarding reconcile gates (fake Docker).
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

seed_app() {
  local ws="$1" id="${2:-appB}"
  mkdir -p "$ws/.devcontainer"
  printf '%s\n' '{"forwardPorts":[3000,5173]}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container "$id" "$id" running \
    "devcontainer.local_folder=$ws" \
    "com.docker.compose.project=projB" \
    image=app \
    network=netB \
    ip=10.0.0.8
}

case_converge_and_idempotent() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_app "$ws"
  dc-forward "$ws" >/dev/null
  log_has 'run '
  [[ -d "$STATE/containers/dc-fwd-appB-3000" ]]
  [[ -d "$STATE/containers/dc-fwd-appB-5173" ]]
  grep -q 'dc.forward.owner=dc-cli' "$STATE/containers/dc-fwd-appB-3000/labels"
  grep -q 'dc.forward.workspace=' "$STATE/containers/dc-fwd-appB-3000/labels"
  : >"$FAKE_DOCKER_LOG"
  dc-forward "$ws" >/dev/null
  log_lacks '^run '
  log_lacks '^rm '
}

case_asymmetric() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container appB appB running \
    "devcontainer.local_folder=$ws" network=netB ip=10.0.0.8
  dc-forward "$ws" 9001:80 >/dev/null
  grep -q 'dc.forward.container=80' "$STATE/containers/dc-fwd-appB-9001/labels"
  grep -q 'TCP:10.0.0.8:80' "$STATE/containers/dc-fwd-appB-9001/command"
}

case_app_pub_pair_vs_host_only() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container appB appB running \
    "devcontainer.local_folder=$ws" \
    network=netB ip=10.0.0.8 \
    "ports=0.0.0.0:9001->80/tcp"
  set +e
  out="$(dc-forward "$ws" 9001:9001 2>&1)"
  rc=$?
  set -e
  # Host-only app publish must not count as pair satisfaction.
  ! printf '%s\n' "$out" | grep -q 'already published on the app'
  # Honest fail (port held by app) or a sidecar for the desired pair.
  if [[ "$rc" -eq 0 ]]; then
    [[ -d "$STATE/containers/dc-fwd-appB-9001" ]]
    grep -q 'dc.forward.container=9001' "$STATE/containers/dc-fwd-appB-9001/labels"
  fi
}

case_duplicate_labels() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  fake_add_container app1 app1 running "devcontainer.local_folder=$ws"
  fake_add_container app2 app2 running "devcontainer.local_folder=$ws"
  set +e
  dc-forward "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  log_lacks '^run '
  log_lacks '^rm '
}

case_inspect_unknown() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_app "$ws"
  fake_add_container orphanO dc-fwd-dead-3000 running \
    "dc.forward.for=ghostT" \
    "dc.forward.host=3000" \
    image=alpine/socat \
    "command=TCP-LISTEN:3000,fork,reuseaddr TCP:1.2.3.4:3000" \
    "ports=0.0.0.0:3000->3000/tcp"
  fake_fail_inspect ghostT
  set +e
  dc-forward "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ -d "$STATE/containers/orphanO" ]]
}

case_live_target_spoof() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_app "$ws"
  # Live app id, forward labels, but not a real sidecar (wrong image/name/command).
  fake_add_container spoofLive evil-live running \
    "dc.forward.for=appB" \
    "dc.forward.host=5173" \
    image=nginx \
    "ports=0.0.0.0:5173->5173/tcp"
  set +e
  dc-forward --stop "$ws" >/dev/null 2>&1
  dc-forward "$ws" >/dev/null 2>&1
  set -e
  [[ -d "$STATE/containers/spoofLive" ]]
  log_lacks 'rm spoofLive'
  log_lacks 'rm -f spoofLive'
}

case_spoof() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_app "$ws"
  fake_add_container spoofX evil running \
    "dc.forward.for=deadT" \
    "dc.forward.host=3000" \
    image=nginx \
    "ports=0.0.0.0:3000->3000/tcp"
  set +e
  dc-forward "$ws" >/dev/null 2>&1
  rc=$?
  set -e
  [[ -d "$STATE/containers/spoofX" ]]
  [[ "$rc" -ne 0 ]]
}

case_explicit_nondrop() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  seed_app "$ws"
  dc-forward "$ws" >/dev/null
  dc-forward "$ws" 8080 >/dev/null
  [[ -d "$STATE/containers/dc-fwd-appB-3000" ]]
  [[ -d "$STATE/containers/dc-fwd-appB-8080" ]]
}

case_up_forward_fail() {
  local ws
  ws="$(mktemp -d "$STATE/ws.XXXX")"
  mkdir -p "$ws/.devcontainer"
  echo '{}' >"$ws/.devcontainer/devcontainer.json"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  # no labeled app after up → forward fails
  set +e
  out="$(dc-up --no-forward "$ws" 2>&1; echo x)"
  set -e
  set +e
  out="$(dc-up "$ws" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'forwarding incomplete\|No matching'
}

echo "== forward gates =="
run_case "converge + idempotent" case_converge_and_idempotent
run_case "asymmetric 9001:80" case_asymmetric
run_case "app-published host-only mismatch" case_app_pub_pair_vs_host_only
run_case "duplicate labels fail closed" case_duplicate_labels
run_case "inspect-unknown no orphan rm" case_inspect_unknown
run_case "spoofed-label no rm" case_spoof
run_case "live-target spoof no rm" case_live_target_spoof
run_case "explicit mode does not drop others" case_explicit_nondrop
run_case "dc-up surfaces forward failure" case_up_forward_fail

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
