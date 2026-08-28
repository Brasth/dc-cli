package main

import (
	"bufio"
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

// loadState separates "not loaded yet" from confirmed empty/stopped.
// Zero value is loadReady so existing test models with injected rows stay valid.
type loadState int

const (
	loadReady loadState = iota
	loadPending
	loadFailed
)

type model struct {
	workspace  string
	fleet      bool
	status     string
	hasConfig  bool
	hasCompose bool
	editor     string
	rows       []container
	stack      []stackSvc
	fwdMaps    []portPair
	buttons    []button
	rowY0      int
	width      int
	height     int
	err        string
	quitting   bool
	more       bool
	hover      string
	hoverStack int // -1 = none
	disk       string
	confirm    string // "", "rm", "try", or "upgrade"
	cursor     int
	leaving    string // "", "start", "shell", "logs"
	pending    string // action key after leave tick: u/e/l or stack:N
	splashOn   bool
	splash     int
	logOpen    bool
	logID      string
	logName    string
	logLines   []string
	logOff     int
	logFollow  bool
	logStop    func()
	logR       *bufio.Reader
	topOpen    bool
	topSnap    statsSnapshot
	topErr     string
	topCursor  int
	topStop    func()
	topR       *bufio.Reader
	topHist    map[string]sparkHist
	topLast    time.Time
	topStale   bool
	pulse      string
	netOpen    bool
	net        netReport
	netErr     string
	load       loadState
	loadGen    int
	loaded     bool // true when current context has a successful snapshot
	hostBlock  bool
	host       hostReport
	updateAvail      bool
	updateInstalled  string
	updateLatest     string
}

type reloadMsg struct {
	gen       int
	workspace string
	fleet     bool
	rows      []container
	stack     []stackSvc
	fwdMaps   []portPair
	disk      string
	nets      netReport
	host      hostReport
	hostErr   error
	err       error
}

type execDoneMsg struct {
	action string
	err    error
}

type leaveTickMsg struct{}

type splashTickMsg struct{}

const splashStep = 70 * time.Millisecond

func (m model) Init() tea.Cmd {
	// load/loadGen are set by main before Run; do not mutate model here.
	if m.splashOn {
		return tea.Batch(m.reloadCmd(), m.pulseCmd(), m.updateCheckCmd(), tea.Tick(splashStep, func(time.Time) tea.Msg {
			return splashTickMsg{}
		}))
	}
	return tea.Batch(m.reloadCmd(), m.pulseCmd(), m.updateCheckCmd())
}

// hardLoading is initial load or a context switch with no trusted snapshot.
func (m model) hardLoading() bool {
	return m.load == loadPending && !m.loaded
}

func (m model) refreshing() bool {
	return m.load == loadPending && m.loaded
}

func (m model) clearContextData() model {
	m.rows = nil
	m.stack = nil
	m.fwdMaps = nil
	m.net = netReport{}
	m.netErr = ""
	m.pulse = ""
	m.disk = ""
	m.cursor = 0
	m.hoverStack = -1
	m.confirm = ""
	return m
}

// beginHardReload clears context-specific data and starts discovery.
func (m model) beginHardReload() (model, tea.Cmd) {
	m.loadGen++
	m.load = loadPending
	m.loaded = false
	m = m.clearContextData()
	return m, m.reloadCmd()
}

// beginSoftReload keeps the current snapshot visible while refreshing.
func (m model) beginSoftReload() (model, tea.Cmd) {
	m.loadGen++
	m.load = loadPending
	return m, m.reloadCmd()
}

func (m model) applyReload(msg reloadMsg) model {
	if msg.gen != m.loadGen || msg.workspace != m.workspace || msg.fleet != m.fleet {
		return m
	}
	m.hasConfig = hasDevcontainer(m.workspace)
	m.hasCompose = hasRootCompose(m.workspace)
	if msg.host.blocked() {
		m.host = msg.host
		m.hostBlock = true
		m.load = loadFailed
		m.err = ""
		m.status = ""
		if !m.loaded {
			m.rows = nil
			m.stack = nil
			m.fwdMaps = nil
			m.disk = ""
			m.net = netReport{}
			m.pulse = ""
		}
		return m
	}
	m.hostBlock = false
	m.host = msg.host
	if msg.err != nil {
		m = m.withErr(msg.err.Error())
		m.load = loadFailed
		if !m.loaded {
			m.rows = nil
			m.stack = nil
			m.fwdMaps = nil
			m.net = netReport{}
		}
		m.clampCursor()
		return m
	}
	m.rows = msg.rows
	m.stack = msg.stack
	m.fwdMaps = msg.fwdMaps
	m.disk = msg.disk
	m.net = msg.nets
	m.load = loadReady
	m.loaded = true
	m.clampCursor()
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case logLineMsg:
		return m.applyLogLine(msg.line)
	case topMsg:
		return m.applyTopMsg(msg)
	case topLineMsg:
		return m.applyTopLine(msg.line)
	case netMsg:
		return m.applyNetMsg(msg)
	case topDoneMsg:
		if m.topOpen && msg.err != nil && !strings.Contains(strings.ToLower(msg.err.Error()), "eof") {
			m.topStale = true
		}
		return m, nil
	case pulseMsg:
		return m.applyPulse(msg)
	case pulseTickMsg:
		if !m.idleForPulse() {
			return m, m.pulseCmd()
		}
		return m, m.fetchPulse()
	case logDoneMsg:
		if m.logOpen {
			m.logFollow = false
			if msg.err != nil && !strings.Contains(strings.ToLower(msg.err.Error()), "eof") {
				m = m.withErr(msg.err.Error())
			}
		}
		return m, nil
	case reloadMsg:
		m = m.applyReload(msg)
	case updateCheckMsg:
		if msg.available && msg.latest != "" {
			m.updateAvail = true
			m.updateInstalled = msg.installed
			m.updateLatest = msg.latest
		}
		return m, nil
	case execDoneMsg:
		m.leaving = ""
		m.pending = ""
		if msg.action == "upgrade" {
			m.quitting = true
			if msg.err != nil {
				m = m.withErr(msg.err.Error())
			}
			return m, tea.Quit
		}
		if msg.err != nil && !benignLeaveErr(msg.action, msg.err) {
			m = m.withErr(msg.err.Error())
		} else {
			m = m.withStatus(backStatus(msg.action))
		}
		// ExecProcess disables mouse on ReleaseTerminal and never restores it.
		m, reload := m.beginSoftReload()
		return m, tea.Batch(tea.EnableMouseAllMotion, reload)
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
		if m.logOpen {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				return m.scrollLogs(-3), nil
			case tea.MouseButtonWheelDown:
				return m.scrollLogs(3), nil
			}
			return m, nil
		}
		if m.topOpen || m.netOpen {
			return m, nil
		}
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
	case action == "u", action == "create-nets":
		return "back from start"
	case action == "e" || strings.HasPrefix(action, "exec-"):
		return "back from shell"
	case action == "m":
		return "back from files"
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
	if m.hostBlock {
		return m.handleHostKey(k)
	}
	if m.splashOn {
		if k == "q" || k == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m.skipSplash(), nil
	}
	if m.logOpen {
		return m.handleLogKey(k)
	}
	if m.topOpen {
		return m.handleTopKey(k)
	}
	if m.netOpen {
		return m.handleNetKey(k)
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
		m = m.withStatus("")
		return m.beginHardReload()
	case "r":
		m = m.withStatus("")
		if m.loaded {
			return m.beginSoftReload()
		}
		return m.beginHardReload()
	case "R":
		if m.fleet {
			return m.withStatus("open a folder (enter / click) to start / shell / stop"), nil
		}
		if reason := m.actionBlockReason("R"); reason != "" {
			return m.refuse(reason)
		}
		return m.restartSelected()
	case "U":
		if !m.updateAvail {
			return m.withStatus("no update available — dc upgrade --check"), nil
		}
		m.confirm = "upgrade"
		return m, nil
	case "d":
		return m.stayCmd("dc-df")
	case "t":
		if reason := m.actionBlockReason("t"); reason != "" {
			return m.refuse(reason)
		}
		return m.openTop()
	case "n":
		if reason := m.actionBlockReason("n"); reason != "" {
			return m.refuse(reason)
		}
		return m.openNets()
	case "j", "down":
		if m.hardLoading() {
			return m, nil
		}
		return m.moveCursor(1), nil
	case "k", "up":
		if m.hardLoading() {
			return m, nil
		}
		return m.moveCursor(-1), nil
	case "enter":
		if reason := m.actionBlockReason("enter"); reason != "" {
			return m.refuse(reason)
		}
		return m.activateRow()
	case "u", "e", "o", "a", "s", "x", "l", "p", "b", "m":
		if m.fleet {
			return m.withStatus("open a folder (enter / click) to start / shell / stop"), nil
		}
		if reason := m.actionBlockReason(k); reason != "" {
			return m.refuse(reason)
		}
		return m.runAction(k)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.fleet {
			return m, nil
		}
		if reason := m.actionBlockReason("url"); reason != "" {
			return m.refuse(reason)
		}
		return m.openWebIndex(int(k[0] - '1'))
	}
	return m, nil
}

