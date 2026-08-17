# Read-only engine + container stats. Source after dc-common.sh.
# shellcheck shell=bash

DC_STATS_TIMEOUT="${DC_STATS_TIMEOUT:-2}"

dc_stats_run_deadline() {
  local secs="$1"
  shift
  local pid killer rc
  "$@" &
  pid=$!
  (
    sleep "$secs"
    kill "$pid" 2>/dev/null || true
    sleep 0.05
    kill -9 "$pid" 2>/dev/null || true
  ) &
  killer=$!
  wait "$pid"
  rc=$?
  kill "$killer" 2>/dev/null || true
  return "$rc"
}

dc_stats_colima_running() {
  local line p st
  command -v colima >/dev/null 2>&1 || return 1
  while read -r p st _rest; do
    [[ "$p" == "PROFILE" || -z "$p" ]] && continue
    if [[ "$st" == "Running" ]]; then
      return 0
    fi
  done < <(colima list 2>/dev/null)
  return 1
}

# colima | desktop | linux
dc_stats_engine() {
  local blob os name kern
  if dc_stats_colima_running; then
    printf '%s\n' colima
    return 0
  fi
  blob="$(docker info --format '{{.OperatingSystem}}|{{.Name}}' 2>/dev/null || true)"
  os="${blob%%|*}"
  name="${blob#*|}"
  if [[ "$os" == *[Dd]ocker\ [Dd]esktop* || "$name" == *docker-desktop* ]]; then
    printf '%s\n' desktop
    return 0
  fi
  kern="$(uname -s 2>/dev/null || true)"
  if [[ "$kern" == Linux ]]; then
    printf '%s\n' linux
    return 0
  fi
  printf '%s\n' desktop
}

dc_stats_skip_box() {
  local id="$1" image="${2:-}" name="${3:-}"
  [[ -n "$(dc_label "$id" "${DC_FWD_LABEL:-dc.forward.for}")" ]] && return 0
  [[ "$image" == *socat* ]] && return 0
  [[ "$name" == dc-fwd-* ]] && return 0
  return 1
}

# TSV: id name service image  (running, no sidecars)
dc_stats_box_rows() {
  local dir="${1:-.}" id name status service image
  local any=0
  while IFS=$'\t' read -r id name status service image; do
    [[ -n "$id" ]] || continue
    [[ "$status" == "running" ]] || continue
    dc_stats_skip_box "$id" "$image" "$name" && continue
    any=1
    printf '%s\t%s\t%s\t%s\n' "$id" "$name" "$service" "$image"
  done < <(dc_stack_rows "$dir")
  [[ "$any" -eq 1 ]] && return 0
  while IFS= read -r id; do
    [[ -n "$id" ]] || continue
    name="$(docker inspect -f '{{.Name}}' "$id" 2>/dev/null | sed 's#^/##')"
    status="$(docker inspect -f '{{.State.Status}}' "$id" 2>/dev/null || true)"
    service="$(dc_label "$id" "${DC_LABEL_SERVICE}")"
    image="$(docker inspect -f '{{.Config.Image}}' "$id" 2>/dev/null || true)"
    [[ "$status" == "running" ]] || continue
    dc_stats_skip_box "$id" "$image" "$name" && continue
    printf '%s\t%s\t%s\t%s\n' "$id" "$name" "$service" "$image"
  done < <(dc_ids_for_workspace "$dir")
}

dc_stats_info_caps() {
  local ncpu mem
  ncpu="$(docker info --format '{{.NCPU}}' 2>/dev/null || true)"
  mem="$(docker info --format '{{.MemTotal}}' 2>/dev/null || true)"
  [[ "$ncpu" =~ ^[0-9]+$ ]] || ncpu=0
  [[ "$mem" =~ ^[0-9]+$ ]] || mem=0
  printf '%s\t%s\n' "$ncpu" "$mem"
}

dc_stats_read_meminfo() {
  local f="$1" total avail
  [[ -f "$f" ]] || return 1
  total="$(awk '/^MemTotal:/ {print $2; exit}' "$f")"
  avail="$(awk '/^MemAvailable:/ {print $2; exit}' "$f")"
  [[ "$total" =~ ^[0-9]+$ ]] || return 1
  [[ "$avail" =~ ^[0-9]+$ ]] || avail=0
  printf '%s\t%s\n' "$((total * 1024))" "$(((total - avail) * 1024))"
}

dc_stats_load_pct() {
  local load nproc
  load="${1:-0}"
  nproc="${2:-1}"
  [[ "$nproc" =~ ^[0-9]+$ && "$nproc" -gt 0 ]] || nproc=1
  awk -v l="$load" -v n="$nproc" 'BEGIN { printf "%.1f", (l / n) * 100 }'
}

