package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Canvilled/dc-cli/internal/actions"
)

func actionWS(t *testing.T, personal, shared string) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	ws, _ := filepath.EvalSymlinks(t.TempDir())
	if personal != "" {
		p, _ := actions.PersonalPath(ws)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(personal), 0o600)
	}
	if shared != "" {
		_ = os.MkdirAll(filepath.Join(ws, ".dc"), 0o755)
		_ = os.WriteFile(actions.SharedPath(ws), []byte(shared), 0o644)
	}
	return ws
}

func loadPicker(t *testing.T, m model) model {
	t.Helper()
	got, cmd := m.handleKey("c")
	m = got.(model)
	if cmd == nil || !m.actOpen {
		t.Fatal("c must open the action picker")
	}
	got, _ = m.Update(cmd())
	return got.(model)
}

const pickerShared = `{"schemaVersion":1,"actions":[{"id":"test","label":"Team test","argv":["rm","-rf","x"]},{"id":"db","label":"DB shell","argv":["psql","-c","a b"],"service":"db"}]}`
const pickerPersonal = `{"schemaVersion":1,"actions":[{"id":"test","label":"My test","argv":["make","test"]}]}`

func TestActionPickerListsAndFilters(t *testing.T) {
	ws := actionWS(t, pickerPersonal, pickerShared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, width: 140, loaded: true})
	es := m.actEntries()
	if len(es) != 2 || es[0].Label != "My test" || es[1].Enabled {
		t.Fatalf("entries=%+v", es)
	}
	s := ansi.Strip(m.View())
	for _, want := range []string{"My test", "DB shell", "off", "disabled", "can modify data"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q:\n%s", want, s)
		}
	}
	m = typeKeys(m, "psq")
	if len(m.actEntries()) != 1 || m.actEntries()[0].ID != "db" {
		t.Fatalf("filter=%q got %+v", m.actFilter, m.actEntries())
	}
	got, _ := m.handleKey("esc")
	if got.(model).actOpen {
		t.Fatal("esc closes")
	}
}

func TestActionPickerReviewRefusesChangedFile(t *testing.T) {
	ws := actionWS(t, "", pickerShared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, width: 120, loaded: true})
	got, _ := m.handleKey("enter")
	m = got.(model)
	_ = os.WriteFile(actions.SharedPath(ws), []byte(strings.Replace(pickerShared, "rm", "curl", 1)), 0o644)
	got, cmd := m.handleKey("y")
	got, _ = got.(model).Update(cmd())
	m = got.(model)
	if m.actSet.Shared.Trusted || !strings.Contains(m.actWarn, "changed") {
		t.Fatalf("changed file must not be trusted: warn=%q", m.actWarn)
	}
}

func TestActionDoneStatusAndStaleWorkspace(t *testing.T) {
	m := model{workspace: "/w/a", hoverStack: -1, loaded: true, leaving: "action"}
	got, cmd := m.Update(actionDoneMsg{workspace: "/w/a", label: "Tests", err: nil})
	mm := got.(model)
	if mm.leaving != "" || !strings.Contains(mm.status, "Tests · exit 0") || cmd == nil || mm.load != loadPending {
		t.Fatalf("status=%q load=%v", mm.status, mm.load)
	}
	m.leaving = "action"
	got, _ = m.Update(actionDoneMsg{workspace: "/w/other", label: "Tests", err: exitErr(t, 3)})
	mm = got.(model)
	if !strings.Contains(mm.err, "exit 3") || mm.load == loadPending {
		t.Fatalf("stale workspace must report but not refresh: err=%q load=%v", mm.err, mm.load)
	}
	m.leaving = "action"
	got, _ = m.Update(actionDoneMsg{workspace: "/w/a", label: "Tests", err: exitErr(t, 130)})
	if !strings.Contains(got.(model).err, "exit 130") {
		t.Fatal("Ctrl+C status must be reported and the board kept")
	}
}

