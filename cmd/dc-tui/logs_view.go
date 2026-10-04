package main

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

var (
	logIP    = mutedStyle
	logTime  = mutedStyle
	logGET   = okStyle.Bold(true)
	logWrite = secondaryStyle.Bold(true)
	logDel   = badStyle.Bold(true)
	logPath  = lipgloss.NewStyle().Foreground(textColor)
	logProto = mutedStyle
	log2xx   = okStyle.Bold(true)
	log3xx   = secondaryStyle.Bold(true)
	log4xx   = warnStyle.Bold(true)
	log5xx   = badStyle.Bold(true)
	logErr   = badStyle.Bold(true)
	logWarn  = warnStyle.Bold(true)
	logInfo  = secondaryStyle
)

func (m model) logsView() string {
	w, _ := m.consoleSize()
	follow := "paused"
	if m.logFollow {
		follow = "follow"
	}
	lines := m.screenHeader("LOGS · "+plainText(m.logName)+" · "+shortID(plainText(m.logID)), follow+" · "+strconv.Itoa(len(m.logLines))+" lines")
	start := min(max(0, m.logOff), m.logMaxOff())
	end := min(len(m.logLines), start+m.logPage())
	if len(m.logLines) == 0 {
		lines = append(lines, mutedStyle.Render("waiting for docker logs…"))
	} else {
		for _, line := range m.logLines[start:end] {
			lines = append(lines, trunc(colorizeLogLine(logDisplayText(line)), w))
		}
	}
	feedback := mutedStyle.Render("Following " + plainText(m.logName))
	if !m.logFollow {
		feedback = mutedStyle.Render("Paused — [f] Follow")
	}
	if m.logEnded {
		feedback = warnStyle.Render("Log stream ended — q back, then l to reopen")
	}
	if m.err != "" {
		feedback = badStyle.Render("✗ " + plainText(m.err))
	}
	return m.finishScreen(lines, feedback, []string{keyHint("j/k", "Scroll") + "  " + keyHint("f", "Follow") + "  " + keyHint("g/G", "Top/end") + "  " + keyHint("q/Esc", "Back")})
}
