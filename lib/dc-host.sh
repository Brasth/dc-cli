# Shared Docker host readiness. Source after optional dc-engine.sh.
# Never installs, starts, or stops an engine. Never sudo.
# shellcheck shell=bash

DC_HOST_CODE="${DC_HOST_CODE:-}"
DC_HOST_SUMMARY="${DC_HOST_SUMMARY:-}"
DC_HOST_DETAIL="${DC_HOST_DETAIL:-}"
DC_HOST_ENGINE_HINT="${DC_HOST_ENGINE_HINT:-unknown}"
DC_HOST_REMEDIATION="${DC_HOST_REMEDIATION:-}"
DC_HOST_ACTIONS="${DC_HOST_ACTIONS:-}"

dc_host_os() {
  uname -s 2>/dev/null || printf 'unknown\n'
}

dc_host_json_str() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '"%s"' "$s"
}

dc_host_set() {
  DC_HOST_CODE="$1"
  DC_HOST_SUMMARY="$2"
  DC_HOST_DETAIL="${3:-}"
  DC_HOST_ENGINE_HINT="${4:-unknown}"
  DC_HOST_REMEDIATION="${5:-}"
  DC_HOST_ACTIONS="${6:-retry,run_doctor}"
}

dc_host_desktop_installed() {
  local os
  os="$(dc_host_os)"
  case "$os" in
    Darwin)
      [[ -d /Applications/Docker.app ]] && return 0
      [[ -d "$HOME/Applications/Docker.app" ]] && return 0
      ;;
    Linux)
      [[ -e "$HOME/.docker/desktop/docker.sock" ]] && return 0
      [[ -e "$HOME/.docker/run/docker.sock" ]] && return 0
      command -v com.docker.cli >/dev/null 2>&1 && return 0
      ;;
  esac
  return 1
}

dc_host_colima_installed() {
  command -v colima >/dev/null 2>&1 && return 0
  [[ -e "$HOME/.colima/default/docker.sock" || -e "$HOME/.colima/docker.sock" ]] && return 0
  return 1
}

dc_host_engine_hint() {
  if dc_host_desktop_installed; then
    printf 'desktop\n'
    return 0
  fi
  if dc_host_colima_installed; then
    printf 'colima\n'
    return 0
  fi
  case "$(dc_host_os)" in
    Linux) printf 'linux\n' ;;
    *) printf 'unknown\n' ;;
  esac
}

dc_host_evidence_of_engine() {
  dc_host_desktop_installed && return 0
  dc_host_colima_installed && return 0
  [[ -e /var/run/docker.sock || -e /run/docker.sock ]] && return 0
  return 1
}

dc_host_desktop_guide_url() {
  printf '%s\n' "https://docs.docker.com/desktop/"
}

dc_host_remediation_for() {
  local code="$1" hint="$2" os
  os="$(dc_host_os)"
  case "$code" in
    ready)
      printf '%s\n' "host ready"
      ;;
    docker_cli_missing)
      if [[ "$os" == Darwin ]]; then
        printf '%s\n' "Install Docker Desktop (${DC_HOST_GUIDE:-$(dc_host_desktop_guide_url)}) or: brew install docker colima && colima start. Then: dc-doctor"
      else
        printf '%s\n' "Install Docker Engine or Desktop, ensure docker is on PATH, then: dc-doctor"
      fi
      ;;
    docker_engine_missing)
      if [[ "$os" == Darwin ]]; then
        printf '%s\n' "No Docker engine found. Beginners: install Docker Desktop (${DC_HOST_GUIDE:-$(dc_host_desktop_guide_url)}). Lightweight: brew install docker colima && colima start. Then: dc-doctor && dc-tui"
      else
        printf '%s\n' "Install Docker Engine or Docker Desktop, start it, then: dc-doctor && dc-tui"
      fi
      ;;
    docker_engine_stopped)
      case "$hint" in
        desktop)
          printf '%s\n' "Start Docker Desktop, wait until it is ready, then: dc-doctor && dc-up"
          ;;
        colima)
          printf '%s\n' "Start Colima: colima start. Then: dc-doctor && dc-up"
          ;;
        linux)
          printf '%s\n' "Start Docker Engine (e.g. sudo systemctl start docker) if you manage it yourself, then: dc-doctor && dc-up"
          ;;
        *)
          printf '%s\n' "Start your Docker engine (Desktop / colima start / dockerd), then: dc-doctor && dc-up"
          ;;
      esac
      ;;
    docker_permission_denied)
      printf '%s\n' "Docker permission denied. On Linux: add your user to the docker group and re-login, or use a rootless context. Then: dc-doctor"
      ;;
    docker_context_invalid)
      printf '%s\n' "Docker context is invalid. Try: unset DOCKER_HOST; docker context use default. Then: dc-engine && dc-doctor"
      ;;
    docker_split_brain)
      if type dc_engine_remediation >/dev/null 2>&1; then
        dc_engine_remediation
      else
        printf '%s\n' "Two Docker engines are live. Pick one with dc-engine --fix (dc-cli will not stop them)"
      fi
      ;;
    *)
      printf '%s\n' "Run: dc-doctor"
      ;;
  esac
}

