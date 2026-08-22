# Recover playbook. Source after dc-host.sh (and optional dc-engine.sh).
# v1: attach to an existing engine. Install-from-zero stays print/copy.
# shellcheck shell=bash

DC_RECOVER_ID="${DC_RECOVER_ID:-}"
DC_RECOVER_SUMMARY="${DC_RECOVER_SUMMARY:-}"
DC_RECOVER_COMMAND="${DC_RECOVER_COMMAND:-}"
DC_RECOVER_APPLY="${DC_RECOVER_APPLY:-}"
DC_RECOVER_APPLY_ALLOWED="${DC_RECOVER_APPLY_ALLOWED:-0}"
DC_RECOVER_VERIFY="${DC_RECOVER_VERIFY:-dc-host}"
DC_RECOVER_ESCALATE="${DC_RECOVER_ESCALATE:-dc-recover --report}"
DC_RECOVER_FOLDER_HINT="${DC_RECOVER_FOLDER_HINT:-}"

dc_recover_json_str() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '"%s"' "$s"
}

dc_recover_clear() {
  DC_RECOVER_ID=""
  DC_RECOVER_SUMMARY=""
  DC_RECOVER_COMMAND=""
  DC_RECOVER_APPLY="none"
  DC_RECOVER_APPLY_ALLOWED=0
  DC_RECOVER_VERIFY="dc-host"
  DC_RECOVER_ESCALATE="dc-recover --report"
}

dc_recover_set() {
  DC_RECOVER_ID="$1"
  DC_RECOVER_SUMMARY="$2"
  DC_RECOVER_COMMAND="$3"
  DC_RECOVER_APPLY="${4:-none}"
  DC_RECOVER_APPLY_ALLOWED="${5:-0}"
  DC_RECOVER_VERIFY="${6:-dc-host}"
}

dc_recover_desktop_guide() {
  if type dc_host_desktop_guide_url >/dev/null 2>&1; then
    dc_host_desktop_guide_url
    return 0
  fi
  printf '%s\n' "https://docs.docker.com/desktop/"
}

# Ranked next step from dc-host + optional folder hint.
dc_recover_plan() {
  local code hint folder extra
  dc_recover_clear
  code="${DC_HOST_CODE:-}"
  hint="${DC_HOST_ENGINE_HINT:-unknown}"
  folder="${DC_RECOVER_FOLDER_HINT:-}"
  extra="${DC_ENGINE_LAST_EXTRA:-}"

  case "$code" in
    docker_cli_missing)
      dc_recover_set install_cli_or_engine \
        "docker is not on PATH. Install an engine, then retry." \
        "See $(dc_recover_desktop_guide) or: brew install docker colima && colima start" \
        open_guide 1
      return 0
      ;;
    docker_engine_missing)
      dc_recover_set install_engine \
        "No Docker engine is installed. Install Desktop or Colima, then retry." \
        "See $(dc_recover_desktop_guide) or: brew install docker colima && colima start" \
        open_guide 1
      return 0
      ;;
    docker_engine_stopped)
      case "$hint" in
        desktop)
          dc_recover_set start_desktop \
            "Docker Desktop is installed but not running." \
            "Start Docker Desktop, wait until ready" \
            launch_desktop 1
          ;;
        colima)
          dc_recover_set start_colima \
            "Colima is installed but not running." \
            "colima start" \
            colima_start 1
          ;;
        linux)
          dc_recover_set start_linux_docker \
            "Docker Engine is installed but not running." \
            "sudo systemctl start docker" \
            sudo_start_docker 1
          ;;
        *)
          dc_recover_set start_unknown_engine \
            "Docker engine is not running." \
            "Start Docker Desktop / colima start / dockerd, then dc-doctor" \
            open_guide 1
          ;;
      esac
      return 0
      ;;
    docker_permission_denied)
      dc_recover_set fix_socket_group \
        "Docker socket permission denied." \
        "sudo usermod -aG docker \"$USER\"  # then re-login" \
        sudo_docker_group 1
      return 0
      ;;
    docker_context_invalid)
      dc_recover_set reset_context \
        "Docker context is invalid." \
        "unset DOCKER_HOST; docker context use default" \
        context_use 1
      return 0
      ;;
    docker_split_brain)
      dc_recover_set pick_engine \
        "More than one Docker engine is live. Keep this CLI engine; stop the extra." \
        "docker context use <recommended>; stop extra (${extra:-unknown})" \
        stop_extra_engine 1
      return 0
      ;;
    ready|"")
      ;;
    *)
      dc_recover_set run_doctor \
        "Host needs diagnosis." \
        "dc-doctor" \
        none 0
      return 0
      ;;
  esac

  case "$folder" in
    enospc)
      dc_recover_set reclaim_disk \
        "No space left. Reclaim Docker cache safely." \
        "dc-df && dc-prune --yes" \
        prune_safe 1 "dc-up"
      ;;
    colima_full)
      dc_recover_set grow_colima_disk \
        "Colima guest disk is still full after prune." \
        "colima stop && colima start --disk ${DC_RECOVER_DISK_GIB:-60}" \
        colima_grow_disk 1
      ;;
    missing_nets)
      dc_recover_set ensure_nets \
        "Declared external networks are missing." \
        "dc-up --create-nets" \
        create_nets 1
      ;;
    port_clash_labeled)
      dc_recover_set take_ports \
        "A labeled foreign stack holds this folder's host port." \
        "dc-up --take-ports" \
        take_ports 1
      ;;
    port_clash_unlabeled)
      dc_recover_set report_holders \
        "An unlabeled process holds the host port. We will not stop it." \
        "Stop the holder listed by dc-up, then retry" \
        none 0
      ;;
    kind_none)
      dc_recover_set try_sandbox \
        "This folder has no .devcontainer or compose file." \
        "dc-try" \
        none 0
      ;;
    *)
      dc_recover_set ready \
        "Docker engine is reachable." \
        "dc-tui or dc-up" \
        none 0
      ;;
  esac
}

