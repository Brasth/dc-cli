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

type button struct {
	key   string
	label string
	x0    int
	x1    int
	y     int
}

type model struct {
	workspace string
	fleet     bool
	status    string
	hasConfig bool
	editor    string
	rows      []container
	buttons   []button
	rowY0     int
	width     int
	err       string
	quitting  bool
	more      bool
}

type reloadMsg struct {
	rows []container
	err  error
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	badStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	btnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1)
	btnDanger   = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1")).Padding(0, 1)
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
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
		workspace: ws,
		fleet:     fleet,
		hasConfig: hasDevcontainer(ws),
		editor:    pickEditor(),
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const helpText = `dc-tui — this folder's devcontainer (click buttons or keys)

  dc-tui [workspace]     this folder (cwd, then git root)
  dc-tui --all           every labeled container (fleet)
  dc-tui --help

In the TUI, click more (or press ?) for the full legend.

Buttons / keys
  start  (u)  dc-up          create/start this folder + auto-forward ports
  shell  (e)  dc-exec        bash inside the container (leaves TUI)
  open   (o)  dc-open        host editor on the bind-mount (zed/code/subl)
  attach (a)  dc-open --attach
                             VS Code Remote INTO the running container
                             (code only; needs a running box)
  ports  (p)  dc-forward     sidecar publish (Colima-safe; localhost:3000)
  stop   (s)  dc-down        stop, keep the container for next start
  rm     (x)  dc-down --rm   stop and delete the container
  logs   (l)  docker logs -f (leaves TUI; Ctrl-C back)
  fleet  (f)  all labeled workspaces; click a row to open it
  more   (?)  this legend
  quit   (q)

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
		return reloadMsg{rows: rows}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case reloadMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.rows = nil
		} else {
			m.err = ""
			m.rows = msg.rows
		}
		m.hasConfig = hasDevcontainer(m.workspace)
	case tea.KeyMsg:
		return m.handleKey(msg.String())
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		return m.handleClick(msg.X, msg.Y)
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

func (m model) handleClick(x, y int) (tea.Model, tea.Cmd) {
	_, buttons, rowY0 := m.layout()
	m.buttons = buttons
	m.rowY0 = rowY0
	for _, b := range buttons {
		if y == b.y && x >= b.x0 && x < b.x1 {
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
	if m.fleet && m.rowY0 > 0 {
		i := y - m.rowY0
		if i >= 0 && i < len(m.rows) {
			return m.openRow(i)
		}
	}
	return m, nil
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
	return m, nil
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
		cmd = exec.Command("dc-open", "--attach", ws)
	case "s":
		cmd = exec.Command("dc-down", ws)
	case "x":
		cmd = exec.Command("dc-down", "--rm", ws)
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
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return reloadMsg{err: err}
		}
		return m.reload()()
	})
}

func (m model) layout() (string, []button, int) {
	var b strings.Builder
	if m.fleet {
		b.WriteString(titleStyle.Render("dc-tui  fleet") + "  " + mutedStyle.Render("every labeled container — click a row") + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("dc-tui") + "  " + headerStyle.Render(m.workspace) + "\n")
		cfg := badStyle.Render("NO  (start disabled — add .devcontainer)")
		if m.hasConfig {
			cfg = okStyle.Render("YES")
		}
		status, id, ports, folder := "not running", "", "", m.workspace
		if len(m.rows) > 0 {
			status, id, ports, folder = m.rows[0].Status, shortID(m.rows[0].ID), m.rows[0].Ports, m.rows[0].LocalFolder
		}
		fmt.Fprintf(&b, "  this folder   %s\n", m.workspace)
		fmt.Fprintf(&b, "  .devcontainer %s\n", cfg)
		fmt.Fprintf(&b, "  container     %s  %s  %s\n", status, id, ports)
		fmt.Fprintf(&b, "  docker label  %s\n", folder)
		fmt.Fprintf(&b, "  host editor   %s  (open uses this; attach needs VS Code)\n\n", m.editor)
	}

	line, buttons := renderButtons(m.fleet)
	y := strings.Count(b.String(), "\n")
	for i := range buttons {
		buttons[i].y = y
	}
	b.WriteString(line + "\n")

	if m.err != "" {
		b.WriteString("\n" + errStyle.Render(m.err) + "\n")
	}

	if m.more {
		b.WriteString("\n" + morePanel(m.editor) + "\n")
	}

	rowY0 := -1
	if m.fleet {
		b.WriteString("\n")
		rowY0 = strings.Count(b.String(), "\n")
		if len(m.rows) == 0 {
			b.WriteString(mutedStyle.Render("  (no labeled containers — start one with dc-up in a project)") + "\n")
		} else {
			b.WriteString(mutedStyle.Render("  status    name                      folder") + "\n")
			for _, r := range m.rows {
				fmt.Fprintf(&b, "  %-8s  %-24s  %s  %s\n", r.Status, r.Name, r.LocalFolder, shortID(r.ID))
			}
		}
	} else if len(m.rows) > 1 {
		b.WriteString("\n" + mutedStyle.Render("  extra matches for this folder:") + "\n")
		for _, r := range m.rows[1:] {
			fmt.Fprintf(&b, "  %-8s  %s  %s\n", r.Status, r.LocalFolder, shortID(r.ID))
		}
	}

	b.WriteString("\n" + hintStyle.Render("click a button  ·  ? more  ·  q quit") + "\n")
	return b.String(), buttons, rowY0
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
		"  shell    bash inside the container — TUI closes while it runs",
		"  open     host editor on the bind-mount  now: " + editor,
		"  attach   VS Code Remote INTO the container (code only, must be running)",
		"  ports    sidecar publish compose/forwardPorts (Colima: not host socat)",
		"  stop     docker stop — keep the container for next start",
		"  rm       stop and delete the container",
		"  logs     follow docker logs — Ctrl-C returns here",
		"  fleet    list every labeled workspace",
		"",
		mutedStyle.Render("open ≠ attach. Zed/Sublime = open only. Docs: https://github.com/Canvilled/dc-cli"),
	}
	return strings.Join(lines, "\n")
}

func renderButtons(fleet bool) (string, []button) {
	type spec struct {
		key, label string
		danger     bool
	}
	var specs []spec
	if fleet {
		specs = []spec{
			{"f", "this folder", false},
			{"r", "reload", false},
			{"?", "more", false},
			{"q", "quit", false},
		}
	} else {
		specs = []spec{
			{"u", "start", false},
			{"e", "shell", false},
			{"o", "open host", false},
			{"a", "attach vscode", false},
			{"p", "ports", false},
			{"s", "stop", false},
			{"x", "rm", true},
			{"l", "logs", false},
			{"f", "fleet", false},
			{"?", "more", false},
			{"q", "quit", false},
		}
	}
	var parts []string
	var buttons []button
	x := 0
	for _, s := range specs {
		st := btnStyle
		if s.danger {
			st = btnDanger
		}
		cell := st.Render(s.label)
		w := lipgloss.Width(cell)
		buttons = append(buttons, button{key: s.key, label: s.label, x0: x, x1: x + w})
		parts = append(parts, cell)
		x += w + 1
	}
	return strings.Join(parts, " "), buttons
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
