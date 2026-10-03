#!/usr/bin/env bash
# Live workspace-hub smoke (opt-in). Needs a running engine, the official
# devcontainer CLI, Go, and a local image (never pulled).
#
#   DC_LIVE_SMOKE=1 bash tests/workspace-hub-smoke/run.sh
#   DC_SMOKE_IMAGE=node:22-bookworm-slim (default)
#   DC_SMOKE_DIR=$HOME/.cache (default; must be shared with the engine VM;
#                may be inside a Git repo — discovery is capped at the temp root)
#
# Creates only its own temp folders and containers: unique compose project
# names, devcontainer.local_folder under its temp dir, and the label
# dc.smoke.run=<run id> on everything else. Cleanup removes only those.
# XDG config/state/data point into the temp dir, so personal actions, trust
# and the workspace registry of the real user are never read or written.
set -euo pipefail

if [[ "${DC_LIVE_SMOKE:-}" != "1" ]]; then
  echo "skip: set DC_LIVE_SMOKE=1 (live engine; creates and removes its own containers only)"
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE="${DC_SMOKE_IMAGE:-node:22-bookworm-slim}"
RUN="dcsmoke$(date +%s)$$"
FAILED=0
ran=0

pass() { echo "  ok  $*"; }
fail() { echo "FAIL $*" >&2; FAILED=$((FAILED + 1)); }
check() {
  local name="$1"
  shift
  ran=$((ran + 1))
  if "$@"; then pass "$name"; else fail "$name"; fi
}

for tool in docker go devcontainer; do
  command -v "$tool" >/dev/null 2>&1 || { echo "smoke needs $tool on PATH" >&2; exit 1; }
done
docker info >/dev/null 2>&1 || { echo "no running Docker engine" >&2; exit 1; }
docker image inspect "$IMAGE" >/dev/null 2>&1 || {
  echo "image $IMAGE is not present locally; this smoke never pulls" >&2
  exit 1
}

# Under $HOME by default: Colima / Desktop only share the home folder with
# the VM, and the devcontainer bind-mounts its folder.
SMOKE_BASE="${DC_SMOKE_DIR:-$HOME/.cache}"
mkdir -p "$SMOKE_BASE"
TMP="$(mktemp -d "$SMOKE_BASE/dc-cli-smoke-XXXXXX")"
TMP="$(cd "$TMP" && pwd -P)"
# DC_SMOKE_DIR may sit inside a Git checkout (even this one): Git discovery
# must never climb out of the temp root, or workspace resolution would pick
# the enclosing repo instead of the temporary folders.
export GIT_CEILING_DIRECTORIES="$TMP"
# An inherited explicit repo location would bypass discovery entirely.
unset GIT_DIR GIT_WORK_TREE GIT_COMMON_DIR GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_NAMESPACE
WS_C="$TMP/compose-ws"
WS_D="$TMP/${RUN}-dev"
WS_T="$TMP/${RUN}-try"
PROJ_C="${RUN}-c"
ORIG_XDG_STATE="${XDG_STATE_HOME:-$HOME/.local/state}"
export XDG_CONFIG_HOME="$TMP/config" XDG_STATE_HOME="$TMP/state" XDG_DATA_HOME="$TMP/data"
export DC_TUI_NO_SPLASH=1

cleanup() {
  set +e
  # Only what this run created: its compose project, its devcontainer folder,
  # and containers carrying its run label.
  local ids id img
  ids="$( {
    docker ps -aq --filter "label=com.docker.compose.project=${PROJ_C}"
    docker ps -aq --filter "label=devcontainer.local_folder=${WS_D}"
    docker ps -aq --filter "label=devcontainer.local_folder=${WS_T}"
    docker ps -aq --filter "label=dc.smoke.run=${RUN}"
  } 2>/dev/null | sort -u)"
  # dc-forward sidecars of those apps (if any were made).
  for id in $ids; do
    docker ps -aq --no-trunc --filter "label=dc.forward.for=$(docker inspect -f '{{.Id}}' "$id" 2>/dev/null)" 2>/dev/null
  done >"$TMP/sidecars" 2>/dev/null
  ids="$(printf '%s\n%s\n' "$ids" "$(cat "$TMP/sidecars" 2>/dev/null)" | awk 'NF' | sort -u)"
  if [[ -n "$ids" ]]; then
    # shellcheck disable=SC2086
    docker rm -f $ids >/dev/null 2>&1
  fi
  docker network rm "${PROJ_C}_default" >/dev/null 2>&1
  # Images the official CLI built for this run's folders only (vsc-<run>-…).
  for img in $(docker images -q --filter "reference=vsc-${RUN}-*" 2>/dev/null); do
    docker rmi -f "$img" >/dev/null 2>&1
  done
  chmod -R u+w "$TMP" 2>/dev/null
  rm -rf "$TMP"
  left="$( {
    docker ps -aq --filter "label=com.docker.compose.project=${PROJ_C}"
    docker ps -aq --filter "label=dc.smoke.run=${RUN}"
  } 2>/dev/null)"
  [[ -z "$left" ]] || echo "WARN smoke containers left: $left" >&2
}
trap cleanup EXIT

