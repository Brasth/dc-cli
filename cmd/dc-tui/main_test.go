package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func workspaceSpecs() [][]btnSpec {
	return model{hasConfig: true}.buttonGroups()
}

func TestRenderPrimaryEqualWidth(t *testing.T) {
	_, buttons := renderGroups(workspaceSpecs(), 80, 0, "")
	var primary []button
	for _, b := range buttons {
		if b.key == "u" || b.key == "e" || b.key == "s" {
			primary = append(primary, b)
		}
	}
	if len(primary) != 3 {
		t.Fatalf("want 3 primary, got %d", len(primary))
	}
	w0 := primary[0].x1 - primary[0].x0
	if w0 < 6 {
		t.Fatalf("tile too narrow: %d", w0)
	}
	for _, b := range primary {
		if b.x1-b.x0 != w0 {
			t.Fatalf("uneven primary %+v want width %d", b, w0)
		}
		if b.y1 <= b.y0 {
			t.Fatalf("empty hitbox %+v", b)
		}
	}
}

func TestRenderPrimaryThenMeta(t *testing.T) {
	line, buttons := renderGroups(workspaceSpecs(), 80, 0, "")
	var start, open, rm button
	for _, b := range buttons {
		switch b.key {
		case "u":
			start = b
		case "o":
			open = b
		case "x":
			rm = b
		}
	}
	if start.label == "" || open.label == "" || rm.label == "" {
		t.Fatalf("missing tiles start=%+v open=%+v rm=%+v\n%s", start, open, rm, line)
	}
	if start.y0 >= open.y0 {
		t.Fatalf("start should sit above open: start.y0=%d open.y0=%d\n%s", start.y0, open.y0, line)
	}
	if rm.y1 <= rm.y0 {
		t.Fatalf("rm hitbox empty: %+v", rm)
	}
	if !strings.Contains(line, "start") || !strings.Contains(line, "shell") {
		t.Fatalf("missing labels in:\n%s", line)
	}
}

func TestRenderNarrowWrapsMore(t *testing.T) {
	_, wide := renderGroups(workspaceSpecs(), 200, 0, "")
	_, narrow := renderGroups(workspaceSpecs(), 24, 0, "")
	maxY := func(bs []button) int {
		m := 0
		for _, b := range bs {
			if b.y0 > m {
				m = b.y0
			}
		}
		return m
	}
	if maxY(narrow) <= maxY(wide) {
		t.Fatalf("narrow should wrap more: wide=%d narrow=%d", maxY(wide), maxY(narrow))
	}
}

func TestStartHitboxAfterHeader(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, width: 80, hoverStack: -1, editor: "zed"}
	s, buttons, _ := m.layout()
	var start button
	for _, b := range buttons {
		if b.key == "u" {
			start = b
		}
	}
	if start.y0 < 4 {
		t.Fatalf("start y0=%d should sit after header\n%s", start.y0, s)
	}
	if strings.Contains(s, "open=host") {
		t.Fatalf("header still lectures open vs attach:\n%s", s)
	}
}

func TestStartHitboxNarrowLongPath(t *testing.T) {
	base := model{hasConfig: true, width: 40, hoverStack: -1, editor: "zed"}
	long := base
	long.workspace = "/Users/huynguyen/src/very-long-project-name-that-would-wrap"
	short := base
	short.workspace = "/tmp/app"
	_, lb, _ := long.layout()
	_, sb, _ := short.layout()
	ys := func(bs []button) int {
		for _, b := range bs {
			if b.key == "u" {
				return b.y0
			}
		}
		return -1
	}
	if ys(lb) != ys(sb) {
		t.Fatalf("long path moved start hitbox: long=%d short=%d", ys(lb), ys(sb))
	}
}

