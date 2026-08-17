package main

import (
	"fmt"
	"strings"
)

func (m model) topView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	var b strings.Builder
	head := logoWord.Render("dc-cli") + mutedStyle.Render("  top  ") +
		headerStyle.Render(trunc(m.topSnap.Engine, 12))
	if m.topSnap.SchemaVersion > 0 {
		head += mutedStyle.Render(fmt.Sprintf("  %d boxes", len(m.topSnap.Containers)))
	}
	if m.topStale {
		head += "  " + warnStyle.Render("stale")
	}
	b.WriteString(clipBlock(head, w) + "\n")
	b.WriteString(clipBlock(guestLine(m.topSnap.Guest), w) + "\n")
	if m.topErr != "" {
		b.WriteString(errStyle.Render(trunc(m.topErr, w)) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(trunc("  SERVICE           CPU     MEM              NET", w)) + "\n")
	if len(m.topSnap.Containers) == 0 && m.topErr == "" {
		b.WriteString(mutedStyle.Render("  (no running boxes)") + "\n")
	}
	sparkW := 12
	if w < 70 {
		sparkW = 8
	}
	for i, c := range m.topSnap.Containers {
		line := formatTopRow(c, w)
		if h, ok := m.topHist[c.ID]; ok {
			sp := sparkline(h.cpu, sparkW)
			if sp != "" {
				line = trunc(line+"  "+sp, w)
			}
		}
		if i == m.topCursor {
			line = rowHover.Width(w).Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("q/esc/t back  j/k select") + "\n")
	return clipBlock(b.String(), w)
}

func formatTopRow(c statsBox, width int) string {
	svc := fmt.Sprintf("%-16s", trunc(boxService(c), 16))
	cpu := fmt.Sprintf("%6.1f%%", c.CPUPct)
	mem := fmt.Sprintf("%-16s", fmtMem(c.MemUsedBytes, c.MemLimitBytes))
	net := fmtBytes(c.NetRxBytes) + " / " + fmtBytes(c.NetTxBytes)
	return trunc("  "+svc+"  "+cpu+"  "+mem+"  "+net, width)
}