mkdir -p "$TMP/bin" "$WS_C" "$WS_D/.devcontainer" "$WS_T"
# The real user's sandbox overrides must stay untouched (XDG is redirected).
REAL_TRY="$ORIG_XDG_STATE/dc-cli/try"
real_try_sig() { if [[ -d "$REAL_TRY" ]]; then find "$REAL_TRY" -type f -exec shasum {} + 2>/dev/null | sort; else echo none; fi; }
real_try_sig >"$TMP/real-try-before"
docker images -q --no-trunc | sort -u >"$TMP/images-before"
(cd "$ROOT" && go build -o "$TMP/bin/dc-actions" ./cmd/dc-actions)
# Compiled dc-actions first, then this checkout's wrappers.
export PATH="$TMP/bin:$ROOT/bin:$PATH"

# Hard stop before any container step: every smoke folder must resolve to
# itself (never an enclosing repo such as this checkout).
for ws in "$WS_C" "$WS_D" "$WS_T"; do
  got="$(bash -c 'source "$1/lib/dc-common.sh"; dc_resolve_workspace "$2"' _ "$ROOT" "$ws")"
  if [[ "$got" != "$ws" ]]; then
    echo "smoke isolation: $ws resolves to $got — refusing to run" >&2
    exit 1
  fi
  if git -C "$ws" rev-parse --show-toplevel >/dev/null 2>&1; then
    echo "smoke isolation: $ws is inside a Git work tree — refusing to run" >&2
    exit 1
  fi
done

cat >"$WS_C/compose.yaml" <<YAML
name: ${PROJ_C}
services:
  app:
    image: ${IMAGE}
    command: ["sleep", "infinity"]
    init: true
    labels:
      dc.smoke.run: "${RUN}"
YAML

cat >"$WS_D/.devcontainer/devcontainer.json" <<JSON
{
  "name": "${RUN}-d",
  "image": "${IMAGE}",
  "runArgs": ["--label", "dc.smoke.run=${RUN}"]
}
JSON

personal_actions() {
  local ws="$1" body="$2" sum
  sum="$(printf '%s' "$ws" | shasum -a 256 | cut -d' ' -f1)"
  mkdir -p "$XDG_CONFIG_HOME/dc-cli/actions"
  printf '%s\n' "$body" >"$XDG_CONFIG_HOME/dc-cli/actions/${sum}.json"
}

state_of() { docker inspect -f '{{.State.Status}} {{.State.StartedAt}}' "$1" 2>/dev/null || echo missing; }

expect_rc() {
  local want="$1" rc
  shift
  set +e
  "$@" >"$TMP/out" 2>"$TMP/err"
  rc=$?
  set -e
  [[ "$rc" -eq "$want" ]] || { echo "    rc=$rc want $want: $(tail -3 "$TMP/err")" >&2; return 1; }
}

echo "workspace-hub smoke  run=$RUN  engine=$(docker context show 2>/dev/null)  image=$IMAGE"

# --- compose-kind -----------------------------------------------------------
echo "compose-kind: $WS_C"
dc-up "$WS_C" >"$TMP/up-c.log" 2>&1 || { cat "$TMP/up-c.log" >&2; fail "dc-up compose-kind"; exit 1; }
cid="$(docker ps -q --filter "label=com.docker.compose.project=${PROJ_C}" --filter "label=com.docker.compose.service=app")"
check "compose-kind app running" test -n "$cid"

personal_actions "$WS_C" '{"schemaVersion":1,"actions":[{"id":"probe","label":"Probe","argv":["sh","-c","echo smoke-ok; exit 7"],"service":"app"},{"id":"-dash","label":"Dash","argv":["true"],"service":"app"}]}'
check "action runs in a running service, exit propagates" expect_rc 7 dc-actions run probe "$WS_C"
check "action stdout is the child's" grep -qx smoke-ok "$TMP/out"
check "id starting with - runs after --" expect_rc 0 dc-actions run -- -dash "$WS_C"

dc-down "$WS_C" >/dev/null 2>&1
before="$(state_of "$cid")"
check "compose-kind stopped" test "${before%% *}" = exited
check "action on a stopped service refuses (exit 1)" expect_rc 1 dc-actions run probe "$WS_C"
check "refused action started nothing" test "$(state_of "$cid")" = "$before"

