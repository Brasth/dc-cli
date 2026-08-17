# This-folder compose network list + ensure. Source after dc-common.sh.
# shellcheck shell=bash

# Valid docker network name. Do not invent names.
dc_net_valid_name() {
  [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]
}

# Compose files for this folder: dockerComposeFile, else root compose.yaml.
dc_net_compose_files() {
  local ws="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    return 0
  fi
  python3 - "$ws" <<'PY'
import json, os, re, sys
ws = sys.argv[1]
files = []

def add(path):
    if not path:
        return
    if not os.path.isabs(path):
        path = os.path.normpath(os.path.join(ws, path))
    if os.path.isfile(path) and path not in files:
        files.append(path)

def load_jsonc(path):
    try:
        raw = open(path, encoding="utf-8", errors="ignore").read()
    except OSError:
        return None
    raw = re.sub(r"//.*?$", "", raw, flags=re.M)
    raw = re.sub(r"/\*.*?\*/", "", raw, flags=re.S)
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return None

cfg = None
cfg_dir = ws
if os.path.isfile(os.path.join(ws, ".devcontainer", "devcontainer.json")):
    cfg = load_jsonc(os.path.join(ws, ".devcontainer", "devcontainer.json"))
    cfg_dir = os.path.join(ws, ".devcontainer")
elif os.path.isfile(os.path.join(ws, ".devcontainer.json")):
    cfg = load_jsonc(os.path.join(ws, ".devcontainer.json"))

rel = None
if isinstance(cfg, dict):
    rel = cfg.get("dockerComposeFile")
if isinstance(rel, str):
    rel = [rel]
if isinstance(rel, list):
    for item in rel:
        if not isinstance(item, str):
            continue
        if os.path.isabs(item):
            add(item)
        else:
            add(os.path.normpath(os.path.join(cfg_dir, item)))

if not files:
    for name in ("compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"):
        add(os.path.join(ws, name))

for p in files:
    print(p)
PY
}

# Project name for this folder: compose config `.name`, else directory basename.
dc_compose_project_name() {
  local ws="${1:-.}" name abs
  [[ -d "$ws" ]] || return 1
  abs="$(cd "$ws" && pwd)"
  name="$(dc_compose_declared_name "$abs" || true)"
  if [[ -z "$name" ]]; then
    name="$(basename "$abs")"
  fi
  [[ -n "$name" ]] || return 1
  printf '%s\n' "$name"
}

# One service → that. Else `app`, then `web`. Else fail (ambiguous).
dc_compose_app_service() {
  local ws="${1:-.}"
  if ! command -v python3 >/dev/null 2>&1; then
    return 1
  fi
  dc_net_compose_json "$ws" | python3 -c '
import json, sys
try:
    data = json.loads(sys.stdin.read() or "{}")
except json.JSONDecodeError:
    raise SystemExit(1)
svcs = list((data.get("services") or {}).keys())
if len(svcs) == 1:
    print(svcs[0])
    raise SystemExit(0)
if "app" in svcs:
    print("app")
    raise SystemExit(0)
if "web" in svcs:
    print("web")
    raise SystemExit(0)
raise SystemExit(1)
'
}

# Print docker compose -f FILE args for this folder (no project name).
dc_compose_file_args() {
  local ws="${1:-.}" f
  while IFS= read -r f; do
    [[ -n "$f" ]] || continue
    printf '%s\n' "$f"
  done < <(dc_net_compose_files "$ws")
}

# Resolved compose project name from `docker compose config` JSON `.name`.
# Empty if compose is missing or unnamed. Callers may fall back to basename.
dc_compose_declared_name() {
  local ws="${1:-.}"
  if ! command -v python3 >/dev/null 2>&1; then
    return 1
  fi
  dc_net_compose_json "$ws" | python3 -c '
import json, sys
try:
    data = json.loads(sys.stdin.read() or "{}")
except json.JSONDecodeError:
    raise SystemExit(1)
name = data.get("name") or ""
if not isinstance(name, str) or not name.strip():
    raise SystemExit(1)
print(name.strip())
'
}

