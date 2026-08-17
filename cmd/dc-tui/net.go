package main

import (
	"encoding/json"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type netReport struct {
	SchemaVersion    int      `json:"schemaVersion"`
	Workspace        string   `json:"workspace"`
	Networks         []netRow `json:"networks"`
	MissingCreatable []string `json:"missingCreatable"`
	MissingBlocked   []string `json:"missingBlocked"`
}

type netRow struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Present   bool   `json:"present"`
	Creatable bool   `json:"creatable"`
	Driver    string `json:"driver"`
	Reason    string `json:"reason"`
}

type netMsg struct {
	rep netReport
	err error
}

// runNet is replaced in tests. Returns dc-net --json stdout.
var runNet = func(args ...string) ([]byte, error) {
	return exec.Command("dc-net", args...).Output()
}

func parseNet(out []byte) (netReport, error) {
	var rep netReport
	if err := json.Unmarshal(out, &rep); err != nil {
		return rep, err
	}
	return rep, nil
}

func (m model) fetchNets() tea.Cmd {
	ws := m.workspace
	return func() tea.Msg {
		out, err := runNet("--json", ws)
		if err != nil {
			return netMsg{err: err}
		}
		rep, err := parseNet(out)
		if err != nil {
			return netMsg{err: err}
		}
		return netMsg{rep: rep}
	}
}

func (m model) openNets() (model, tea.Cmd) {
	if m.fleet {
		return m.refuse("nets is this folder only.")
	}
	m.netOpen = true
	m.status = ""
	m.err = ""
	return m, m.fetchNets()
}

func (m model) closeNets() model {
	m.netOpen = false
	m.netErr = ""
	return m
}

func (m model) handleNetKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "esc", "n":
		m = m.closeNets()
		return m.withStatus("back from nets"), nil
	case "y":
		if len(m.net.MissingBlocked) > 0 {
			return m.withStatus("cannot create blocked network (overlay / ipam / unknown)"), nil
		}
		if len(m.net.MissingCreatable) == 0 {
			return m.withStatus("nothing to create"), nil
		}
		m = m.closeNets()
		return m.startLeave("start", "create-nets")
	}
	return m, nil
}

func (m model) applyNetMsg(msg netMsg) (model, tea.Cmd) {
	if !m.netOpen {
		return m, nil
	}
	if msg.err != nil {
		m.netErr = compactLines(msg.err.Error(), 2)
		return m, nil
	}
	m.net = msg.rep
	m.netErr = ""
	return m, nil
}

func netHeaderLine(rep netReport) string {
	if len(rep.Networks) == 0 {
		return ""
	}
	miss := len(rep.MissingCreatable) + len(rep.MissingBlocked)
	if miss == 0 {
		return "ok"
	}
	if len(rep.MissingCreatable) > 0 {
		return "missing " + rep.MissingCreatable[0]
	}
	return "blocked " + rep.MissingBlocked[0]
}