dc_host_classify_unreachable() {
  local err="$1"
  local low hint
  low="$(printf '%s' "$err" | tr '[:upper:]' '[:lower:]')"
  hint="$(dc_host_engine_hint)"

  if printf '%s' "$low" | grep -Eq 'permission denied|access denied|operation not permitted'; then
    dc_host_set docker_permission_denied \
      "Docker permission denied" \
      "$err" \
      "$hint" \
      "$(dc_host_remediation_for docker_permission_denied "$hint")" \
      "retry,run_doctor"
    return 0
  fi

  if printf '%s' "$low" | grep -Eq 'context .+ does not exist|current context|docker context|known hosts|npipe://|error during connect.*context'; then
    dc_host_set docker_context_invalid \
      "Docker context is invalid" \
      "$err" \
      "$hint" \
      "$(dc_host_remediation_for docker_context_invalid "$hint")" \
      "retry,run_doctor"
    return 0
  fi

  if dc_host_evidence_of_engine; then
    dc_host_set docker_engine_stopped \
      "Docker engine is not running" \
      "$err" \
      "$hint" \
      "$(dc_host_remediation_for docker_engine_stopped "$hint")" \
      "retry,open_guide,copy_commands,run_doctor"
    return 0
  fi

  dc_host_set docker_engine_missing \
    "No Docker engine is installed" \
    "$err" \
    "$hint" \
    "$(dc_host_remediation_for docker_engine_missing "$hint")" \
    "open_guide,copy_commands,run_doctor,retry"
}

dc_host_diagnose() {
  local err="" rc=0 hint
  DC_HOST_GUIDE="$(dc_host_desktop_guide_url)"

  if ! command -v docker >/dev/null 2>&1; then
    hint="$(dc_host_engine_hint)"
    dc_host_set docker_cli_missing \
      "docker is not on PATH" \
      "" \
      "$hint" \
      "$(dc_host_remediation_for docker_cli_missing "$hint")" \
      "open_guide,copy_commands,run_doctor,retry"
    return 0
  fi

  set +e
  err="$(docker info 2>&1 >/dev/null)"
  rc=$?
  set -e

  if [[ "$rc" -ne 0 ]]; then
    dc_host_classify_unreachable "$err"
    return 0
  fi

  # Reachable: optional split-brain via engine helpers.
  if type dc_engine_refresh >/dev/null 2>&1; then
    dc_engine_refresh || true
    if [[ -n "${DC_ENGINE_LAST_EXTRA:-}" ]]; then
      hint="${DC_ENGINE_LAST_ENGINE:-unknown}"
      dc_host_set docker_split_brain \
        "More than one Docker engine is live" \
        "CLI=${DC_ENGINE_LAST_ENGINE:-unknown} extra=${DC_ENGINE_LAST_EXTRA}" \
        "$hint" \
        "$(dc_host_remediation_for docker_split_brain "$hint")" \
        "run_doctor,retry"
      return 0
    fi
    hint="${DC_ENGINE_LAST_ENGINE:-$(dc_host_engine_hint)}"
  else
    hint="$(dc_host_engine_hint)"
  fi

  dc_host_set ready \
    "Docker engine reachable" \
    "" \
    "$hint" \
    "$(dc_host_remediation_for ready "$hint")" \
    "retry"
}

