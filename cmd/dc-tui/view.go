package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	badStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	btnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Background(lipgloss.Color("109")).Bold(true).Padding(0, 1).Align(lipgloss.Center)
	btnHover     = lipgloss.NewStyle().Foreground(lipgloss.Color("234")).Background(lipgloss.Color("159")).Bold(true).Padding(0, 1).Align(lipgloss.Center)
	btnMeta      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1).Align(lipgloss.Center)
	btnMetaHover = lipgloss.NewStyle().Foreground(lipgloss.Color("234")).Background(lipgloss.Color("246")).Padding(0, 1).Align(lipgloss.Center)
	btnDisabled  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(lipgloss.Color("236")).Padding(0, 1).Align(lipgloss.Center)
	btnDanger    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("167")).Padding(0, 1).Align(lipgloss.Center)
	btnDangerH   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("203")).Bold(true).Padding(0, 1).Align(lipgloss.Center)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Width(10)
	rowHover     = lipgloss.NewStyle().Background(lipgloss.Color("237"))
)

func kv(k, v string) string {
	return labelStyle.Render(k) + " " + v
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.splashOn {
		return m.splashView()
	}
	if m.logOpen {
		return m.logsView()
	}
	if m.topOpen {
		return m.topView()
	}
	if m.netOpen {
		return m.netView()
	}
	s, _, _ := m.layout()
	return s
}

