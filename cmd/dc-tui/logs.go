package main

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

const logCap = 4000

type logLineMsg struct{ line string }

type logDoneMsg struct{ err error }

// startLogFollow is replaced in tests. Returns a line reader and a stop func.
var startLogFollow = dockerLogFollow

func dockerLogFollow(id string) (io.ReadCloser, func(), error) {
	cmd := exec.Command("docker", "logs", "--tail", "300", "-f", id)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return nil, nil, err
	}
	var once sync.Once
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
		close(done)
	}()
	stop := func() {
		once.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
		<-done
	}
	return pr, stop, nil
}

// logTarget is the selected stack sibling, else dc-ls rows[0]. Empty stack
// falls back so a lone labeled app still has logs.
func (m model) logTarget() (id, name string) {
	if m.fleet {
		return "", ""
	}
	if len(m.stack) > 0 {
		i := m.cursor
		if i < 0 {
			i = 0
		}
		if i >= len(m.stack) {
			i = len(m.stack) - 1
		}
		s := m.stack[i]
		if s.ID != "" {
			name = s.Name
			if name == "" {
				name = s.Service
			}
			return s.ID, name
		}
	}
	if len(m.rows) > 0 && m.rows[0].ID != "" {
		return m.rows[0].ID, m.rows[0].Name
	}
	return "", ""
}

func (m model) canFollowLogs() bool {
	id, _ := m.logTarget()
	return id != ""
}

func (m model) openLogs() (model, tea.Cmd) {
	if m.fleet {
		return m.refuse("open a folder (enter / click) to start / shell / stop")
	}
	id, name := m.logTarget()
	if id == "" {
		return m.refuse("No container to log.")
	}
	m = m.closeLogs()
	r, stop, err := startLogFollow(id)
	if err != nil {
		return m.refuse("logs: " + err.Error())
	}
	m.logOpen = true
	m.logID = id
	m.logName = name
	if m.logName == "" {
		m.logName = shortID(id)
	}
	m.logLines = nil
	m.logOff = 0
	m.logFollow = true
	m.logStop = stop
	m.logR = bufio.NewReader(r)
	m.status = ""
	m.err = ""
	return m, readLogLine(m.logR)
}

func (m model) closeLogs() model {
	if m.logStop != nil {
		m.logStop()
	}
	m.logOpen = false
	m.logID = ""
	m.logName = ""
	m.logLines = nil
	m.logOff = 0
	m.logFollow = false
	m.logStop = nil
	m.logR = nil
	return m
}

func readLogLine(r *bufio.Reader) tea.Cmd {
	if r == nil {
		return nil
	}
	return func() tea.Msg {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			return logLineMsg{line: strings.TrimRight(line, "\r\n")}
		}
		if err != nil {
			return logDoneMsg{err: err}
		}
		return logLineMsg{line: ""}
	}
}

func (m model) applyLogLine(line string) (model, tea.Cmd) {
	if !m.logOpen {
		return m, nil
	}
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > logCap {
		drop := len(m.logLines) - logCap
		m.logLines = append([]string(nil), m.logLines[drop:]...)
		m.logOff -= drop
		if m.logOff < 0 {
			m.logOff = 0
		}
	}
	if m.logFollow {
		m.logOff = m.logMaxOff()
	}
	return m, readLogLine(m.logR)
}

func (m model) logPage() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	page := h - 3
	if page < 4 {
		return 4
	}
	return page
}

func (m model) logMaxOff() int {
	maxOff := len(m.logLines) - m.logPage()
	if maxOff < 0 {
		return 0
	}
	return maxOff
}

func (m model) scrollLogs(delta int) model {
	if !m.logOpen {
		return m
	}
	m.logOff += delta
	maxOff := m.logMaxOff()
	if m.logOff < 0 {
		m.logOff = 0
	}
	if m.logOff > maxOff {
		m.logOff = maxOff
	}
	m.logFollow = m.logOff >= maxOff
	return m
}

func (m model) handleLogKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "esc", "ctrl+c", "l":
		m = m.closeLogs()
		return m.withStatus("back from logs"), nil
	case "j", "down":
		return m.scrollLogs(1), nil
	case "k", "up":
		return m.scrollLogs(-1), nil
	case "pgdown":
		return m.scrollLogs(m.logPage()), nil
	case "pgup":
		return m.scrollLogs(-m.logPage()), nil
	case "g", "home":
		m.logOff = 0
		m.logFollow = false
		return m, nil
	case "G", "end":
		m.logOff = m.logMaxOff()
		m.logFollow = true
		return m, nil
	case "f":
		m.logFollow = !m.logFollow
		if m.logFollow {
			m.logOff = m.logMaxOff()
		}
		return m, nil
	}
	return m, nil
}
