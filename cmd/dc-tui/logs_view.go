package main

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logIP    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	logTime  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	logGET   = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	logWrite = lipgloss.NewStyle().Foreground(lipgloss.Color("109")).Bold(true)
	logDel   = lipgloss.NewStyle().Foreground(lipgloss.Color("167")).Bold(true)
	logPath  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	logProto = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	log2xx   = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)
	log3xx   = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	log4xx   = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
	log5xx   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	logErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	logWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
	logInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
)

func (m model) logsView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	follow := "paused"
	if m.logFollow {
		follow = "follow"
	}
	head := logoWord.Render("dc-cli") + mutedStyle.Render("  logs  ") +
		headerStyle.Render(trunc(m.logName, 24)) + mutedStyle.Render("  "+shortID(m.logID)) +
		mutedStyle.Render("  "+follow) +
		mutedStyle.Render("  "+strconv.Itoa(len(m.logLines))+" lines")
	hint := hintStyle.Render("q/esc back  j/k scroll  f follow  g/G top/end")
	var b strings.Builder
	b.WriteString(clipBlock(head, w) + "\n")
	b.WriteString(clipBlock(hint, w) + "\n")
	page := m.logPage()
	start := m.logOff
	if start < 0 {
		start = 0
	}
	end := start + page
	if end > len(m.logLines) {
		end = len(m.logLines)
	}
	if len(m.logLines) == 0 {
		b.WriteString(mutedStyle.Render("waiting for docker logs…") + "\n")
	} else {
		for _, line := range m.logLines[start:end] {
			b.WriteString(trunc(colorizeLogLine(line), w) + "\n")
		}
	}
	return b.String()
}
