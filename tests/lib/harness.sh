#!/usr/bin/env bash
# Shared fake-Docker fixture helpers. Source from test runners.
# shellcheck shell=bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

harness_setup() {
  STATE="$(mktemp -d "${TMPDIR:-/tmp}/dc-fake.XXXXXX")"
  export DC_FAKE_STATE="$STATE"
  export FAKE_DOCKER_LOG="$STATE/docker.log"
  export DC_MUTATION_LOCK_ROOT="$STATE/locks"
  export DC_MUTATION_LOCK_WAIT="${DC_MUTATION_LOCK_WAIT:-2}"
  mkdir -p "$STATE/bin" "$STATE/fail" "$STATE/volumes" "$STATE/locks"
  ln -sf "$ROOT/tests/lib/fake-docker" "$STATE/bin/docker"
  chmod +x "$ROOT/tests/lib/fake-docker"
  export PATH="$STATE/bin:$ROOT/bin:$PATH"
  : >"$FAKE_DOCKER_LOG"
}

harness_teardown() {
  [[ -n "${STATE:-}" && -d "$STATE" ]] && rm -rf "$STATE"
}

fake_add_container() {
  local id="$1" name="$2" status="${3:-running}"
  shift 3 || true
  local dir="$STATE/containers/$id"
  mkdir -p "$dir"
  printf '%s\n' "$name" >"$dir/name"
  printf '%s\n' "$status" >"$dir/status"
  printf '%s\n' "alpine" >"$dir/image"
  printf '%s\n' "bridge" >"$dir/network"
  printf '%s\n' "10.0.0.2" >"$dir/ip"
  : >"$dir/labels"
  : >"$dir/ports"
  : >"$dir/mounts"
  local arg
  for arg in "$@"; do
    case "$arg" in
      ports=*) printf '%s\n' "${arg#ports=}" >"$dir/ports" ;;
      image=*) printf '%s\n' "${arg#image=}" >"$dir/image" ;;
      network=*) printf '%s\n' "${arg#network=}" >"$dir/network" ;;
      ip=*) printf '%s\n' "${arg#ip=}" >"$dir/ip" ;;
      command=*) printf '%s\n' "${arg#command=}" >"$dir/command" ;;
      mount=*) printf '%s\n' "${arg#mount=}" >>"$dir/mounts" ;;
      env=*) printf '%s\n' "${arg#env=}" >>"$dir/env" ;;
      exposed=*) printf '%s\n' "${arg#exposed=}" >>"$dir/exposed" ;;
      bin=*) printf '%s\n' "${arg#bin=}" >>"$dir/bins" ;;
      *) printf '%s\n' "$arg" >>"$dir/labels" ;;
    esac
  done
}

fake_add_volume() {
  mkdir -p "$STATE/volumes/$1"
}

fake_fail_inspect() {
  mkdir -p "$STATE/fail/inspect"
  : >"$STATE/fail/inspect/$1"
}

log_has() {
  grep -E -- "$1" "$FAKE_DOCKER_LOG" >/dev/null
}

log_lacks() {
  ! grep -E -- "$1" "$FAKE_DOCKER_LOG" >/dev/null
}

assert_eq() {
  local got="$1" want="$2" msg="${3:-assert_eq}"
  if [[ "$got" != "$want" ]]; then
    echo "FAIL $msg: got='$got' want='$want'" >&2
    return 1
  fi
}

pass() { echo "  ok  $*"; }
