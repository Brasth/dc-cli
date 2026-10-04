package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const leaveWait = 50 * time.Millisecond

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
		if m.needsTry() {
			m.confirm = "try"
			return m.withStatus(""), nil
		}
		return m.startLeave("start", "u")
	case "e":
		return m.startLeave("shell", "e")
	case "o":
		return m.stayCmd("dc-open", m.workspace)
	case "p":
		return m.stayCmd("dc-forward", m.workspace)
	case "a":
		if m.isComposeKind() {
			return m.refuse("attach is N/A for compose-kind (no Dev Container / VS Code URI).")
		}
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
	// The terminal goes to a foreground command: close + reap the events
	// stream now; it resumes (with a rescan) when the board is back.
	m = m.pauseActivity("board was away ("+kind+") — rescanned on return", false)
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

// stackExecCommand shells into a stack row. dc-exec re-resolves membership
// from a fresh read (service / name / id prefix within this workspace's
// stack); the board's snapshot id is only the reference, never the target.
func stackExecCommand(ws string, s stackSvc) *exec.Cmd {
	ref := s.ID
	if ref == "" {
		ref = s.Service
	}
	if ref == "" {
		ref = s.Name
	}
	return exec.Command("dc-exec", "--service", ref, ws)
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
	if pending == "action" {
		return m.actionExecCmd()
	}
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
		cmd := stackExecCommand(m.workspace, s)
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
	case "try":
		cmd = exec.Command("dc-try", "--yes", ws)
	case "upgrade":
		cmd = exec.Command("dc-upgrade", "--yes")
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
	return m.switchWorkspace(folder)
}

// runStay is the stay-in-board exec. Tests replace it so confirm y never hits Docker.
var runStay = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// stayDoneMsg is a finished stay command. workspace/fleet say which board
// context launched it; a result for another context only paints status.
type stayDoneMsg struct {
	name      string
	workspace string
	fleet     bool
	out       string
	err       error
}

// stayCmd runs a stay-in-board command off the Update loop (no timeout: it is
// a user action). One at a time; the board keeps painting meanwhile.
func (m model) stayCmd(name string, args ...string) (tea.Model, tea.Cmd) {
	if m.busy != "" {
		return m.withStatus(m.busy + " still running — wait for it to finish"), nil
	}
	m.busy = name
	m = m.withStatus("running " + name + "…")
	ws, fleet := m.workspace, m.fleet
	argv := append([]string(nil), args...)
	return m, func() tea.Msg {
		out, err := runStay(name, argv...)
		return stayDoneMsg{name: name, workspace: ws, fleet: fleet, out: out, err: err}
	}
}

func (m model) applyStayDone(msg stayDoneMsg) (model, tea.Cmd) {
	m.busy = ""
	text := compactLines(strings.TrimSpace(msg.out), 4)
	if msg.err != nil {
		if text == "" {
			text = msg.err.Error()
		}
		return m.withErr(text), nil
	}
	m = m.withStatus(text)
	m.feedbackSuccess = true
	if msg.name == "dc-prune" {
		diskCache.invalidate(m.engine)
	}
	if msg.workspace != m.workspace || msg.fleet != m.fleet {
		return m, nil
	}
	return m.requestRefresh()
}

func diskLooksCritical(compact, guest string) bool {
	return highestPercent(compact, guest) >= 85
}

func highestPercent(parts ...string) int {
	best := 0
	for _, s := range parts {
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				continue
			}
			n := 0
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				n = n*10 + int(s[j]-'0')
				j++
			}
			if j < len(s) && s[j] == '%' && n > best {
				best = n
			}
			i = j
		}
	}
	return best
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
