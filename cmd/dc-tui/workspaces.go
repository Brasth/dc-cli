package main

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Canvilled/dc-cli/internal/workspaces"
)

// Recent-workspace picker (w). Registry state lives in
// ${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json; every registry
// call runs off the Update loop and reports back as wsListMsg.

// wsItem is one picker row.
type wsItem struct {
	entry   workspaces.Entry
	label   string
	missing bool
}

// wsListMsg carries the registry after a load / record / favorite / forget.
// warn is a nonfatal registry problem (malformed file, lock timeout, unwritable).
type wsListMsg struct {
	items []wsItem
	warn  string
	// recorded: a background Record finished; only refresh the picker if open.
	recorded bool
}

// openRegistry is swapped in tests.
var openRegistry = workspaces.Open

func registryItems(es []workspaces.Entry) []wsItem {
	es = workspaces.Sorted(es)
	labels := workspaces.Labels(es)
	items := make([]wsItem, len(es))
	for i, e := range es {
		items[i] = wsItem{entry: e, label: labels[i], missing: !workspaces.Exists(e.Path)}
	}
	return items
}

// registryCmd runs one registry operation and returns the fresh list.
func registryCmd(recorded bool, op func(*workspaces.Store) ([]workspaces.Entry, error)) tea.Cmd {
	return func() tea.Msg {
		st, err := openRegistry()
		if err != nil {
			return wsListMsg{warn: "recent workspaces unavailable: " + err.Error(), recorded: recorded}
		}
		es, err := op(st)
		msg := wsListMsg{recorded: recorded}
		if err != nil {
			msg.warn = "recent workspaces: " + err.Error()
			// Still show what can be read (malformed → empty, newer → entries).
			es, _ = st.Load()
		}
		msg.items = registryItems(es)
		return msg
	}
}

// recordWorkspaceCmd remembers a folder the board opened (config or not).
// Recording never starts anything.
func recordWorkspaceCmd(ws string) tea.Cmd {
	if ws == "" {
		return nil
	}
	return registryCmd(true, func(s *workspaces.Store) ([]workspaces.Entry, error) { return s.Record(ws) })
}

func (m model) applyWsList(msg wsListMsg) model {
	if msg.recorded && !m.wsOpen {
		if msg.warn != "" {
			m.wsWarn = msg.warn
		}
		return m
	}
	m.wsItems = msg.items
	m.wsWarn = msg.warn
	m.wsLoading = false
	m.clampWsCursor()
	return m
}

func (m model) openWorkspacePicker() (model, tea.Cmd) {
	m.wsOpen = true
	m.wsFilter = ""
	m.wsCursor = 0
	m.wsLoading = true
	m.confirm = ""
	m.more = false
	return m, registryCmd(false, func(s *workspaces.Store) ([]workspaces.Entry, error) { return s.Load() })
}

func (m model) closeWorkspacePicker() model {
	m.wsOpen = false
	m.wsFilter = ""
	m.wsCursor = 0
	return m
}

// wsVisible is the filtered list (case-insensitive substring on label + path).
func (m model) wsVisible() []wsItem {
	if m.wsFilter == "" {
		return m.wsItems
	}
	q := strings.ToLower(m.wsFilter)
	var out []wsItem
	for _, it := range m.wsItems {
		if strings.Contains(strings.ToLower(it.label), q) || strings.Contains(strings.ToLower(it.entry.Path), q) {
			out = append(out, it)
		}
	}
	return out
}

func (m *model) clampWsCursor() {
	n := len(m.wsVisible())
	if m.wsCursor >= n {
		m.wsCursor = n - 1
	}
	if m.wsCursor < 0 {
		m.wsCursor = 0
	}
}

func (m model) wsSelected() (wsItem, bool) {
	vis := m.wsVisible()
	if m.wsCursor < 0 || m.wsCursor >= len(vis) {
		return wsItem{}, false
	}
	return vis[m.wsCursor], true
}

// handleWorkspaceKey: typing filters, ↑/↓ move, enter opens, ctrl+f
// favorite, ctrl+d forget, esc back. q is a filter character here.
func (m model) handleWorkspaceKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "ctrl+c":
		return m.quit()
	case "esc":
		return m.closeWorkspacePicker(), nil
	case "up", "ctrl+p":
		if m.wsCursor > 0 {
			m.wsCursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.wsCursor < len(m.wsVisible())-1 {
			m.wsCursor++
		}
		return m, nil
	case "backspace":
		if m.wsFilter != "" {
			_, size := utf8.DecodeLastRuneInString(m.wsFilter)
			m.wsFilter = m.wsFilter[:len(m.wsFilter)-size]
			m.clampWsCursor()
		}
		return m, nil
	case "enter":
		it, ok := m.wsSelected()
		if !ok {
			return m, nil
		}
		if it.missing || !workspaces.Exists(it.entry.Path) {
			m.wsWarn = "folder missing on disk: " + it.entry.Path + " (ctrl+d forgets it)"
			return m, nil
		}
		m = m.closeWorkspacePicker()
		return m.switchWorkspace(it.entry.Path)
	case "ctrl+f":
		it, ok := m.wsSelected()
		if !ok {
			return m, nil
		}
		fav := !it.entry.Favorite
		path := it.entry.Path
		return m, registryCmd(false, func(s *workspaces.Store) ([]workspaces.Entry, error) { return s.SetFavorite(path, fav) })
	case "ctrl+d":
		it, ok := m.wsSelected()
		if !ok {
			return m, nil
		}
		path := it.entry.Path
		return m, registryCmd(false, func(s *workspaces.Store) ([]workspaces.Entry, error) { return s.Forget(path) })
	case "space":
		k = " "
	}
	if utf8.RuneCountInString(k) == 1 {
		m.wsFilter += k
		m.wsCursor = 0
		return m, nil
	}
	return m, nil
}

// switchContext is the one path for changing what the board looks at
// (picker, fleet enter, fleet toggle): close every stream and view tied to
// the old context, kill its probes, clear its data, and hard-reload.
func (m model) switchContext(ws string, fleet bool) (model, tea.Cmd) {
	m = m.closeLogs()
	m = m.closeTop()
	m = m.closeNets()
	m = m.stopActivity()
	m = m.closeWorkspacePicker()
	m = m.closeActionPicker()
	m.more = false
	m.confirm = ""
	m.pulse = ""
	m.engine = ""
	m = m.renewProbes()
	m.fleet = fleet
	m.workspace = ws
	m.hasConfig = hasDevcontainer(ws)
	m.hasCompose = hasRootCompose(ws)
	m = m.withStatus("")
	return m.beginHardReload()
}

// switchWorkspace opens a folder on the board and records it.
func (m model) switchWorkspace(ws string) (model, tea.Cmd) {
	m, reload := m.switchContext(ws, false)
	return m, tea.Batch(reload, recordWorkspaceCmd(ws))
}