func exitErr(t *testing.T, code int) error {
	t.Helper()
	err := execShell("exit " + itoaTest(code))
	if err == nil {
		t.Fatal("want error")
	}
	return err
}

func TestActionPickerRefusedInFleetAndClosedOnSwitch(t *testing.T) {
	got, cmd := model{fleet: true, hoverStack: -1}.handleKey("c")
	if cmd != nil || got.(model).actOpen {
		t.Fatal("fleet must refuse actions")
	}
	ws := actionWS(t, pickerPersonal, "")
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	m, _ = m.switchContext(t.TempDir(), false)
	if m.actOpen || m.actSet != nil {
		t.Fatal("switch must close the action picker")
	}
	// A late load for the old workspace is dropped.
	m.actOpen = true
	m = m.applyActionsLoaded(actionsLoadedMsg{workspace: ws, set: &actions.Set{}})
	if m.actSet != nil {
		t.Fatal("stale actions load must be dropped")
	}
}

func execShell(script string) error {
	return exec.Command("sh", "-c", script).Run()
}

func itoaTest(n int) string { return strconv.Itoa(n) }

// pressEnter presses enter and, when a pre-run check starts, runs it and
// applies its result (the check reads both files fresh).
func pressEnter(t *testing.T, m model) (model, tea.Cmd) {
	t.Helper()
	got, cmd := m.handleKey("enter")
	m = got.(model)
	if cmd == nil || !m.actPreparing {
		return m, cmd
	}
	ready, ok := cmd().(actionReadyMsg)
	if !ok {
		t.Fatal("want actionReadyMsg")
	}
	got, cmd = m.Update(ready)
	return got.(model), cmd
}

func TestActionPickerReviewShowsAllSharedThenEnables(t *testing.T) {
	ws := actionWS(t, pickerPersonal, pickerShared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, width: 140, height: 40, loaded: true})
	got, _ := m.handleKey("down")
	m = got.(model)
	got, cmd := m.handleKey("enter") // disabled → review, never runs
	m = got.(model)
	if cmd != nil || !m.actReview || m.leaving != "" || m.actPreparing {
		t.Fatal("disabled entry must open review, not run")
	}
	s := ansi.Strip(m.View())
	for _, want := range []string{"Team test", "overridden", "rm -rf x", "target: service db", "psql -c 'a b'", "sha256"} {
		if !strings.Contains(s, want) {
			t.Fatalf("review missing %q:\n%s", want, s)
		}
	}
	got, cmd = m.handleKey("y")
	m = got.(model)
	got, _ = m.Update(cmd())
	m = got.(model)
	if !m.actSet.Shared.Trusted || !m.actEntries()[1].Enabled {
		t.Fatalf("y must enable exactly the reviewed file: %+v", m.actEntries())
	}
}

