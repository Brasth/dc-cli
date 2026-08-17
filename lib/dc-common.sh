# Shared resolve + Docker label list for dc-cli.
# Sourced by dc-ls, dc-open, dc-tui, dc-down, dc-ps.
# shellcheck shell=bash

DC_LABEL_FOLDER="devcontainer.local_folder"
DC_LABEL_COMPOSE="com.docker.compose.project"
DC_LABEL_SERVICE="com.docker.compose.service"

_dc_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$_dc_lib_dir/dc-floor.sh" ]]; then
  # shellcheck source=/dev/null
  source "$_dc_lib_dir/dc-floor.sh"
fi

# Kit version from VERSION next to bin/, then $DC_CLI_VERSION, then git tag, else dev.
dc_cli_version() {
  local f ver
  if [[ -n "${DC_CLI_VERSION:-}" ]]; then
    printf '%s\n' "${DC_CLI_VERSION#v}"
    return 0
  fi
  for f in "$_dc_lib_dir/../VERSION" "$_dc_lib_dir/../../VERSION"; do
    if [[ -f "$f" ]]; then
      ver="$(tr -d '[:space:]' <"$f" || true)"
      ver="${ver#v}"
      if [[ -n "$ver" ]]; then
        printf '%s\n' "$ver"
        return 0
      fi
    fi
  done
  if command -v git >/dev/null 2>&1 && git -C "$_dc_lib_dir/.." describe --tags --abbrev=0 >/dev/null 2>&1; then
    ver="$(git -C "$_dc_lib_dir/.." describe --tags --abbrev=0)"
    printf '%s\n' "${ver#v}"
    return 0
  fi
  printf '%s\n' "dev"
}

dc_cli_print_version() {
  printf '%s %s\n' "${1:-dc-cli}" "$(dc_cli_version)"
}

dc_json_escape() {
  local s="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; sys.stdout.write(json.dumps(sys.argv[1])[1:-1])' "$s"
    return
  fi
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}

dc_has_devcontainer() {
  local dir="$1"
  [[ -f "$dir/.devcontainer/devcontainer.json" || -f "$dir/.devcontainer.json" ]]
}

# Print abs path of $1 (default .). If no .devcontainer and git root differs, use git root.
# When they differ, mention both on stderr.
dc_resolve_workspace() {
  local dir="${1:-.}"
  local abs root
  if [[ ! -d "$dir" ]]; then
    echo "Not a directory: $dir" >&2
    return 1
  fi
  abs="$(cd "$dir" && pwd)"
  if dc_has_devcontainer "$abs"; then
    printf '%s\n' "$abs"
    return 0
  fi
  if git -C "$abs" rev-parse --show-toplevel >/dev/null 2>&1; then
    root="$(git -C "$abs" rev-parse --show-toplevel)"
    if [[ "$root" != "$abs" ]]; then
      if [[ "${DC_RESOLVE_VERBOSE:-}" == "1" ]]; then
        echo "dc: no .devcontainer in $abs; using git root $root" >&2
      fi
      printf '%s\n' "$root"
      return 0
    fi
  fi
  printf '%s\n' "$abs"
}

# Logical + physical paths (macOS /Users -> /Volumes symlink).
dc_path_forms() {
  local dir="$1" log phys
  if [[ ! -d "$dir" ]]; then
    return 1
  fi
  log="$(cd "$dir" && pwd)"
  phys="$(cd "$dir" && pwd -P)"
  printf '%s\n' "$log"
  if [[ "$phys" != "$log" ]]; then
    printf '%s\n' "$phys"
  fi
}

# Candidate label values: cwd + git root, each as logical and physical.
dc_workspace_candidates() {
  local dir="${1:-.}"
  local resolved
  if [[ ! -d "$dir" ]]; then
    echo "Not a directory: $dir" >&2
    return 1
  fi
  dc_path_forms "$dir"
  resolved="$(dc_resolve_workspace "$dir")"
  dc_path_forms "$resolved" | awk 'NF && !seen[$0]++'
}

# Same project even when one path is a symlink (/Users -> /Volumes).
dc_same_workspace() {
  local a="$1" b="$2" pa pb
  [[ -n "$a" && -n "$b" ]] || return 1
  if [[ "$a" == "$b" ]]; then
    return 0
  fi
  [[ -d "$a" && -d "$b" ]] || return 1
  pa="$(cd "$a" && pwd -P)"
  pb="$(cd "$b" && pwd -P)"
  [[ "$pa" == "$pb" ]]
}

# Exact folder string the official CLI stored on the labeled app.
# Use this for `devcontainer exec --workspace-folder` — it matches labels
# literally, so `.` / a symlink realpath will miss a running box.
dc_cli_workspace_folder() {
  local dir="${1:-.}" id folder
  id="$(dc_ids_for_workspace "$dir" | head -n1)"
  if [[ -n "$id" ]]; then
    folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
    if [[ -n "$folder" && "$folder" != "<no value>" ]]; then
      printf '%s\n' "$folder"
      return 0
    fi
  fi
  (cd "$dir" && pwd)
}

dc_labeled_ids() {
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  docker ps -aq --filter "label=${DC_LABEL_FOLDER}" 2>/dev/null || true
}

# TSV: id<TAB>name<TAB>status<TAB>local_folder<TAB>compose<TAB>ports
dc_inspect_row() {
  local id="$1"
  local name status folder compose ports
  name="$(docker inspect -f '{{.Name}}' "$id" 2>/dev/null | sed 's#^/##')" || return 1
  status="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || echo missing)"
  folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
  compose="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_COMPOSE}\"}}" "$id" 2>/dev/null || true)"
  ports="$(docker ps -a --filter "id=$id" --format '{{.Ports}}' 2>/dev/null || true)"
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$id" "$name" "$status" "$folder" "$compose" "$ports"
}

