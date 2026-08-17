package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const pulseEvery = 3 * time.Second

type statsSnapshot struct {
	SchemaVersion int        `json:"schemaVersion"`
	Engine        string     `json:"engine"`
	Guest         statsGuest `json:"guest"`
	Containers    []statsBox `json:"containers"`
}

type statsGuest struct {
	Label           string   `json:"label"`
	CPUs            int      `json:"cpus"`
	MemoryBytes     int64    `json:"memoryBytes"`
	CPUPct          *float64 `json:"cpuPct"`
	MemoryUsedBytes *int64   `json:"memoryUsedBytes"`
	Live            bool     `json:"live"`
}

type statsBox struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Service       string  `json:"service"`
	CPUPct        float64 `json:"cpuPct"`
	MemUsedBytes  int64   `json:"memUsedBytes"`
	MemLimitBytes int64   `json:"memLimitBytes"`
	NetRxBytes    int64   `json:"netRxBytes"`
	NetTxBytes    int64   `json:"netTxBytes"`
}

type topMsg struct {
	snap statsSnapshot
	err  error
}

type pulseMsg struct {
	line string
	err  error
}

type pulseTickMsg struct{}

// runStats is replaced in tests. Returns dc-stats --json stdout.
var runStats = func(args ...string) ([]byte, error) {
	return exec.Command("dc-stats", args...).Output()
}

func (m model) idleForPulse() bool {
	return !m.splashOn && !m.logOpen && !m.topOpen && m.leaving == "" && !m.fleet && !m.quitting
}

func (m model) pulseCmd() tea.Cmd {
	return tea.Tick(pulseEvery, func(time.Time) tea.Msg {
		return pulseTickMsg{}
	})
}

func parseStats(out []byte) (statsSnapshot, error) {
	var snap statsSnapshot
	if err := json.Unmarshal(out, &snap); err != nil {
		return snap, err
	}
	return snap, nil
}

func (m model) fetchStats() tea.Cmd {
	ws := m.workspace
	return func() tea.Msg {
		out, err := runStats("--json", ws)
		if err != nil {
			return topMsg{err: err}
		}
		snap, err := parseStats(out)
		if err != nil {
			return topMsg{err: err}
		}
		return topMsg{snap: snap}
	}
}

func (m model) fetchPulse() tea.Cmd {
	ws := m.workspace
	return func() tea.Msg {
		out, err := runStats("--json", ws)
		if err != nil {
			return pulseMsg{err: err}
		}
		snap, err := parseStats(out)
		if err != nil {
			return pulseMsg{err: err}
		}
		line := pulseLine(snap)
		if line == "" {
			return pulseMsg{}
		}
		return pulseMsg{line: line}
	}
}

func pulseLine(snap statsSnapshot) string {
	if len(snap.Containers) == 0 {
		return ""
	}
	box := snap.Containers[0]
	for _, c := range snap.Containers {
		if c.Service == "app" {
			box = c
			break
		}
	}
	return fmt.Sprintf("cpu %.1f%%  mem %s", box.CPUPct, fmtMem(box.MemUsedBytes, box.MemLimitBytes))
}

func (m model) openTop() (model, tea.Cmd) {
	if m.fleet {
		return m.refuse("top is this folder only.")
	}
	if len(m.rows) == 0 || m.rows[0].ID == "" {
		return m.refuse("No container to sample.")
	}
	m.topOpen = true
	m.topCursor = 0
	m.topHist = map[string]sparkHist{}
	m.status = ""
	m.err = ""
	m, stream := m.beginTopStream()
	return m, tea.Batch(m.fetchStats(), stream)
}

func (m model) closeTop() model {
	if m.topStop != nil {
		m.topStop()
	}
	m.topOpen = false
	m.topStop = nil
	m.topR = nil
	m.topHist = nil
	m.topStale = false
	return m
}

func (m model) handleTopKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "esc", "ctrl+c", "t":
		m = m.closeTop()
		return m.withStatus("back from top"), nil
	case "j", "down":
		if n := len(m.topSnap.Containers); n > 0 {
			m.topCursor++
			if m.topCursor >= n {
				m.topCursor = n - 1
			}
		}
		return m, nil
	case "k", "up":
		m.topCursor--
		if m.topCursor < 0 {
			m.topCursor = 0
		}
		return m, nil
	}
	return m, nil
}

func (m model) applyTopMsg(msg topMsg) (model, tea.Cmd) {
	if !m.topOpen {
		return m, nil
	}
	if msg.err != nil {
		m.topErr = compactLines(msg.err.Error(), 2)
		return m, nil
	}
	m.topSnap = msg.snap
	m.topErr = ""
	if m.topCursor >= len(m.topSnap.Containers) {
		m.topCursor = 0
	}
	return m, nil
}

func (m model) applyPulse(msg pulseMsg) (model, tea.Cmd) {
	if msg.err == nil && msg.line != "" {
		m.pulse = msg.line
	}
	return m, m.pulseCmd()
}

func fmtBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.0fK", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.0fM", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fG", float64(n)/(1024*1024*1024))
	}
}

func fmtMem(used, limit int64) string {
	if limit <= 0 {
		return fmtBytes(used) + " / —"
	}
	return fmtBytes(used) + " / " + fmtBytes(limit)
}

func guestLine(g statsGuest) string {
	cap := fmtBytes(g.MemoryBytes)
	if !g.Live {
		return fmt.Sprintf("GUEST  %s  %d cpu  %s cap  live n/a", g.Label, g.CPUs, cap)
	}
	used := int64(0)
	if g.MemoryUsedBytes != nil {
		used = *g.MemoryUsedBytes
	}
	cpu := 0.0
	if g.CPUPct != nil {
		cpu = *g.CPUPct
	}
	return fmt.Sprintf("GUEST  %s  %d cpu  %s / %s  cpu %.1f%%", g.Label, g.CPUs, fmtBytes(used), cap, cpu)
}

func boxService(b statsBox) string {
	if b.Service != "" {
		return b.Service
	}
	if b.Name != "" {
		return b.Name
	}
	return "-"
}
