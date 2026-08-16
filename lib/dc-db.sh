# DB target classify + declared-port helpers. Source after dc-common.sh.
# shellcheck shell=bash

DC_DB_WELL_KNOWN=$'postgres|5432\nmysql|3306\nredis|6379\nmongo|27017'

dc_db_well_known_port() {
  local driver="$1" line d p
  while IFS='|' read -r d p; do
    [[ "$d" == "$driver" ]] && { printf '%s\n' "$p"; return 0; }
  done <<<"$DC_DB_WELL_KNOWN"
  return 1
}

# driver or empty. Require image or well-known port; service name only if it is a known db name.
dc_db_classify() {
  local image="${1:-}" service="${2:-}" ports="${3:-}"
  local low img svc
  img="$(printf '%s' "$image" | tr '[:upper:]' '[:lower:]')"
  svc="$(printf '%s' "$service" | tr '[:upper:]' '[:lower:]')"
  low="$(printf '%s' "$ports" | tr '[:upper:]' '[:lower:]')"
  if [[ "$img" == *postgres* || "$img" == *postgis* ]]; then
    printf '%s\n' postgres; return 0
  fi
  if [[ "$img" == *mariadb* || "$img" == *mysql* ]]; then
    printf '%s\n' mysql; return 0
  fi
  if [[ "$img" == *redis* ]]; then
    printf '%s\n' redis; return 0
  fi
  if [[ "$img" == *mongo* ]]; then
    printf '%s\n' mongo; return 0
  fi
  if [[ "$low" == *5432* ]]; then printf '%s\n' postgres; return 0; fi
  if [[ "$low" == *3306* ]]; then printf '%s\n' mysql; return 0; fi
  if [[ "$low" == *6379* ]]; then printf '%s\n' redis; return 0; fi
  if [[ "$low" == *27017* ]]; then printf '%s\n' mongo; return 0; fi
  case "$svc" in
    postgres|postgresql|pg) printf '%s\n' postgres; return 0 ;;
    mysql|mariadb) printf '%s\n' mysql; return 0 ;;
    redis) printf '%s\n' redis; return 0 ;;
    mongo|mongodb) printf '%s\n' mongo; return 0 ;;
  esac
  return 1
}

# stdin: KEY=VAL. stdout: user|password|database
dc_db_creds_from_env() {
  local driver="$1" line key val
  local user="" pass="" db=""
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" == *=* ]] || continue
    key="${line%%=*}"
    val="${line#*=}"
    case "$driver:$key" in
      postgres:POSTGRES_USER) user="$val" ;;
      postgres:POSTGRES_PASSWORD) pass="$val" ;;
      postgres:POSTGRES_DB) db="$val" ;;
      mysql:MYSQL_USER) user="$val" ;;
      mysql:MYSQL_PASSWORD) pass="$val" ;;
      mysql:MYSQL_DATABASE) db="$val" ;;
      mysql:MYSQL_ROOT_PASSWORD) [[ -z "$pass" ]] && pass="$val"; [[ -z "$user" ]] && user="root" ;;
      mongo:MONGO_INITDB_ROOT_USERNAME) user="$val" ;;
      mongo:MONGO_INITDB_ROOT_PASSWORD) pass="$val" ;;
      mongo:MONGO_INITDB_DATABASE) db="$val" ;;
      redis:REDIS_PASSWORD) pass="$val" ;;
    esac
  done
  case "$driver" in
    postgres) [[ -n "$user" ]] || user="postgres"; [[ -n "$db" ]] || db="postgres" ;;
    mysql) [[ -n "$user" ]] || user="root" ;;
    mongo) [[ -n "$db" ]] || db="admin" ;;
    redis) [[ -n "$db" ]] || db="0" ;;
  esac
  printf '%s|%s|%s\n' "$user" "$pass" "$db"
}

