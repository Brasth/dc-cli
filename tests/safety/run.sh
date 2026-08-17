#!/usr/bin/env bash
# Phase 1 safety behavioral gates (fake Docker, no daemon).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/tests/lib/harness.sh"

FAILED=0
ran=0

run_case() {
  local name="$1"
  shift
  ran=$((ran + 1))
  harness_setup
  if ( set -euo pipefail; "$@" ); then
    pass "$name"
  else
    echo "FAIL $name" >&2
    FAILED=$((FAILED + 1))
  fi
  harness_teardown
}

# --- cases ---

case_take_positive_foreign() {
  local a b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA" \
    "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
nfile="${DC_FAKE_STATE}/up.count"
n=0
[[ -f "$nfile" ]] && n="$(cat "$nfile")"
n=$((n + 1))
echo "$n" >"$nfile"
if [[ "$n" -eq 1 ]]; then
  echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
  exit 1
fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  dc-up --take-ports --no-forward "$b" >/dev/null
  log_has 'compose -p projA stop'
  log_lacks 'stop unrelated'
}

case_take_labeled_compose_in_devcontainer() {
  local a b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  mkdir -p "$a/.devcontainer"
  echo '{}' >"$a/.devcontainer/devcontainer.json"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA" \
    "com.docker.compose.project.working_dir=$a/.devcontainer" \
    "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
nfile="${DC_FAKE_STATE}/up.count"
n=0
[[ -f "$nfile" ]] && n="$(cat "$nfile")"
n=$((n + 1))
echo "$n" >"$nfile"
if [[ "$n" -eq 1 ]]; then
  echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
  exit 1
fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  dc-up --take-ports --no-forward "$b" >/dev/null
  log_has 'compose -p projA stop'
}

case_take_unlabeled() {
  local b
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container stray stray-box running "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --take-ports --no-forward "$b" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'unlabeled'
  log_lacks 'stop stray'
  log_lacks 'rm stray'
}

case_take_ambiguous_compose() {
  local a c b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  c="$(mktemp -d "$STATE/wsC.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=shared" \
    "ports=0.0.0.0:9001->9001/tcp"
  fake_add_container appC app-c running \
    "devcontainer.local_folder=$c" \
    "com.docker.compose.project=shared"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --take-ports --no-forward "$b" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'ambiguous-compose'
  log_lacks 'compose -p shared stop'

  set +e
  out="$(dc-down "$a" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'refuse compose-wide'
}

case_take_same_workspace() {
  local b
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appB app-b running \
    "devcontainer.local_folder=$b" \
    "com.docker.compose.project=projB" \
    "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --take-ports --no-forward "$b" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'same workspace'
  log_lacks 'compose -p projB stop'
}

case_take_sidecar() {
  local a b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA"
  fake_add_container fwdA dc-fwd-appa-9001 running \
    "dc.forward.for=appA" \
    "dc.forward.host=9001" \
    "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
nfile="${DC_FAKE_STATE}/up.count"
n=0
[[ -f "$nfile" ]] && n="$(cat "$nfile")"
n=$((n + 1))
echo "$n" >"$nfile"
if [[ "$n" -eq 1 ]]; then
  echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
  exit 1
fi
exit 0
EOF
  chmod +x "$STATE/bin/devcontainer"
  dc-up --take-ports --no-forward "$b" >/dev/null
  log_has 'compose -p projA stop'
}

case_take_sidecar_inspect_unknown() {
  local b
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container fwdX dc-fwd-x-9001 running \
    "dc.forward.for=ghostA" \
    "dc.forward.host=9001" \
    "ports=0.0.0.0:9001->9001/tcp"
  fake_fail_inspect ghostA
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --take-ports --no-forward "$b" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -qE 'sidecar-inspect-unknown|inspect-unknown'
  log_lacks 'rm fwdX'
  log_lacks 'stop fwdX'
}

case_nontty_no_flag() {
  local a b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA" \
    "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --no-forward "$b" </dev/null 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'non-interactive'
  log_lacks 'compose -p projA stop'
}

