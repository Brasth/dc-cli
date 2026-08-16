package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Brand mark: host frame around a guest, phosphor pip (assets/branding/logo-mark.svg).
// Splash: pip patrols the outer host frame once, then docks on the guest.
// Header: parked dock pose. No idle orbit.
const (
	splashBuild  = 3
	splashHold   = 3
	logoMinWidth = 28
)

var (
	logoFrame = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8680"))
	logoPip   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6FCF7B")).Bold(true)
	logoWord  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4F1EA"))
	logoTag   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8680"))
)

// Clockwise around the outer host frame. Corners linger (slow-in/out).
// Compact is 4×10; splash is 5×12.
var compactPipPath = [][2]int{
	{0, 3}, {0, 6},
	{0, 9}, {0, 9},
	{2, 9},
	{3, 9}, {3, 9},
	{3, 5}, {3, 1},
	{1, 0},
}

var splashPipPath = [][2]int{
	{0, 3}, {0, 7},
	{0, 11}, {0, 11},
	{2, 11},
	{4, 11}, {4, 11},
	{4, 6}, {4, 1},
	{2, 0},
}

var splashLast = splashBuild + len(splashPipPath) + 1

func logoCompact(frame int) string {
	if frame < 1 {
		return strings.Repeat(" \n", 3) + " "
	}
	if frame < 2 {
		return colorFrame([]string{
			"╭        ╮",
			"          ",
			"          ",
			"╰        ╯",
		})
	}
	lines := []string{
		"╭────────╮",
		"│        │",
		"│        │",
		"╰────────╯",
	}
	if frame >= 3 {
		lines[1] = "│ ╭────╮ │"
		lines[2] = "│ ╰────╯ │"
	}
	pip := false
	if frame >= splashBuild+1 && frame < splashLast {
		i := frame - splashBuild - 1
		if i >= 0 && i < len(compactPipPath) {
			lines = overlayPip(lines, compactPipPath[i][0], compactPipPath[i][1])
			pip = true
		}
	}
	if frame >= splashLast {
		lines[1] = "│ ╭────╮●│"
		pip = true
	}
	return colorMark(lines, pip)
}

func logoSplash(frame, width int) string {
	var block string
	switch {
	case frame < 1:
		block = "\n\n\n\n"
	case frame < 2:
		block = colorFrame([]string{
			"╭          ╮",
			"            ",
			"            ",
			"            ",
			"╰          ╯",
		})
	default:
		lines := []string{
			"╭──────────╮",
			"│          │",
			"│          │",
			"│          │",
			"╰──────────╯",
		}
		if frame >= 3 {
			lines[1] = "│  ╭────╮  │"
			lines[2] = "│  │    │  │"
			lines[3] = "│  ╰────╯  │"
		}
		pip := false
		if frame >= splashBuild+1 && frame < splashLast {
			i := frame - splashBuild - 1
			if i >= 0 && i < len(splashPipPath) {
				lines = overlayPip(lines, splashPipPath[i][0], splashPipPath[i][1])
				pip = true
			}
		}
		if frame >= splashLast {
			lines[1] = "│  ╭────╮● │"
			pip = true
		}
		block = colorMark(lines, pip)
	}
	word := ""
	if frame >= splashLast {
		word = "\n" + logoWord.Render("dc-cli") + "  " + logoTag.Render(cliVersion()) + "  " + logoTag.Render("host frame · guest")
	}
	body := block + word
	if width > 12 {
		body = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	}
	return body + "\n\n" + logoTag.Render("any key skips")
}

func overlayPip(lines []string, row, col int) []string {
	if row < 0 || row >= len(lines) {
		return lines
	}
	rs := []rune(lines[row])
	if col < 0 || col >= len(rs) {
		return lines
	}
	out := append([]string(nil), lines...)
	cp := append([]rune(nil), rs...)
	cp[col] = '●'
	out[row] = string(cp)
	return out
}

func colorFrame(lines []string) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = logoFrame.Render(l)
	}
	return strings.Join(out, "\n")
}

func colorMark(lines []string, pip bool) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		if pip && strings.Contains(l, "●") {
			parts := strings.SplitN(l, "●", 2)
			out[i] = logoFrame.Render(parts[0]) + logoPip.Render("●") + logoFrame.Render(parts[1])
			continue
		}
		out[i] = logoFrame.Render(l)
	}
	return strings.Join(out, "\n")
}

func (m model) splashView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	return "\n" + logoSplash(m.splash, w) + "\n"
}

func (m model) headerLogo() string {
	if m.width > 0 && m.width < logoMinWidth {
		return logoWord.Render("dc-cli")
	}
	return logoCompact(splashLast)
}

func joinLogo(mark, info string, width int) string {
	if width > 0 && width < logoMinWidth {
		return info
	}
	gap := "  "
	return lipgloss.JoinHorizontal(lipgloss.Top, mark, gap, info)
}
