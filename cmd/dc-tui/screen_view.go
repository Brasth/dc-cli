package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m model) screenHeader(title, meta string) []string {
	w, _ := m.consoleSize()
	name := plainText(filepath.Base(m.workspace))
	if m.fleet {
		name = "fleet"
	}
	lines := []string{splitHeader(logoWord.Render("dc-cli")+mutedStyle.Render(" "+cliVersion())+"  "+titleStyle.Render(name), secondaryStyle.Render(plainText(meta)), w),
		splitHeader(mutedStyle.Render(plainText(m.workspace)), mutedStyle.Render("engine "+plainText(strings.TrimPrefix(engineLabel(m.engine), "context:"))), w), ""}
	return append(lines, paneTitle(title, w)...)
}

func (m model) hostView() string {
	w, _ := m.consoleSize()
	if m.engine == "" {
		m.engine = m.host.EngineHint
	}
	lines := m.screenHeader("RECOVER", "Docker unavailable")
	lines = append(lines, badStyle.Render(plainText(m.host.Summary)))
	next := m.host.NextCommand
	if next == "" {
		next = m.host.Remediation
	}
	if next != "" {
		lines = append(lines, okStyle.Render(trunc("Run: "+plainText(next), w)))
	}
	if m.host.NextID != "" {
		lines = append(lines, kv("next", plainText(m.host.NextID)))
	}
	if m.host.Code != "" {
		lines = append(lines, kv("problem", plainText(m.host.Code)))
	}
	if m.host.Detail != nil {
		lines = append(lines, wrapLines(plainText(*m.host.Detail), w, "", "")...)
	}
	if !m.host.canApply() && (m.host.Code == "docker_cli_missing" || m.host.Code == "docker_engine_missing") {
		guide := m.host.GuideURL
		if guide == "" {
			guide = "https://docs.docker.com/desktop/"
		}
		lines = append(lines, okStyle.Render("Empty machine — install an engine, then retry"), mutedStyle.Render(guide), mutedStyle.Render("Lightweight: brew install docker colima && colima start"))
	}
	hints := []string{keyHint("r", "Check again") + "  " + keyHint("d", "Desktop guide") + "  " + keyHint("c", "Copy Colima setup"), keyHint("w", "Workspaces") + "  " + keyHint("q", "Quit")}
	if m.host.canApply() {
		hints[1] = keyHint("f", "Apply next step") + "  " + hints[1]
	}
	return m.finishScreen(lines, m.boardFeedback(), hints)
}

func (m model) reportView() string {
	w, h := m.consoleSize()
	lines := m.screenHeader("OUTPUT · PgUp/PgDn", "dc-cli command output")
	body := strings.Split(m.status, "\n")
	page := max(1, h-len(lines)-2)
	start := min(m.outputOff, max(0, len(body)-page))
	for _, line := range body[start:min(len(body), start+page)] {
		lines = append(lines, secondaryStyle.Render(trunc(plainText(line), w)))
	}
	return m.finishScreen(lines, mutedStyle.Render(fmt.Sprintf("Lines %d–%d of %d", start+1, min(len(body), start+page), len(body))), []string{keyHint("j/k", "Scroll") + "  " + keyHint("PgUp/PgDn", "Page") + "  " + keyHint("q/Esc", "Back")})
}

func logDisplayText(s string) string {
	// Expand tabs at eight-cell stops before measuring or highlighting.
	s = ansi.Strip(s)
	var b strings.Builder
	parts := strings.Split(s, "\t")
	for i, part := range parts {
		b.WriteString(plainText(part))
		if i < len(parts)-1 {
			b.WriteString(strings.Repeat(" ", 8-ansi.StringWidth(b.String())%8))
		}
	}
	return b.String()
}
