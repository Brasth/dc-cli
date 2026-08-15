package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	key      string
	label    string
	disabled bool
	x0       int
	x1       int
	y0       int
	y1       int
}

type model struct {
	workspace  string
	fleet      bool
	status     string
	hasConfig  bool
	editor     string
	rows       []container
	stack      []stackSvc
	buttons    []button
	rowY0      int
	width      int
	err        string
	quitting   bool
	more       bool
	hover      string
	hoverStack int // -1 = none
	disk       string
	confirm    string // "" or "rm"
	cursor     int
	leaving    string // "", "start", "shell", "logs"
	pending    string // action key after leave tick: u/e/l or stack:N
	splashOn   bool
	splash     int
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

type leaveTickMsg struct{}

type splashTickMsg struct{}

const splashStep = 90 * time.Millisecond

func (m model) Init() tea.Cmd {
	if m.splashOn {
		return tea.Batch(m.reload(), tea.Tick(splashStep, func(time.Time) tea.Msg {
			return splashTickMsg{}
		}))
	}
	return m.reload()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case reloadMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
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
		m.clampCursor()
	case execDoneMsg:
		m.leaving = ""
		m.pending = ""
		if msg.err != nil && !benignLeaveErr(msg.action, msg.err) {
			m.err = msg.err.Error()
			m.status = ""
		} else {
			m.err = ""
			m.status = backStatus(msg.action)
		}
		return m, m.reload()
	case leaveTickMsg:
		return m.runPending()
	case splashTickMsg:
		if !m.splashOn {
			return m, nil
		}
		m.splash++
		if m.splash >= splashLast+splashHold {
			m.splashOn = false
			return m, nil
		}
		return m, tea.Tick(splashStep, func(time.Time) tea.Msg {
			return splashTickMsg{}
		})
	case tea.KeyMsg:
		return m.handleKey(msg.String())
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionMotion:
			m.hover = m.hitButton(msg.X, msg.Y)
			if i := m.hitRow(msg.X, msg.Y); i >= 0 {
				m.cursor = i
				m.hoverStack = i
			} else {
				m.hoverStack = -1
			}
			return m, nil
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				return m.handleClick(msg.X, msg.Y)
			}
		}
	}
	return m, nil
}

func backStatus(action string) string {
	switch {
	case action == "l":
		return "back from logs"
	case action == "u":
		return "back from start"
	case action == "e" || strings.HasPrefix(action, "exec-"):
		return "back from shell"
	default:
		return ""
	}
}

func (m model) skipSplash() model {
	m.splashOn = false
	m.splash = splashLast
	return m
}

func (m model) handleKey(k string) (tea.Model, tea.Cmd) {
	if m.splashOn {
		if k == "q" || k == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m.skipSplash(), nil
	}
	if m.leaving != "" {
		if k == "q" || k == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}
	if m.confirm != "" {
		return m.handleConfirmKey(k)
	}
	switch k {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.quitting = true
		return m, tea.Quit
	case "?", "h":
		m.more = !m.more
		return m, nil
	case "f":
		m.fleet = !m.fleet
		m.more = false
		m.cursor = 0
		m.status = ""
		return m, m.reload()
	case "r":
		return m, m.reload()
	case "d":
		return m.stayCmd("dc-df")
	case "j", "down":
		return m.moveCursor(1), nil
	case "k", "up":
		return m.moveCursor(-1), nil
	case "enter":
		return m.activateRow()
	case "u", "e", "o", "a", "s", "x", "l", "p":
		if m.fleet {
			m.status = "open a folder (enter / click) to start / shell / stop"
			return m, nil
		}
		return m.runAction(k)
	}
	return m, nil
}

func (m model) handleConfirmKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "y":
		m.confirm = ""
		return m.stayCmd("dc-down", "--rm", m.workspace)
	case "n", "esc":
		m.confirm = ""
		m.status = "rm cancelled"
		return m, nil
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) confirmAction() string {
	if m.confirm == "rm" {
		return "dc-down --rm"
	}
	return ""
}

func (m model) rowCount() int {
	if m.fleet {
		return len(m.rows)
	}
	return len(m.stack)
}

func (m *model) clampCursor() {
	n := m.rowCount()
	if n <= 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m model) moveCursor(delta int) model {
	if m.rowCount() == 0 {
		return m
	}
	m.cursor += delta
	m.clampCursor()
	m.hoverStack = m.cursor
	return m
}

func (m model) activateRow() (tea.Model, tea.Cmd) {
	m.clampCursor()
	if m.fleet {
		if len(m.rows) == 0 {
			m.status = "empty — dc-up in a project"
			return m, nil
		}
		return m.openRow(m.cursor)
	}
	if len(m.stack) > 0 {
		return m.execStack(m.cursor)
	}
	return m.runAction("e")
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

func (m model) hitRow(x, y int) int {
	_, _, rowY0 := m.layout()
	if rowY0 <= 0 {
		return -1
	}
	i := y - rowY0
	if i >= 0 && i < m.rowCount() {
		return i
	}
	return -1
}

func (m model) handleClick(x, y int) (tea.Model, tea.Cmd) {
	if m.splashOn {
		return m.skipSplash(), nil
	}
	if m.leaving != "" {
		return m, nil
	}
	_, buttons, rowY0 := m.layout()
	m.buttons = buttons
	m.rowY0 = rowY0
	if m.confirm != "" {
		if m.hitButton(x, y) == "x" {
			return m, nil
		}
		m.confirm = ""
		m.status = "rm cancelled"
		return m, nil
	}
	for _, b := range buttons {
		if y >= b.y0 && y < b.y1 && x >= b.x0 && x < b.x1 {
			return m.clickKey(b.key)
		}
	}
	if i := m.hitRow(x, y); i >= 0 {
		m.cursor = i
		return m.activateRow()
	}
	return m, nil
}

func (m model) clickKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "?":
		m.more = !m.more
		return m, nil
	case "f":
		m.fleet = !m.fleet
		m.more = false
		m.cursor = 0
		m.status = ""
		return m, m.reload()
	case "r":
		return m, m.reload()
	default:
		if m.fleet {
			m.status = "open a folder (enter / click) to start / shell / stop"
			return m, nil
		}
		return m.runAction(key)
	}
}

func (m model) refuse(reason string) (model, tea.Cmd) {
	m.status = reason
	m.err = ""
	return m, nil
}