# docker start if needed. Wait until running (or fail if it dies immediately).
dc_ensure_running() {
  local id="$1" st i
  st="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || echo missing)"
  if [[ "$st" == "running" ]]; then
    return 0
  fi
  if [[ "$st" == "missing" ]]; then
    echo "Container missing: $id" >&2
    return 1
  fi
  echo "start  $id  (was $st)"
  docker start "$id" >/dev/null
  for i in 1 2 3 4 5 6 7 8 9 10; do
    st="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || echo missing)"
    if [[ "$st" == "running" ]]; then
      return 0
    fi
    if [[ "$st" == "exited" || "$st" == "dead" || "$st" == "missing" ]]; then
      echo "Container $id started then $st. Logs: docker logs $id" >&2
      return 1
    fi
    sleep 0.3
  done
  echo "Container $id did not become running (status=$st)" >&2
  return 1
}

# Resolve ref against dc_stack_rows. Three-pass: exact service, exact name,
# then id prefix. Service names like "db" must not steal an app whose hex
# id starts with db. Prints the matching TSV row. Miss → 1 (no inspect of strangers).
dc_stack_resolve() {
  local dir="${1:-.}" ref="${2:-}"
  local id name status svc image line
  local -a rows=()
  if [[ -z "$ref" ]]; then
    echo "dc_stack_resolve: need a service name" >&2
    return 2
  fi
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    rows+=("$line")
  done < <(dc_stack_rows "$dir")

  for line in "${rows[@]+"${rows[@]}"}"; do
    IFS=$'\t' read -r id name status svc image <<<"$line"
    if [[ "$svc" == "$ref" ]]; then
      printf '%s\n' "$line"
      return 0
    fi
  done
  for line in "${rows[@]+"${rows[@]}"}"; do
    IFS=$'\t' read -r id name status svc image <<<"$line"
    if [[ "$name" == "$ref" ]]; then
      printf '%s\n' "$line"
      return 0
    fi
  done
  for line in "${rows[@]+"${rows[@]}"}"; do
    IFS=$'\t' read -r id name status svc image <<<"$line"
    if [[ "$id" == "$ref"* ]]; then
      printf '%s\n' "$line"
      return 0
    fi
  done
  return 1
}

# Restart a compose sibling that is already in dc_stack_rows.
# Match via dc_stack_resolve. Miss → fail closed. Do not docker inspect strangers.
dc_restart_in_stack() {
  local dir="${1:-.}" ref="${2:-}"
  local id name status svc image row
  if [[ -z "$ref" ]]; then
    echo "dc_restart_in_stack: need a service name" >&2
    return 2
  fi
  row="$(dc_stack_resolve "$dir" "$ref")" || {
    echo "No compose service \"$ref\" in this workspace stack. Try: dc-exec --list" >&2
    return 1
  }
  IFS=$'\t' read -r id name status svc image <<<"$row"
  if [[ -z "$id" ]]; then
    echo "No compose service \"$ref\" in this workspace stack. Try: dc-exec --list" >&2
    return 1
  fi
  echo "restart  ${svc:-$name}  ${name:-$id}  ${id:0:12}"
  docker restart "$id" >/dev/null
  local st i
  for i in 1 2 3 4 5 6 7 8 9 10; do
    st="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || echo missing)"
    if [[ "$st" == "running" ]]; then
      return 0
    fi
    if [[ "$st" == "exited" || "$st" == "dead" || "$st" == "missing" ]]; then
      echo "Container $id restarted then $st. Logs: docker logs $id" >&2
      return 1
    fi
    sleep 0.3
  done
  echo "Container $id did not become running (status=$st)" >&2
  return 1
}

dc_compose_project_for() {
  local id="$1"
  docker inspect -f "{{index .Config.Labels \"${DC_LABEL_COMPOSE}\"}}" "$id" 2>/dev/null || true
}

# TSV: id<TAB>name<TAB>status<TAB>service<TAB>image  (compose siblings; skip our sidecars)
dc_stack_rows() {
  local dir="${1:-.}" id proj cid name status service image fwd
  mapfile -t ids < <(dc_ids_for_workspace "$dir")
  id=""
  for cid in "${ids[@]+"${ids[@]}"}"; do
    [[ -n "$cid" ]] || continue
    id="$cid"
    break
  done
  [[ -n "$id" ]] || return 0
  proj="$(dc_compose_project_for "$id")"
  [[ -n "$proj" ]] || return 0
  while IFS= read -r cid; do
    [[ -n "$cid" ]] || continue
    fwd="$(docker inspect -f '{{index .Config.Labels "dc.forward.for"}}' "$cid" 2>/dev/null || true)"
    [[ -z "$fwd" || "$fwd" == "<no value>" ]] || continue
    name="$(docker inspect -f '{{.Name}}' "$cid" | sed 's#^/##')"
    status="$(docker inspect -f '{{.State.Status}}' "$cid")"
    service="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_SERVICE}\"}}" "$cid" 2>/dev/null || true)"
    image="$(docker inspect -f '{{.Config.Image}}' "$cid" 2>/dev/null || true)"
    printf '%s\t%s\t%s\t%s\t%s\n' "$cid" "$name" "$status" "$service" "$image"
  done < <(docker ps -aq --filter "label=${DC_LABEL_COMPOSE}=${proj}" 2>/dev/null)
}

