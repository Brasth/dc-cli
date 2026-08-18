# Host-side opener for dc-files. Sourced by bin/dc-files only.
# Guest writes NUL paths to /tmp/dc-cli-open.q; we attach VS Code or remap a bind-mount.

DC_GUEST_OPENER_PATH="${DC_GUEST_OPENER_PATH:-/tmp/dc-cli-open}"
DC_GUEST_OPEN_QUEUE="${DC_GUEST_OPEN_QUEUE:-/tmp/dc-cli-open.q}"
DC_FILES_WATCH_PID=""

_dc_files_open_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

dc_files_host_open_on() {
  case "${DC_FILES_EDITOR:-}" in
    vim|vi|0|no|off) return 1 ;;
  esac
  return 0
}

dc_files_safe_guest_path() {
  local p="$1"
  [[ "$p" == /* ]] || return 1
  [[ "$p" != *$'\n'* && "$p" != *$'\r'* ]] || return 1
  [[ "$p" != *'$'* && "$p" != *'`'* ]] || return 1
  if [[ "$p" == */.. || "$p" == */../* || "$p" == .. || "$p" == ../* ]]; then
    return 1
  fi
  return 0
}

dc_inject_guest_opener() {
  local id="$1" q="${2:-$DC_GUEST_OPEN_QUEUE}" src
  [[ -n "$id" ]] || {
    echo "dc-files: no container id to inject opener" >&2
    return 1
  }
  src="${DC_FILES_OPENER_SRC:-$_dc_files_open_dir/dc-files-opener-guest.sh}"
  if [[ ! -f "$src" ]]; then
    echo "dc-files: opener stub missing at $src" >&2
    return 1
  fi
  docker exec "$id" rm -f "$DC_GUEST_OPENER_PATH" >/dev/null 2>&1 || true
  if ! docker cp "$src" "$id:${DC_GUEST_OPENER_PATH}"; then
    echo "dc-files: could not copy opener into ${id:0:12}" >&2
    return 1
  fi
  # 755 stub + 666 queue so official remoteUser can exec/append (inject is root).
  docker exec "$id" chmod 755 "$DC_GUEST_OPENER_PATH" >/dev/null 2>&1 || true
  docker exec "$id" sh -c ": > '$q'" >/dev/null 2>&1 || true
  docker exec "$id" chmod 666 "$q" >/dev/null 2>&1 || true
  printf '%s\n' "$DC_GUEST_OPENER_PATH"
}

dc_files_guest_to_host() {
  local id="$1" guest="$2" typ src dest
  while IFS=$'\t' read -r typ src dest; do
    [[ "$typ" == "bind" && -n "$src" && -n "$dest" ]] || continue
    if [[ "$guest" == "$dest" ]]; then
      printf '%s\n' "$src"
      return 0
    fi
    if [[ "$guest" == "$dest"/* ]]; then
      printf '%s%s\n' "$src" "${guest#"$dest"}"
      return 0
    fi
  done < <(docker inspect -f '{{range .Mounts}}{{.Type}}{{"\t"}}{{.Source}}{{"\t"}}{{.Destination}}{{"\n"}}{{end}}' "$id" 2>/dev/null)
  return 1
}

dc_files_workspace_inner() {
  local id="$1" guest="$2" ws="$3" folder
  if [[ -n "$ws" ]]; then
    dc_container_workspace_path "$id" "$ws"
    return 0
  fi
  folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
  if [[ -n "$folder" && "$folder" != "<no value>" ]]; then
    dc_container_workspace_path "$id" "$folder"
    return 0
  fi
  dirname "$guest"
}

dc_files_open_host() {
  local id="$1" guest="$2" ws="${3:-${DC_FILES_WS:-}}"
  local name bin inner folder_uri file_uri host picked host_bin can_attach=0
  dc_files_safe_guest_path "$guest" || {
    echo "dc-files: refuse unsafe path" >&2
    return 1
  }
  if name="$(dc_pick_files_editor)" && bin="$(dc_editor_bin "$name")"; then
    inner="$(dc_files_workspace_inner "$id" "$guest" "$ws")"
    if [[ "$name" == cursor ]]; then
      if [[ -n "$ws" && "$(dc_workspace_kind "$ws")" == devcontainer ]]; then
        folder_uri="$(dc_dev_container_uri "$ws" "$inner")"
        file_uri="$(dc_dev_container_uri "$ws" "$guest")"
        can_attach=1
      fi
    else
      folder_uri="$(dc_attached_container_uri "$id" "$inner")"
      file_uri="$(dc_attached_container_uri "$id" "$guest")"
      can_attach=1
    fi
    if [[ "$can_attach" -eq 1 ]]; then
      echo "dc-files: opening in $name (this container) — first attach may take a bit" >&2
      if ! "$bin" --folder-uri "$folder_uri" --file-uri "$file_uri"; then
        echo "dc-files: $name attach failed" >&2
        return 1
      fi
      return 0
    fi
  fi
  if host="$(dc_files_guest_to_host "$id" "$guest")"; then
    if picked="$(dc_pick_editor)" && host_bin="$(dc_editor_bin "$picked")"; then
      echo "dc-files: opening host $host with $picked (not attached)" >&2
      "$host_bin" "$host" || true
      return 0
    fi
  fi
  echo "dc-files: no attach editor (code/cursor/…), and $guest is not on a bind-mount. Install one, use dc-open, or DC_FILES_EDITOR=vim." >&2
  return 1
}

dc_files_watch_start() {
  local id="$1" q="${2:-$DC_GUEST_OPEN_QUEUE}"
  DC_FILES_WATCH_PID=""
  DC_FILES_WATCH_Q="$q"
  (
    while IFS= read -r -d '' path; do
      [[ -n "$path" ]] || continue
      dc_files_open_host "$id" "$path" || true
    done < <(docker exec -i "$id" sh -c "tail -n +1 -f ${q} 2>/dev/null")
  ) &
  DC_FILES_WATCH_PID=$!
}

dc_files_watch_stop() {
  if [[ -n "${DC_FILES_WATCH_PID:-}" ]]; then
    kill "$DC_FILES_WATCH_PID" 2>/dev/null || true
    wait "$DC_FILES_WATCH_PID" 2>/dev/null || true
    DC_FILES_WATCH_PID=""
  fi
  if [[ -n "${DC_FILES_WATCH_Q:-}" ]]; then
    pkill -f "tail -n +1 -f ${DC_FILES_WATCH_Q}" 2>/dev/null || true
    DC_FILES_WATCH_Q=""
  fi
}

dc_files_run_fm() {
  local use_official="$1" workspace="$2" target="$3" found="$4" start_path="$5"
  local opener run_id rc=0
  local -a cmd
  if ! dc_files_host_open_on; then
    if [[ "$use_official" -eq 1 ]]; then
      if [[ -n "$start_path" ]]; then
        exec dc-exec "$workspace" -- "$found" "$start_path"
      fi
      exec dc-exec "$workspace" -- "$found"
    fi
    if [[ -n "$start_path" ]]; then
      exec dc-exec --id "$target" -- "$found" "$start_path"
    fi
    exec dc-exec --id "$target" -- "$found"
  fi
  run_id="$target"
  if [[ -z "$run_id" ]]; then
    run_id="$(dc_ids_for_workspace "$workspace" | head -n1)"
  fi
  if [[ -z "$run_id" ]]; then
    echo "dc-files: no labeled app in $workspace (dc-up first)" >&2
    return 1
  fi
  dc_ensure_running "$run_id" || return 1
  DC_FILES_WS="$(dc_resolve_workspace "$workspace" 2>/dev/null || printf '%s' "$workspace")"
  DC_GUEST_OPEN_QUEUE="/tmp/dc-cli-open.$$.q"
  if ! opener="$(dc_inject_guest_opener "$run_id" "$DC_GUEST_OPEN_QUEUE")"; then
    echo "dc-files: could not inject opener stub" >&2
    return 1
  fi
  dc_files_watch_start "$run_id" "$DC_GUEST_OPEN_QUEUE"
  trap 'dc_files_watch_stop' EXIT INT TERM
  cmd=(env "EDITOR=${opener}" "VISUAL=${opener}" "DC_OPEN_QUEUE=${DC_GUEST_OPEN_QUEUE}" "$found")
  if [[ -n "$start_path" ]]; then
    cmd+=("$start_path")
  fi
  if [[ "$use_official" -eq 1 ]]; then
    dc-exec "$workspace" -- "${cmd[@]}" || rc=$?
  else
    dc-exec --id "$target" -- "${cmd[@]}" || rc=$?
  fi
  dc_files_watch_stop
  trap - EXIT INT TERM
  return "$rc"
}
