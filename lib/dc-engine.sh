# Read-only Docker engine identity. Source after docker is on PATH.
# Classify the CLI target. Never use `colima list` as the source of truth.
# shellcheck shell=bash

DC_ENGINE_PROBE_TIMEOUT="${DC_ENGINE_PROBE_TIMEOUT:-2}"

dc_engine_run_deadline() {
  local secs="$1"
  shift
  local pid killer rc=0
  "$@" &
  pid=$!
  (
    sleep "$secs"
    kill "$pid" 2>/dev/null || true
    sleep 0.05
    kill -9 "$pid" 2>/dev/null || true
  ) &
  killer=$!
  set +e
  wait "$pid"
  rc=$?
  kill "$killer" 2>/dev/null
  wait "$killer" 2>/dev/null
  set -e
  return "$rc"
}

dc_engine_home() {
  printf '%s\n' "${DC_ENGINE_HOME:-$HOME}"
}

# unix://path | tcp://... → comparable host string
dc_engine_norm_host() {
  local h="$1"
  [[ -n "$h" ]] || return 1
  if [[ "$h" == unix://* ]]; then
    printf '%s\n' "$h"
    return 0
  fi
  if [[ "$h" == /* ]]; then
    printf 'unix://%s\n' "$h"
    return 0
  fi
  printf '%s\n' "$h"
}

# Follow symlink. Empty if not a unix path or missing.
dc_engine_sock_path() {
  local host="$1" path
  host="$(dc_engine_norm_host "$host" 2>/dev/null || true)"
  [[ "$host" == unix://* ]] || return 1
  path="${host#unix://}"
  [[ -e "$path" ]] || return 1
  printf '%s\n' "$path"
}

dc_engine_realpath() {
  local path="$1"
  if command -v realpath >/dev/null 2>&1; then
    realpath "$path" 2>/dev/null && return 0
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$path" 2>/dev/null && return 0
  fi
  printf '%s\n' "$path"
}

# Device:inode of the target (follow symlink). Empty if unavailable.
dc_engine_inode() {
  local path="$1"
  if stat -L -f '%d:%i' "$path" >/dev/null 2>&1; then
    stat -L -f '%d:%i' "$path"
    return 0
  fi
  if stat -c '%d:%i' "$path" >/dev/null 2>&1; then
    stat -c '%d:%i' "$path"
    return 0
  fi
  return 1
}

# Dedup key: same Desktop daemon via /var/run and ~/.docker/run is one key.
# Colima's default profile publishes two sockets (not a symlink).
dc_engine_identity_key() {
  local host="$1" path ino rp
  host="$(dc_engine_norm_host "$host")" || return 1
  path="$(dc_engine_sock_path "$host" 2>/dev/null || true)"
  [[ -n "$path" ]] && rp="$(dc_engine_realpath "$path")"
  [[ -z "${rp:-}" ]] && rp="${host#unix://}"
  case "$rp" in
    */.colima/docker.sock|*/.colima/default/docker.sock)
      printf 'colima-default\n'
      return 0
      ;;
  esac
  if [[ -n "$path" ]]; then
    ino="$(dc_engine_inode "$path" || true)"
    if [[ -n "$ino" ]]; then
      printf 'ino:%s\n' "$ino"
      return 0
    fi
    printf 'path:%s\n' "${rp:-$path}"
    return 0
  fi
  printf 'host:%s\n' "$host"
}

dc_engine_cli_host() {
  local host ctx
  if [[ -n "${DOCKER_HOST:-}" ]]; then
    dc_engine_norm_host "$DOCKER_HOST"
    return 0
  fi
  if command -v docker >/dev/null 2>&1; then
    ctx="$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
    if [[ -n "$ctx" ]]; then
      dc_engine_norm_host "$ctx"
      return 0
    fi
  fi
  printf '%s\n' "unix:///var/run/docker.sock"
}

dc_engine_context_name() {
  local ctx
  if command -v docker >/dev/null 2>&1; then
    ctx="$(docker context show 2>/dev/null || true)"
    if [[ -z "$ctx" ]]; then
      ctx="$(docker context inspect --format '{{.Name}}' 2>/dev/null || true)"
    fi
  fi
  printf '%s\n' "${ctx:-unknown}"
}

# colima | desktop | orbstack | linux | unknown
dc_engine_classify() {
  local host="$1"
  local path rp
  [[ -n "$host" ]] || {
    printf '%s\n' unknown
    return 0
  }
  path="${host#unix://}"
  if [[ "$host" == unix://* && -e "$path" ]]; then
    rp="$(dc_engine_realpath "$path")"
    path="$rp"
    host="unix://${rp}"
  fi
  if [[ "$host" == *desktop-linux* || "$path" == *"/.docker/run/docker.sock" ]]; then
    printf '%s\n' desktop
    return 0
  fi
  if [[ "$path" == *"/.colima/"* || "$path" == *"/.colima/docker.sock" ]]; then
    printf '%s\n' colima
    return 0
  fi
  if [[ "$path" == *"/.orbstack/"* ]]; then
    printf '%s\n' orbstack
    return 0
  fi
  if [[ "$path" == /var/run/docker.sock ]]; then
    if [[ "$(uname -s 2>/dev/null || true)" == Linux ]]; then
      printf '%s\n' linux
    else
      printf '%s\n' desktop
    fi
    return 0
  fi
  printf '%s\n' unknown
}

dc_engine_probe_live() {
  local host="$1"
  local secs="${DC_ENGINE_PROBE_TIMEOUT:-2}"
  [[ -n "$host" ]] || return 1
  command -v docker >/dev/null 2>&1 || return 1
  dc_engine_run_deadline "$secs" env DOCKER_HOST="$host" docker info >/dev/null 2>&1
}

# CLI host first, then extra sockets. Dedup by inode so /var/run aliases collapse.
dc_engine_known_hosts() {
  local home sock host cli key
  local -A seen=()
  home="$(dc_engine_home)"
  cli="$(dc_engine_cli_host)"
  if [[ -n "$cli" ]]; then
    key="$(dc_engine_identity_key "$cli")"
    seen["$key"]=1
    printf '%s\n' "$cli"
  fi
  for sock in "$home"/.colima/*/docker.sock "$home/.colima/docker.sock" \
    "$home/.docker/run/docker.sock" "$home/.orbstack/run/docker.sock"; do
    [[ -e "$sock" ]] || continue
    host="$(dc_engine_norm_host "$sock")"
    key="$(dc_engine_identity_key "$host")"
    [[ -n "${seen[$key]:-}" ]] && continue
    seen["$key"]=1
    printf '%s\n' "$host"
  done
  # /var/run is a Desktop/Colima alias. Tests set DC_ENGINE_HOME and skip it.
  if [[ -z "${DC_ENGINE_HOME:-}" && -e /var/run/docker.sock ]]; then
    host="$(dc_engine_norm_host /var/run/docker.sock)"
    key="$(dc_engine_identity_key "$host")"
    if [[ -z "${seen[$key]:-}" ]]; then
      printf '%s\n' "$host"
    fi
  fi
}

# TSV: engine<TAB>context<TAB>host<TAB>dockerHostSet<TAB>extraLive(comma)
dc_engine_report() {
  local cli ctx engine dset=0 extra_csv="" host eng
  local -A extra=()
  cli="$(dc_engine_cli_host)"
  ctx="$(dc_engine_context_name)"
  engine="$(dc_engine_classify "$cli")"
  [[ -n "${DOCKER_HOST:-}" ]] && dset=1
  local cli_key
  cli_key="$(dc_engine_identity_key "$cli")"
  while IFS= read -r host; do
    [[ -n "$host" ]] || continue
    [[ "$(dc_engine_identity_key "$host")" == "$cli_key" ]] && continue
    if dc_engine_probe_live "$host"; then
      eng="$(dc_engine_classify "$host")"
      extra["$eng"]=1
    fi
  done < <(dc_engine_known_hosts)
  if [[ ${#extra[@]} -gt 0 ]]; then
    extra_csv="$(printf '%s\n' "${!extra[@]}" | LC_ALL=C sort | paste -sd, -)"
  fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$engine" "$ctx" "$cli" "$dset" "$extra_csv"
}

# 0 if extraLive nonempty
dc_engine_split() {
  local extra
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  [[ -n "$extra" ]]
}

dc_engine_remediation() {
  printf '%s\n' "unset DOCKER_HOST; docker context use desktop-linux or colima; stop the extra engine (colima stop / quit Docker Desktop)"
}

# dc-up: refuse split unless DC_UP_ALLOW_SPLIT=1. Sets DC_ENGINE_LAST_*.
dc_engine_guard_up() {
  local engine ctx host dset extra
  IFS=$'\t' read -r engine ctx host dset extra < <(dc_engine_report)
  DC_ENGINE_LAST_ENGINE="$engine"
  DC_ENGINE_LAST_CONTEXT="$ctx"
  DC_ENGINE_LAST_HOST="$host"
  if [[ -z "$extra" ]]; then
    return 0
  fi
  if [[ "${DC_UP_ALLOW_SPLIT:-}" == "1" ]]; then
    echo "dc-up: warning: split-brain (CLI=$engine extra=$extra); DC_UP_ALLOW_SPLIT=1 continuing" >&2
    return 0
  fi
  echo "dc-up: split-brain — more than one Docker engine is live." >&2
  echo "  CLI    $engine  context=$ctx" >&2
  echo "  socket $host" >&2
  echo "  extra  $extra" >&2
  echo "  $(dc_engine_remediation)" >&2
  echo "  or: DC_UP_ALLOW_SPLIT=1 dc-up   (not recommended)" >&2
  return 1
}

dc_engine_up_line() {
  printf 'dc-up: engine %s context=%s\n' \
    "${DC_ENGINE_LAST_ENGINE:-unknown}" \
    "${DC_ENGINE_LAST_CONTEXT:-unknown}"
}