dc_stack_json() {
  local dir="${1:-.}" id name status service image first=1
  printf '['
  while IFS=$'\t' read -r id name status service image; do
    [[ -n "$id" ]] || continue
    [[ "$first" -eq 1 ]] || printf ','
    first=0
    printf '{"id":"%s","name":"%s","status":"%s","service":"%s","image":"%s"}' \
      "$(dc_json_escape "$id")" \
      "$(dc_json_escape "$name")" \
      "$(dc_json_escape "$status")" \
      "$(dc_json_escape "$service")" \
      "$(dc_json_escape "$image")"
  done < <(dc_stack_rows "$dir")
  printf ']\n'
}

dc_ids_for_workspace() {
  local dir="${1:-.}"
  local abs id folder
  local -a cands=()
  mapfile -t cands < <(dc_workspace_candidates "$dir")
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
    for abs in "${cands[@]}"; do
      if dc_same_workspace "$folder" "$abs"; then
        printf '%s\n' "$id"
        break
      fi
    done
  done < <(dc_labeled_ids)
}

# Print JSON array to stdout. Args: --workspace DIR | --all
dc_ls_json() {
  local mode="all" dir="."
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --all) mode="all" ;;
      --workspace)
        shift
        mode="workspace"
        dir="${1:-.}"
        ;;
      *)
        echo "dc_ls_json: unknown arg $1" >&2
        return 2
        ;;
    esac
    shift || true
  done

  local -a ids=()
  if ! command -v docker >/dev/null 2>&1; then
    echo "[]"
    return 0
  fi

  if [[ "$mode" == "workspace" ]]; then
    mapfile -t ids < <(dc_ids_for_workspace "$dir")
  else
    mapfile -t ids < <(dc_labeled_ids)
  fi

  local id first=1
  printf '['
  for id in "${ids[@]+"${ids[@]}"}"; do
    [[ -n "${id:-}" ]] || continue
    local name status folder compose ports
    IFS=$'\t' read -r id name status folder compose ports < <(dc_inspect_row "$id") || continue
    [[ "$first" -eq 1 ]] || printf ','
    first=0
    printf '{"id":"%s","name":"%s","status":"%s","local_folder":"%s","compose":"%s","ports":"%s"}' \
      "$(dc_json_escape "$id")" \
      "$(dc_json_escape "$name")" \
      "$(dc_json_escape "$status")" \
      "$(dc_json_escape "$folder")" \
      "$(dc_json_escape "$compose")" \
      "$(dc_json_escape "$ports")"
  done
  printf ']\n'
}

dc_ls_table() {
  local mode="all" dir="."
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --all) mode="all" ;;
      --workspace)
        shift
        mode="workspace"
        dir="${1:-.}"
        ;;
    esac
    shift || true
  done
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' ID NAME STATUS FOLDER COMPOSE PORTS
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  local -a ids=()
  if [[ "$mode" == "workspace" ]]; then
    mapfile -t ids < <(dc_ids_for_workspace "$dir")
  else
    mapfile -t ids < <(dc_labeled_ids)
  fi
  local id
  for id in "${ids[@]+"${ids[@]}"}"; do
    [[ -n "${id:-}" ]] || continue
    dc_inspect_row "$id"
  done
}

# Print executable path for zed|code|subl. Checks PATH then macOS .app bundles.
dc_editor_bin() {
  local name="$1" bin
  if command -v "$name" >/dev/null 2>&1; then
    command -v "$name"
    return 0
  fi
  case "$name" in
    zed)
      for bin in \
        "/Applications/Zed.app/Contents/MacOS/cli" \
        "$HOME/Applications/Zed.app/Contents/MacOS/cli" \
        "/Applications/Zed.app/Contents/MacOS/zed"
      do
        if [[ -x "$bin" ]]; then
          printf '%s\n' "$bin"
          return 0
        fi
      done
      ;;
    code)
      for bin in \
        "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" \
        "$HOME/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
      do
        if [[ -x "$bin" ]]; then
          printf '%s\n' "$bin"
          return 0
        fi
      done
      ;;
    subl)
      for bin in \
        "/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl" \
        "$HOME/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"
      do
        if [[ -x "$bin" ]]; then
          printf '%s\n' "$bin"
          return 0
        fi
      done
      ;;
  esac
  return 1
}

# Print editor name (zed|code|subl) that we can actually launch.
dc_pick_editor() {
  local want="${1:-}"
  if [[ -z "$want" && -n "${DC_EDITOR:-}" ]]; then
    want="$DC_EDITOR"
  fi
  if [[ -n "$want" ]]; then
    if dc_editor_bin "$want" >/dev/null; then
      printf '%s\n' "$want"
      return 0
    fi
    return 1
  fi
  local e
  for e in zed code subl; do
    if dc_editor_bin "$e" >/dev/null; then
      printf '%s\n' "$e"
      return 0
    fi
  done
  return 1
}

dc_hex() {
  local s="$1"
  if command -v xxd >/dev/null 2>&1; then
    printf '%s' "$s" | xxd -p | tr -d '\n'
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import sys; sys.stdout.write(sys.argv[1].encode().hex())' "$s"
    return
  fi
  printf '%s' "$s" | od -An -tx1 | tr -d ' \n'
}

# Destination path inside the container for a host folder mount.
dc_container_workspace_path() {
  local id="$1" host="$2"
  local src dest
  while IFS=$'\t' read -r src dest; do
    if [[ "$src" == "$host" ]]; then
      printf '%s\n' "$dest"
      return 0
    fi
  done < <(docker inspect -f '{{range .Mounts}}{{.Source}}{{"\t"}}{{.Destination}}{{"\n"}}{{end}}' "$id" 2>/dev/null)
  printf '/workspaces/%s\n' "$(basename "$host")"
}

DC_FWD_LABEL="dc.forward.for"
DC_FWD_IMAGE="${DC_FWD_IMAGE:-alpine/socat}"

