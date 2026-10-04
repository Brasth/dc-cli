package main

import (
	"fmt"
)

func (m model) netView() string {
	w, h := m.consoleSize()
	lines := m.screenHeader("NETWORKS", fmt.Sprintf("%d declared", len(m.net.Networks)))
	lines = append(lines, mutedStyle.Render("  NAME                  KIND       STATE"))
	page := max(1, h-len(lines)-2)
	start := min(m.netOff, max(0, len(m.net.Networks)-page))
	if len(m.net.Networks) == 0 {
		lines = append(lines, mutedStyle.Render("(no declared compose networks)"))
	}
	for _, n := range m.net.Networks[start:min(len(m.net.Networks), start+page)] {
		lines = append(lines, trunc(formatNetRow(n), w))
	}
	feedback := m.boardFeedback()
	if len(m.net.MissingCreatable) > 0 {
		feedback = warnStyle.Render("create missing external nets and start? y/n")
	}
	if len(m.net.MissingBlocked) > 0 {
		feedback = badStyle.Render("blocked — overlay / ipam / unknown. not created.")
	}
	if m.netErr != "" {
		feedback = badStyle.Render("✗ " + plainText(m.netErr))
	}
	return m.finishScreen(lines, feedback, []string{keyHint("j/k", "Scroll") + "  " + keyHint("y", "Create missing + start") + "  " + keyHint("q/Esc/n", "Back")})
}

func formatNetRow(n netRow) string {
	st := "missing"
	style := warnStyle
	if n.Present {
		st = "present"
		style = okStyle
	} else if n.Reason != "missing" && n.Reason != "compose-managed" {
		style = badStyle
	} else if n.Reason == "compose-managed" {
		style = mutedStyle
	}
	note := n.Reason
	if n.Present {
		note = ""
	}
	return "  " + fmt.Sprintf("%-20s", cell(plainText(n.Name), 20)) + "  " +
		fmt.Sprintf("%-8s", plainText(n.Kind)) + "  " + style.Render(st) +
		mutedStyle.Render("  "+plainText(note))
}