func (m model) layout() (string, []button, int) {
	var b strings.Builder
	w := m.width
	if w <= 0 {
		w = 80
	}
	infoW := max(12, w-12)
	if m.fleet {
		info := logoWord.Render("dc-cli") + mutedStyle.Render("  "+cliVersion()) + mutedStyle.Render("  fleet") + "  " + mutedStyle.Render("j/k · enter")
		b.WriteString(clipBlock(joinLogo(m.headerLogo(), info, w), w) + "\n\n")
	} else {
		base := filepath.Base(m.workspace)
		var info strings.Builder
		info.WriteString(logoWord.Render("dc-cli") + mutedStyle.Render("  "+cliVersion()) + "  " + headerStyle.Render(trunc(base, infoW)) + "\n")
		info.WriteString(mutedStyle.Render(trunc(m.workspace, infoW)) + "\n")
		cfg := badStyle.Render("no workspace")
		if m.hasConfig {
			cfg = okStyle.Render("ready")
		} else if m.hasCompose {
			cfg = okStyle.Render("compose")
		}
		st, id, ports := warnStyle.Render("stopped"), "", ""
		if len(m.rows) > 0 {
			id, ports = shortID(m.rows[0].ID), m.rows[0].Ports
			switch m.rows[0].Status {
			case "running":
				st = okStyle.Render("running")
			case "exited":
				st = badStyle.Render("exited")
			default:
				st = warnStyle.Render(m.rows[0].Status)
			}
		}
		meta := st + mutedStyle.Render("  ") + cfg
		if id != "" {
			meta += mutedStyle.Render("  ") + mutedStyle.Render(id)
		}
		if ports != "" {
			meta += mutedStyle.Render("  ") + ports
		}
		info.WriteString(trunc(meta, infoW) + "\n")
		info.WriteString(kv("editor", trunc(m.editor, max(8, infoW-12))) + "\n")
		if m.pulse != "" {
			info.WriteString(kv("load", trunc(m.pulse+"  t=top", max(8, infoW-12))) + "\n")
		}
		if m.disk != "" {
			info.WriteString(kv("disk", trunc(m.disk+"  d=df", max(8, infoW-12))))
		}
		if line := netHeaderLine(m.net); line != "" {
			if m.disk != "" {
				info.WriteString("\n")
			}
			info.WriteString(kv("nets", trunc(line+"  n=nets", max(8, infoW-12))))
		}
		b.WriteString(clipBlock(joinLogo(m.headerLogo(), strings.TrimRight(info.String(), "\n"), w), w) + "\n\n")
	}

	y0 := strings.Count(b.String(), "\n")
	line, buttons := renderGroups(m.buttonGroups(), w, y0, m.hover)
	b.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		b.WriteString("\n")
	}

	if m.leaving != "" {
		b.WriteString("\n" + warnStyle.Render(trunc(leaveLine(m.leaving), w)) + "\n")
	}
	if m.confirm == "rm" {
		b.WriteString("\n" + warnStyle.Render("remove stack containers? y/n") + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(trunc(m.status, w)) + "\n")
	}
	if m.err != "" {
		b.WriteString("\n" + errStyle.Render(trunc(m.err, w)) + "\n")
	}
	if m.more {
		b.WriteString("\n" + morePanel(m.editor, w) + "\n")
	}

	rowY0 := -1
	if m.fleet {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("  status    workspace") + "\n")
		rowY0 = strings.Count(b.String(), "\n")
		if len(m.rows) == 0 {
			b.WriteString(mutedStyle.Render("  (empty — dc-up in a project)") + "\n")
		} else {
			for i, r := range m.rows {
				line := formatFleetRow(r, w)
				if i == m.cursor {
					line = rowHover.Width(w).Render(line)
				}
				b.WriteString(line + "\n")
			}
		}
	} else if len(m.stack) > 0 {
		b.WriteString("\n" + mutedStyle.Render("  stack") + hintStyle.Render("   j/k · enter · e is app") + "\n")
		rowY0 = strings.Count(b.String(), "\n")
		for i, s := range m.stack {
			line := trunc(formatStackRow(s, w), w)
			if i == m.cursor || i == m.hoverStack {
				line = rowHover.Width(w).Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	if m.confirm == "rm" {
		b.WriteString("\n" + hintStyle.Render("y confirm  n/esc cancel  q quit") + "\n")
	} else if !m.fleet && len(m.webLinks()) > 0 {
		b.WriteString("\n" + hintStyle.Render("u start  e shell  s stop  b db  m files  n nets  1-9 url  j/k  enter  ? more  q quit") + "\n")
	} else {
		b.WriteString("\n" + hintStyle.Render("u start  e shell  s stop  b db  m files  n nets  j/k  enter  ? more  q quit") + "\n")
	}
	return clipBlock(b.String(), w), buttons, rowY0
}

func leaveLine(kind string) string {
	switch kind {
	case "logs":
		return "opening logs"
	case "start":
		return "leaving to start — board returns when it finishes"
	case "files":
		return "leaving to files — quit the manager to return"
	default:
		return "leaving to shell — exit to return"
	}
}

func formatStackRow(s stackSvc, width int) string {
	svc := s.Service
	if svc == "" {
		svc = "-"
	}
	st := mutedStyle.Render(fmt.Sprintf("%-8s", s.Status))
	switch s.Status {
	case "running":
		st = okStyle.Render(fmt.Sprintf("%-8s", "up"))
	case "exited":
		st = badStyle.Render(fmt.Sprintf("%-8s", "down"))
	}
	name := trunc(s.Name, max(8, width-30))
	return "  " + st + "  " + fmt.Sprintf("%-16s", trunc(svc, 16)) + "  " + mutedStyle.Render(name)
}

func formatFleetRow(r container, width int) string {
	st := mutedStyle.Render(fmt.Sprintf("%-8s", r.Status))
	if r.Status == "running" {
		st = okStyle.Render(fmt.Sprintf("%-8s", "up"))
	}
	folder := r.LocalFolder
	if folder == "" {
		folder = r.Name
	}
	return "  " + st + "  " + trunc(folder, max(8, width-12))
}

func morePanel(editor string, width int) string {
	lines := []string{
		"more — what each action does",
		"  start    create/start this folder (.devcontainer → official CLI; compose-kind → docker compose)",
		"  shell    bash in the labeled app — color prompt, ls, hl for logs",
		"  stack    j/k + enter or click — starts the box if down, then exec",
		"  open     host editor on the bind-mount  now: " + editor,
		"  attach   VS Code Remote URI; Zed: Project → Open Remote → Connect Dev Container",
		"  ports    sidecar publish compose/forwardPorts",
		"  url      click or 1-9 — open a published website in the browser",
		"  stop     full compose stack — not app-only",
		"  rm       compose down (remove stack containers) — asks y/n",
		"  logs     follow docker logs for the selected stack row — highlighted, q returns",
		"  R        restart selected stack sibling (not the labeled app). r still reloads",
		"  top      CPU / RAM for this folder (t). Disk stays d / dc-df",
		"  nets     this folder's declared compose nets (n). y creates missing externals then start",
		"  db       open TablePlus (etc.) on a declared db port (b)",
		"  files    yazi/nnn in the box, else Linux yazi copied into /tmp (m)",
		"  fleet    list every labeled workspace",
		"  disk     dc-df report (d key). Reclaim: dc-prune --yes (CLI)",
		"",
		"open ≠ attach. Zed attaches itself. Sublime cannot.",
	}
	if width > 0 {
		for i, line := range lines {
			lines[i] = trunc(line, width)
		}
	}
	lines[0] = titleStyle.Render(lines[0])
	lines[len(lines)-1] = mutedStyle.Render(lines[len(lines)-1])
	return strings.Join(lines, "\n")
}

type btnSpec struct {
	key, label string
	danger     bool
	primary    bool
	disabled   bool
}

func (m model) buttonGroups() [][]btnSpec {
	if m.fleet {
		return [][]btnSpec{{
			{key: "f", label: "folder"},
			{key: "r", label: "reload"},
			{key: "?", label: "more"},
			{key: "q", label: "quit"},
		}}
	}
	groups := [][]btnSpec{
		{
			{key: "u", label: "start", primary: true, disabled: !m.canStart()},
			{key: "e", label: "shell", primary: true},
			{key: "s", label: "stop", primary: true},
		},
		{
			{key: "o", label: "open"},
			{key: "a", label: "attach"},
			{key: "p", label: "ports"},
			{key: "l", label: "logs", disabled: !m.canFollowLogs()},
			{key: "t", label: "top", disabled: len(m.rows) == 0 || m.rows[0].ID == ""},
			{key: "n", label: "nets"},
		},
		{
			{key: "b", label: "db"},
			{key: "m", label: "files"},
		},
		{
			{key: "f", label: "fleet"},
			{key: "?", label: "more"},
			{key: "q", label: "quit"},
			{key: "x", label: "rm", danger: true},
		},
	}
	if links := m.webLinks(); len(links) > 0 {
		specs := make([]btnSpec, len(links))
		for i, l := range links {
			label := l.Label
			if i < 9 {
				label = strconv.Itoa(i+1) + " " + l.Label
			}
			specs[i] = btnSpec{key: "url:" + l.URL, label: label, primary: true}
		}
		groups = append(groups, specs)
	}
	return groups
}

func renderGroups(groups [][]btnSpec, width, y0 int, hover string) (string, []button) {
	if width <= 0 {
		width = 80
	}
	var lines []string
	var buttons []button
	y := y0
	for gi, specs := range groups {
		if gi > 0 {
			lines = append(lines, "")
			y++
		}
		chunk, btns, nextY := renderRow(specs, width, y, hover)
		lines = append(lines, chunk)
		buttons = append(buttons, btns...)
		y = nextY
	}
	return strings.Join(lines, "\n") + "\n", buttons
}

func renderRow(specs []btnSpec, width, y0 int, hover string) (string, []button, int) {
	inner := 0
	for _, s := range specs {
		if n := len(s.label); n > inner {
			inner = n
		}
	}
	tileW := inner + 2
	gap := 1
	perRow := (width + gap) / (tileW + gap)
	if perRow < 1 {
		perRow = 1
	}
	var lines []string
	var buttons []button
	var row []string
	x, y, col := 0, y0, 0
	flush := func() {
		if len(row) == 0 {
			return
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, row...))
		row = nil
		x = 0
		col = 0
		y++
	}
	for _, s := range specs {
		st := tileStyle(s, hover)
		cell := st.Width(tileW).Render(s.label)
		h := lipgloss.Height(cell)
		if col >= perRow {
			flush()
		}
		if col > 0 {
			row = append(row, strings.Repeat(" ", gap))
			x += gap
		}
		buttons = append(buttons, button{
			key: s.key, label: s.label, disabled: s.disabled,
			x0: x, x1: x + tileW,
			y0: y, y1: y + h,
		})
		row = append(row, cell)
		x += tileW
		col++
	}
	flush()
	return strings.Join(lines, "\n"), buttons, y
}

func tileStyle(s btnSpec, hover string) lipgloss.Style {
	if s.disabled {
		return btnDisabled
	}
	if s.danger {
		if hover == s.key {
			return btnDangerH
		}
		return btnDanger
	}
	if s.primary {
		if hover == s.key {
			return btnHover
		}
		return btnStyle
	}
	if hover == s.key {
		return btnMetaHover
	}
	return btnMeta
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func trunc(s string, n int) string {
	if n <= 0 || ansi.StringWidth(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return ansi.Truncate(s, n, "…")
}

func clipBlock(s string, w int) string {
	if w <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = trunc(line, w)
	}
	return strings.Join(lines, "\n")
}
