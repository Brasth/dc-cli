package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Canvilled/dc-cli/internal/workspaces"
)

// TestMain keeps every test in this package away from the user's real
// ~/.local/state registry.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dc-tui-state-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", dir)
	// No test reaches a real `docker events` unless it installs one itself.
	startEventProcess = idleEventProcess
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// isolateRegistry gives the test its own state dir and returns the store.
func isolateRegistry(t *testing.T) *workspaces.Store {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	st, err := workspaces.Open()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// runWsCmd executes a registry cmd and applies its wsListMsg.
func runWsCmd(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected registry cmd")
	}
	msg, ok := cmd().(wsListMsg)
	if !ok {
		t.Fatal("want wsListMsg")
	}
	got, _ := m.Update(msg)
	return got.(model)
}

func typeKeys(m model, s string) model {
	for _, r := range s {
		got, _ := m.handleKey(string(r))
		m = got.(model)
	}
	return m
}

func seedRegistry(t *testing.T, st *workspaces.Store, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := st.Record(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPickerOpensFiltersAndCloses(t *testing.T) {
	st := isolateRegistry(t)
	root := t.TempDir()
	api, web := filepath.Join(root, "api"), filepath.Join(root, "web")
	_ = os.MkdirAll(api, 0o755)
	_ = os.MkdirAll(web, 0o755)
	seedRegistry(t, st, api, web)

	m := model{workspace: api, hasConfig: true, hoverStack: -1, width: 100, loaded: true}
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	if !m.wsOpen || len(m.wsItems) != 2 {
		t.Fatalf("open=%v items=%d", m.wsOpen, len(m.wsItems))
	}
	if m.wsItems[0].entry.Path != mustCanon(t, web) {
		t.Fatalf("most recent first, got %s", m.wsItems[0].entry.Path)
	}
	s := ansi.Strip(m.View())
	if !strings.Contains(s, "workspaces") || !strings.Contains(s, "current") {
		t.Fatalf("picker view:\n%s", s)
	}
	// q is a filter character, not quit.
	m = typeKeys(m, "apiq")
	if m.quitting || m.wsFilter != "apiq" {
		t.Fatalf("filter=%q quitting=%v", m.wsFilter, m.quitting)
	}
	got, _ = m.handleKey("backspace")
	m = got.(model)
	if m.wsFilter != "api" || len(m.wsVisible()) != 1 || m.wsVisible()[0].label != "api" {
		t.Fatalf("filter=%q visible=%+v", m.wsFilter, m.wsVisible())
	}
	got, _ = m.handleKey("esc")
	m = got.(model)
	if m.wsOpen || m.workspace != api {
		t.Fatal("esc must go back to the same board")
	}
}

func mustCanon(t *testing.T, p string) string {
	t.Helper()
	c, err := workspaces.Canonical(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPickerArrowsAndEnterSwitches(t *testing.T) {
	st := isolateRegistry(t)
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	_ = os.MkdirAll(a, 0o755)
	_ = os.MkdirAll(b, 0o755)
	seedRegistry(t, st, a, b) // b newest → first

	logStopped, topStopped := false, false
	m := model{
		workspace: a, hasConfig: true, hoverStack: -1, loaded: true, load: loadReady, loadGen: 4,
		rows:   []container{{ID: "x", Status: "running"}},
		probes: newProbeSession(),
		disk:   "docker 10%",
		engine: "context:a",
		cursor: 2,
	}
	oldCtx := m.probes.context()
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	// Anything still attached to the old context must be torn down by the switch.
	m.logStop = func() { logStopped = true }
	m.topStop = func() { topStopped = true }
	m.logOpen, m.topOpen, m.netOpen = true, true, true
	got, _ = m.handleKey("down")
	m = got.(model)
	got, _ = m.handleKey("up")
	m = got.(model)
	got, _ = m.handleKey("down") // a (older) is second
	m = got.(model)
	got, cmd = m.handleKey("enter")
	m = got.(model)
	if m.workspace != mustCanon(t, a) {
		t.Fatalf("workspace=%s", m.workspace)
	}
	if !logStopped || !topStopped || m.logOpen || m.topOpen || m.netOpen || m.wsOpen {
		t.Fatalf("switch must close logs/stats/nets/picker: logs=%v top=%v nets=%v", m.logOpen, m.topOpen, m.netOpen)
	}
	if oldCtx.Err() == nil {
		t.Fatal("switch must kill the old context's probes")
	}
	if len(m.rows) != 0 || m.disk != "" || m.engine != "" || m.loaded || m.loadGen != 5 {
		t.Fatalf("switch must clear context data and hard-reload: rows=%v disk=%q gen=%d", m.rows, m.disk, m.loadGen)
	}
	if cmd == nil {
		t.Fatal("switch must reload + record")
	}
}

func TestPickerMissingFolderStaysAndErrors(t *testing.T) {
	st := isolateRegistry(t)
	gone := filepath.Join(t.TempDir(), "gone")
	_ = os.MkdirAll(gone, 0o755)
	seedRegistry(t, st, gone)
	_ = os.RemoveAll(gone)

	m := model{workspace: "/tmp/app", hoverStack: -1, width: 100}
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	if len(m.wsItems) != 1 || !m.wsItems[0].missing {
		t.Fatalf("missing folder must stay listed: %+v", m.wsItems)
	}
	if !strings.Contains(ansi.Strip(m.View()), "missing") {
		t.Fatalf("view must flag missing:\n%s", ansi.Strip(m.View()))
	}
	got, cmd = m.handleKey("enter")
	m = got.(model)
	if cmd != nil || !m.wsOpen || m.workspace != "/tmp/app" {
		t.Fatal("opening a missing folder must not switch")
	}
	if !strings.Contains(m.wsWarn, "missing") {
		t.Fatalf("warn=%q", m.wsWarn)
	}
}

func TestPickerFavoriteAndForget(t *testing.T) {
	st := isolateRegistry(t)
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	_ = os.MkdirAll(a, 0o755)
	_ = os.MkdirAll(b, 0o755)
	seedRegistry(t, st, a, b)
	m := model{workspace: "/tmp/app", hoverStack: -1, width: 100}
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	got, _ = m.handleKey("down") // a
	m = got.(model)
	got, cmd = m.handleKey("ctrl+f")
	m = runWsCmd(t, got.(model), cmd)
	if !m.wsItems[0].entry.Favorite || m.wsItems[0].entry.Path != mustCanon(t, a) {
		t.Fatalf("favorite must sort first: %+v", m.wsItems)
	}
	if !strings.Contains(m.View(), "★") {
		t.Fatal("favorite star missing")
	}
	got, cmd = m.handleKey("ctrl+f") // cursor reset? toggle selected again
	m = runWsCmd(t, got.(model), cmd)
	m.wsCursor = 0
	got, cmd = m.handleKey("ctrl+d")
	m = runWsCmd(t, got.(model), cmd)
	es, _ := st.Load()
	if len(es) != 1 || len(m.wsItems) != 1 {
		t.Fatalf("forget must drop one entry: %+v", es)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("forget is registry-only; folders must stay")
		}
	}
}

func TestPickerAvailableWhenDockerDown(t *testing.T) {
	isolateRegistry(t)
	m := model{workspace: "/tmp/app", hostBlock: true, host: hostReport{Code: "docker_engine_stopped"}, hoverStack: -1}
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	if !m.wsOpen {
		t.Fatal("w must open the picker on the recover screen")
	}
	if !strings.Contains(ansi.Strip(m.View()), "workspaces") {
		t.Fatal("picker must render over the recover screen")
	}
	got, _ = m.handleKey("esc")
	if !strings.Contains(ansi.Strip(got.(model).View()), "Recover") {
		t.Fatal("esc must return to the recover screen")
	}
}

func TestPickerMalformedRegistryNonfatal(t *testing.T) {
	st := isolateRegistry(t)
	_ = os.MkdirAll(filepath.Dir(st.Path), 0o755)
	_ = os.WriteFile(st.Path, []byte("{oops"), 0o600)
	m := model{workspace: "/tmp/app", hoverStack: -1, width: 120}
	got, cmd := m.handleKey("w")
	m = runWsCmd(t, got.(model), cmd)
	if !m.wsOpen || !strings.Contains(m.wsWarn, "malformed") {
		t.Fatalf("warn=%q", m.wsWarn)
	}
	// Recording elsewhere is refused but harmless; file preserved.
	m2 := runWsCmd(t, model{hoverStack: -1}, recordWorkspaceCmd(t.TempDir()))
	if !strings.Contains(m2.wsWarn, "malformed") {
		t.Fatalf("record warn=%q", m2.wsWarn)
	}
	b, _ := os.ReadFile(st.Path)
	if string(b) != "{oops" {
		t.Fatal("malformed registry must be preserved")
	}
}

func TestInitRecordsWorkspaceWithoutConfig(t *testing.T) {
	st := isolateRegistry(t)
	dir := t.TempDir() // no .devcontainer, no compose: kind none
	cmd := recordWorkspaceCmd(dir)
	runWsCmd(t, model{hoverStack: -1}, cmd)
	es, err := st.Load()
	if err != nil || len(es) != 1 || es[0].Path != mustCanon(t, dir) {
		t.Fatalf("es=%+v err=%v", es, err)
	}
}

func TestRecordInBackgroundKeepsPickerClosed(t *testing.T) {
	isolateRegistry(t)
	m := model{hoverStack: -1}
	got, _ := m.Update(wsListMsg{recorded: true, items: []wsItem{{label: "x"}}})
	if got.(model).wsOpen || len(got.(model).wsItems) != 0 {
		t.Fatal("background record must not open or fill the picker")
	}
}

func TestSymlinkSpellingsDedupInPicker(t *testing.T) {
	st := isolateRegistry(t)
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "ln")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	runWsCmd(t, model{hoverStack: -1}, recordWorkspaceCmd(real))
	runWsCmd(t, model{hoverStack: -1}, recordWorkspaceCmd(link))
	es, _ := st.Load()
	if len(es) != 1 {
		t.Fatalf("symlink must dedup: %+v", es)
	}
}

func TestFleetToggleUsesSharedSwitch(t *testing.T) {
	m := model{
		workspace: "/tmp/app", hoverStack: -1, loaded: true, load: loadReady, loadGen: 1,
		rows:   []container{{ID: "x"}},
		probes: newProbeSession(),
	}
	ctx := m.probes.context()
	got, cmd := m.handleKey("f")
	mm := got.(model)
	if !mm.fleet || cmd == nil || len(mm.rows) != 0 {
		t.Fatal("f must still toggle fleet with a hard reload")
	}
	if ctx.Err() == nil {
		t.Fatal("fleet toggle must kill the old context's probes")
	}
	got, _ = mm.handleKey("f")
	if got.(model).fleet {
		t.Fatal("f must toggle back")
	}
}

func TestFleetEnterRecordsWorkspace(t *testing.T) {
	st := isolateRegistry(t)
	installFakeProbes(t, defaultFake()) // the reload in the batch must not touch Docker
	dir := t.TempDir()
	m := model{fleet: true, hoverStack: -1, rows: []container{{LocalFolder: dir, Status: "exited"}}}
	got, cmd := m.handleKey("enter")
	mm := got.(model)
	if mm.fleet || mm.workspace != dir || cmd == nil {
		t.Fatalf("fleet enter must open the folder: %+v", mm.workspace)
	}
	// Run the batch: one of its cmds records the workspace.
	for _, c := range cmd().(tea.BatchMsg) {
		if c == nil {
			continue
		}
		if c2, ok := c().(wsListMsg); ok && c2.recorded {
			es, _ := st.Load()
			if len(es) != 1 {
				t.Fatalf("es=%+v", es)
			}
			return
		}
	}
	t.Fatal("fleet enter must record the opened folder")
}
