package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// workspaceView renders the w picker: filter line, rows (favorites first,
// then most recent), missing folders flagged, registry warnings.
func (m model) workspaceView() string {
	w, h := m.consoleSize()
	lines := m.screenHeader("WORKSPACES · recent folders this board opened", "favorites first")
	lines = append(lines, kv("filter", plainText(m.wsFilter)+"▏"))
	vis := m.wsVisible()
	switch {
	case m.wsLoading && len(m.wsItems) == 0:
		lines = append(lines, mutedStyle.Render("(loading…)"))
	case len(m.wsItems) == 0:
		lines = append(lines, mutedStyle.Render("(none yet — folders appear here after the board opens them)"))
	case len(vis) == 0:
		lines = append(lines, mutedStyle.Render("(no match)"))
	default:
		page := max(1, h-len(lines)-3)
		start := max(0, m.wsCursor-page+1)
		for i := start; i < len(vis) && i < start+page; i++ {
			lines = append(lines, m.workspaceRow(vis[i], i == m.wsCursor, w))
		}
	}
	feedback := mutedStyle.Render("Type to filter · works while Docker is down")
	if m.wsWarn != "" {
		feedback = warnStyle.Render("! " + plainText(m.wsWarn))
	}
	return m.finishScreen(lines, feedback, []string{keyHint("↑/↓", "Move") + "  " + keyHint("enter", "Open") + "  " + keyHint("esc", "Back"), keyHint("ctrl+f", "Favorite") + "  " + keyHint("ctrl+d", "Forget")})
}

func (m model) workspaceRow(it wsItem, selected bool, w int) string {
	mark := "  "
	if selected {
		mark = "> "
	}
	if it.entry.Favorite {
		mark += "★ "
	}
	tags := ""
	if it.missing {
		tags += "  missing"
	}
	if !m.fleet && sameWorkspace(it.entry.Path, m.workspace) {
		tags += "  current"
	}
	// Tags always stay visible; the path gives way first, then the label.
	room := max(8, w-ansi.StringWidth(tags))
	label := trunc(" "+mark+plainText(it.label), room)
	path := trunc("  "+plainText(it.entry.Path), max(0, room-ansi.StringWidth(label)))
	rowStyle := lipgloss.NewStyle()
	if it.missing {
		rowStyle = mutedStyle
	}
	out := rowStyle.Render(label) + mutedStyle.Render(path) + badStyle.Render(tags)
	if selected {
		return selectedRow(out, w)
	}
	return out
}
