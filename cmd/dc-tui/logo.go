package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Brand mark: host frame around a guest, phosphor pip (assets/branding/logo-mark.svg).
const (
	splashLast   = 5
	splashHold   = 3
	logoMinWidth = 28
)

var (
	logoFrame = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8680"))
	logoPip   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6FCF7B")).Bold(true)
	logoWord  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4F1EA"))
	logoTag   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8680"))
)

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
	if frame >= 4 {
		lines[1] = "│ ╭────╮●│"
	}
	return colorMark(lines, frame >= 4)
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
		if frame >= 4 {
			lines[1] = "│  ╭────╮● │"
		}
		block = colorMark(lines, frame >= 4)
	}
	word := ""
	if frame >= splashLast {
		word = "\n" + logoWord.Render("dc-cli") + "  " + logoTag.Render("host frame · guest")
	}
	body := block + word
	if width > 12 {
		body = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	}
	return body + "\n\n" + logoTag.Render("any key skips")
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