# Print network|ip for a container. Prefer $DC_FWD_NET if set, else first with an IP.
dc_container_net_ip() {
  local id="$1" blob first="" prefer="" tok net ip
  blob="$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}|{{$v.IPAddress}} {{end}}' "$id" 2>/dev/null || true)"
  for tok in $blob; do
    [[ "$tok" == *"|"* ]] || continue
    net="${tok%%|*}"
    ip="${tok#*|}"
    [[ -n "$ip" ]] || continue
    [[ -n "$first" ]] || first="${net}|${ip}"
    if [[ -n "${DC_FWD_NET:-}" && "$net" == "${DC_FWD_NET}" ]]; then
      prefer="${net}|${ip}"
    fi
  done
  if [[ -n "$prefer" ]]; then
    printf '%s\n' "$prefer"
    return 0
  fi
  if [[ -n "$first" ]]; then
    printf '%s\n' "$first"
    return 0
  fi
  return 1
}

# Config.Env as KEY=VAL lines. Fail closed (non-zero) on inspect error.
dc_inspect_env() {
  local id="$1" out rc
  [[ -n "$id" ]] || return 1
  set +e
  out="$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$id" 2>/dev/null)"
  rc=$?
  set -e
  if [[ "$rc" -ne 0 ]]; then
    return 1
  fi
  printf '%s' "$out"
  [[ -n "$out" && "$out" != *$'\n' ]] && printf '\n'
  return 0
}

# ExposedPorts keys (5432/tcp). Fail closed on inspect error.
dc_inspect_exposed() {
  local id="$1" out rc
  [[ -n "$id" ]] || return 1
  set +e
  out="$(docker inspect -f '{{range $p, $_ := .Config.ExposedPorts}}{{println $p}}{{end}}' "$id" 2>/dev/null)"
  rc=$?
  set -e
  if [[ "$rc" -ne 0 ]]; then
    return 1
  fi
  printf '%s' "$out"
  [[ -n "$out" && "$out" != *$'\n' ]] && printf '\n'
  return 0
}

# Host ports already published on this container (not our sidecars).
dc_published_host_ports() {
  dc_published_port_pairs "$1" | awk -F'|' '{print $1}'
}

# host|container pairs published on the app (PortBindings / Ports).
dc_published_port_pairs() {
  local id="$1"
  [[ -n "$id" ]] || return 0
  docker inspect -f '{{range $p,$c := .NetworkSettings.Ports}}{{range $c}}{{.HostPort}}|{{$p}}{{"\n"}}{{end}}{{end}}' "$id" 2>/dev/null |
    awk -F'|' 'NF>=1 && $1 != "" && $1 != "<no value>" {
      split($2, a, "/")
      split(a[1], b, ":")
      c=b[1]
      if (c == "") next
      print $1 "|" c
    }'
}

# Host ports mentioned in an error log ("Bind for 0.0.0.0:9001 failed").
dc_ports_from_log() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  # Prefer explicit Bind for ... :PORT; also "port 9001 is already"
  grep -Eo 'Bind for [^:]+:[0-9]+|port [0-9]+ is already|0\.0\.0\.0:[0-9]+' "$file" 2>/dev/null |
    grep -Eo '[0-9]{2,5}' |
    awk 'NF && !seen[$0]++'
}

# True if log looks like a host port collision.
dc_log_is_port_conflict() {
  local file="$1"
  [[ -f "$file" ]] || return 1
  grep -qiE 'port is already allocated|address already in use|bind for .* failed: port' "$file" 2>/dev/null
}

# Containers publishing host port $1 (running). TSV: id name compose folder ports
dc_holders_of_host_port() {
  local port="$1" id name ports compose folder
  [[ -n "$port" ]] || return 0
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    ports="$(docker ps --filter "id=$id" --format '{{.Ports}}' 2>/dev/null || true)"
    # Match :9001-> or 0.0.0.0:9001 or [::]:9001
    if ! printf '%s' "$ports" | grep -Eq "(^|[,[])(:|::)?${port}->|0\.0\.0\.0:${port}|\\[::\\]:${port}"; then
      # also HostConfig PortBindings for edge cases
      if ! docker inspect -f '{{range $p,$b := .HostConfig.PortBindings}}{{range $b}}{{.HostPort}} {{end}}{{end}}' "$id" 2>/dev/null |
        grep -Eq "(^| )${port}( |$)"; then
        continue
      fi
    fi
    name="$(docker inspect -f '{{.Name}}' "$id" 2>/dev/null | sed 's#^/##')"
    compose="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_COMPOSE}\"}}" "$id" 2>/dev/null || true)"
    folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
    [[ "$folder" == "<no value>" ]] && folder=""
    [[ "$compose" == "<no value>" ]] && compose=""
    printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$name" "${compose:--}" "${folder:--}" "$ports"
  done < <(docker ps -q 2>/dev/null)
}

# Unique holder lines for any of the given ports (stdin or args).
# Prints TSV via dc_holders_of_host_port, de-duped by id.
dc_holders_of_ports() {
  local p id
  declare -A seen=()
  for p in "$@"; do
    [[ -n "$p" ]] || continue
    while IFS=$'\t' read -r id name compose folder ports; do
      [[ -n "$id" ]] || continue
      [[ -n "${seen[$id]:-}" ]] && continue
      seen[$id]=1
      printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$name" "$compose" "$folder" "$ports"
    done < <(dc_holders_of_host_port "$p")
  done
}

# Stop a positively identified foreign holder. Unlabeled / unknown / ambiguous
# holders are report-only (no docker stop / compose stop fallback).
# Usage: dc_stop_port_holder ID [CURRENT_WORKSPACE]
dc_stop_port_holder() {
  dc_with_mutation_lock _dc_stop_port_holder_locked "$@"
}

