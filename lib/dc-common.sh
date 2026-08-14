# Shared resolve + Docker label list for dc-cli.
# Sourced by dc-ls, dc-open, dc-tui, dc-down, dc-ps.
# shellcheck shell=bash

DC_LABEL_FOLDER="devcontainer.local_folder"
DC_LABEL_COMPOSE="com.docker.compose.project"
DC_LABEL_SERVICE="com.docker.compose.service"

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

# Host ports already published on this container (not our sidecars).
dc_published_host_ports() {
  local id="$1"
  docker inspect -f '{{range $p,$c := .NetworkSettings.Ports}}{{range $c}}{{.HostPort}}\n{{end}}{{end}}' "$id" 2>/dev/null |
    awk 'NF && $0 != "<no value>"'
}

# Wanted HOST ports from compose + .devcontainer (no project edits).
dc_wanted_host_ports() {
  local ws="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    printf '%s\n' 3000 5173 8080 4000 5000 8000 9229
    return 0
  fi
  python3 - "$ws" <<'PY'
import json, os, re, sys
ws = sys.argv[1]
found = []

def add(p):
    try:
        n = int(str(p).split(":")[0])
    except ValueError:
        return
    if 1 <= n <= 65535 and n not in found:
        found.append(n)

def walk(node):
    if isinstance(node, dict):
        for k, v in node.items():
            if k in ("forwardPorts", "appPort", "ports") and not isinstance(v, dict):
                if isinstance(v, list):
                    for item in v:
                        if isinstance(item, dict):
                            add(item.get("hostPort") or item.get("published") or item.get("containerPort") or item.get("target"))
                        else:
                            add(item)
                else:
                    add(v)
            else:
                walk(v)
    elif isinstance(node, list):
        for x in node:
            walk(x)

svc_ok = re.compile(r"^(app|web|next|frontend|ui|devcontainer)\s*:\s*$")
port_line = re.compile(r"[-:]\s*[\"']?(\d{2,5})(?::\d{2,5})?[\"']?\s*$")
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
                            add(m.group(1))
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
                        add(n)

if not found:
    found = [3000]
for n in found:
    print(n)
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