func TestBenignExecErr(t *testing.T) {
	if !benignExecErr(nil) {
		t.Fatal("nil should be benign")
	}
	if !benignExecErr(errStr("exit status 1")) {
		t.Fatal("exit 1 (shell) should resume TUI")
	}
	if !benignExecErr(errStr("could not restore terminal: something")) {
		t.Fatal("restore-terminal should resume")
	}
	if benignExecErr(errStr("dc-ls: command not found")) {
		t.Fatal("real errors must still surface")
	}
	if benignLeaveErr("u", errStr("exit status 1")) {
		t.Fatal("dc-up exit 1 is a real failure")
	}
	if !benignLeaveErr("e", errStr("exit status 1")) {
		t.Fatal("shell exit 1 is still benign")
	}
}

func TestDisabledStartReason(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1}
	got, cmd := m.handleKey("u")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("disabled start must not run")
	}
	if mm.status == "" {
		t.Fatal("expected a reason")
	}
	if !strings.Contains(mm.status, ".devcontainer") {
		t.Fatalf("reason=%q", mm.status)
	}
}

func TestFleetKeyNotSilent(t *testing.T) {
	m := model{fleet: true, hoverStack: -1}
	got, cmd := m.handleKey("u")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("fleet u must not run an action")
	}
	if mm.status == "" {
		t.Fatal("fleet leftover key must explain itself")
	}
	got, cmd = mm.handleKey("b")
	mm = got.(model)
	if cmd != nil {
		t.Fatal("fleet b must not run dc-db")
	}
	got, cmd = mm.handleKey("m")
	mm = got.(model)
	if cmd != nil || mm.leaving != "" {
		t.Fatal("fleet m must not leave to files")
	}
}

func TestStayDbStubsRunStay(t *testing.T) {
	old := runStay
	t.Cleanup(func() { runStay = old })
	var gotName string
	var gotArgs []string
	runStay = func(name string, args ...string) (string, error) {
		gotName = name
		gotArgs = append([]string{}, args...)
		return "opened postgres on 127.0.0.1:5433 (tableplus)", nil
	}
	got, _ := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1}.handleKey("b")
	mm := got.(model)
	if mm.leaving != "" {
		t.Fatalf("db stay must not leave the board, leaving=%q", mm.leaving)
	}
	if gotName != "dc-db" {
		t.Fatalf("ran %q", gotName)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "/tmp/app" {
		t.Fatalf("args=%v", gotArgs)
	}
	if !strings.Contains(mm.status, "127.0.0.1:5433") {
		t.Fatalf("status=%q", mm.status)
	}
	if strings.Contains(mm.status, "password") || strings.Contains(mm.err, "://") {
		t.Fatalf("status leaked url/secret: %q %q", mm.status, mm.err)
	}
}

func TestLeaveFiles(t *testing.T) {
	got, cmd := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1}.handleKey("m")
	mm := got.(model)
	if mm.leaving != "files" {
		t.Fatalf("leaving=%q", mm.leaving)
	}
	if mm.pending != "m" {
		t.Fatalf("pending=%q", mm.pending)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
	if !strings.Contains(leaveLine("files"), "files") {
		t.Fatalf("leaveLine=%q", leaveLine("files"))
	}
	if backStatus("m") != "back from files" {
		t.Fatalf("back=%q", backStatus("m"))
	}
}

func TestConfirmRmCancel(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1}
	got, cmd := m.handleKey("x")
	if cmd != nil {
		t.Fatal("x should only enter confirm")
	}
	mm := got.(model)
	if mm.confirm != "rm" {
		t.Fatalf("confirm=%q", mm.confirm)
	}
	if mm.confirmAction() != "dc-down --rm" {
		t.Fatalf("confirmAction=%q", mm.confirmAction())
	}
	got, cmd = mm.handleKey("n")
	mm = got.(model)
	if cmd != nil {
		t.Fatal("cancel must not exec")
	}
	if mm.confirm != "" {
		t.Fatalf("confirm still %q", mm.confirm)
	}
	if mm.status != "rm cancelled" {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestConfirmRmYesUsesStayHook(t *testing.T) {
	old := runStay
	t.Cleanup(func() { runStay = old })
	var saw []string
	runStay = func(name string, args ...string) (string, error) {
		saw = append(saw, name)
		saw = append(saw, args...)
		return "removed", nil
	}
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1, confirm: "rm"}
	got, _ := m.handleKey("y")
	mm := got.(model)
	if mm.confirm != "" {
		t.Fatalf("confirm still %q", mm.confirm)
	}
	if mm.status != "removed" {
		t.Fatalf("status=%q", mm.status)
	}
	if mm.err != "" {
		t.Fatalf("err=%q", mm.err)
	}
	want := []string{"dc-down", "--rm", "/tmp/app"}
	if strings.Join(saw, " ") != strings.Join(want, " ") {
		t.Fatalf("ran %v want %v", saw, want)
	}
}

