package main

import (
	"fmt"
	"strings"
)

func (m model) topView() string {
	w, h := m.consoleSize()
	meta := plainText(m.topSnap.Engine) + fmt.Sprintf(" · %d boxes", len(m.topSnap.Containers))
	if m.topStale {
		meta += " · stale"
	}
	lines := m.screenHeader("TOP · live dc-stats", meta)
	lines = append(lines, guestLine(m.topSnap.Guest), mutedStyle.Render("  SERVICE           CPU     MEM              NET"))
	page := max(1, h-len(lines)-2)
	start := max(0, m.topCursor-page+1)
	if len(m.topSnap.Containers) == 0 {
		lines = append(lines, mutedStyle.Render("(no running boxes)"))
	}
	for i := start; i < len(m.topSnap.Containers) && i < start+page; i++ {
		c := m.topSnap.Containers[i]
		c.Name = plainText(c.Name)
		c.Service = plainText(c.Service)
		line := formatTopRow(c, w)
		if hist, ok := m.topHist[c.ID]; ok {
			line = trunc(line+"  "+sparkline(hist.cpu, 12), w)
		}
		if i == m.topCursor {
			line = selectedRow("> "+strings.TrimPrefix(line, "  "), w)
		}
		lines = append(lines, line)
	}
	feedback := mutedStyle.Render("Live measurements")
	if m.topStale {
		feedback = warnStyle.Render("Stream lost — stale snapshot; reconnecting")
	}
	if m.topErr != "" {
		feedback = badStyle.Render("✗ " + plainText(m.topErr))
	}
	return m.finishScreen(lines, feedback, []string{keyHint("j/k", "Select") + "  " + keyHint("q/Esc/t", "Back")})
}

func formatTopRow(c statsBox, width int) string {
	svc := fmt.Sprintf("%-16s", trunc(boxService(c), 16))
	cpu := fmt.Sprintf("%6.1f%%", c.CPUPct)
	mem := fmt.Sprintf("%-16s", fmtMem(c.MemUsedBytes, c.MemLimitBytes))
	net := fmtBytes(c.NetRxBytes) + " / " + fmtBytes(c.NetTxBytes)
	return trunc("  "+svc+"  "+cpu+"  "+mem+"  "+net, width)
}
