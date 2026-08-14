package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type container struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	LocalFolder string `json:"local_folder"`
	Compose     string `json:"compose"`
	Ports       string `json:"ports"`
}

type stackSvc struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Service string `json:"service"`
	Image   string `json:"image"`
}

type button struct {
	key   string
	label string
	x0    int
	x1    int
	y0    int
	y1    int
}

type model struct {
	workspace string
	fleet     bool
	status    string
	hasConfig bool
	editor    string
	rows      []container
	stack     []stackSvc
	buttons   []button
	rowY0     int
	width     int
	err       string
	quitting  bool
	more      bool
	hover      string
	hoverStack int // -1 = none
	disk      string // compact line from dc-df --json
}

type reloadMsg struct {
	rows  []container
	stack []stackSvc
	disk  string
	err   error
}

type execDoneMsg struct {
	action string
	err    error
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	badStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	btnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Background(lipgloss.Color("109")).Padding(0, 1).Align(lipgloss.Center)
	btnHover    = lipgloss.NewStyle().Foreground(lipgloss.Color("234")).Background(lipgloss.Color("159")).Bold(true).Padding(0, 1).Align(lipgloss.Center)
	btnDanger   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("167")).Padding(0, 1).Align(lipgloss.Center)
	btnDangerH  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("203")).Bold(true).Padding(0, 1).Align(lipgloss.Center)
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Width(10)
	rowHover    = lipgloss.NewStyle().Background(lipgloss.Color("237"))
)

func main() {
	workspace := "."
	fleet := false
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "-h", "--help":
			fmt.Print(helpText)
			os.Exit(0)
		case "--all":
			fleet = true
		case "-":
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", a)
			os.Exit(2)
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", a)
				os.Exit(2)
			}
			workspace = a
		}
	}
	ws, err := resolveWorkspace(workspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	m := model{
		workspace:  ws,
		fleet:      fleet,
		hasConfig:  hasDevcontainer(ws),
		editor:     pickEditor(),
		hoverStack: -1,
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_ = exec.Command("stty", "sane").Run()
		if !benignExecErr(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

const helpText = `dc-tui — this folder's devcontainer (click buttons or keys)

  dc-tui [workspace]     this folder (cwd, then git root)
  dc-tui --all           every labeled container (fleet)
  dc-tui --help

In the TUI, click more (or press ?) for the full legend.

Buttons / keys
  start  (u)  dc-up          create/start this folder + auto-forward ports
  shell  (e)  dc-exec        bash in the labeled app (leaves TUI)
                             click a stack row: start if down, then exec
  open   (o)  dc-open        host editor on the bind-mount (zed/code/subl)
  attach (a)  dc-open --attach
                             VS Code Remote INTO the running container
                             (code only; needs a running box)
  ports  (p)  dc-forward     sidecar publish (Colima-safe; localhost:3000)
  stop   (s)  dc-down        stop, keep the container for next start
  rm     (x)  dc-down --rm   stop and delete the container
  logs   (l)  docker logs -f (leaves TUI; Ctrl-C back)
  disk   (d)  dc-df report (stays in TUI)
  fleet  (f)  all labeled workspaces; click a row to open it
  more   (?)  this legend
  quit   (q)

Disk reclaim is CLI-only (needs --yes):
  dc-prune --yes
  dc-prune --all --yes
  dc-prune --volume NAME --yes

Open vs attach: open = edit files on the Mac/Linux host.
Attach = VS Code's terminal/debugger run inside Linux.
Zed and Sublime cannot attach.

start is refused if this folder has no .devcontainer.
Use CLI dc-up --ports only if you accept REPLACE of project config.
`

func (m model) Init() tea.Cmd {
	return m.reload()
}

func (m model) reload() tea.Cmd {
	ws := m.workspace
	fleet := m.fleet
	return func() tea.Msg {
		args := []string{"--json"}
		if fleet {
			args = append(args, "--all")
		} else {
			args = append(args, "--workspace", ws)
		}
		out, err := exec.Command("dc-ls", args...).Output()
		if err != nil {
			return reloadMsg{err: err}
		}
		var rows []container
		if err := json.Unmarshal(out, &rows); err != nil {
			return reloadMsg{err: err}
		}
		var stack []stackSvc
		if !fleet {
			if sout, err := exec.Command("dc-exec", "--list", "--json", ws).Output(); err == nil {
				_ = json.Unmarshal(sout, &stack)
			}
		}
		disk := ""
		if dout, err := exec.Command("dc-df", "--json").Output(); err == nil {
			var df struct {
				Compact string `json:"compact"`
			}
			if json.Unmarshal(dout, &df) == nil {
				disk = strings.TrimSpace(df.Compact)
			}
		}
		return reloadMsg{rows: rows, stack: stack, disk: disk}
	}
}

func benignExecErr(err error) bool {
	if err == nil {
		return true
	}
	s := strings.ToLower(err.Error())
	// Interactive shells often exit 1 / 130 / 143; Bubble Tea also reports
	// "could not restore terminal" when stty races after exec.
	switch {
	case strings.Contains(s, "exit status 1"),
		strings.Contains(s, "exit status 130"),
		strings.Contains(s, "exit status 143"),
		strings.Contains(s, "interrupt"),
		strings.Contains(s, "could not restore terminal"),
		strings.Contains(s, "the input device is not a tty"),
		strings.Contains(s, "signal: hangup"):
		return true
	}
	return false
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case reloadMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.rows = nil
			m.stack = nil
		} else {
			m.err = ""
			m.rows = msg.rows
			m.stack = msg.stack
			if msg.disk != "" {
				m.disk = msg.disk
			}
		}
		m.hasConfig = hasDevcontainer(m.workspace)
	case execDoneMsg:
		if msg.err != nil && !benignExecErr(msg.err) {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, m.reload()
	case tea.KeyMsg:
		return m.handleKey(msg.String())
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionMotion:
			m.hover = m.hitButton(msg.X, msg.Y)
			m.hoverStack = m.hitStack(msg.X, msg.Y)
			return m, nil
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				return m.handleClick(msg.X, msg.Y)
			}
		}
	}
	return m, nil
}