dc_recover_json() {
  local allowed=false
  [[ "${DC_RECOVER_APPLY_ALLOWED:-0}" == "1" ]] && allowed=true
  printf '{"schemaVersion":1,"command":"dc-recover","host":{"status":%s,"code":%s,"summary":%s,"detail":%s,"engineHint":%s,"guideUrl":%s},"next":{"id":%s,"summary":%s,"command":%s,"apply":%s,"applyAllowed":%s,"verify":%s,"escalate":%s}}\n' \
    "$( [[ "${DC_HOST_CODE:-}" == "ready" ]] && printf '"ok"' || printf '"blocker"' )" \
    "$(dc_recover_json_str "${DC_HOST_CODE:-unknown}")" \
    "$(dc_recover_json_str "${DC_HOST_SUMMARY:-}")" \
    "$( [[ -n "${DC_HOST_DETAIL:-}" ]] && dc_recover_json_str "$DC_HOST_DETAIL" || printf 'null' )" \
    "$(dc_recover_json_str "${DC_HOST_ENGINE_HINT:-unknown}")" \
    "$(dc_recover_json_str "${DC_HOST_GUIDE:-$(dc_recover_desktop_guide)}")" \
    "$(dc_recover_json_str "${DC_RECOVER_ID:-}")" \
    "$(dc_recover_json_str "${DC_RECOVER_SUMMARY:-}")" \
    "$(dc_recover_json_str "${DC_RECOVER_COMMAND:-}")" \
    "$(dc_recover_json_str "${DC_RECOVER_APPLY:-none}")" \
    "$allowed" \
    "$(dc_recover_json_str "${DC_RECOVER_VERIFY:-dc-host}")" \
    "$(dc_recover_json_str "${DC_RECOVER_ESCALATE:-dc-recover --report}")"
}