func TestActionReviewLongAndManyEntriesAllReachable(t *testing.T) {
	long := strings.Repeat("segment-", 40) + "END"
	var acts []string
	for i := 0; i < 30; i++ {
		acts = append(acts, `{"id":"a`+strconv.Itoa(i)+`","label":"L`+strconv.Itoa(i)+`","argv":["run","`+long+`","tail-`+strconv.Itoa(i)+`"]}`)
	}
	shared := `{"schemaVersion":1,"actions":[` + strings.Join(acts, ",") + `]}`
	ws := actionWS(t, "", shared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, width: 60, height: 20, loaded: true})
	got, _ := m.handleKey("ctrl+t")
	m = got.(model)
	if !m.actReview || m.actReviewSeenEnd {
		t.Fatal("long review must start unseen")
	}
	// y before the end is refused.
	got, cmd := m.handleKey("y")
	m = got.(model)
	if cmd != nil || !m.actReview || !strings.Contains(m.actWarn, "scroll to the end") {
		t.Fatalf("y before the end must be refused: warn=%q", m.actWarn)
	}
	// Collect every screen while paging down; every byte of every argv must appear.
	var seen strings.Builder
	for i := 0; i < 200 && !m.actReviewSeenEnd; i++ {
		seen.WriteString(ansi.Strip(m.View()))
		for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
			if ansi.StringWidth(line) > 60 {
				t.Fatalf("line wider than screen: %q", line)
			}
		}
		got, _ = m.handleKey("pgdown")
		m = got.(model)
	}
	seen.WriteString(ansi.Strip(m.View()))
	if !m.actReviewSeenEnd {
		t.Fatal("end never reached")
	}
	flat := strings.Join(strings.Fields(strings.ReplaceAll(seen.String(), "\n", " ")), "")
	wantLong := strings.Join(strings.Fields(long), "")
	for i := 0; i < 30; i++ {
		if !strings.Contains(seen.String(), "tail-"+strconv.Itoa(i)) {
			t.Fatalf("entry %d argv end never shown", i)
		}
	}
	if !strings.Contains(flat, wantLong) {
		t.Fatal("long argv must be shown in full (wrapped, not truncated)")
	}
	got, cmd = m.handleKey("y")
	if cmd == nil {
		t.Fatal("y after the end must approve")
	}
	got, _ = got.(model).Update(cmd())
	if !got.(model).actSet.Shared.Trusted {
		t.Fatal("approval after full review must enable")
	}
}

func TestActionReviewEscapesControlCharacters(t *testing.T) {
	shared := `{"schemaVersion":1,"actions":[{"id":"x\u001b[2J","label":"L\nfake","argv":["echo","\u001b[31mred"]}]}`
	ws := actionWS(t, "", shared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, width: 100, height: 30, loaded: true})
	got, _ := m.handleKey("ctrl+t")
	v := got.(model).View()
	if strings.Contains(v, "\x1b[2J") || strings.Contains(v, "\x1b[31m") {
		t.Fatal("raw escape sequences from the file must not reach the screen")
	}
	if !strings.Contains(ansi.Strip(v), `\x1b[31mred`) {
		t.Fatalf("escaped argv must be shown:\n%s", ansi.Strip(v))
	}
}

func TestActionPickerRunsEnabledInForeground(t *testing.T) {
	ws := actionWS(t, pickerPersonal, "")
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	got, cmd := m.handleKey("enter")
	m = got.(model)
	if cmd == nil || !m.actPreparing || m.leaving != "" {
		t.Fatal("enter must start the async pre-run check, not leave yet")
	}
	// Second enter while checking is ignored.
	got2, cmd2 := m.handleKey("enter")
	if cmd2 != nil || !got2.(model).actPreparing {
		t.Fatal("only one action at a time")
	}
	got, cmd = m.Update(cmd())
	m = got.(model)
	if cmd == nil || m.leaving != "action" || m.actOpen || m.actPending == nil {
		t.Fatalf("verified entry must leave to run: leaving=%q", m.leaving)
	}
	if m.actPending.entry.ID != "test" || m.actPending.workspace != ws {
		t.Fatalf("pending=%+v", m.actPending)
	}
	got2, cmd2 = m.runProjectAction(m.actPending.entry)
	if cmd2 != nil || got2.(model).leaving != "action" {
		t.Fatal("second action while leaving must be ignored")
	}
	args := actions.Args(ws, m.actPending.entry.Action)
	if args[0] != "--no-start" || args[len(args)-2] != "make" {
		t.Fatalf("args=%v", args)
	}
}

func TestActionPickerRevokedTrustRefusesRun(t *testing.T) {
	ws := actionWS(t, "", pickerShared)
	st, _ := actions.OpenTrust()
	_ = st.Approve(ws, actions.HashContent([]byte(pickerShared)))
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	if !m.actEntries()[0].Enabled {
		t.Fatal("trusted shared must be enabled")
	}
	_ = st.Revoke(ws)
	m, cmd := pressEnter(t, m)
	if cmd != nil || m.leaving != "" || m.actPending != nil || !strings.Contains(m.actWarn, "no longer trusted") {
		t.Fatalf("revoked trust must block the run: warn=%q", m.actWarn)
	}
	if m.actEntries()[0].Enabled {
		t.Fatal("list must show the fresh (disabled) state")
	}
}