func (m model) handleKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "?", "h":
		m.more = !m.more
		return m, nil
	case "f":
		m.fleet = !m.fleet
		m.more = false
		return m, m.reload()
	case "r":
		return m, m.reload()
	case "d":
		// disk report stays in TUI (stayCmd)
		return m.stayCmd("dc-df")
	case "u", "e", "o", "a", "s", "x", "l", "p":
		if m.fleet {
			return m, nil
		}
		return m.runAction(k)
	case "enter":
		if m.fleet && len(m.rows) > 0 {
			return m.openRow(0)
		}
	}
	return m, nil
}

func (m model) hitButton(x, y int) string {
	_, buttons, _ := m.layout()
	for _, b := range buttons {
		if y >= b.y0 && y < b.y1 && x >= b.x0 && x < b.x1 {
			return b.key
		}
	}
	return ""
}

func (m model) hitStack(x, y int) int {
	if m.fleet {
		return -1
	}
	_, _, rowY0 := m.layout()
	if rowY0 <= 0 {
		return -1
	}
	i := y - rowY0
	if i >= 0 && i < len(m.stack) {
		return i
	}
	return -1
}

func (m model) handleClick(x, y int) (tea.Model, tea.Cmd) {
	_, buttons, rowY0 := m.layout()
	m.buttons = buttons
	m.rowY0 = rowY0
	for _, b := range buttons {
		if y >= b.y0 && y < b.y1 && x >= b.x0 && x < b.x1 {
			switch b.key {
			case "q":
				m.quitting = true
				return m, tea.Quit
			case "?":
				m.more = !m.more
				return m, nil
			case "f":
				m.fleet = !m.fleet
				m.more = false
				return m, m.reload()
			case "r":
				return m, m.reload()
			default:
				if !m.fleet {
					return m.runAction(b.key)
				}
			}
		}
	}
	if m.rowY0 > 0 {
		i := y - m.rowY0
		if m.fleet {
			if i >= 0 && i < len(m.rows) {
				return m.openRow(i)
			}
		} else if i >= 0 && i < len(m.stack) {
			return m.execStack(i)
		}
	}
	return m, nil
}

func (m model) execStack(i int) (tea.Model, tea.Cmd) {
	s := m.stack[i]
	label := s.Service
	if label == "" {
		label = s.Name
	}
	cmd := exec.Command("dc-exec", "--id", s.ID)
	cmd.Stdin = os.Stdin
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{action: "exec-" + label, err: err}
	})
}

func (m model) openRow(i int) (tea.Model, tea.Cmd) {
	folder := m.rows[i].LocalFolder
	if folder == "" {
		m.err = "row has no local_folder"
		return m, nil
	}
	if st, err := os.Stat(folder); err != nil || !st.IsDir() {
		m.err = "folder missing on disk: " + folder
		return m, nil
	}
	m.fleet = false
	m.workspace = folder
	m.hasConfig = hasDevcontainer(folder)
	return m, m.reload()
}

