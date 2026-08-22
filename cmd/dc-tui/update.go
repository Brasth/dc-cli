package main

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type updateCheckMsg struct {
	available bool
	installed string
	latest    string
}

// runUpdateCheck shells to dc-upgrade --check. Tests replace this.
var runUpdateCheck = func() updateCheckMsg {
	cmd := exec.Command("dc-upgrade", "--check")
	out, err := cmd.CombinedOutput()
	msg := parseUpdateCheck(string(out))
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			msg.available = true
			return msg
		}
		// exit 2 / missing binary / network — stay silent on the board
		return updateCheckMsg{}
	}
	return msg
}

func parseUpdateCheck(out string) updateCheckMsg {
	var msg updateCheckMsg
	state := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "installed:"):
			msg.installed = strings.TrimSpace(strings.TrimPrefix(line, "installed:"))
		case strings.HasPrefix(line, "latest:"):
			msg.latest = strings.TrimSpace(strings.TrimPrefix(line, "latest:"))
		case strings.HasPrefix(line, "state:"):
			state = strings.TrimSpace(strings.TrimPrefix(line, "state:"))
		}
	}
	msg.available = state == "available"
	return msg
}

func (m model) updateCheckCmd() tea.Cmd {
	return func() tea.Msg {
		return runUpdateCheck()
	}
}

func (m model) updateBanner() string {
	if !m.updateAvail || m.updateLatest == "" {
		return ""
	}
	from := m.updateInstalled
	if from == "" {
		from = cliVersion()
	}
	return "update " + from + " → " + m.updateLatest + " · U or dc upgrade"
}
