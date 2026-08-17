package main

import (
	"fmt"
	"strings"
)

func (m model) netView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	var b strings.Builder
	head := logoWord.Render("dc-cli") + mutedStyle.Render("  nets  ") +
		headerStyle.Render("this folder")
	if n := len(m.net.Networks); n > 0 {
		head += mutedStyle.Render(fmt.Sprintf("  %d declared", n))
	}
	b.WriteString(clipBlock(head, w) + "\n")
	if m.netErr != "" {
		b.WriteString(errStyle.Render(trunc(m.netErr, w)) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(trunc("  NAME                  KIND       STATE", w)) + "\n")
	if len(m.net.Networks) == 0 && m.netErr == "" {
		b.WriteString(mutedStyle.Render("  (no declared compose networks)") + "\n")
	}
	for _, n := range m.net.Networks {
		b.WriteString(trunc(formatNetRow(n), w) + "\n")
	}
	b.WriteString("\n")
	if len(m.net.MissingCreatable) > 0 {
		b.WriteString(warnStyle.Render(trunc("create missing external nets and start? y/n", w)) + "\n")
	} else if len(m.net.MissingBlocked) > 0 {
		b.WriteString(errStyle.Render(trunc("blocked — overlay / ipam / unknown. not created.", w)) + "\n")
		b.WriteString(hintStyle.Render("q/esc/n back") + "\n")
	} else {
		b.WriteString(hintStyle.Render("q/esc/n back") + "\n")
	}
	return clipBlock(b.String(), w)
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
	return "  " + fmt.Sprintf("%-20s", trunc(n.Name, 20)) + "  " +
		fmt.Sprintf("%-8s", n.Kind) + "  " + style.Render(st) +
		mutedStyle.Render("  "+note)
}