func (m model) stayCmd(name string, args ...string) (tea.Model, tea.Cmd) {
	c := exec.Command(name, args...)
	out, err := c.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		m.err = msg
		return m, nil
	}
	if msg != "" {
		// keep last few lines so ports status fits
		lines := strings.Split(msg, "\n")
		if len(lines) > 6 {
			lines = lines[len(lines)-6:]
		}
		m.err = strings.Join(lines, " · ")
	} else {
		m.err = ""
	}
	return m, m.reload()
}

func (m model) runAction(key string) (tea.Model, tea.Cmd) {
	ws := m.workspace
	var cmd *exec.Cmd
	switch key {
	case "u":
		if !hasDevcontainer(ws) {
			m.err = "No .devcontainer in " + ws + ". Add one, or use CLI: dc-up --ports (REPLACE)."
			return m, nil
		}
		cmd = exec.Command("dc-up", ws)
	case "e":
		cmd = exec.Command("dc-exec", ws)
	case "o":
		// Spawn; do not tear down the TUI (ExecProcess looks like a crash).
		return m.stayCmd("dc-open", ws)
	case "p":
		return m.stayCmd("dc-forward", ws)
	case "a":
		return m.stayCmd("dc-open", "--attach", ws)
	case "s":
		return m.stayCmd("dc-down", ws)
	case "x":
		return m.stayCmd("dc-down", "--rm", ws)
	case "l":
		if len(m.rows) == 0 || m.rows[0].ID == "" {
			m.err = "No container to log."
			return m, nil
		}
		cmd = exec.Command("docker", "logs", "-f", m.rows[0].ID)
	default:
		return m, nil
	}
	cmd.Stdin = os.Stdin
	action := key
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{action: action, err: err}
	})
}

func kv(k, v string) string {
	return labelStyle.Render(k) + " " + v
}

