package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type hostReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Code          string `json:"code"`
	Summary       string `json:"summary"`
	Detail        *string `json:"detail"`
	EngineHint    string `json:"engineHint"`
	Remediation   string `json:"remediation"`
	Actions       string `json:"actions"`
	GuideURL      string `json:"guideUrl"`
	NextID        string `json:"nextId"`
	NextCommand   string `json:"nextCommand"`
	NextApply     string `json:"nextApply"`
	ApplyAllowed  bool   `json:"applyAllowed"`
}

// runHostDiagnose is swapped in tests.
var runHostDiagnose = defaultRunHostDiagnose

func defaultRunHostDiagnose() (hostReport, error) {
	script := `
set -euo pipefail
lib=""
if [[ -n "${DC_HOST_LIB:-}" && -f "${DC_HOST_LIB}" ]]; then
  lib="${DC_HOST_LIB}"
elif [[ -f "${XDG_DATA_HOME:-$HOME/.local/share}/dc-cli/generations/current/lib/dc-host.sh" ]]; then
  lib="${XDG_DATA_HOME:-$HOME/.local/share}/dc-cli/generations/current/lib/dc-host.sh"
elif [[ -f "$HOME/.config/devcontainer/dc-host.sh" ]]; then
  lib="$HOME/.config/devcontainer/dc-host.sh"
else
  up="$(command -v dc-up 2>/dev/null || true)"
  if [[ -n "$up" ]]; then
    # Prefer generation payload next to the real binary when shimmed.
    real="$up"
    if command -v realpath >/dev/null 2>&1; then
      real="$(realpath "$up" 2>/dev/null || echo "$up")"
    fi
    cand="$(dirname "$real")/../lib/dc-host.sh"
    [[ -f "$cand" ]] && lib="$cand"
  fi
fi
[[ -n "$lib" && -f "$lib" ]] || { echo '{"schemaVersion":1,"command":"dc-host","status":"blocker","code":"docker_cli_missing","summary":"dc-host.sh missing","detail":null,"engineHint":"unknown","remediation":"reinstall helpers via install.sh","actions":"run_doctor,retry","guideUrl":"https://docs.docker.com/desktop/"}'; exit 0; }
eng="$(dirname "$lib")/dc-engine.sh"
# shellcheck disable=SC1090
[[ -f "$eng" ]] && . "$eng"
# shellcheck disable=SC1090
rec="$(dirname "$lib")/dc-recover.sh"
# shellcheck disable=SC1090
. "$lib"
# shellcheck disable=SC1090
[[ -f "$rec" ]] && . "$rec"
dc_host_diagnose
if type dc_recover_plan >/dev/null 2>&1; then
  dc_recover_plan
fi
dc_host_json
`
	out, err := exec.Command("bash", "-c", script).Output()
	if err != nil {
		return hostReport{}, err
	}
	var hr hostReport
	if err := json.Unmarshal(out, &hr); err != nil {
		return hostReport{}, fmt.Errorf("host json: %w", err)
	}
	return hr, nil
}

func (h hostReport) blocked() bool {
	return h.Code != "" && h.Code != "ready"
}

func (h hostReport) canApply() bool {
	if !h.ApplyAllowed {
		return false
	}
	switch h.NextApply {
	case "launch_desktop", "colima_start", "sudo_start_docker", "sudo_docker_group",
		"context_use", "stop_extra_engine", "colima_grow_disk", "prune_safe",
		"create_nets", "take_ports", "try_sandbox":
		return true
	default:
		return false
	}
}

var runHostRecover = defaultRunHostRecover

func defaultRunHostRecover() error {
	cmd := exec.Command("dc-recover", "--yes")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openHostGuide(url string) error {
	if url == "" {
		url = "https://docs.docker.com/desktop/"
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("open unsupported on %s", runtime.GOOS)
	}
}

func copyHostText(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("clipboard unsupported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func colimaSetupText() string {
	return "brew install docker colima && colima start && dc-doctor"
}

func findRepoHostLib() string {
	if p := os.Getenv("DC_HOST_LIB"); p != "" {
		return p
	}
	return filepath.Clean(filepath.Join("lib", "dc-host.sh"))
}