# --- devcontainer -----------------------------------------------------------
echo "devcontainer: $WS_D"
dc-up --no-forward "$WS_D" >"$TMP/up-d.log" 2>&1 || { tail -20 "$TMP/up-d.log" >&2; fail "dc-up devcontainer"; exit 1; }
did="$(docker ps -q --filter "label=devcontainer.local_folder=${WS_D}")"
check "devcontainer app running" test -n "$did"
personal_actions "$WS_D" '{"schemaVersion":1,"actions":[{"id":"probe","label":"Probe","argv":["sh","-c","echo dev-ok; exit 3"]}]}'
check "action runs in the labeled app (devcontainer exec)" expect_rc 3 dc-actions run probe "$WS_D"
check "devcontainer action stdout" grep -qx dev-ok "$TMP/out"
dc-down "$WS_D" >/dev/null 2>&1
before="$(state_of "$did")"
check "devcontainer stopped" test "${before%% *}" = exited
check "action on a stopped app refuses (exit 1)" expect_rc 1 dc-actions run probe "$WS_D"
check "refused action started nothing (devcontainer)" test "$(state_of "$did")" = "$before"

# --- sandbox (dc-try, kind=none) ---------------------------------------------
echo "sandbox: $WS_T"
DC_TRY_IMAGE_GENERIC="$IMAGE" dc-try --profile generic --yes --no-forward "$WS_T" >"$TMP/try.log" 2>&1 ||
  { tail -20 "$TMP/try.log" >&2; fail "dc-try sandbox"; exit 1; }
tid="$(docker ps -q --filter "label=devcontainer.local_folder=${WS_T}")"
check "sandbox running" test -n "$tid"
check "sandbox has no project config written" test ! -e "$WS_T/.devcontainer" -a ! -e "$WS_T/.devcontainer.json"
override="$(ls "$XDG_STATE_HOME"/dc-cli/try/*/override.json 2>/dev/null | head -1)"
check "sandbox override lives in the isolated XDG state" test -n "$override"
check "sandbox override uses the local image" grep -q "\"image\": \"${IMAGE}\"" "$override"
sandbox_members() {
  local out
  out="$(dc-ls --json --workspace "$WS_T")"
  [[ "$(printf '%s' "$out" | grep -o '"id": *"' | wc -l | tr -d ' ')" == 1 ]] && printf '%s' "$out" | grep -q "$tid"
}
check "sandbox membership is exactly its container" sandbox_members
personal_actions "$WS_T" '{"schemaVersion":1,"actions":[{"id":"probe","label":"Probe","argv":["sh","-c","echo try-ok; exit 5"]}]}'
check "action runs in the sandbox" expect_rc 5 dc-actions run probe "$WS_T"
check "sandbox action stdout" grep -qx try-ok "$TMP/out"

echo "activity: live docker events against the sandbox"
if (cd "$ROOT" && DC_LIVE_ACTIVITY_WS="$WS_T" DC_LIVE_ACTIVITY_KIND=none DC_LIVE_RUN="$RUN" DC_LIVE_IMAGE="$IMAGE" \
  go test -count=1 -run '^TestLiveActivity$' -v ./cmd/dc-tui) >"$TMP/live-try.log" 2>&1; then
  pass "activity (sandbox): known app restart, foreign labeled app excluded, switch reaps"
  grep -E "live timeline" "$TMP/live-try.log" | sed 's/^/    /' || true
else
  fail "activity live test (sandbox)"
  tail -30 "$TMP/live-try.log" >&2
fi
ran=$((ran + 1))

dc-down "$WS_T" >/dev/null 2>&1
before="$(state_of "$tid")"
check "sandbox stopped" test "${before%% *}" = exited
check "action on a stopped sandbox refuses (exit 1)" expect_rc 1 dc-actions run probe "$WS_T"
check "refused action started nothing (sandbox)" test "$(state_of "$tid")" = "$before"
real_try_sig >"$TMP/real-try-after"
check "real sandbox overrides unchanged" cmp -s "$TMP/real-try-before" "$TMP/real-try-after"

# --- board activity stream (real engine, Go test hooks) ---------------------
echo "activity: live docker events against $WS_C"
dc-up "$WS_C" >/dev/null 2>&1
live() {
  (cd "$ROOT" && DC_LIVE_ACTIVITY_WS="$WS_C" DC_LIVE_RUN="$RUN" DC_LIVE_IMAGE="$IMAGE" \
    go test -count=1 -run '^TestLiveActivity$' -v ./cmd/dc-tui) >"$TMP/live.log" 2>&1
}
if live; then
  pass "activity: pinned stream, known restart, verified new sibling, forged + sidecar excluded, switch reaps"
  grep -E "live timeline" "$TMP/live.log" | sed 's/^/    /' || true
else
  fail "activity live test"
  tail -30 "$TMP/live.log" >&2
fi
ran=$((ran + 1))

# Nothing may have been pulled (no forward sidecar image, no base image).
docker images -q --no-trunc | sort -u >"$TMP/images-after"
check "no image pulled" test -z "$(comm -13 "$TMP/images-before" "$TMP/images-after")"

echo
if [[ "$FAILED" -eq 0 ]]; then
  echo "workspace-hub smoke: ${ran}/${ran} passed"
else
  echo "workspace-hub smoke: ${FAILED} of ${ran} failed" >&2
  exit 1
fi