# host|container lines for compose service NAME under workspace.
dc_db_compose_service_ports() {
  local ws="$1" service="$2"
  [[ -n "$ws" && -n "$service" && -d "$ws" ]] || return 0
  command -v python3 >/dev/null 2>&1 || return 0
  python3 - "$ws" "$service" <<'PY'
import os, re, sys
ws, want = sys.argv[1], sys.argv[2]
want_l = want.lower()
svc_line = re.compile(r"^(\s*)([A-Za-z0-9._-]+)\s*:\s*$")
port_line = re.compile(
    r"""[-:]\s*["']?(?:127\.0\.0\.1:|0\.0\.0\.0:)?(\d{2,5})(?::(\d{2,5}))?(?:/\w+)?["']?\s*$"""
)
for root, dirs, files in os.walk(ws):
    dirs[:] = [d for d in dirs if d not in (".git", "node_modules", ".pnpm-store", "dist", "build", ".next")]
    if root.count(os.sep) - ws.count(os.sep) > 3:
        dirs[:] = []
        continue
    for name in files:
        low = name.lower()
        if not (low.endswith((".yml", ".yaml")) and ("compose" in low or name.startswith("docker-compose"))):
            continue
        path = os.path.join(root, name)
        try:
            text = open(path, encoding="utf-8", errors="ignore").read()
        except OSError:
            continue
        in_svc = False
        in_ports = False
        svc_indent = 0
        inline_ports = re.compile(r"^\s*ports\s*:\s*\[(.*)\]\s*$")
        for line in text.splitlines():
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            indent = len(line) - len(line.lstrip())
            m = svc_line.match(line)
            if m and m.group(2).lower() == want_l:
                in_svc = True
                svc_indent = indent
                in_ports = False
                continue
            if in_svc and indent <= svc_indent and not line.strip().startswith("-"):
                in_svc = False
                in_ports = False
            if in_svc:
                im = inline_ports.match(line)
                if im:
                    for token in re.findall(r"['\"]?(\d{2,5}(?::\d{2,5})?)['\"]?", im.group(1)):
                        if ":" in token:
                            host, cont = token.split(":", 1)
                        else:
                            host = cont = token
                        print("%s|%s" % (host, cont))
                    continue
            if in_svc and re.match(r"\s*ports\s*:", line):
                in_ports = True
                continue
            if in_svc and in_ports:
                if re.match(r"\s*-\s", line):
                    pm = port_line.search(line)
                    if pm:
                        host = pm.group(1)
                        # Single number in compose = container port (ephemeral host). Skip.
                        if pm.group(2):
                            print("%s|%s" % (host, pm.group(2)))
                else:
                    in_ports = False
PY
}

# host ports listed in forwardPorts/appPort (same number both sides).
dc_db_forward_ports() {
  local ws="$1"
  [[ -n "$ws" && -d "$ws" ]] || return 0
  command -v python3 >/dev/null 2>&1 || return 0
  python3 - "$ws" <<'PY'
import json, os, re, sys
ws = sys.argv[1]
found = []

def add(raw):
    try:
        s = str(raw)
        if ":" in s:
            a, b = s.split(":", 1)
            h, c = int(a), int(b.split("/")[0])
        else:
            h = c = int(s)
    except ValueError:
        return
    if not (1 <= h <= 65535 and 1 <= c <= 65535):
        return
    key = (h, c)
    if key in found:
        return
    found.append(key)

def walk(node):
    if isinstance(node, dict):
        for k, v in node.items():
            if k in ("forwardPorts", "appPort") and not isinstance(v, dict):
                if isinstance(v, list):
                    for item in v:
                        if isinstance(item, dict):
                            hp = item.get("hostPort") or item.get("published")
                            cp = item.get("containerPort") or item.get("target")
                            add("%s:%s" % (hp or cp, cp or hp))
                        else:
                            add(item)
                else:
                    add(v)
            else:
                walk(v)
    elif isinstance(node, list):
        for x in node:
            walk(x)

for root, dirs, files in os.walk(ws):
    dirs[:] = [d for d in dirs if d not in (".git", "node_modules", ".pnpm-store", "dist", "build", ".next")]
    if root.count(os.sep) - ws.count(os.sep) > 3:
        dirs[:] = []
        continue
    for name in files:
        path = os.path.join(root, name)
        if name != "devcontainer.json" and not path.endswith(".devcontainer.json"):
            continue
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
for h, c in found:
    print("%s|%s" % (h, c))
PY
}

# stdout: host|container|source   source=compose|forwardPorts
dc_db_declared_pair() {
  local ws="$1" service="$2" cport="$3" driver="${4:-}"
  local h c known
  known="$(dc_db_well_known_port "$driver" 2>/dev/null || true)"
  [[ -n "$cport" ]] || cport="$known"
  while IFS='|' read -r h c; do
    [[ -n "$h" ]] || continue
    if [[ -z "$cport" || "$c" == "$cport" ]]; then
      printf '%s|%s|compose\n' "$h" "$c"
      return 0
    fi
  done < <(dc_db_compose_service_ports "$ws" "$service")
  while IFS='|' read -r h c; do
    [[ -n "$h" ]] || continue
    if [[ "$c" == "$cport" || "$h" == "$cport" ]]; then
      printf '%s|%s|forwardPorts\n' "$h" "$c"
      return 0
    fi
    if [[ -n "$known" && ( "$c" == "$known" || "$h" == "$known" ) ]]; then
      printf '%s|%s|forwardPorts\n' "$h" "$c"
      return 0
    fi
  done < <(dc_db_forward_ports "$ws")
  return 1
}