func TestLeaveOnShell(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1}
	got, cmd := m.handleKey("e")
	mm := got.(model)
	if mm.leaving != "shell" {
		t.Fatalf("leaving=%q", mm.leaving)
	}
	if mm.pending != "e" {
		t.Fatalf("pending=%q", mm.pending)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
}

func TestLeaveIgnoresSecondShell(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1, leaving: "shell", pending: "e"}
	got, cmd := m.handleKey("e")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("second shell during leave must not tick again")
	}
	if mm.pending != "e" {
		t.Fatalf("pending changed: %q", mm.pending)
	}
}

func TestLeaveOnLogs(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		rows:       []container{{ID: "abc123"}},
	}
	got, cmd := m.handleKey("l")
	mm := got.(model)
	if mm.leaving != "logs" {
		t.Fatalf("leaving=%q", mm.leaving)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
}

func TestCursorClampAndEnterFleet(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	m := model{
		fleet:      true,
		hoverStack: -1,
		rows: []container{
			{LocalFolder: dir1, Status: "running"},
			{LocalFolder: dir2, Status: "exited"},
		},
	}
	got, _ := m.handleKey("j")
	mm := got.(model)
	if mm.cursor != 1 {
		t.Fatalf("cursor=%d", mm.cursor)
	}
	got, cmd := mm.handleKey("j")
	mm = got.(model)
	if mm.cursor != 1 {
		t.Fatalf("cursor should clamp at last row, got %d", mm.cursor)
	}
	got, cmd = mm.handleKey("enter")
	mm = got.(model)
	if mm.fleet {
		t.Fatal("enter should leave fleet")
	}
	if mm.workspace != dir2 {
		t.Fatalf("workspace=%s want %s", mm.workspace, dir2)
	}
	if cmd == nil {
		t.Fatal("expected reload after openRow")
	}
}

func TestStayCmdSplitsStatusAndErr(t *testing.T) {
	if compactLines("a\nb\nc\nd\ne", 4) != "b · c · d · e" {
		t.Fatalf("compact=%q", compactLines("a\nb\nc\nd\ne", 4))
	}
	if backStatus("e") != "back from shell" {
		t.Fatalf("back shell=%q", backStatus("e"))
	}
	if backStatus("l") != "back from logs" {
		t.Fatalf("back logs=%q", backStatus("l"))
	}
	old := runStay
	t.Cleanup(func() { runStay = old })
	runStay = func(name string, args ...string) (string, error) {
		return "port taken", errStr("exit status 1")
	}
	got, _ := model{workspace: "/tmp/app", hoverStack: -1, status: "old ok"}.stayCmd("dc-forward", "/tmp/app")
	mm := got.(model)
	if mm.err == "" {
		t.Fatal("stayCmd failure must set err")
	}
	if mm.status != "" {
		t.Fatalf("err should clear status, status=%q", mm.status)
	}
}