dc_recover_print_human() {
  echo "dc-recover"
  echo "  host   ${DC_HOST_CODE:-unknown}${DC_HOST_ENGINE_HINT:+  hint=${DC_HOST_ENGINE_HINT}}"
  echo "  next   ${DC_RECOVER_ID:-unknown}"
  echo "  ${DC_RECOVER_SUMMARY:-}"
  if [[ -n "${DC_RECOVER_COMMAND:-}" ]]; then
    echo
    echo "  command:"
    echo "    ${DC_RECOVER_COMMAND}"
  fi
  if [[ "${DC_RECOVER_APPLY_ALLOWED:-0}" == "1" && "${DC_RECOVER_APPLY:-none}" != "none" && "${DC_RECOVER_APPLY}" != "open_guide" && "${DC_RECOVER_APPLY}" != "copy" ]]; then
    echo
    echo "  apply with: dc-recover --yes"
  fi
  echo
  echo "  still stuck: dc-recover --report"
}

dc_recover_apply_launch_desktop() {
  case "$(uname -s 2>/dev/null || true)" in
    Darwin)
      open -a Docker
      ;;
    Linux)
      if command -v gtk-launch >/dev/null 2>&1 && gtk-launch docker-desktop >/dev/null 2>&1; then
        return 0
      fi
      if [[ -x /opt/docker-desktop/bin/docker-desktop ]]; then
        /opt/docker-desktop/bin/docker-desktop >/dev/null 2>&1 &
        return 0
      fi
      echo "dc-recover: start Docker Desktop from the app menu, then retry" >&2
      return 1
      ;;
    *)
      echo "dc-recover: start Docker Desktop, then retry" >&2
      return 1
      ;;
  esac
}

dc_recover_apply_colima_start() {
  command -v colima >/dev/null 2>&1 || { echo "dc-recover: colima not on PATH" >&2; return 1; }
  echo "dc-recover: colima start"
  colima start
}

dc_recover_apply_sudo_start_docker() {
  echo "dc-recover: sudo systemctl start docker"
  if sudo systemctl start docker 2>/dev/null; then
    return 0
  fi
  if pgrep -x dockerd >/dev/null 2>&1; then
    return 0
  fi
  echo "dc-recover: systemctl unavailable — starting dockerd"
  sudo dockerd >/tmp/dockerd-dc-recover.log 2>&1 &
  local i
  for i in $(seq 1 50); do
    [[ -S /var/run/docker.sock || -S /run/docker.sock ]] && return 0
    sleep 0.2
  done
  echo "dc-recover: dockerd did not become ready" >&2
  return 1
}

dc_recover_apply_sudo_docker_group() {
  echo "dc-recover: sudo usermod -aG docker ${USER}"
  sudo usermod -aG docker "$USER"
  echo "dc-recover: re-login (or newgrp docker) before dc-tui will work."
}

dc_recover_apply_context_use() {
  if type dc_engine_refresh >/dev/null 2>&1; then
    dc_engine_refresh || true
  fi
  if type dc_engine_apply_fix >/dev/null 2>&1; then
    dc_engine_apply_fix || true
    return 0
  fi
  echo "dc-recover: unset DOCKER_HOST in this shell, then: docker context use default" >&2
  return 1
}

dc_recover_apply_stop_extra() {
  local extra
  extra="${DC_ENGINE_LAST_EXTRA:-}"
  if type dc_engine_refresh >/dev/null 2>&1; then
    dc_engine_refresh || true
    extra="${DC_ENGINE_LAST_EXTRA:-$extra}"
  fi
  if type dc_engine_apply_fix >/dev/null 2>&1; then
    dc_engine_apply_fix || true
  fi
  extra="${DC_ENGINE_LAST_EXTRA:-$extra}"
  if [[ -z "$extra" ]]; then
    echo "dc-recover: no extra engine recorded"
    return 0
  fi
  case "$extra" in
    *colima*)
      command -v colima >/dev/null 2>&1 || { echo "dc-recover: colima not on PATH" >&2; return 1; }
      echo "dc-recover: colima stop (extra engine)"
      colima stop
      ;;
    *desktop*)
      case "$(uname -s 2>/dev/null || true)" in
        Darwin)
          echo "dc-recover: quit Docker Desktop (extra engine)"
          osascript -e 'quit app "Docker"' 2>/dev/null || open -a Docker
          ;;
        *)
          echo "dc-recover: quit Docker Desktop yourself (extra engine)" >&2
          return 1
          ;;
      esac
      ;;
    *linux*)
      echo "dc-recover: sudo systemctl stop docker.socket docker.service (extra engine)"
      sudo systemctl stop docker.socket docker.service
      ;;
    *)
      echo "dc-recover: stop the extra engine ($extra) yourself" >&2
      return 1
      ;;
  esac
}