func (m model) handleHostKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "r":
		m = m.withStatus("checking Docker…")
		return m.beginHardReload()
	case "f":
		if !m.host.canApply() {
			return m.withStatus("nothing safe to apply — use d / c, or dc-recover --report"), nil
		}
		if err := runHostRecover(); err != nil {
			return m.withErr("dc-recover: " + err.Error()), nil
		}
		m = m.withStatus("applied fix — checking Docker…")
		return m.beginHardReload()
	case "d":
		url := m.host.GuideURL
		if err := openHostGuide(url); err != nil {
			return m.withErr(err.Error()), nil
		}
		return m.withStatus("opened Docker Desktop guide"), nil
	case "c":
		if err := copyHostText(colimaSetupText()); err != nil {
			return m.withErr(err.Error()), nil
		}
		return m.withStatus("copied Colima setup command"), nil
	default:
		return m, nil
	}
}

func (m model) handleConfirmKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "y":
		which := m.confirm
		m.confirm = ""
		if which == "try" {
			return m.startLeave("start", "try")
		}
		if which == "upgrade" {
			return m.startLeave("upgrade", "upgrade")
		}
		return m.stayCmd("dc-down", "--rm", m.workspace)
	case "n", "esc":
		which := m.confirm
		m.confirm = ""
		if which == "try" {
			return m.withStatus("try cancelled"), nil
		}
		if which == "upgrade" {
			return m.withStatus("upgrade cancelled"), nil
		}
		return m.withStatus("rm cancelled"), nil
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) confirmAction() string {
	switch m.confirm {
	case "rm":
		return "dc-down --rm"
	case "try":
		return "dc-try --yes"
	case "upgrade":
		return "dc-upgrade --yes"
	default:
		return ""
	}
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
	if reason := m.actionBlockReason("enter"); reason != "" {
		return m.refuse(reason)
	}
	m.clampCursor()
	if m.fleet {
		if len(m.rows) == 0 {
			return m.withStatus("empty — dc-up in a project, or dc try for no config"), nil
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
	if m.logOpen || m.topOpen || m.netOpen {
		return m, nil
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
		which := m.confirm
		m.confirm = ""
		if which == "try" {
			return m.withStatus("try cancelled"), nil
		}
		return m.withStatus("rm cancelled"), nil
	}
	for _, b := range buttons {
		if y >= b.y0 && y < b.y1 && x >= b.x0 && x < b.x1 {
			// Loading-disabled tiles refuse here. Other disabled tiles still go
			// through clickKey so runAction can emit concrete reasons (no config).
			if b.disabled {
				if reason := m.actionBlockReason(b.key); reason != "" {
					return m.refuse(reason)
				}
			}
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
		m = m.withStatus("")
		return m.beginHardReload()
	case "r":
		m = m.withStatus("")
		if m.loaded {
			return m.beginSoftReload()
		}
		return m.beginHardReload()
	default:
		if strings.HasPrefix(key, "url:") {
			if reason := m.actionBlockReason("url"); reason != "" {
				return m.refuse(reason)
			}
			return m.openWebURL(strings.TrimPrefix(key, "url:"))
		}
		if m.fleet {
			return m.withStatus("open a folder (enter / click) to start / shell / stop"), nil
		}
		if reason := m.actionBlockReason(key); reason != "" {
			return m.refuse(reason)
		}
		return m.runAction(key)
	}
}

// actionBlockReason returns a human reason when an action must wait for discovery.
// Safe controls (quit/help/fleet/reload/disk) never block here.
// Soft-refresh failure keeps the last snapshot actionable; only hard-pending
// or failed-without-snapshot discovery blocks container actions.
func (m model) actionBlockReason(key string) string {
	noSnapshot := m.hardLoading() || (m.load == loadFailed && !m.loaded)
	if !noSnapshot {
		return ""
	}
	switch key {
	case "q", "?", "h", "f", "r", "d":
		return ""
	case "u":
		// Config-based start remains available while discovery is pending/unknown.
		if m.canStart() {
			return ""
		}
	}
	if m.hardLoading() {
		return "still checking containers"
	}
	return "status unknown — press r to retry"
}

func (m model) refuse(reason string) (model, tea.Cmd) {
	return m.withStatus(reason), nil
}

func (m model) withErr(s string) model {
	m.err = s
	m.status = ""
	return m
}

func (m model) withStatus(s string) model {
	m.status = s
	m.err = ""
	return m
}