func TestDbFilesOwnRow(t *testing.T) {
	_, buttons := renderGroups(workspaceSpecs(), 80, 0, "")
	var db, files, logs, fleet button
	for _, b := range buttons {
		switch b.key {
		case "b":
			db = b
		case "m":
			files = b
		case "l":
			logs = b
		case "f":
			fleet = b
		}
	}
	if db.label == "" || files.label == "" {
		t.Fatal("db and files tiles must be on the board")
	}
	if db.y0 <= logs.y0 {
		t.Fatalf("db should sit below logs: db.y0=%d logs.y0=%d", db.y0, logs.y0)
	}
	if fleet.y0 <= db.y0 {
		t.Fatalf("fleet should sit below db: fleet.y0=%d db.y0=%d", fleet.y0, db.y0)
	}
	if db.y0 != files.y0 {
		t.Fatalf("db and files should share a row: db.y0=%d files.y0=%d", db.y0, files.y0)
	}
}

func TestHeaderShowsVersion(t *testing.T) {
	t.Setenv("DC_CLI_VERSION", "0.10.0")
	m := model{workspace: filepath.Join(os.TempDir(), "app"), hasConfig: true, width: 80, hoverStack: -1, editor: "zed"}
	s := ansi.Strip(m.View())
	if !strings.Contains(s, "0.10.0") {
		t.Fatalf("header missing version:\n%s", s)
	}
	if !strings.Contains(s, "db") || !strings.Contains(s, "files") {
		t.Fatalf("board missing db/files:\n%s", s)
	}
}

func TestDisabledStartTile(t *testing.T) {
	groups := model{hasConfig: false}.buttonGroups()
	_, buttons := renderGroups(groups, 80, 0, "")
	for _, b := range buttons {
		if b.key == "u" && !b.disabled {
			t.Fatal("start should be disabled without config")
		}
	}
}