_dc_stop_port_holder_locked() {
  local id="$1" current_ws="${2:-}"
  local name kind reason folder compose target_id target_ws presence
  presence="$(dc_inspect_presence "$id")"
  if [[ "$presence" != "present" ]]; then
    echo "report-only holder ${id:0:12} reason=inspect-${presence}" >&2
    return 1
  fi
  name="$(docker inspect -f '{{.Name}}' "$id" 2>/dev/null | sed 's#^/##')"
  IFS='|' read -r kind reason folder compose target_id target_ws < <(dc_classify_port_holder "$id" "$current_ws")
  case "$reason" in
    same-workspace)
      echo "skip self  ${id:0:12}  (same workspace)"
      return 0
      ;;
    unlabeled|folder-missing|ambiguous-compose|sidecar-unresolved|sidecar-ambiguous|inspect-unknown|sidecar-inspect-unknown)
      echo "report-only holder ${id:0:12}  ${name:-?}  reason=${reason}" >&2
      return 1
      ;;
  esac
  case "$kind" in
    sidecar)
      if [[ "$reason" == "orphan-absent" ]]; then
        if [[ "$(dc_inspect_presence "$id")" != "present" ]]; then
          echo "report-only holder ${id:0:12} reason=inspect-unknown" >&2
          return 1
        fi
        echo "stop holder sidecar (orphan-absent)  ${name:-?}  ${id:0:12}"
        docker rm -f "$id" >/dev/null
        return 0
      fi
      if [[ -n "$target_ws" && -d "$target_ws" ]] && command -v dc-down >/dev/null 2>&1; then
        echo "stop holder sidecar via dc-down  $target_ws  ($name)"
        dc-down "$target_ws"
        return
      fi
      echo "stop holder sidecar  ${name:-?}  ${id:0:12}"
      docker rm -f "$id" >/dev/null
      return 0
      ;;
    app)
      if [[ -n "$folder" && -d "$folder" ]] && command -v dc-down >/dev/null 2>&1; then
        echo "stop holder stack via dc-down  $folder  ($name)"
        dc-down "$folder"
        return
      fi
      echo "report-only holder ${id:0:12}  ${name:-?}  reason=folder-missing" >&2
      return 1
      ;;
  esac
  echo "report-only holder ${id:0:12}  ${name:-?}  reason=${reason:-unlabeled}" >&2
  return 1
}

# Wanted HOST ports from compose + .devcontainer (no project edits).
dc_wanted_host_ports() {
  dc_wanted_port_pairs "$1" | awk -F'|' '{print $1}'
}

# host|container|provenance   provenance = compose|forwardPorts|fallback
dc_wanted_port_pairs() {
  local ws="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    printf '%s\n' '3000|3000|fallback'
    return 0
  fi
  python3 - "$ws" <<'PY'
import json, os, re, sys
ws = sys.argv[1]
found = []  # (host, container, provenance)

def add(host, container=None, prov="forwardPorts"):
    try:
        raw = str(host)
        if ":" in raw and container is None:
            a, b = raw.split(":", 1)
            h, c = int(a), int(b.split("/")[0])
        else:
            h = int(raw)
            c = int(container) if container is not None else h
    except ValueError:
        return
    if not (1 <= h <= 65535 and 1 <= c <= 65535):
        return
    key = (h, c)
    if any(x[0] == h and x[1] == c for x in found):
        return
    found.append((h, c, prov))

def walk(node):
    if isinstance(node, dict):
        for k, v in node.items():
            if k in ("forwardPorts", "appPort", "ports") and not isinstance(v, dict):
                if isinstance(v, list):
                    for item in v:
                        if isinstance(item, dict):
                            hp = item.get("hostPort") or item.get("published")
                            cp = item.get("containerPort") or item.get("target")
                            add(hp or cp, cp or hp, "forwardPorts")
                        else:
                            add(item, None, "forwardPorts")
                else:
                    add(v, None, "forwardPorts")
            else:
                walk(v)
    elif isinstance(node, list):
        for x in node:
            walk(x)

svc_ok = re.compile(r"^(app|web|next|frontend|ui|devcontainer)\s*:\s*$")
port_line = re.compile(r"[-:]\s*[\"']?(\d{2,5})(?::(\d{2,5}))?[\"']?\s*$")
for root, dirs, files in os.walk(ws):
    dirs[:] = [d for d in dirs if d not in (".git", "node_modules", ".pnpm-store", "dist", "build", ".next")]
    if root.count(os.sep) - ws.count(os.sep) > 3:
        dirs[:] = []
        continue
    for name in files:
        path = os.path.join(root, name)
        low = name.lower()
        if low.endswith((".yml", ".yaml")) and ("compose" in low or name.startswith("docker-compose")):
            try:
                text = open(path, encoding="utf-8", errors="ignore").read()
            except OSError:
                continue
            in_app = False
            in_ports = False
            app_indent = 0
            for line in text.splitlines():
                if not line.strip() or line.lstrip().startswith("#"):
                    continue
                indent = len(line) - len(line.lstrip())
                if svc_ok.match(line.strip()):
                    in_app = True
                    app_indent = indent
                    in_ports = False
                    continue
                if in_app and indent <= app_indent and not line.strip().startswith("-"):
                    in_app = False
                    in_ports = False
                if in_app and re.match(r"\s*ports\s*:", line):
                    in_ports = True
                    continue
                if in_app and in_ports:
                    if re.match(r"\s*-\s", line):
                        m = port_line.search(line)
                        if m:
                            add(m.group(1), m.group(2) or m.group(1), "compose")
                    else:
                        in_ports = False
        if name == "devcontainer.json" or path.endswith(".devcontainer.json"):
            try:
                raw = open(path, encoding="utf-8", errors="ignore").read()
            except OSError:
                continue
            raw = re.sub(r"//.*?$", "", raw, flags=re.M)
            raw = re.sub(r"/\*.*?\*/", "", raw, flags=re.S)
            try:
                walk(json.loads(raw))
            except json.JSONDecodeError:
                for m in re.finditer(r"\"(?:forwardPorts|appPort)\"\s*:\s*(\[[^\]]*\]|\d+)", raw):
                    for n in re.findall(r"\d{2,5}", m.group(1)):
                        add(n, n, "forwardPorts")

if not found:
    found = [(3000, 3000, "fallback")]
for h, c, p in found:
    print("%s|%s|%s" % (h, c, p))
PY
}