func TestActionPickerSharedChangedAfterOpenRefusesOld(t *testing.T) {
	ws := actionWS(t, "", pickerShared)
	st, _ := actions.OpenTrust()
	_ = st.Approve(ws, actions.HashContent([]byte(pickerShared)))
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	// File edited and re-approved behind the board: still not what was shown.
	changed := strings.Replace(pickerShared, `"rm","-rf","x"`, `"curl","evil"`, 1)
	_ = os.WriteFile(actions.SharedPath(ws), []byte(changed), 0o644)
	_ = st.Approve(ws, actions.HashContent([]byte(changed)))
	m, cmd := pressEnter(t, m)
	if cmd != nil || m.leaving != "" || m.actPending != nil {
		t.Fatal("changed shared file must refuse the old selection")
	}
	if !strings.Contains(m.actWarn, "changed") {
		t.Fatalf("warn=%q", m.actWarn)
	}
}

func TestActionPickerPersonalChangedRefuses(t *testing.T) {
	ws := actionWS(t, pickerPersonal, "")
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	p, _ := actions.PersonalPath(ws)
	_ = os.WriteFile(p, []byte(strings.Replace(pickerPersonal, `"make","test"`, `"make","deploy"`, 1)), 0o600)
	m, cmd := pressEnter(t, m)
	if cmd != nil || m.actPending != nil || !strings.Contains(m.actWarn, "changed since it was shown") {
		t.Fatalf("edited personal must not run silently: warn=%q", m.actWarn)
	}
}

func TestActionPickerPrecedenceFlipRefuses(t *testing.T) {
	ws := actionWS(t, "", pickerShared)
	st, _ := actions.OpenTrust()
	_ = st.Approve(ws, actions.HashContent([]byte(pickerShared)))
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	// A personal "test" appears after the shared one was selected.
	p, _ := actions.PersonalPath(ws)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(pickerPersonal), 0o600)
	m, cmd := pressEnter(t, m)
	if cmd != nil || m.actPending != nil || !strings.Contains(m.actWarn, "now resolves") {
		t.Fatalf("source flip must refuse: warn=%q", m.actWarn)
	}
}

func TestActionPickerPersonalRunsWhenSharedBecomesInvalid(t *testing.T) {
	ws := actionWS(t, pickerPersonal, pickerShared)
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	_ = os.WriteFile(actions.SharedPath(ws), []byte("{broken"), 0o644)
	m, cmd := pressEnter(t, m)
	if cmd == nil || m.leaving != "action" || m.actPending == nil || m.actPending.entry.Source != actions.SourcePersonal {
		t.Fatalf("valid personal must still run: warn=%q", m.actWarn)
	}
}

func TestActionReadyStaleGenerationDropped(t *testing.T) {
	ws := actionWS(t, pickerPersonal, "")
	m := loadPicker(t, model{workspace: ws, hoverStack: -1, loaded: true})
	got, cmd := m.handleKey("enter")
	m = got.(model)
	ready := cmd().(actionReadyMsg)
	// User closed and reopened the picker meanwhile.
	got, _ = m.handleKey("esc")
	m = got.(model)
	m = loadPicker(t, m)
	got, c := m.Update(ready)
	if c != nil || got.(model).leaving != "" || got.(model).actPending != nil {
		t.Fatal("a pre-run check from an older picker must not run anything")
	}
	// Same for a late load.
	old := actionsLoadedMsg{workspace: ws, gen: ready.gen, set: &actions.Set{}}
	if got.(model).applyActionsLoaded(old).actSet == old.set {
		t.Fatal("stale load must be dropped")
	}
}