func (m model) layout() (string, []button, int) {
	var b strings.Builder
	w := m.width
	if w <= 0 {
		w = 80
	}
	if m.fleet {
		b.WriteString(titleStyle.Render("dc-tui") + mutedStyle.Render("  fleet") + "  " + mutedStyle.Render("click a workspace") + "\n\n")
	} else {
		base := filepath.Base(m.workspace)
		b.WriteString(titleStyle.Render("dc-tui") + "  " + headerStyle.Render(base) + "\n")
		b.WriteString(mutedStyle.Render(m.workspace) + "\n")
		cfg := badStyle.Render("no .devcontainer")
		if m.hasConfig {
			cfg = okStyle.Render("ready")
		}
		status, id, ports := warnStyle.Render("stopped"), "", ""
		if len(m.rows) > 0 {
			id, ports = shortID(m.rows[0].ID), m.rows[0].Ports
			switch m.rows[0].Status {
			case "running":
				status = okStyle.Render("running")
			case "exited":
				status = badStyle.Render("exited")
			default:
				status = warnStyle.Render(m.rows[0].Status)
			}
		}
		meta := status + mutedStyle.Render("  ") + cfg
		if id != "" {
			meta += mutedStyle.Render("  ") + mutedStyle.Render(id)
		}
		if ports != "" {
			meta += mutedStyle.Render("  ") + ports
		}
		b.WriteString(meta + "\n")
		b.WriteString(kv("editor", m.editor+"  open=host  attach=vscode") + "\n")
		if m.disk != "" {
			b.WriteString(kv("disk", m.disk+"  d=df") + "\n")
		}
		b.WriteString("\n")
	}

	y0 := strings.Count(b.String(), "\n")
	line, buttons := renderButtons(m.fleet, w, m.hover, y0)
	b.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString("\n" + errStyle.Render(m.err) + "\n")
	}

	if m.more {
		b.WriteString("\n" + morePanel(m.editor) + "\n")
	}

	rowY0 := -1
	if m.fleet {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("  status    workspace") + "\n")
		rowY0 = strings.Count(b.String(), "\n")
		if len(m.rows) == 0 {
			b.WriteString(mutedStyle.Render("  (empty — dc-up in a project)") + "\n")
		} else {
			for _, r := range m.rows {
				b.WriteString(formatFleetRow(r) + "\n")
			}
		}
	} else if len(m.stack) > 0 {
		b.WriteString("\n" + mutedStyle.Render("  stack") + hintStyle.Render("   click a row · e is app") + "\n")
		rowY0 = strings.Count(b.String(), "\n")
		for i, s := range m.stack {
			line := formatStackRow(s)
			if i == m.hoverStack {
				line = rowHover.Width(w).Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n" + hintStyle.Render("u start  e shell  p ports  d disk  ? more  q quit") + "\n")
	return b.String(), buttons, rowY0
}

func formatStackRow(s stackSvc) string {
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
	return "  " + st + "  " + fmt.Sprintf("%-16s", svc) + "  " + mutedStyle.Render(s.Name)
}

func formatFleetRow(r container) string {
	st := mutedStyle.Render(fmt.Sprintf("%-8s", r.Status))
	if r.Status == "running" {
		st = okStyle.Render(fmt.Sprintf("%-8s", "up"))
	}
	folder := r.LocalFolder
	if folder == "" {
		folder = r.Name
	}
	return "  " + st + "  " + folder
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	s, _, _ := m.layout()
	return s
}

func morePanel(editor string) string {
	lines := []string{
		titleStyle.Render("more — what each action does"),
		"  start    create/start this folder (needs .devcontainer) then dc-forward",
		"  shell    bash in the labeled app — TUI closes while it runs",
		"  stack    click a row — starts the box if it is down, then exec",
		"  open     host editor on the bind-mount  now: " + editor,
		"  attach   VS Code Remote INTO the container (code only, must be running)",
		"  ports    sidecar publish compose/forwardPorts (Colima: not host socat)",
		"  stop     docker stop — keep the container for next start",
		"  rm       stop and delete the container",
		"  logs     follow docker logs — Ctrl-C returns here",
		"  fleet    list every labeled workspace",
		"  disk     dc-df report (d key). Reclaim: dc-prune --yes (CLI)",
		"",
		mutedStyle.Render("open ≠ attach. Zed/Sublime = open only. Docs: https://github.com/Canvilled/dc-cli"),
	}
	return strings.Join(lines, "\n")
}

type btnSpec struct {
	key, label string
	danger     bool
}

func buttonSpecs(fleet bool) []btnSpec {
	if fleet {
		return []btnSpec{
			{"f", "folder", false},
			{"r", "reload", false},
			{"?", "more", false},
			{"q", "quit", false},
		}
	}
	return []btnSpec{
		{"u", "start", false},
		{"e", "shell", false},
		{"o", "open", false},
		{"a", "attach", false},
		{"p", "ports", false},
		{"s", "stop", false},
		{"x", "rm", true},
		{"l", "logs", false},
		{"f", "fleet", false},
		{"?", "more", false},
		{"q", "quit", false},
	}
}

func renderButtons(fleet bool, width int, hover string, y0 int) (string, []button) {
	specs := buttonSpecs(fleet)
	if width <= 0 {
		width = 80
	}
	// Equal tiles. Longest label + horizontal pad (style Padding 0,1).
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
	// Workspace: 6 primary + 5 meta. Prefer 6-wide so rows stay even.
	if !fleet && perRow > 6 {
		perRow = 6
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
		st := btnStyle.Width(tileW)
		if s.danger {
			st = btnDanger.Width(tileW)
		}
		if hover == s.key {
			if s.danger {
				st = btnDangerH.Width(tileW)
			} else {
				st = btnHover.Width(tileW)
			}
		}
		cell := st.Render(s.label)
		h := lipgloss.Height(cell)
		if col >= perRow {
			flush()
		}
		if col > 0 {
			row = append(row, strings.Repeat(" ", gap))
			x += gap
		}
		buttons = append(buttons, button{
			key: s.key, label: s.label,
			x0: x, x1: x + tileW,
			y0: y, y1: y + h,
		})
		row = append(row, cell)
		x += tileW
		col++
	}
	flush()
	return strings.Join(lines, "\n") + "\n", buttons
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func hasDevcontainer(dir string) bool {
	for _, p := range []string{
		filepath.Join(dir, ".devcontainer", "devcontainer.json"),
		filepath.Join(dir, ".devcontainer.json"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func resolveWorkspace(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	if hasDevcontainer(abs) {
		return abs, nil
	}
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return abs, nil
	}
	root := strings.TrimSpace(string(out))
	if root != "" && root != abs {
		return root, nil
	}
	return abs, nil
}

func editorBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	var cands []string
	switch name {
	case "zed":
		cands = []string{
			"/Applications/Zed.app/Contents/MacOS/cli",
			filepath.Join(home, "Applications/Zed.app/Contents/MacOS/cli"),
			"/Applications/Zed.app/Contents/MacOS/zed",
		}
	case "code":
		cands = []string{
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			filepath.Join(home, "Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"),
		}
	case "subl":
		cands = []string{
			"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl",
			filepath.Join(home, "Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"),
		}
	}
	for _, p := range cands {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func pickEditor() string {
	if e := os.Getenv("DC_EDITOR"); e != "" {
		if editorBin(e) != "" {
			return e
		}
		return e + " (missing)"
	}
	for _, e := range []string{"zed", "code", "subl"} {
		if editorBin(e) != "" {
			return e
		}
	}
	return "(none — install Zed / VS Code / Sublime)"
}
