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

const helpText = `dc-tui — control this folder's devcontainer (click or keys)

  dc-tui [workspace]     this folder (cwd, then git root)
  dc-tui --all           fleet of labeled containers
  dc-tui --help

Click a button or fleet row. Keys:
  u  dc-up          e  dc-exec         o  dc-open (host)
  a  VS Code attach s  dc-down         x  dc-down --rm
  l  docker logs    f  fleet           q  quit

Up/exec/logs leave the TUI so you see real CLI output.
Editors stay on the host. Only VS Code can attach inside.
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
	case "f":
		m.fleet = !m.fleet
		return m, m.reload()
	case "r":
		return m, m.reload()
	case "u", "e", "o", "a", "s", "x", "l":
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
			case "f":
				m.fleet = !m.fleet
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
		cmd = exec.Command("dc-open", ws)
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
		b.WriteString(titleStyle.Render("dc-tui --all") + "  " + mutedStyle.Render("click a row · f workspace · q quit") + "\n\n")
	} else {
		b.WriteString(titleStyle.Render("dc-tui") + "  " + headerStyle.Render(m.workspace) + "\n")
		cfg := badStyle.Render("NO")
		if m.hasConfig {
			cfg = okStyle.Render("YES")
		}
		status, id, ports, folder := "(none)", "", "", m.workspace
		if len(m.rows) > 0 {
			status, id, ports, folder = m.rows[0].Status, shortID(m.rows[0].ID), m.rows[0].Ports, m.rows[0].LocalFolder
		}
		fmt.Fprintf(&b, "  config   .devcontainer   %s\n", cfg)
		fmt.Fprintf(&b, "  label    %s  %s  %s  %s\n", folder, status, id, ports)
		fmt.Fprintf(&b, "  editor   %s\n\n", m.editor)
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

	rowY0 := -1
	if m.fleet {
		b.WriteString("\n")
		rowY0 = strings.Count(b.String(), "\n")
		if len(m.rows) == 0 {
			b.WriteString(mutedStyle.Render("  (no labeled containers)") + "\n")
		} else {
			for _, r := range m.rows {
				fmt.Fprintf(&b, "  %-8s  %-24s  %s  %s\n", r.Status, r.Name, r.LocalFolder, shortID(r.ID))
			}
		}
	} else if len(m.rows) > 1 {
		b.WriteString("\n" + mutedStyle.Render("  extra matches:") + "\n")
		for _, r := range m.rows[1:] {
			fmt.Fprintf(&b, "  %-8s  %s  %s\n", r.Status, r.LocalFolder, shortID(r.ID))
		}
	}

	b.WriteString("\n" + hintStyle.Render("click buttons · keys u/e/o/a/s/x/l/f/q · r reload") + "\n")
	return b.String(), buttons, rowY0
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	s, _, _ := m.layout()
	return s
}

func renderButtons(fleet bool) (string, []button) {
	type spec struct {
		key, label string
		danger     bool
	}
	var specs []spec
	if fleet {
		specs = []spec{{"f", "workspace", false}, {"r", "reload", false}, {"q", "quit", false}}
	} else {
		specs = []spec{
			{"u", "up", false},
			{"e", "exec", false},
			{"o", "open", false},
			{"a", "attach", false},
			{"s", "stop", false},
			{"x", "rm", true},
			{"l", "logs", false},
			{"f", "fleet", false},
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

func pickEditor() string {
	if e := os.Getenv("DC_EDITOR"); e != "" {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
		return e + " (missing)"
	}
	for _, e := range []string{"zed", "code", "subl"} {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	return "(none)"
}