dc_recover_apply_colima_grow() {
  local n="${DC_RECOVER_DISK_GIB:-60}"
  command -v colima >/dev/null 2>&1 || { echo "dc-recover: colima not on PATH" >&2; return 1; }
  echo "dc-recover: colima stop && colima start --disk ${n}"
  colima stop
  colima start --disk "$n"
}

dc_recover_apply_prune_safe() {
  command -v dc-prune >/dev/null 2>&1 || { echo "dc-recover: dc-prune not on PATH" >&2; return 1; }
  echo "dc-recover: dc-prune --yes"
  dc-prune --yes
}

dc_recover_apply_create_nets() {
  command -v dc-up >/dev/null 2>&1 || { echo "dc-recover: dc-up not on PATH" >&2; return 1; }
  echo "dc-recover: dc-up --create-nets"
  dc-up --create-nets
}

dc_recover_apply_take_ports() {
  command -v dc-up >/dev/null 2>&1 || { echo "dc-recover: dc-up not on PATH" >&2; return 1; }
  echo "dc-recover: dc-up --take-ports"
  dc-up --take-ports
}

dc_recover_apply_open_guide() {
  local url
  url="$(dc_recover_desktop_guide)"
  echo "dc-recover: open $url"
  case "$(uname -s 2>/dev/null || true)" in
    Darwin) open "$url" ;;
    Linux) xdg-open "$url" >/dev/null 2>&1 || true ;;
    *) echo "$url" ;;
  esac
}

# Dispatch one allowlisted apply. Unknown / none → 1.
dc_recover_apply() {
  local apply="${1:-${DC_RECOVER_APPLY:-none}}"
  case "$apply" in
    launch_desktop) dc_recover_apply_launch_desktop ;;
    colima_start) dc_recover_apply_colima_start ;;
    sudo_start_docker) dc_recover_apply_sudo_start_docker ;;
    sudo_docker_group) dc_recover_apply_sudo_docker_group ;;
    context_use) dc_recover_apply_context_use ;;
    stop_extra_engine) dc_recover_apply_stop_extra ;;
    colima_grow_disk) dc_recover_apply_colima_grow ;;
    prune_safe) dc_recover_apply_prune_safe ;;
    create_nets) dc_recover_apply_create_nets ;;
    take_ports) dc_recover_apply_take_ports ;;
    open_guide) dc_recover_apply_open_guide ;;
    copy|none|"")
      echo "dc-recover: nothing to apply (${apply:-none})" >&2
      return 1
      ;;
    *)
      echo "dc-recover: apply not allowed: $apply" >&2
      return 1
      ;;
  esac
}

dc_recover_write_report() {
  local dest="$1"
  mkdir -p "$dest"
  {
    echo "dc-cli recover report"
    echo "redact credentials before sharing. do not paste .env or docker inspect."
    echo "host code: ${DC_HOST_CODE:-unknown}"
    echo "next: ${DC_RECOVER_ID:-unknown}"
  } >"$dest/README.txt"
  dc_recover_json >"$dest/recover.json"
  if type dc_host_json >/dev/null 2>&1; then
    dc_host_json >"$dest/host.json"
  else
    printf '%s\n' "{\"code\":\"${DC_HOST_CODE:-unknown}\"}" >"$dest/host.json"
  fi
  if command -v dc-engine >/dev/null 2>&1; then
    dc-engine --json >"$dest/engine.json" 2>/dev/null || printf '%s\n' '{}' >"$dest/engine.json"
  fi
  if command -v dc-doctor >/dev/null 2>&1; then
    dc-doctor --json . >"$dest/doctor.json" 2>/dev/null || true
  fi
  if command -v dc-df >/dev/null 2>&1; then
    dc-df --json >"$dest/df.json" 2>/dev/null || true
  fi
}
