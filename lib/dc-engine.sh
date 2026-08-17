# Read-only Docker engine identity. Source after docker is on PATH.
# Classify the CLI target. Never use `colima list` as the source of truth.
# shellcheck shell=bash

DC_ENGINE_PROBE_TIMEOUT="${DC_ENGINE_PROBE_TIMEOUT:-2}"

dc_engine_run_deadline() {
  local secs="$1"
  shift
  local pid killer rc=0 max
  "$@" &
  pid=$!
  max=$((secs * 20))
  [[ "$max" -gt 0 ]] || max=1
  (
    n=0
    while [[ "$n" -lt "$max" ]]; do
      kill -0 "$pid" 2>/dev/null || exit 0
      sleep 0.05
      n=$((n + 1))
    done
    kill "$pid" 2>/dev/null || true
    sleep 0.05
    kill -9 "$pid" 2>/dev/null || true
  ) &
  killer=$!
  set +e
  wait "$pid"
  rc=$?
  kill "$killer" 2>/dev/null || true
  wait "$killer" 2>/dev/null || true
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

# Device:inode of the target (must follow symlink). Empty if unavailable.
# GNU `stat -f` is --file-system. GNU `stat -c` without -L is the link itself.
dc_engine_inode() {
  local path="$1" out
  out="$(stat -L -c '%d:%i' "$path" 2>/dev/null || true)"
  if [[ "$out" == [0-9]*:[0-9]* ]]; then
    printf '%s\n' "$out"
    return 0
  fi
  out="$(stat -L -f '%d:%i' "$path" 2>/dev/null || true)"
  if [[ "$out" == [0-9]*:[0-9]* ]]; then
    printf '%s\n' "$out"
    return 0
  fi
  return 1
}

# Dedup key: same Desktop daemon via /var/run and ~/.docker/run is one key.
# Colima's default profile publishes two sockets (not a symlink).
# Linux Desktop publishes ~/.docker/desktop/docker.sock (not ~/.docker/run).
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
    */.docker/desktop/docker.sock|*/.docker/desktop/docker.sock.raw)
      printf 'desktop-linux\n'
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
  case "$path" in
    */.docker/run/docker.sock|*/.docker/desktop/docker.sock|*/.docker/desktop/docker.sock.raw)
      printf '%s\n' desktop
      return 0
      ;;
  esac
  if [[ "$host" == *desktop-linux* ]]; then
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
  if [[ "$path" == /var/run/docker.sock || "$path" == /run/docker.sock ]]; then
    if [[ "$(uname -s 2>/dev/null || true)" == Linux ]]; then
      printf '%s\n' linux
    else
      printf '%s\n' desktop
    fi
    return 0
  fi
  printf '%s\n' unknown
}

# stdout: daemon ID (may be empty). 0 if docker info answered in time.
dc_engine_probe_info() {
  local host="$1"
  local secs="${DC_ENGINE_PROBE_TIMEOUT:-2}"
  local out rc=0
  [[ -n "$host" ]] || return 1
  command -v docker >/dev/null 2>&1 || return 1
  set +e
  out="$(dc_engine_run_deadline "$secs" env DOCKER_HOST="$host" docker info --format '{{.ID}}' 2>/dev/null)"
  rc=$?
  set -e
  out="${out//$'\r'/}"
  out="${out%%$'\n'*}"
  [[ "$out" == "<no value>" ]] && out=""
  printf '%s\n' "$out"
  return "$rc"
}

dc_engine_probe_live() {
  dc_engine_probe_info "$1" >/dev/null
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
    "$home/.docker/run/docker.sock" "$home/.docker/desktop/docker.sock" \
    "$home/.orbstack/run/docker.sock"; do
    [[ -e "$sock" ]] || continue
    host="$(dc_engine_norm_host "$sock")"
    key="$(dc_engine_identity_key "$host")"
    [[ -n "${seen[$key]:-}" ]] && continue
    seen["$key"]=1
    printf '%s\n' "$host"
  done
  # System sockets. Tests set DC_ENGINE_HOME and skip them.
  if [[ -z "${DC_ENGINE_HOME:-}" ]]; then
    for sock in /var/run/docker.sock /run/docker.sock; do
      [[ -e "$sock" ]] || continue
      host="$(dc_engine_norm_host "$sock")"
      key="$(dc_engine_identity_key "$host")"
      [[ -n "${seen[$key]:-}" ]] && continue
      seen["$key"]=1
      printf '%s\n' "$host"
    done
  fi
}