case_compose_revalidate() {
  local a
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA"
  # After snapshot, id is gone before down: simulate by failing inspect of appA
  # We revalidate at start of down; if we delete first, down should skip.
  rm -rf "$STATE/containers/appA"
  set +e
  out="$(dc-down "$a" 2>&1)"
  rc=$?
  set -e
  # no matching container → exit 0 (nothing to do)
  [[ "$rc" -eq 0 ]]
  printf '%s\n' "$out" | grep -q 'No matching'
}

case_prune_honesty() {
  out="$(dc-prune)"
  printf '%s\n' "$out" | grep -q '\[engine-wide\]'
  printf '%s\n' "$out" | grep -q '\[owned-only\]'
  printf '%s\n' "$out" | grep -q 'Engine-wide steps'
}

case_volume_protection() {
  fake_add_volume dbdata
  fake_add_container c1 c1 running mount=dbdata
  set +e
  out="$(dc-prune --volume dbdata --yes 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'mounted'
  [[ -d "$STATE/volumes/dbdata" ]]

  rm -rf "$STATE/containers/c1"
  out="$(dc-prune --volume dbdata)"
  printf '%s\n' "$out" | grep -q 'Would delete'
  [[ -d "$STATE/volumes/dbdata" ]]

  dc-prune --volume dbdata --yes >/dev/null
  [[ ! -d "$STATE/volumes/dbdata" ]]

  set +e
  out="$(dc-prune --volume missing --yes 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
}

case_volume_query_error() {
  fake_add_volume dbdata
  mkdir -p "$STATE/fail"
  : >"$STATE/fail/volume_ps"
  set +e
  out="$(dc-prune --volume dbdata --yes 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'volume-query-error'
  [[ -d "$STATE/volumes/dbdata" ]]
}

case_orphan_inspect_unknown() {
  fake_add_container fwdO dc-fwd-o running \
    "dc.forward.for=ghostT" \
    "dc.forward.host=3000"
  fake_fail_inspect ghostT
  set +e
  out="$(dc-prune --yes 2>&1)"
  rc=$?
  set -e
  [[ -d "$STATE/containers/fwdO" ]]
  # unknown target is not listed as orphan; apply should still succeed (no orphan step)
  # but if it were listed, we fail closed. dc_orphan_forward_ids must omit unknown.
  log_lacks 'rm -f fwdO'
  log_lacks 'rm fwdO'
}

case_orphan_absent_removed() {
  fake_add_container fwdO dc-fwd-o running \
    "dc.forward.for=deadT" \
    "dc.forward.host=3000"
  # no deadT container → inspect No such object → absent
  dc-prune --yes >/dev/null
  [[ ! -d "$STATE/containers/fwdO" ]]
}

case_prune_image_fail() {
  mkdir -p "$STATE/fail"
  : >"$STATE/fail/image_prune"
  set +e
  out="$(dc-prune --yes 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'step failed'
}

case_lock_concurrent() {
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  local key lockdir
  key="$(dc_engine_lock_key)"
  lockdir="$(dc_mutation_lock_root)/${key}.lock"
  mkdir -p "$lockdir"
  {
    printf 'pid=%s\n' "$$"
    printf 'start=%s\n' "$(dc_pid_start "$$")"
    printf 'engine=%s\n' "$key"
    printf 'token=held\n'
  } >"$lockdir/owner"
  set +e
  out="$(DC_MUTATION_LOCK_WAIT=1 dc_with_mutation_lock echo should-not 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'lock busy'
}

case_lock_nested() {
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  dc_with_mutation_lock bash -c '
    source "'"$ROOT"'/lib/dc-common.sh"
    dc_with_mutation_lock echo nested-ok
  ' | grep -qx 'nested-ok'
}