dc_fwd_name() {
  local id="$1" host="$2"
  printf 'dc-fwd-%s-%s\n' "${id:0:12}" "$host"
}

dc_fwd_ids_for() {
  local id="$1"
  docker ps -aq --filter "label=${DC_FWD_LABEL}=${id}" 2>/dev/null || true
}

dc_fwd_workspace_key() {
  local id="$1" folder
  folder="$(dc_label "$id" "$DC_LABEL_FOLDER")"
  if [[ -n "$folder" ]]; then
    printf '%s\n' "$folder"
    return 0
  fi
  return 1
}

dc_unique_app_id() {
  local dir="${1:-.}" explicit="${2:-}"
  local -a ids=()
  local id n=0
  if [[ -n "$explicit" ]]; then
    if [[ "$(dc_inspect_presence "$explicit")" != "present" ]]; then
      echo "dc-forward: --id $explicit not found" >&2
      return 1
    fi
    printf '%s\n' "$explicit"
    return 0
  fi
  mapfile -t ids < <(dc_ids_for_workspace "$dir")
  for id in "${ids[@]+"${ids[@]}"}"; do
    [[ -n "$id" ]] && n=$((n + 1))
  done
  if [[ "$n" -eq 0 ]]; then
    echo "No matching running/labeled container. Start with dc-up first." >&2
    return 1
  fi
  if [[ "$n" -gt 1 ]]; then
    echo "dc-forward: duplicate labeled apps; pass --id. matches:" >&2
    for id in "${ids[@]}"; do
      [[ -n "$id" ]] || continue
      echo "  $id" >&2
    done
    return 1
  fi
  printf '%s\n' "${ids[0]}"
}

dc_fwd_ids_for_workspace() {
  local key="$1" cid owner ws forid folder
  [[ -n "$key" ]] || return 0
  while IFS= read -r cid; do
    [[ -n "$cid" ]] || continue
    owner="$(dc_label "$cid" "dc.forward.owner")"
    ws="$(dc_label "$cid" "dc.forward.workspace")"
    forid="$(dc_label "$cid" "$DC_FWD_LABEL")"
    if [[ "$owner" == "dc-cli" && -n "$ws" ]]; then
      if [[ "$ws" == "$key" ]] || dc_same_workspace "$ws" "$key" 2>/dev/null; then
        printf '%s\n' "$cid"
        continue
      fi
    fi
    # Legacy: for→this workspace while target is Present, only if full fingerprint holds.
    # Labels alone are never enough (spoof ban).
    if [[ -n "$forid" && "$owner" != "dc-cli" ]]; then
      if [[ "$(dc_inspect_presence "$forid")" == "present" ]]; then
        folder="$(dc_label "$forid" "$DC_LABEL_FOLDER")"
        if [[ -n "$folder" ]] && { [[ "$folder" == "$key" ]] || dc_same_workspace "$folder" "$key" 2>/dev/null; }; then
          if dc_fwd_fingerprint_ok "$cid"; then
            printf '%s\n' "$cid"
          fi
        fi
      fi
    fi
  done < <(docker ps -aq --filter "label=${DC_FWD_LABEL}" 2>/dev/null)
}

dc_fwd_fingerprint_ok() {
  local id="$1" want_host="${2:-}"
  local name image cmd host lab_host
  name="$(docker inspect -f '{{.Name}}' "$id" 2>/dev/null | sed 's#^/##')"
  image="$(docker inspect -f '{{.Config.Image}}' "$id" 2>/dev/null || true)"
  cmd="$(docker inspect -f '{{join .Config.Cmd " "}}' "$id" 2>/dev/null || true)"
  lab_host="$(dc_label "$id" "dc.forward.host")"
  [[ -n "$(dc_label "$id" "$DC_FWD_LABEL")" ]] || return 1
  [[ -n "$lab_host" ]] || return 1
  [[ "$name" == dc-fwd-* ]] || return 1
  [[ "$image" == *socat* || "$image" == "${DC_FWD_IMAGE}"* ]] || return 1
  printf '%s' "$cmd" | grep -q 'TCP-LISTEN' || return 1
  if [[ -n "$want_host" && "$lab_host" != "$want_host" ]]; then
    return 1
  fi
  return 0
}

# Sidecars whose target is tri-state definitively absent (never inspect-unknown).
dc_orphan_forward_ids() {
  local cid target presence
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  while IFS= read -r cid; do
    [[ -n "$cid" ]] || continue
    target="$(dc_label "$cid" "$DC_FWD_LABEL")"
    [[ -n "$target" ]] || continue
    presence="$(dc_inspect_presence "$target")"
    if [[ "$presence" == "absent" ]]; then
      printf '%s\n' "$cid"
    fi
  done < <(docker ps -aq --filter "label=${DC_FWD_LABEL}" 2>/dev/null)
}