dc_host_is_ready() {
  [[ "${DC_HOST_CODE:-}" == "ready" ]]
}

dc_host_print_human() {
  local os guide
  os="$(dc_host_os)"
  guide="${DC_HOST_GUIDE:-$(dc_host_desktop_guide_url)}"

  echo "Host Docker: ${DC_HOST_CODE:-unknown}"
  echo "  ${DC_HOST_SUMMARY:-}"
  if [[ -n "${DC_HOST_DETAIL:-}" ]]; then
    echo "  detail: ${DC_HOST_DETAIL}"
  fi
  if [[ -n "${DC_HOST_ENGINE_HINT:-}" && "${DC_HOST_ENGINE_HINT}" != "unknown" ]]; then
    echo "  engine hint: ${DC_HOST_ENGINE_HINT}"
  fi
  if [[ "${DC_HOST_CODE:-}" == "ready" ]]; then
    return 0
  fi
  echo
  echo "What to do:"
  case "${DC_HOST_CODE:-}" in
    docker_cli_missing|docker_engine_missing)
      echo "  Recommended (beginners): Docker Desktop"
      echo "    $guide"
      if [[ "$os" == Darwin ]]; then
        echo "  Lightweight alternative:"
        echo "    brew install docker colima"
        echo "    colima start"
      fi
      ;;
    docker_engine_stopped)
      case "${DC_HOST_ENGINE_HINT:-}" in
        desktop)
          echo "  Start Docker Desktop, wait until ready"
          ;;
        colima)
          echo "  colima start"
          ;;
        *)
          echo "  Start your Docker engine, then retry"
          ;;
      esac
      ;;
    docker_permission_denied)
      echo "  Fix Docker socket permissions / group membership, then retry"
      ;;
    docker_context_invalid)
      echo "  unset DOCKER_HOST"
      echo "  docker context use default"
      ;;
    docker_split_brain)
      if type dc_engine_fix_text >/dev/null 2>&1; then
        dc_engine_fix_text
      else
        echo "  dc-engine --fix"
      fi
      ;;
  esac
  if [[ -n "${DC_HOST_REMEDIATION:-}" && "${DC_HOST_CODE:-}" != "docker_split_brain" ]]; then
    echo "  ${DC_HOST_REMEDIATION}"
  fi
  echo
  echo "Then:"
  echo "  dc-doctor"
  echo "  dc-tui"
}

dc_host_json() {
  printf '{"schemaVersion":1,"command":"dc-host","status":%s,"code":%s,"summary":%s,"detail":%s,"engineHint":%s,"remediation":%s,"actions":%s,"guideUrl":%s}\n' \
    "$( [[ "${DC_HOST_CODE:-}" == "ready" ]] && printf '"ok"' || printf '"blocker"' )" \
    "$(dc_host_json_str "${DC_HOST_CODE:-unknown}")" \
    "$(dc_host_json_str "${DC_HOST_SUMMARY:-}")" \
    "$( [[ -n "${DC_HOST_DETAIL:-}" ]] && dc_host_json_str "$DC_HOST_DETAIL" || printf 'null' )" \
    "$(dc_host_json_str "${DC_HOST_ENGINE_HINT:-unknown}")" \
    "$(dc_host_json_str "${DC_HOST_REMEDIATION:-}")" \
    "$(dc_host_json_str "${DC_HOST_ACTIONS:-}")" \
    "$(dc_host_json_str "${DC_HOST_GUIDE:-$(dc_host_desktop_guide_url)}")"
}

dc_host_colima_copy_text() {
  printf '%s\n' "brew install docker colima && colima start && dc-doctor"
}
