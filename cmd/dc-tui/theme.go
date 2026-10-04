package main

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Phosphor Console tokens. CompleteColor keeps the semantic hierarchy on
// true-color, ANSI-256 and ANSI-16 terminals; Lip Gloss handles no-color output.
var (
	textColor      = lipgloss.CompleteColor{TrueColor: "#F4F1EA", ANSI256: "255", ANSI: "7"}
	mutedColor     = lipgloss.CompleteColor{TrueColor: "#8A8680", ANSI256: "245", ANSI: "8"}
	secondaryColor = lipgloss.CompleteColor{TrueColor: "#B5B0A7", ANSI256: "249", ANSI: "7"}
	accentColor    = lipgloss.CompleteColor{TrueColor: "#6FCF7B", ANSI256: "114", ANSI: "10"}
	warningColor   = lipgloss.CompleteColor{TrueColor: "#E2B65C", ANSI256: "179", ANSI: "11"}
	errorColor     = lipgloss.CompleteColor{TrueColor: "#E47777", ANSI256: "210", ANSI: "9"}
	borderColor    = lipgloss.CompleteColor{TrueColor: "#454842", ANSI256: "238", ANSI: "8"}
	selectedColor  = lipgloss.CompleteColor{TrueColor: "#233126", ANSI256: "22", ANSI: "0"}
	surfaceColor   = lipgloss.CompleteColor{TrueColor: "#151715", ANSI256: "234", ANSI: "0"}
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(textColor)
	mutedStyle     = lipgloss.NewStyle().Foreground(mutedColor)
	secondaryStyle = lipgloss.NewStyle().Foreground(secondaryColor)
	okStyle        = lipgloss.NewStyle().Foreground(accentColor)
	badStyle       = lipgloss.NewStyle().Foreground(errorColor)
	warnStyle      = lipgloss.NewStyle().Foreground(warningColor)
	hintStyle      = secondaryStyle
	errStyle       = badStyle
	statusStyle    = secondaryStyle
	headerStyle    = titleStyle
	labelStyle     = mutedStyle.Width(10)
	rowHover       = lipgloss.NewStyle().Foreground(textColor).Background(selectedColor)
	hoverStyle     = lipgloss.NewStyle().Foreground(textColor).Background(surfaceColor)
	ruleStyle      = lipgloss.NewStyle().Foreground(borderColor)
	btnStyle       = lipgloss.NewStyle().Foreground(accentColor)
	btnHover       = lipgloss.NewStyle().Foreground(surfaceColor).Background(accentColor).Bold(true)
	btnMeta        = secondaryStyle
	btnMetaHover   = hoverStyle
	btnDisabled    = mutedStyle
	btnDanger      = secondaryStyle
	btnDangerH     = badStyle.Bold(true)
)

// Only display text is sanitized; command arguments and workspace paths used
// for execution remain untouched. Never let container names emit terminal codes.
func plainText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(s))
}

func cell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = trunc(s, width)
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}

func paneTitle(title string, width int) []string {
	return []string{secondaryStyle.Bold(true).Render(trunc(title, width)), ruleStyle.Render(strings.Repeat("─", max(0, width)))}
}

func keyHint(key, label string) string {
	return okStyle.Render("["+key+"]") + secondaryStyle.Render(" "+label)
}

// finishScreen pins feedback and shortcuts below content and bounds both axes.
func (m model) finishScreen(lines []string, feedback string, hints []string) string {
	w, h := m.consoleSize()
	bottom := append([]string{feedback}, hints...)
	if len(bottom) > h {
		bottom = bottom[len(bottom)-h:]
	}
	room := h - len(bottom)
	if len(lines) > room {
		lines = lines[:room]
	}
	for len(lines) < room {
		lines = append(lines, "")
	}
	lines = append(lines, bottom...)
	return clipBlock(strings.Join(lines, "\n"), w)
}

// Strip nested SGR resets before painting a whole row, otherwise a child's
// reset cancels the selection background halfway through the row.
func selectedRow(line string, width int) string {
	return rowHover.Render(cell(ansi.Strip(line), width))
}
func hoveredRow(line string, width int) string {
	return hoverStyle.Render(cell(ansi.Strip(line), width))
}