func TestClickDisabledStart(t *testing.T) {
	m := model{workspace: "/tmp/app", width: 80, hoverStack: -1, editor: "zed"}
	_, buttons, _ := m.layout()
	var start button
	for _, b := range buttons {
		if b.key == "u" {
			start = b
		}
	}
	if start.label == "" {
		t.Fatal("no start tile")
	}
	got, cmd := m.handleClick(start.x0, start.y0)
	mm := got.(model)
	if cmd != nil {
		t.Fatal("click disabled start must not run")
	}
	if !strings.Contains(mm.status, ".devcontainer") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestEnterStackLeaves(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		stack: []stackSvc{
			{ID: "1", Service: "app"},
			{ID: "2", Service: "db"},
		},
	}
	got, _ := m.handleKey("j")
	got, cmd := got.(model).handleKey("enter")
	mm := got.(model)
	if mm.leaving != "shell" {
		t.Fatalf("leaving=%q", mm.leaving)
	}
	if mm.pending != "stack:1" {
		t.Fatalf("pending=%q", mm.pending)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
}

func TestLogoMarkHasFrameAndPip(t *testing.T) {
	mark := logoCompact(splashLast)
	if !strings.Contains(mark, "╭") || !strings.Contains(mark, "●") {
		t.Fatalf("compact mark missing frame or pip:\n%s", mark)
	}
	splash := logoSplash(splashLast, 40)
	if !strings.Contains(splash, "dc-cli") || !strings.Contains(splash, "●") {
		t.Fatalf("splash missing wordmark or pip:\n%s", splash)
	}
	early := logoSplash(2, 40)
	if strings.Contains(early, "dc-cli") {
		t.Fatal("wordmark should wait until the last splash frame")
	}
}

func pipCell(s string) (row, col int) {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i, l := range lines {
		if j := strings.IndexRune(l, '●'); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

func TestPipPatrolsThenDocks(t *testing.T) {
	if len(splashPipPath) < 4 {
		t.Fatal("path too short to read as a lap")
	}
	seen := map[[2]int]int{}
	for i := splashBuild + 1; i < splashLast; i++ {
		r, c := pipCell(logoSplash(i, 0))
		if r < 0 {
			t.Fatalf("lap frame %d missing pip", i)
		}
		seen[[2]int{r, c}]++
	}
	if len(seen) < 6 {
		t.Fatalf("pip should move around the frame, unique cells=%d", len(seen))
	}
	dockR, dockC := pipCell(logoSplash(splashLast, 0))
	if dockR < 0 {
		t.Fatal("docked splash missing pip")
	}
	// Dock sits on the inner guest, not still on the last outer cell.
	last := splashPipPath[len(splashPipPath)-1]
	if dockR == last[0] && dockC == last[1] {
		t.Fatalf("dock should leave the patrol path, still at %d,%d", dockR, dockC)
	}
	cr, cc := pipCell(logoCompact(splashLast))
	if cr < 0 || cc < 0 {
		t.Fatal("header rest pose missing pip")
	}
}

func TestSplashSkipsOnKey(t *testing.T) {
	m := model{workspace: "/tmp/app", splashOn: true, splash: 2, hoverStack: -1}
	got, cmd := m.handleKey(" ")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("skip splash should not run an action")
	}
	if mm.splashOn {
		t.Fatal("space should skip splash")
	}
	if !strings.Contains(mm.View(), "start") {
		t.Fatalf("board should show after skip:\n%s", mm.View())
	}
}

func TestSplashTickEnds(t *testing.T) {
	m := model{splashOn: true, splash: splashLast + splashHold - 1}
	got, cmd := m.Update(splashTickMsg{})
	mm := got.(model)
	if mm.splashOn {
		t.Fatal("last hold tick should end splash")
	}
	if cmd != nil {
		t.Fatal("no more splash ticks after end")
	}
}

func TestViewHasPrimaryHint(t *testing.T) {
	m := model{workspace: filepath.Join(os.TempDir(), "app"), hasConfig: true, width: 80, hoverStack: -1, editor: "zed"}
	s := m.View()
	if !strings.Contains(s, "start") || !strings.Contains(s, "shell") || !strings.Contains(s, "stop") {
		t.Fatalf("missing primary verbs:\n%s", s)
	}
}

func TestExecDoneStartFailure(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1, status: "old", leaving: "start"}
	got, cmd := m.Update(execDoneMsg{action: "u", err: errStr("exit status 1")})
	mm := got.(model)
	if mm.err == "" {
		t.Fatal("dc-up exit 1 must set err")
	}
	if mm.status != "" {
		t.Fatalf("start failure must not paint back-from-start, status=%q", mm.status)
	}
	if mm.leaving != "" {
		t.Fatalf("leaving=%q", mm.leaving)
	}
	if cmd == nil {
		t.Fatal("expected reload after start")
	}
	got, _ = mm.Update(reloadMsg{rows: []container{{ID: "abc", Status: "exited"}}})
	mm = got.(model)
	if mm.err == "" {
		t.Fatal("reload must not wipe start failure")
	}
	if len(mm.rows) != 1 || mm.rows[0].Status != "exited" {
		t.Fatalf("rows not refreshed: %+v", mm.rows)
	}
}

func TestExecDoneReenablesMouse(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, leaving: "start"}
	_, cmd := m.Update(execDoneMsg{action: "u"})
	if cmd == nil {
		t.Fatal("return from start must reload and re-enable mouse")
	}
}

