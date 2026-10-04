package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	if m.wsOpen {
		return m.workspaceView()
	}
	if m.actOpen {
		return m.actionPickerView()
	}
	if m.hostBlock {
		return m.hostView()
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
	if m.activity.open {
		return m.activityView()
	}
	if strings.Contains(m.status, "\n") {
		return m.reportView()
	}
	s, _, _ := m.layout()
	return s
}

func (m model) layout() (string, []button, int) {
	frame := m.consoleLayout()
	return frame.view, frame.buttons, frame.rowY
}

// optionalRows are the header rows for disk / nets / ports: the value when
// there is one, plus loading / unavailable / stale when it is not fresh.
func (m model) optionalRows() [][2]string {
	var rows [][2]string
	note := sectionNote
	if m.disk != "" {
		v := m.disk + "  d=df"
		if m.diskCritical {
			v = m.disk + "  CRITICAL  P=prune"
		}
		if n := note(m.diskSec, true); n != "" {
			v += "  (" + n + ")"
		}
		rows = append(rows, [2]string{"disk", v})
	} else if n := note(m.diskSec, false); n != "" {
		rows = append(rows, [2]string{"disk", n + "  d=df"})
	}
	if line := netHeaderLine(m.net); line != "" {
		v := line + "  n=nets"
		if n := note(m.netsSec, true); n != "" {
			v += "  (" + n + ")"
		}
		rows = append(rows, [2]string{"nets", v})
	} else if n := note(m.netsSec, false); n != "" {
		rows = append(rows, [2]string{"nets", n + "  n=nets"})
	}
	if n := note(m.portsSec, len(m.fwdMaps) > 0); n != "" {
		rows = append(rows, [2]string{"ports", n + "  p=ports"})
	}
	return rows
}

func leaveLine(kind string) string {
	switch kind {
	case "logs":
		return "opening logs"
	case "start":
		return "leaving to start — board returns when it finishes"
	case "files":
		return "leaving to files — quit the manager to return"
	case "upgrade":
		return "leaving to upgrade — board exits when it finishes"
	case "action":
		return "leaving to run the action — board returns when it finishes (Ctrl+C stops it)"
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
		"  start    .devcontainer/compose → dc-up; else confirm → dc-try sandbox (no project edits)",
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
		"  files    yazi/nnn in the box; Enter opens code/cursor on this container (m)",
		"  fleet    list every labeled workspace",
		"  w        recent workspaces — type to filter, enter open, ctrl+f favorite, ctrl+d forget",
		"  c        project actions (.dc/actions.json + personal) — shared ones need review first",
		"  v        activity — this workspace's container events (latest 200, memory only). c clears",
		"  disk     dc-df report (d key). P = dc-prune --yes when disk looks critical",
		"  upgrade  U when a newer release is available → dc-upgrade --yes",
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

func (m model) workspaceStatusParts() (st, id, ports string) {
	switch {
	case m.hardLoading():
		return warnStyle.Render("checking…"), "", ""
	case m.load == loadFailed && !m.loaded:
		return badStyle.Render("unknown"), "", ""
	case len(m.rows) == 0:
		return warnStyle.Render("stopped"), "", ""
	default:
		id, ports = shortID(m.rows[0].ID), m.rows[0].Ports
		switch m.rows[0].Status {
		case "running":
			st = okStyle.Render("running")
		case "exited":
			st = badStyle.Render("exited")
		default:
			st = warnStyle.Render(m.rows[0].Status)
		}
		return st, id, ports
	}
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
	blocked := m.hardLoading() || (m.load == loadFailed && !m.loaded)
	groups := [][]btnSpec{
		{
			{key: "u", label: "start", primary: true, disabled: !m.canStart()},
			{key: "e", label: "shell", primary: true, disabled: blocked},
			{key: "s", label: "stop", primary: true, disabled: blocked},
		},
		{
			{key: "o", label: "open", disabled: blocked},
			{key: "a", label: "attach", disabled: blocked},
			{key: "p", label: "ports", disabled: blocked},
			{key: "l", label: "logs", disabled: blocked || !m.canFollowLogs()},
			{key: "t", label: "top", disabled: blocked || len(m.rows) == 0 || m.rows[0].ID == ""},
			{key: "n", label: "nets", disabled: blocked},
		},
		{
			{key: "b", label: "db", disabled: blocked},
			{key: "m", label: "files", disabled: blocked},
		},
		{
			{key: "f", label: "fleet"},
			{key: "?", label: "more"},
			{key: "q", label: "quit"},
			{key: "x", label: "rm", danger: true, disabled: blocked},
		},
	}
	if links := m.webLinks(); len(links) > 0 && !blocked {
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
