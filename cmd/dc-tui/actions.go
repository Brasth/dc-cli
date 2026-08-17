package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const leaveWait = 50 * time.Millisecond

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
		fwd := mergePairs(listFwdMaps(ws), listStackPorts(stack))
		var nets netReport
		if nout, err := runNet("--json", ws); err == nil {
			if parsed, err := parseNet(nout); err == nil {
				nets = parsed
			}
		}
		return reloadMsg{rows: rows, stack: stack, fwdMaps: fwd, disk: disk, nets: nets}
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

// start (u) exit 1 is a real failure. Shells/logs still swallow it.
func benignLeaveErr(action string, err error) bool {
	if err == nil {
		return true
	}
	if action == "u" || action == "create-nets" {
		s := strings.ToLower(err.Error())
		switch {
		case strings.Contains(s, "exit status 130"),
			strings.Contains(s, "exit status 143"),
			strings.Contains(s, "interrupt"),
			strings.Contains(s, "could not restore terminal"),
			strings.Contains(s, "the input device is not a tty"),
			strings.Contains(s, "signal: hangup"):
			return true
		default:
			return false
		}
	}
	return benignExecErr(err)
}

func (m model) runAction(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "u":
		if !m.hasConfig {
			return m.refuse("No .devcontainer in " + m.workspace + ". Add one, or use CLI: dc-up --ports (REPLACE).")
		}
		return m.startLeave("start", "u")
	case "e":
		return m.startLeave("shell", "e")
	case "o":
		return m.stayCmd("dc-open", m.workspace)
	case "p":
		return m.stayCmd("dc-forward", m.workspace)
	case "a":
		return m.stayCmd("dc-open", "--attach", m.workspace)
	case "s":
		return m.stayCmd("dc-down", m.workspace)
	case "x":
		m.confirm = "rm"
		return m.withStatus(""), nil
	case "l":
		return m.openLogs()
	case "t":
		return m.openTop()
	case "n":
		return m.openNets()
	case "b":
		return m.stayCmd("dc-db", m.workspace)
	case "m":
		return m.startLeave("files", "m")
	default:
		return m, nil
	}
}

func (m model) startLeave(kind, pending string) (model, tea.Cmd) {
	if m.leaving != "" {
		return m, nil
	}
	m.leaving = kind
	m.pending = pending
	return m, tea.Tick(leaveWait, func(time.Time) tea.Msg {
		return leaveTickMsg{}
	})
}

func (m model) execStack(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.stack) {
		return m, nil
	}
	return m.startLeave("shell", "stack:"+strconv.Itoa(i))
}

// restartSelected restarts the stack-cursor sibling. Labeled app row
// (same id as dc-ls rows[0]) is refused — use u/s. Fleet is refused by the key handler.
func (m model) restartSelected() (tea.Model, tea.Cmd) {
	if len(m.stack) == 0 {
		return m.refuse("No stack row to restart.")
	}
	m.clampCursor()
	if m.cursor < 0 || m.cursor >= len(m.stack) {
		return m.refuse("No stack row to restart.")
	}
	s := m.stack[m.cursor]
	if s.ID == "" {
		return m.refuse("No stack row to restart.")
	}
	if len(m.rows) > 0 && s.ID == m.rows[0].ID {
		return m.refuse("restart is for stack siblings. Use u/s for the labeled app.")
	}
	svc := s.Service
	if svc == "" {
		svc = s.Name
	}
	if svc == "" {
		return m.refuse("stack row has no service name")
	}
	return m.stayCmd("dc-exec", "--service", svc, "--restart", m.workspace)
}

func (m model) runPending() (tea.Model, tea.Cmd) {
	pending := m.pending
	m.pending = ""
	if strings.HasPrefix(pending, "stack:") {
		i, err := strconv.Atoi(strings.TrimPrefix(pending, "stack:"))
		if err != nil || i < 0 || i >= len(m.stack) {
			m.leaving = ""
			return m.refuse("stack row gone")
		}
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
	ws := m.workspace
	var cmd *exec.Cmd
	switch pending {
	case "u":
		cmd = exec.Command("dc-up", ws)
	case "create-nets":
		cmd = exec.Command("dc-up", "--create-nets", ws)
	case "e":
		cmd = exec.Command("dc-exec", ws)
	case "m":
		cmd = exec.Command("dc-files", ws)
	default:
		m.leaving = ""
		return m, nil
	}
	cmd.Stdin = os.Stdin
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{action: pending, err: err}
	})
}

func (m model) openRow(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.rows) {
		return m, nil
	}
	folder := m.rows[i].LocalFolder
	if folder == "" {
		return m.withErr("row has no local_folder"), nil
	}
	if st, err := os.Stat(folder); err != nil || !st.IsDir() {
		return m.withErr("folder missing on disk: " + folder), nil
	}
	m.fleet = false
	m.workspace = folder
	m.hasConfig = hasDevcontainer(folder)
	m.cursor = 0
	return m.withStatus(""), m.reload()
}

// runStay is the stay-in-board exec. Tests replace it so confirm y never hits Docker.
var runStay = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

func (m model) stayCmd(name string, args ...string) (tea.Model, tea.Cmd) {
	out, err := runStay(name, args...)
	msg := compactLines(strings.TrimSpace(out), 4)
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		return m.withErr(msg), nil
	}
	return m.withStatus(msg), m.reload()
}

func compactLines(msg string, n int) string {
	if msg == "" {
		return ""
	}
	lines := strings.Split(msg, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " · ")
}