func TestExecDoneShellExit1IsBack(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, err: "old", leaving: "shell"}
	got, _ := m.Update(execDoneMsg{action: "e", err: errStr("exit status 1")})
	mm := got.(model)
	if mm.err != "" {
		t.Fatalf("shell exit 1 should stay benign, err=%q", mm.err)
	}
	if mm.status != "back from shell" {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestStartHitboxLongPorts(t *testing.T) {
	base := model{workspace: "/tmp/app", hasConfig: true, width: 40, hoverStack: -1, editor: "zed"}
	long := base
	long.rows = []container{{
		ID:     "203467d304351234",
		Status: "running",
		Ports:  "0.0.0.0:3000->3000/tcp, 0.0.0.0:5432->5432/tcp, 0.0.0.0:9001->9001/tcp",
	}}
	short := base
	_, lb, _ := long.layout()
	_, sb, _ := short.layout()
	ys := func(bs []button) int {
		for _, b := range bs {
			if b.key == "u" {
				return b.y0
			}
		}
		return -1
	}
	if ys(lb) != ys(sb) {
		t.Fatalf("long ports moved start hitbox: long=%d short=%d", ys(lb), ys(sb))
	}
}

func TestHoveredStackRowFitsWidth(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		width:      40,
		cursor:     0,
		hoverStack: 0,
		editor:     "zed",
		stack: []stackSvc{
			{Name: "very-long-container-name-that-would-wrap", Service: "app", Status: "running"},
			{Name: "db-1", Service: "db", Status: "running"},
		},
	}
	s, _, rowY0 := m.layout()
	if rowY0 < 0 {
		t.Fatal("missing stack rows")
	}
	lines := strings.Split(s, "\n")
	if rowY0+1 >= len(lines) {
		t.Fatalf("not enough lines after rowY0=%d", rowY0)
	}
	app, db := lines[rowY0], lines[rowY0+1]
	if ansi.StringWidth(app) > 40 {
		t.Fatalf("hovered app row width %d > 40", ansi.StringWidth(app))
	}
	if ansi.StringWidth(db) > 40 {
		t.Fatalf("db row width %d > 40", ansi.StringWidth(db))
	}
	if !strings.Contains(db, "db") {
		t.Fatalf("wrap stole the next hitbox row: app=%q db=%q", app, db)
	}
	if m.hitRow(2, rowY0+1) != 1 {
		t.Fatalf("click on second visual row should hit db, got %d", m.hitRow(2, rowY0+1))
	}
}

func TestLayoutNoWrapAtNarrowWidth(t *testing.T) {
	m := model{
		workspace:  "/Users/huynguyen/src/very-long-project-name-that-would-wrap",
		hasConfig:  true,
		width:      40,
		hoverStack: -1,
		editor:     "zed",
		disk:       "docker 42% · colima 61% · extra long disk line that would wrap",
		status:     "this status line is also quite long and should not wrap the board",
		more:       true,
		rows: []container{{
			ID:     "203467d304351234",
			Status: "running",
			Ports:  "0.0.0.0:3000->3000/tcp, 0.0.0.0:5432->5432/tcp, 0.0.0.0:9001->9001/tcp",
		}},
	}
	s, _, _ := m.layout()
	for i, line := range strings.Split(s, "\n") {
		if ansi.StringWidth(line) > 40 {
			t.Fatalf("line %d width %d > 40", i, ansi.StringWidth(line))
		}
	}
}

func TestOpenRowClearsStatusOnErr(t *testing.T) {
	m := model{fleet: true, hoverStack: -1, status: "old ok", rows: []container{{Name: "orphan"}}}
	got, cmd := m.openRow(0)
	mm := got.(model)
	if cmd != nil {
		t.Fatal("missing folder must not reload")
	}
	if mm.err == "" {
		t.Fatal("want err")
	}
	if mm.status != "" {
		t.Fatalf("status leftover %q", mm.status)
	}
}

func TestHelpAndMoreTeachZedAttach(t *testing.T) {
	if strings.Contains(helpText, "Zed and Sublime cannot attach") {
		t.Fatal("help still says Zed cannot attach")
	}
	if !strings.Contains(helpText, "Connect Dev Container") {
		t.Fatal("help must teach Zed first-party attach")
	}
	more := morePanel("zed", 120)
	if strings.Contains(more, "Zed/Sublime = open only") {
		t.Fatal("more still says Zed is open only")
	}
	if strings.Contains(more, "cannot attach") && !strings.Contains(more, "Sublime cannot") {
		t.Fatalf("more still lumps Zed with cannot attach:\n%s", more)
	}
	if !strings.Contains(more, "Connect Dev Container") {
		t.Fatalf("more must teach Zed first-party attach:\n%s", more)
	}
	if !strings.Contains(more, "Zed attaches itself") {
		t.Fatalf("more must say Zed attaches itself:\n%s", more)
	}
	if !strings.Contains(more, "db") || !strings.Contains(more, "files") {
		t.Fatalf("more must document db and files:\n%s", more)
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