case_lock_dead_owner() {
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  local key lockdir
  key="$(dc_engine_lock_key)"
  lockdir="$(dc_mutation_lock_root)/${key}.lock"
  mkdir -p "$lockdir"
  {
    printf 'pid=9999999\n'
    printf 'start=Mon Jan  1 00:00:00 2000\n'
    printf 'engine=%s\n' "$key"
    printf 'token=dead\n'
  } >"$lockdir/owner"
  out="$(DC_MUTATION_LOCK_WAIT=1 dc_with_mutation_lock echo reclaimed)"
  [[ "$out" == "reclaimed" ]]
}

case_lock_age_not_enough() {
  # shellcheck source=/dev/null
  source "$ROOT/lib/dc-common.sh"
  local key lockdir
  key="$(dc_engine_lock_key)"
  lockdir="$(dc_mutation_lock_root)/${key}.lock"
  mkdir -p "$lockdir"
  {
    printf 'pid=%s\n' "$$"
    printf 'start=%s\n' "$(dc_pid_start "$$")"
    printf 'engine=%s\n' "$key"
    printf 'token=live\n'
  } >"$lockdir/owner"
  touch -t 200001010000 "$lockdir" "$lockdir/owner" 2>/dev/null || true
  set +e
  out="$(DC_MUTATION_LOCK_WAIT=1 dc_with_mutation_lock echo stolen 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -vq stolen || return 1
}

case_partial_mixed() {
  local a b
  a="$(mktemp -d "$STATE/wsA.XXXX")"
  b="$(mktemp -d "$STATE/wsB.XXXX")"
  fake_add_container appA app-a running \
    "devcontainer.local_folder=$a" \
    "com.docker.compose.project=projA" \
    "ports=0.0.0.0:9001->9001/tcp"
  fake_add_container stray stray-box running "ports=0.0.0.0:9001->9001/tcp"
  cat >"$STATE/bin/devcontainer" <<'EOF'
#!/usr/bin/env bash
nfile="${DC_FAKE_STATE}/up.count"
n=0
[[ -f "$nfile" ]] && n="$(cat "$nfile")"
n=$((n + 1))
echo "$n" >"$nfile"
# retry still blocked by unlabeled holder
echo "Bind for 0.0.0.0:9001 failed: port is already allocated" >&2
exit 1
EOF
  chmod +x "$STATE/bin/devcontainer"
  set +e
  out="$(dc-up --take-ports --no-forward "$b" 2>&1)"
  rc=$?
  set -e
  [[ "$rc" -ne 0 ]]
  printf '%s\n' "$out" | grep -q 'unlabeled'
  log_has 'compose -p projA stop'
  log_lacks 'stop stray'
}

echo "== safety gates =="
run_case "port takeover positive foreign" case_take_positive_foreign
run_case "port takeover labeled compose-in-devcontainer" case_take_labeled_compose_in_devcontainer
run_case "port takeover unlabeled" case_take_unlabeled
run_case "port takeover / down ambiguous compose" case_take_ambiguous_compose
run_case "port takeover same workspace" case_take_same_workspace
run_case "port takeover forward sidecar" case_take_sidecar
run_case "port takeover sidecar inspect-unknown" case_take_sidecar_inspect_unknown
run_case "nontty without flag does not stop" case_nontty_no_flag
run_case "compose revalidate missing id" case_compose_revalidate
run_case "prune scope honesty" case_prune_honesty
run_case "single-volume protection" case_volume_protection
run_case "volume query error" case_volume_query_error
run_case "orphan inspect-unknown no rm" case_orphan_inspect_unknown
run_case "orphan absent removed" case_orphan_absent_removed
run_case "prune image step failure" case_prune_image_fail
run_case "lock concurrent fail-closed" case_lock_concurrent
run_case "lock nested inherit" case_lock_nested
run_case "lock dead-owner reclaim" case_lock_dead_owner
run_case "lock age-alone never reclaim" case_lock_age_not_enough
run_case "partial mixed holders" case_partial_mixed

echo
if [[ "$FAILED" -ne 0 ]]; then
  echo "$FAILED/$ran failed"
  exit 1
fi
echo "$ran/$ran passed"