dc_stats_colima_guest() {
  local p st arch cpus mem disk rest load nproc used tot live=0 cpu=""
  local mi
  cpus=0
  mem=0
  while read -r p st arch cpus mem disk rest; do
    [[ "$p" == "PROFILE" || -z "$p" ]] && continue
    [[ "$st" == "Running" ]] || continue
    break
  done < <(colima list 2>/dev/null)
  [[ "$cpus" =~ ^[0-9]+$ ]] || cpus=0
  if [[ "$mem" == *GiB || "$mem" == *G ]]; then
    mem="$(awk -v s="${mem%%[A-Za-z]*}" 'BEGIN { printf "%d", s * 1024 * 1024 * 1024 }')"
  elif [[ ! "$mem" =~ ^[0-9]+$ ]]; then
    mem=0
  fi
  mi="$(colima ssh -- cat /proc/meminfo 2>/dev/null || true)"
  if [[ -n "$mi" ]]; then
    tot="$(printf '%s\n' "$mi" | awk '/^MemTotal:/ {print $2; exit}')"
    used="$(printf '%s\n' "$mi" | awk '/^MemAvailable:/ {print $2; exit}')"
    if [[ "$tot" =~ ^[0-9]+$ ]]; then
      [[ "$used" =~ ^[0-9]+$ ]] || used=0
      mem="$((tot * 1024))"
      used="$(((tot - used) * 1024))"
      nproc="$(colima ssh -- nproc 2>/dev/null || true)"
      load="$(colima ssh -- awk '{print $1}' /proc/loadavg 2>/dev/null || true)"
      cpu="$(dc_stats_load_pct "${load:-0}" "${nproc:-$cpus}")"
      live=1
      printf '%s\t%s\t%s\t%s\t%s\n' "$cpus" "$mem" "$used" "$cpu" "$live"
      return 0
    fi
  fi
  printf '%s\t%s\t\t\t0\n' "$cpus" "$mem"
}

dc_stats_linux_guest() {
  local root="${DC_STATS_PROC_ROOT:-/proc}" tot used load nproc cpu
  local pair
  pair="$(dc_stats_read_meminfo "$root/meminfo")" || return 1
  tot="${pair%%$'\t'*}"
  used="${pair#*$'\t'}"
  nproc="$(nproc 2>/dev/null || true)"
  [[ "$nproc" =~ ^[0-9]+$ ]] || nproc="$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1)"
  load="$(awk '{print $1}' "$root/loadavg" 2>/dev/null || echo 0)"
  cpu="$(dc_stats_load_pct "$load" "$nproc")"
  printf '%s\t%s\t%s\t%s\t1\n' "$nproc" "$tot" "$used" "$cpu"
}

# prints: label cpus memoryBytes [memoryUsedBytes] [cpuPct] live(0|1)
dc_stats_guest_fields() {
  local engine="$1" ncpu mem pair
  case "$engine" in
    colima)
      pair="$(dc_stats_colima_guest)"
      printf 'colima\t%s\n' "$pair"
      ;;
    linux)
      if pair="$(dc_stats_linux_guest)"; then
        printf 'host\t%s\n' "$pair"
      else
        pair="$(dc_stats_info_caps)"
        printf 'host\t%s\t%s\t\t\t0\n' "${pair%%$'\t'*}" "${pair#*$'\t'}"
      fi
      ;;
    *)
      pair="$(dc_stats_info_caps)"
      printf 'desktop\t%s\t%s\t\t\t0\n' "${pair%%$'\t'*}" "${pair#*$'\t'}"
      ;;
  esac
}

dc_stats_parse_docker() {
  python3 -c '
import json, re, sys

def b(s):
    s = (s or "").strip()
    m = re.match(r"([0-9.]+)\s*([A-Za-z]+)", s)
    if not m:
        return 0
    n, u = float(m.group(1)), m.group(2)
    mul = {
        "B": 1, "kB": 1000, "KB": 1000, "MB": 1000**2, "GB": 1000**3, "TB": 1000**4,
        "KiB": 1024, "MiB": 1024**2, "GiB": 1024**3, "TiB": 1024**4,
    }
    return int(n * mul.get(u, 1))

meta = {}
for line in sys.argv[1].split("\n"):
    if not line.strip():
        continue
    parts = line.split("\t")
    cid = parts[0]
    name = parts[1] if len(parts) > 1 else ""
    svc = parts[2] if len(parts) > 2 else ""
    meta[cid] = (name, svc)

out = []
for raw in sys.stdin:
    raw = raw.strip()
    if not raw:
        continue
    try:
        o = json.loads(raw)
    except Exception:
        continue
    cid = o.get("ID") or o.get("Container") or o.get("Name") or ""
    name, svc = "", ""
    for k, v in meta.items():
        if cid.startswith(k) or k.startswith(cid) or o.get("Name") == v[0]:
            cid, name, svc = k, v[0], v[1]
            break
    mem = o.get("MemUsage") or ""
    left, _, right = mem.partition("/")
    net = o.get("NetIO") or ""
    rx, _, tx = net.partition("/")
    cpu = (o.get("CPUPerc") or "0").strip().rstrip("%")
    try:
        cpu_f = float(cpu)
    except Exception:
        cpu_f = 0.0
    lim = b(right)
    out.append({
        "id": cid,
        "name": name or o.get("Name") or cid,
        "service": svc,
        "cpuPct": cpu_f,
        "memUsedBytes": b(left),
        "memLimitBytes": lim,
        "netRxBytes": b(rx),
        "netTxBytes": b(tx),
    })
print(json.dumps(out))
' "$1"
}

dc_stats_fmt_bytes() {
  local n="${1:-0}"
  awk -v n="$n" 'BEGIN {
    if (n < 0) n = 0
    if (n < 1024) { printf "%dB", n; exit }
    if (n < 1048576) { printf "%.0fK", n/1024; exit }
    if (n < 1073741824) { printf "%.0fM", n/1048576; exit }
    printf "%.1fG", n/1073741824
  }'
}