# TSV: engine<TAB>context<TAB>host<TAB>dockerHostSet<TAB>extraLive(comma)
# Same daemon ID on two paths is one engine (Linux Desktop + /var/run proxy).
dc_engine_report() {
  local cli ctx engine dset=0 extra_csv="" host eng extra_id extra_rc
  local cli_key cli_id="" extra_hosts_csv=""
  local -A extra=()
  local -a extra_hosts=()
  cli="$(dc_engine_cli_host)"
  ctx="$(dc_engine_context_name)"
  engine="$(dc_engine_classify "$cli")"
  [[ -n "${DOCKER_HOST:-}" ]] && dset=1
  cli_key="$(dc_engine_identity_key "$cli")"
  while IFS= read -r host; do
    [[ -n "$host" ]] || continue
    [[ "$(dc_engine_identity_key "$host")" == "$cli_key" ]] && continue
    set +e
    extra_id="$(dc_engine_probe_info "$host")"
    extra_rc=$?
    set -e
    [[ "$extra_rc" -eq 0 ]] || continue
    if [[ -z "$cli_id" ]]; then
      cli_id="$(dc_engine_probe_info "$cli" 2>/dev/null || true)"
    fi
    if [[ -n "$cli_id" && -n "$extra_id" && "$cli_id" == "$extra_id" ]]; then
      continue
    fi
    eng="$(dc_engine_classify "$host")"
    extra["$eng"]=1
    extra_hosts+=("$host")
  done < <(dc_engine_known_hosts)
  if [[ ${#extra[@]} -gt 0 ]]; then
    extra_csv="$(printf '%s\n' "${!extra[@]}" | LC_ALL=C sort | paste -sd, -)"
  fi
  if [[ ${#extra_hosts[@]} -gt 0 ]]; then
    extra_hosts_csv="$(IFS=','; echo "${extra_hosts[*]}")"
  fi
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$engine" "$ctx" "$cli" "$dset" "$extra_csv" "$extra_hosts_csv"
}

# Capture TSV in this shell. $(report) / < <(report) run in a subshell and drop LAST_*.
dc_engine_refresh() {
  local engine ctx host dset extra extra_hosts
  IFS=$'\t' read -r engine ctx host dset extra extra_hosts < <(dc_engine_report)
  DC_ENGINE_LAST_ENGINE="${engine:-unknown}"
  DC_ENGINE_LAST_CONTEXT="${ctx:-unknown}"
  DC_ENGINE_LAST_HOST="${host:-}"
  DC_ENGINE_LAST_DSET="${dset:-0}"
  DC_ENGINE_LAST_EXTRA="${extra:-}"
  DC_ENGINE_LAST_EXTRA_HOSTS="${extra_hosts:-}"
}

# 0 if extraLive nonempty
dc_engine_split() {
  local extra
  extra="$(dc_engine_report | awk -F'\t' '{print $5}')"
  [[ -n "$extra" ]]
}

dc_engine_recommended_context() {
  case "${1:-${DC_ENGINE_LAST_ENGINE:-unknown}}" in
    desktop) printf '%s\n' desktop-linux ;;
    colima) printf '%s\n' colima ;;
    orbstack) printf '%s\n' orbstack ;;
    linux) printf '%s\n' default ;;
    *) printf '%s\n' "${DC_ENGINE_LAST_CONTEXT:-default}" ;;
  esac
}

# One-line JSON / compact doctor rem. Full recipe is dc_engine_fix_text.
dc_engine_remediation() {
  local engine="${DC_ENGINE_LAST_ENGINE:-unknown}"
  local extra="${DC_ENGINE_LAST_EXTRA:-}"
  local rec
  rec="$(dc_engine_recommended_context "$engine")"
  if [[ "$engine" == desktop && "$extra" == *linux* ]]; then
    printf '%s\n' "CLI is Docker Desktop; extra native dockerd on /var/run. Keep Desktop: sudo systemctl stop docker.socket docker.service; docker context use desktop-linux. Then dc-engine --fix. Or DC_UP_ALLOW_SPLIT=1 (not recommended)"
    return 0
  fi
  if [[ "$engine" == linux && "$extra" == *desktop* ]]; then
    printf '%s\n' "CLI is native dockerd; extra Docker Desktop is live. Keep native: quit Docker Desktop; docker context use default. Or keep Desktop: docker context use desktop-linux. Then dc-engine --fix"
    return 0
  fi
  printf '%s\n' "unset DOCKER_HOST; docker context use ${rec}; stop the extra engine (colima stop / quit Docker Desktop / sudo systemctl stop docker). Then dc-engine --fix"
}

# Human recipe. Never sudo, never stop an engine.
dc_engine_fix_text() {
  local engine="${DC_ENGINE_LAST_ENGINE:-unknown}"
  local ctx="${DC_ENGINE_LAST_CONTEXT:-unknown}"
  local host="${DC_ENGINE_LAST_HOST:-}"
  local extra="${DC_ENGINE_LAST_EXTRA:-}"
  local extra_hosts="${DC_ENGINE_LAST_EXTRA_HOSTS:-}"
  local dset="${DC_ENGINE_LAST_DSET:-0}"
  local rec
  rec="$(dc_engine_recommended_context "$engine")"
  if [[ -z "$extra" ]]; then
    echo "One live engine (${engine}, context=${ctx}). Nothing to fix."
    echo "  socket ${host}"
    echo "  dc-up from the project folder."
    return 0
  fi
  echo "Two live Docker engines. Pick one. dc-cli will not stop them for you."
  echo "  CLI    ${engine}  context=${ctx}"
  echo "  socket ${host}"
  echo "  extra  ${extra}${extra_hosts:+  ${extra_hosts}}"
  if [[ "$dset" == "1" ]]; then
    echo
    echo "  1. In this shell:  unset DOCKER_HOST"
  fi
  echo
  if [[ "$engine" == desktop && "$extra" == *linux* ]]; then
    echo "  Keep Docker Desktop (this CLI socket):"
    echo "    sudo systemctl stop docker.socket docker.service"
    echo "    sudo systemctl disable --now docker.socket docker.service"
    echo "    docker context use desktop-linux"
    echo
    echo "  Keep native dockerd instead:"
    echo "    quit Docker Desktop"
    echo "    docker context use default"
  elif [[ "$engine" == linux && "$extra" == *desktop* ]]; then
    echo "  Keep native dockerd (this CLI socket):"
    echo "    quit Docker Desktop"
    echo "    docker context use default"
    echo
    echo "  Keep Docker Desktop instead:"
    echo "    sudo systemctl stop docker.socket docker.service"
    echo "    docker context use desktop-linux"
  elif [[ "$extra" == *colima* || "$engine" == colima ]]; then
    echo "  Keep this CLI engine (${engine}):"
    echo "    docker context use ${rec}"
    echo "    stop the other: colima stop   or quit Docker Desktop"
  else
    echo "  docker context use ${rec}"
    echo "  stop the extra engine (colima stop / quit Docker Desktop / sudo systemctl stop docker)"
  fi
  echo
  echo "  Then: dc-doctor && dc-up"
  echo "  Hatch only: DC_UP_ALLOW_SPLIT=1 dc-up"
}

# Safe apply: docker context use only. Never sudo. Never stop another engine.
# 0 applied or already on target. 1 split still needs a manual stop.
dc_engine_apply_fix() {
  local rec ctx
  rec="$(dc_engine_recommended_context "${DC_ENGINE_LAST_ENGINE:-unknown}")"
  ctx="${DC_ENGINE_LAST_CONTEXT:-}"
  if [[ -n "${DOCKER_HOST:-}" ]]; then
    echo "dc-engine: DOCKER_HOST is set (${DOCKER_HOST}). Unset it in this shell:" >&2
    echo "  unset DOCKER_HOST" >&2
  fi
  if [[ "$ctx" != "$rec" ]]; then
    echo "dc-engine: docker context use ${rec}"
    docker context use "$rec"
  else
    echo "dc-engine: context already ${ctx}"
  fi
  if [[ -n "${DC_ENGINE_LAST_EXTRA:-}" ]]; then
    echo "dc-engine: extra engine still live — stop it with the commands above (we do not sudo)." >&2
    return 1
  fi
  return 0
}

# dc-up: refuse split unless DC_UP_ALLOW_SPLIT=1. Sets DC_ENGINE_LAST_*.
dc_engine_guard_up() {
  local engine extra
  dc_engine_refresh
  engine="${DC_ENGINE_LAST_ENGINE:-unknown}"
  extra="${DC_ENGINE_LAST_EXTRA:-}"
  if [[ -z "$extra" ]]; then
    return 0
  fi
  if [[ "${DC_UP_ALLOW_SPLIT:-}" == "1" ]]; then
    echo "dc-up: warning: split-brain (CLI=$engine extra=$extra); DC_UP_ALLOW_SPLIT=1 continuing" >&2
    return 0
  fi
  echo "dc-up: split-brain — more than one Docker engine is live." >&2
  dc_engine_fix_text >&2
  echo "  or: dc-engine --fix" >&2
  return 1
}

dc_engine_up_line() {
  printf 'dc-up: engine %s context=%s\n' \
    "${DC_ENGINE_LAST_ENGINE:-unknown}" \
    "${DC_ENGINE_LAST_CONTEXT:-unknown}"
}