# Merged compose JSON (networks only needed). Empty object if none / fail.
dc_net_compose_json() {
  local ws="$1"
  local -a files=()
  local -a args=(compose)
  local f out rc
  mapfile -t files < <(dc_net_compose_files "$ws")
  if [[ ${#files[@]} -eq 0 ]]; then
    printf '%s\n' '{}'
    return 0
  fi
  for f in "${files[@]}"; do
    args+=(-f "$f")
  done
  args+=(config --format json)
  set +e
  out="$(cd "$ws" && docker "${args[@]}" 2>/dev/null)"
  rc=$?
  set -e
  if [[ "$rc" -ne 0 || -z "$out" ]]; then
    printf '%s\n' '{}'
    return 0
  fi
  printf '%s\n' "$out"
}

# TSV from compose JSON: name|key|kind|driver|custom_ipam
dc_net_declared_tsv() {
  local ws="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    return 0
  fi
  dc_net_compose_json "$ws" | python3 -c '
import json, sys
try:
    data = json.loads(sys.stdin.read() or "{}")
except json.JSONDecodeError:
    raise SystemExit(0)
nets = data.get("networks") or {}
if not isinstance(nets, dict):
    raise SystemExit(0)
for key, spec in nets.items():
    if not isinstance(spec, dict):
        spec = {}
    ext = spec.get("external", False)
    name = spec.get("name") or key
    kind = "compose"
    if ext is True:
        kind = "external"
    elif isinstance(ext, dict):
        kind = "external"
        name = ext.get("name") or name
    driver = spec.get("driver") or ""
    ipam = spec.get("ipam") or {}
    cfg = []
    if isinstance(ipam, dict):
        cfg = ipam.get("config") or []
    custom = "1" if cfg else "0"
    print("%s|%s|%s|%s|%s" % (name, key, kind, driver, custom))
'
}

# present | missing | unknown
dc_net_presence() {
  local name="$1" err rc
  set +e
  err="$(docker network inspect -f '{{.Name}}' "$name" 2>&1)"
  rc=$?
  set -e
  if [[ "$rc" -eq 0 ]]; then
    printf '%s\n' present
    return 0
  fi
  if printf '%s' "$err" | grep -qiE 'not found|no such'; then
    printf '%s\n' missing
    return 0
  fi
  printf '%s\n' unknown
}

# TSV report: name|kind|present|creatable|driver|reason
dc_net_report_tsv() {
  local ws="$1" name key kind driver custom present creatable reason low
  while IFS='|' read -r name key kind driver custom; do
    [[ -n "$name" ]] || continue
    creatable=0
    reason=ok
    if ! dc_net_valid_name "$name"; then
      present=0
      reason=invalid-name
      printf '%s|%s|%s|%s|%s|%s\n' "$name" "$kind" "$present" "$creatable" "$driver" "$reason"
      continue
    fi
    present=0
    case "$(dc_net_presence "$name")" in
      present) present=1 ;;
      unknown)
        reason=inspect-unknown
        printf '%s|%s|%s|%s|%s|%s\n' "$name" "$kind" "$present" "$creatable" "$driver" "$reason"
        continue
        ;;
    esac
    if [[ "$present" -eq 1 ]]; then
      printf '%s|%s|%s|%s|%s|%s\n' "$name" "$kind" "$present" "$creatable" "$driver" ok
      continue
    fi
    if [[ "$kind" != "external" ]]; then
      printf '%s|%s|%s|%s|%s|%s\n' "$name" "$kind" "$present" "$creatable" "$driver" compose-managed
      continue
    fi
    low="$(printf '%s' "$driver" | tr '[:upper:]' '[:lower:]')"
    case "$low" in
      overlay|macvlan|ipvlan|host)
        reason=unsupported-driver
        ;;
      *)
        if [[ "$custom" == "1" ]]; then
          reason=custom-ipam
        else
          reason=missing
          creatable=1
        fi
        ;;
    esac
    printf '%s|%s|%s|%s|%s|%s\n' "$name" "$kind" "$present" "$creatable" "$driver" "$reason"
  done < <(dc_net_declared_tsv "$ws")
}

dc_net_create() {
  dc_with_mutation_lock _dc_net_create_locked "$@"
}

_dc_net_create_locked() {
  local name="$1"
  dc_net_valid_name "$name" || return 1
  if [[ "$(dc_net_presence "$name")" == "present" ]]; then
    return 0
  fi
  docker network create --driver bridge "$name" >/dev/null
}

# Create missing creatable names. Prints names created.
dc_net_create_missing() {
  local ws="$1" name kind present creatable driver reason
  while IFS='|' read -r name kind present creatable driver reason; do
    [[ "$creatable" == "1" ]] || continue
    dc_net_create "$name" || return 1
    printf '%s\n' "$name"
  done < <(dc_net_report_tsv "$ws")
}

# JSON document from report TSV on stdin. workspace path is $1.
dc_net_report_json() {
  local ws="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    printf '{"schemaVersion":1,"command":"dc-net","workspace":"%s","networks":[],"missingCreatable":[],"missingBlocked":[]}\n' "$ws"
    return 0
  fi
  python3 -c '
import json, sys
ws = sys.argv[1]
rows = []
miss_c = []
miss_b = []
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    parts = line.split("|")
    if len(parts) < 6:
        continue
    name, kind, present, creatable, driver, reason = parts[:6]
    pres = present == "1"
    cre = creatable == "1"
    rows.append({
        "name": name,
        "kind": kind,
        "present": pres,
        "creatable": cre,
        "driver": driver,
        "reason": reason,
    })
    if not pres and cre:
        miss_c.append(name)
    elif not pres and reason != "compose-managed":
        miss_b.append(name)
print(json.dumps({
    "schemaVersion": 1,
    "command": "dc-net",
    "workspace": ws,
    "networks": rows,
    "missingCreatable": miss_c,
    "missingBlocked": miss_b,
}, separators=(",", ":")))
' "$ws"
}
