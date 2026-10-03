package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// workspaceView renders the w picker: filter line, rows (favorites first,
// then most recent), missing folders flagged, registry warnings.
func (m model) workspaceView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("workspaces") + mutedStyle.Render("  recent folders this board opened") + "\n\n")
	b.WriteString(kv("filter", m.wsFilter+"▏") + "\n\n")
	vis := m.wsVisible()
	switch {
	case m.wsLoading && len(m.wsItems) == 0:
		b.WriteString(mutedStyle.Render("  (loading…)") + "\n")
	case len(m.wsItems) == 0:
		b.WriteString(mutedStyle.Render("  (none yet — folders appear here after the board opens them)") + "\n")
	case len(vis) == 0:
		b.WriteString(mutedStyle.Render("  (no match)") + "\n")
	default:
		limit := len(vis)
		if m.height > 0 {
			if room := m.height - 9; room > 3 && limit > room {
				limit = room
			}
		}
		start := 0
		if m.wsCursor >= limit {
			start = m.wsCursor - limit + 1
		}
		for i := start; i < len(vis) && i < start+limit; i++ {
			b.WriteString(m.workspaceRow(vis[i], i == m.wsCursor, w) + "\n")
		}
	}
	if m.wsWarn != "" {
		b.WriteString("\n" + warnStyle.Render(trunc(m.wsWarn, w)) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render(trunc("type to filter  ↑/↓ move  enter open  ctrl+f favorite  ctrl+d forget  esc back", w)) + "\n")
	return clipBlock(b.String(), w)
}

func (m model) workspaceRow(it wsItem, selected bool, w int) string {
	mark := "  "
	if it.entry.Favorite {
		mark = "★ "
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
	label := trunc(" "+mark+it.label, room)
	path := trunc("  "+it.entry.Path, max(0, room-ansi.StringWidth(label)))
	rowStyle := lipgloss.NewStyle()
	if it.missing {
		rowStyle = mutedStyle
	}
	out := rowStyle.Render(label) + mutedStyle.Render(path) + badStyle.Render(tags)
	if selected {
		return rowHover.Width(w).Render(out)
	}
	return out
}