# Image IDs used by this workspace's compose stack (or labeled app).
dc_workspace_image_ids() {
  local dir="${1:-.}" id
  while IFS=$'\t' read -r id _rest; do
    [[ -n "$id" ]] || continue
    docker inspect -f '{{.Image}}' "$id" 2>/dev/null || true
  done < <(dc_stack_rows "$dir")
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    docker inspect -f '{{.Image}}' "$id" 2>/dev/null || true
  done < <(dc_ids_for_workspace "$dir")
}

# --- Safety helpers (ownership, tri-state inspect, mutation lock) ---

dc_label() {
  local id="$1" key="$2" v
  v="$(docker inspect -f "{{index .Config.Labels \"${key}\"}}" "$id" 2>/dev/null || true)"
  [[ "$v" == "<no value>" ]] && v=""
  printf '%s' "$v"
}

# present | absent | unknown — engine/transport errors are unknown, never absent.
dc_inspect_presence() {
  local id="$1" out rc
  if [[ -z "$id" ]]; then
    printf '%s\n' unknown
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    printf '%s\n' unknown
    return 0
  fi
  set +e
  out="$(docker inspect -f '{{.Id}}' "$id" 2>&1)"
  rc=$?
  set -e
  if [[ "$rc" -eq 0 && -n "$out" && "$out" != *"Error"* && "$out" != *"error"* ]]; then
    printf '%s\n' present
    return 0
  fi
  if printf '%s' "$out" | grep -qiE 'no such object|no such container|no such image'; then
    printf '%s\n' absent
    return 0
  fi
  printf '%s\n' unknown
}

dc_is_positive_dc_container() {
  local id="$1" folder
  folder="$(dc_label "$id" "$DC_LABEL_FOLDER")"
  [[ -n "$folder" ]]
}

# Distinct labeled workspace folders claiming compose project $1.
dc_compose_claimants() {
  local proj="$1" id folder p existing match
  local -a out=()
  [[ -n "$proj" && "$proj" != "<no value>" ]] || return 0
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    p="$(dc_label "$id" "$DC_LABEL_COMPOSE")"
    [[ "$p" == "$proj" ]] || continue
    folder="$(dc_label "$id" "$DC_LABEL_FOLDER")"
    [[ -n "$folder" ]] || continue
    match=0
    for existing in "${out[@]+"${out[@]}"}"; do
      if [[ "$existing" == "$folder" ]] || dc_same_workspace "$existing" "$folder" 2>/dev/null; then
        match=1
        break
      fi
    done
    [[ "$match" -eq 1 ]] && continue
    out+=("$folder")
  done < <(dc_labeled_ids)
  for folder in "${out[@]+"${out[@]}"}"; do
    printf '%s\n' "$folder"
  done
}

# Pipe-delimited: id|status|folder|compose|fwd — fail if not present.
dc_revalidate_container() {
  local id="$1" presence status folder compose fwd
  presence="$(dc_inspect_presence "$id")"
  if [[ "$presence" != "present" ]]; then
    return 1
  fi
  status="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || true)"
  folder="$(dc_label "$id" "$DC_LABEL_FOLDER")"
  compose="$(dc_label "$id" "$DC_LABEL_COMPOSE")"
  fwd="$(dc_label "$id" "$DC_FWD_LABEL")"
  printf '%s|%s|%s|%s|%s\n' "$id" "$status" "$folder" "$compose" "$fwd"
}

# Pipe-delimited: kind|reason|folder|compose|target_id|target_ws
dc_classify_port_holder() {
  local id="$1" current_ws="${2:-}"
  local presence folder compose fwd target_presence target_ws claimants n
  presence="$(dc_inspect_presence "$id")"
  if [[ "$presence" != "present" ]]; then
    printf 'unknown|inspect-unknown||||\n'
    return 0
  fi
  folder="$(dc_label "$id" "$DC_LABEL_FOLDER")"
  compose="$(dc_label "$id" "$DC_LABEL_COMPOSE")"
  fwd="$(dc_label "$id" "$DC_FWD_LABEL")"

  if [[ -n "$fwd" ]]; then
    target_presence="$(dc_inspect_presence "$fwd")"
    if [[ "$target_presence" == "unknown" ]]; then
      printf 'sidecar|sidecar-inspect-unknown|%s|%s|%s|\n' "$folder" "$compose" "$fwd"
      return 0
    fi
    if [[ "$target_presence" == "present" ]]; then
      target_ws="$(dc_label "$fwd" "$DC_LABEL_FOLDER")"
      if [[ -z "$target_ws" ]]; then
        printf 'sidecar|sidecar-unresolved|%s|%s|%s|\n' "$folder" "$compose" "$fwd"
        return 0
      fi
      if [[ -n "$current_ws" ]] && dc_same_workspace "$target_ws" "$current_ws" 2>/dev/null; then
        printf 'sidecar|same-workspace|%s|%s|%s|%s\n' "$folder" "$compose" "$fwd" "$target_ws"
        return 0
      fi
      printf 'sidecar|foreign|%s|%s|%s|%s\n' "$folder" "$compose" "$fwd" "$target_ws"
      return 0
    fi
    printf 'sidecar|orphan-absent|%s|%s|%s|\n' "$folder" "$compose" "$fwd"
    return 0
  fi

  if [[ -z "$folder" ]]; then
    printf 'unlabeled|unlabeled||%s||\n' "$compose"
    return 0
  fi
  if [[ -n "$current_ws" ]] && dc_same_workspace "$folder" "$current_ws" 2>/dev/null; then
    printf 'app|same-workspace|%s|%s||\n' "$folder" "$compose"
    return 0
  fi
  if [[ ! -d "$folder" ]]; then
    printf 'app|folder-missing|%s|%s||\n' "$folder" "$compose"
    return 0
  fi
  if [[ -n "$compose" ]]; then
    n=0
    while IFS= read -r claimants; do
      [[ -n "$claimants" ]] || continue
      n=$((n + 1))
    done < <(dc_compose_claimants "$compose")
    if [[ "$n" -gt 1 ]]; then
      printf 'ambiguous|ambiguous-compose|%s|%s||\n' "$folder" "$compose"
      return 0
    fi
  fi
  printf 'app|foreign|%s|%s||\n' "$folder" "$compose"
}

# Succeed-or-fail list of containers mounting volume NAME. Failure ≠ empty.
dc_volume_mount_inventory() {
  local name="$1" out rc
  [[ -n "$name" ]] || return 1
  if ! command -v docker >/dev/null 2>&1; then
    return 1
  fi
  set +e
  out="$(docker ps -aq --filter "volume=${name}" 2>&1)"
  rc=$?
  set -e
  if [[ "$rc" -ne 0 ]]; then
    return 1
  fi
  if printf '%s' "$out" | grep -qiE 'error|cannot connect|permission denied'; then
    return 1
  fi
  printf '%s\n' "$out"
  return 0
}

dc_engine_lock_key() {
  local host ctx raw
  host="${DOCKER_HOST:-}"
  if [[ -z "$host" ]] && command -v docker >/dev/null 2>&1; then
    ctx="$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
    host="${ctx:-}"
  fi
  host="${host:-unix:///var/run/docker.sock}"
  raw="${host}|${USER:-$(id -un 2>/dev/null || echo user)}"
  dc_hex "$raw"
}

dc_mutation_lock_root() {
  printf '%s\n' "${DC_MUTATION_LOCK_ROOT:-${XDG_RUNTIME_DIR:-/tmp}/dc-cli/locks}"
}

dc_pid_start() {
  local pid="$1"
  ps -p "$pid" -o lstart= 2>/dev/null | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

dc_mutation_lock_held() {
  local key="${1:-}"
  [[ -n "${DC_MUTATION_LOCK_TOKEN:-}" ]] || return 1
  if [[ -n "$key" && "${DC_MUTATION_LOCK_ENGINE:-}" != "$key" ]]; then
    return 1
  fi
  return 0
}

# Conjunctive dead-owner recovery. Never age/mtime alone.
dc_mutation_lock_try_reclaim_dead() {
  local lockdir="$1" meta pid start now
  meta="${lockdir}/owner"
  [[ -d "$lockdir" && -f "$meta" ]] || return 1
  pid="$(sed -n 's/^pid=//p' "$meta" | head -1)"
  start="$(sed -n 's/^start=//p' "$meta" | head -1)"
  [[ -n "$pid" && -n "$start" ]] || return 1
  if ! [[ "$pid" =~ ^[0-9]+$ ]]; then
    return 1
  fi
  if kill -0 "$pid" 2>/dev/null; then
    now="$(dc_pid_start "$pid")"
    if [[ -n "$now" && "$now" == "$start" ]]; then
      return 1
    fi
  fi
  rm -rf "$lockdir"
  return 0
}

# Top-level acquire or inherit. Usage: dc_with_mutation_lock cmd [args...]
dc_with_mutation_lock() {
  local key lockroot lockdir token wait_s elapsed rc nest
  if [[ $# -lt 1 ]]; then
    echo "dc_with_mutation_lock: missing command" >&2
    return 2
  fi
  key="$(dc_engine_lock_key)"
  if dc_mutation_lock_held "$key"; then
    nest="${DC_MUTATION_LOCK_NEST:-1}"
    export DC_MUTATION_LOCK_NEST=$((nest + 1))
    "$@"
    rc=$?
    export DC_MUTATION_LOCK_NEST="$nest"
    return "$rc"
  fi

  lockroot="$(dc_mutation_lock_root)"
  if ! mkdir -p "$lockroot" 2>/dev/null; then
    echo "dc: mutation lock dir unwritable: $lockroot" >&2
    return 1
  fi
  lockdir="${lockroot}/${key}.lock"
  token="$(dc_hex "$$-$(date +%s)-${RANDOM:-0}")"
  wait_s="${DC_MUTATION_LOCK_WAIT:-8}"
  elapsed=0
  while ! mkdir "$lockdir" 2>/dev/null; do
    if dc_mutation_lock_try_reclaim_dead "$lockdir"; then
      continue
    fi
    if [[ "$elapsed" -ge "$wait_s" ]]; then
      echo "dc: mutation lock busy for engine ${key:0:12} (waited ${wait_s}s)" >&2
      return 1
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  {
    printf 'pid=%s\n' "$$"
    printf 'start=%s\n' "$(dc_pid_start "$$")"
    printf 'engine=%s\n' "$key"
    printf 'token=%s\n' "$token"
  } >"${lockdir}/owner"

  export DC_MUTATION_LOCK_TOKEN="$token"
  export DC_MUTATION_LOCK_ENGINE="$key"
  export DC_MUTATION_LOCK_OWNER_PID="$$"
  export DC_MUTATION_LOCK_NEST=1
  export DC_MUTATION_LOCK_PATH="$lockdir"

  set +e
  "$@"
  rc=$?
  set -e
  if [[ "${DC_MUTATION_LOCK_OWNER_PID:-}" == "$$" ]]; then
    rm -rf "$lockdir"
    unset DC_MUTATION_LOCK_TOKEN DC_MUTATION_LOCK_ENGINE DC_MUTATION_LOCK_OWNER_PID
    unset DC_MUTATION_LOCK_NEST DC_MUTATION_LOCK_PATH
  fi
  return "$rc"
}