# TSV: id service image driver cport declared live user pass db src
dc_db_targets() {
  local dir="${1:-.}" id name status service image driver
  local env exposed pubs cport live decl_h decl_c decl_src user pass db blob
  while IFS=$'\t' read -r id name status service image; do
    [[ -n "$id" ]] || continue
    if ! env="$(dc_inspect_env "$id")"; then
      echo "dc-db: inspect failed for ${id:0:12}" >&2
      return 1
    fi
    if ! exposed="$(dc_inspect_exposed "$id")"; then
      echo "dc-db: inspect failed for ${id:0:12}" >&2
      return 1
    fi
    pubs="$(dc_published_port_pairs "$id" || true)"
    blob="${exposed}"$'\n'"${pubs}"
    driver="$(dc_db_classify "$image" "$service" "$blob" || true)"
    [[ -n "$driver" ]] || continue
    cport="$(dc_db_well_known_port "$driver" || true)"
    if [[ -n "$exposed" && -n "$cport" ]]; then
      local ep
      while IFS= read -r ep; do
        ep="${ep%%/*}"
        if [[ "$ep" == "$cport" ]]; then
          break
        fi
      done <<<"$exposed"
    fi
    live=""
    while IFS='|' read -r h c; do
      [[ -n "$h" ]] || continue
      if [[ -n "$cport" && "$c" == "$cport" ]]; then
        live="$h"
        break
      fi
    done <<<"$pubs"
    decl_h=""; decl_c=""; decl_src=""
    if IFS='|' read -r decl_h decl_c decl_src < <(dc_db_declared_pair "$dir" "${service:-$name}" "$cport" "$driver"); then
      :
    else
      decl_h=""; decl_c=""; decl_src=""
    fi
    IFS='|' read -r user pass db < <(printf '%s\n' "$env" | dc_db_creds_from_env "$driver")
    # Dash placeholders so bash read keeps columns when hosts/password are empty.
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
      "$id" "${service:-$name}" "$image" "$driver" "${cport:--}" \
      "${decl_h:--}" "${live:--}" "${user:--}" "${pass:--}" "${db:--}" "${decl_src:--}"
  done < <(dc_stack_rows "$dir")
}

dc_urlencode() {
  local s="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import urllib.parse,sys; sys.stdout.write(urllib.parse.quote(sys.argv[1], safe=""))' "$s"
    return
  fi
  printf '%s' "$s"
}

# scheme://user:pass@127.0.0.1:host/db
dc_db_url() {
  local driver="$1" user="$2" pass="$3" host="$4" port="$5" db="$6"
  local scheme userinfo path
  case "$driver" in
    postgres) scheme="postgresql"; path="/${db:-postgres}" ;;
    mysql) scheme="mysql"; path="/${db}" ;;
    redis) scheme="redis"; path="/${db:-0}" ;;
    mongo) scheme="mongodb"; path="/${db:-admin}" ;;
    *) scheme="$driver"; path="/${db}" ;;
  esac
  userinfo=""
  if [[ -n "$user" && -n "$pass" ]]; then
    userinfo="$(dc_urlencode "$user"):$(dc_urlencode "$pass")@"
  elif [[ -n "$user" ]]; then
    userinfo="$(dc_urlencode "$user")@"
  elif [[ -n "$pass" ]]; then
    userinfo=":$(dc_urlencode "$pass")@"
  fi
  printf '%s://%s%s:%s%s\n' "$scheme" "$userinfo" "$host" "$port" "$path"
}

# Print client id: tableplus|sequelace|dbeaver|rainfrog|lazysql
dc_db_pick_client() {
  local want="${1:-}" driver="${2:-}"
  if [[ -z "$want" && -n "${DC_DB_CLIENT:-}" ]]; then
    want="$DC_DB_CLIENT"
  fi
  want="$(printf '%s' "$want" | tr '[:upper:]' '[:lower:]')"
  case "$want" in
    tableplus|sequelace|sequel-ace|dbeaver|rainfrog|lazysql)
      printf '%s\n' "${want/sequel-ace/sequelace}"
      return 0
      ;;
    "") ;;
    *)
      echo "dc-db: unknown --client $want (tableplus|sequelace|dbeaver|rainfrog|lazysql)" >&2
      return 1
      ;;
  esac
  if [[ -d "/Applications/TablePlus.app" || -d "$HOME/Applications/TablePlus.app" ]]; then
    printf '%s\n' tableplus; return 0
  fi
  if [[ "$driver" == "mysql" ]] && [[ -d "/Applications/Sequel Ace.app" || -d "$HOME/Applications/Sequel Ace.app" ]]; then
    printf '%s\n' sequelace; return 0
  fi
  if command -v dbeaver >/dev/null 2>&1; then
    printf '%s\n' dbeaver; return 0
  fi
  if command -v rainfrog >/dev/null 2>&1; then
    printf '%s\n' rainfrog; return 0
  fi
  if command -v lazysql >/dev/null 2>&1; then
    printf '%s\n' lazysql; return 0
  fi
  return 1
}

dc_db_redact_url() {
  python3 -c 'import re,sys; print(re.sub(r":([^:@/]+)@", ":****@", sys.argv[1]))' "$1"
}
