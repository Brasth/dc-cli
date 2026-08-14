# Shared resolve + Docker label list for dc-cli.
# Sourced by dc-ls, dc-open, dc-tui, dc-down, dc-ps.
# shellcheck shell=bash

DC_LABEL_FOLDER="devcontainer.local_folder"
DC_LABEL_COMPOSE="com.docker.compose.project"

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

# Candidate label values for a workspace: abs(cwd) and resolved (git root).
dc_workspace_candidates() {
  local dir="${1:-.}"
  local abs resolved
  if [[ ! -d "$dir" ]]; then
    echo "Not a directory: $dir" >&2
    return 1
  fi
  abs="$(cd "$dir" && pwd)"
  resolved="$(dc_resolve_workspace "$abs")"
  printf '%s\n' "$abs"
  if [[ "$resolved" != "$abs" ]]; then
    printf '%s\n' "$resolved"
  fi
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

dc_ids_for_workspace() {
  local dir="${1:-.}"
  local abs id folder
  local -a cands=()
  mapfile -t cands < <(dc_workspace_candidates "$dir")
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    folder="$(docker inspect -f "{{index .Config.Labels \"${DC_LABEL_FOLDER}\"}}" "$id" 2>/dev/null || true)"
    for abs in "${cands[@]}"; do
      if [[ "$folder" == "$abs" ]]; then
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

