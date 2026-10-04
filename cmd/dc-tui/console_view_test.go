package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func consoleFixture(w, h int) model {
	return model{workspace: "/Users/dev/src/acme-storefront", width: w, height: h, hasConfig: true, loaded: true,
		editor: "zed", engine: "context:colima", cursor: 1, hoverStack: -1, pulse: "cpu 3.2%  mem 412M / 7.8G", disk: "docker 42% · colima 61%",
		rows: []container{{ID: "203467d30435", Status: "running", Ports: "127.0.0.1:5173->5173/tcp"}},
		stack: []stackSvc{
			{ID: "203467d30435", Service: "app", Name: "acme-storefront-app-1", Status: "running", Image: "mcr.microsoft.com/devcontainers/typescript-node:22"},
			{ID: "8c1f0e2a9b7d", Service: "db", Name: "acme-storefront-db-1", Status: "running", Image: "postgres:16"},
			{ID: "redis", Service: "redis", Name: "acme-storefront-redis-1", Status: "running", Image: "redis:7-alpine"},
			{ID: "worker", Service: "worker", Name: "acme-storefront-worker-1", Status: "exited", Image: "acme/worker:dev"},
			{ID: "mailpit", Service: "mailpit", Name: "acme-storefront-mailpit-1", Status: "running", Image: "axllent/mailpit"},
		},
	}
}

func assertFits(t *testing.T, m model, s string) {
	t.Helper()
	lines := strings.Split(s, "\n")
	if len(lines) > m.height {
		t.Fatalf("%dx%d: rendered %d rows", m.width, m.height, len(lines))
	}
	for i, line := range lines {
		if width := ansi.StringWidth(line); width > m.width {
			t.Fatalf("line %d has %d cells at width %d: %q", i, width, m.width, line)
		}
	}
}

func TestConsoleResponsiveAndHitboxes(t *testing.T) {
	t.Setenv("DC_CLI_VERSION", "0.21.0")
	for _, size := range [][2]int{{120, 36}, {100, 30}, {80, 24}, {60, 20}, {40, 12}, {120, 12}, {20, 8}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := consoleFixture(size[0], size[1])
			frame := m.consoleLayout()
			assertFits(t, m, frame.view)
			plain := ansi.Strip(frame.view)
			if !strings.Contains(plain, "> db") {
				t.Fatalf("selected service missing:\n%s", plain)
			}
			if m.width >= 110 && m.height >= 28 && !strings.Contains(plain, "SELECTED SERVICE") {
				t.Fatal("wide layout missing details")
			}
			if m.width >= 80 && m.width < 110 && !strings.Contains(plain, "selected db") {
				t.Fatal("medium layout missing compact details")
			}
			if m.width >= 60 && m.height >= 16 && !strings.Contains(plain, "ROLE") {
				t.Fatal("regular layout missing role column")
			}
			for _, b := range frame.buttons {
				if got := m.hitButton(b.x0, b.y0); got != b.key {
					t.Fatalf("hit %s got %s", b.key, got)
				}
			}
			if got := m.hitRow(1, frame.rowY+1); got != 1 {
				t.Fatalf("second row targets %d", got)
			}
			if m.hitRow(1, frame.rowY+frame.rowVisible) != -1 {
				t.Fatal("below viewport must not activate a hidden service")
			}
			if frame.rowWidth < m.width && m.hitRow(frame.rowWidth+2, frame.rowY) != -1 {
				t.Fatal("details pane must not activate a row")
			}
		})
	}
}

func TestConsoleScrolledSelectionAndMouse(t *testing.T) {
	m := consoleFixture(80, 24)
	for i := 0; i < 60; i++ {
		m.stack = append(m.stack, stackSvc{ID: fmt.Sprint(i), Service: fmt.Sprintf("svc-%02d", i), Status: "running"})
	}
	m.cursor = len(m.stack) - 1
	frame := m.consoleLayout()
	assertFits(t, m, frame.view)
	if !strings.Contains(ansi.Strip(frame.view), "> svc-59") {
		t.Fatal("last selected service is offscreen")
	}
	if m.hitRow(1, frame.rowY+frame.rowVisible-1) != m.cursor {
		t.Fatal("scrolled click must target actual service index")
	}
	got, cmd := m.Update(tea.MouseMsg{X: 2, Y: frame.rowY, Action: tea.MouseActionMotion})
	hovered := got.(model)
	if cmd != nil || hovered.cursor != m.cursor || hovered.hoverStack != frame.rowStart {
		t.Fatal("hover changed keyboard selection")
	}
	view := ansi.Strip(hovered.View())
	if !strings.Contains(view, "· svc-") || !strings.Contains(view, "> svc-59") {
		t.Fatal("selection and hover markers must be distinct")
	}
}

func TestConsoleUntrustedNamesAndMonochrome(t *testing.T) {
	m := consoleFixture(100, 30)
	m.workspace = "/tmp/\x1b]2;owned\a工作e\u0301"
	m.stack[1].Service = "\x1b[2J数据库e\u0301\n\t" + strings.Repeat("长", 80)
	m.stack[1].Image = "\x1b]8;;https://untrusted\aimage\x1b]8;;\a"
	m.stack[1].Name = "\x1b[31muntrusted\x1b[0m\rname"
	s := m.View()
	assertFits(t, m, s)
	if strings.Contains(s, "\x1b[2J") || strings.Contains(s, "\x1b]") || strings.Contains(s, "\r") || strings.Contains(s, "\t") {
		t.Fatal("untrusted terminal controls escaped the display sanitizer")
	}
	previous := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
	lipgloss.SetColorProfile(termenv.Ascii)
	s = m.View()
	assertFits(t, m, s)
	if strings.Contains(s, "\x1b") || !strings.Contains(s, ">") || !strings.Contains(s, "running") {
		t.Fatal("monochrome lost selection or textual status")
	}
}

func TestConsoleStatesAndConfirmations(t *testing.T) {
	for _, size := range [][2]int{{120, 36}, {80, 24}, {40, 12}} {
		for _, confirm := range []string{"rm", "try", "upgrade", "prune"} {
			m := consoleFixture(size[0], size[1])
			m.confirm = confirm
			m.updateLatest = "0.22.0"
			m.more = true
			s := m.View()
			assertFits(t, m, s)
			if !strings.Contains(ansi.Strip(s), m.confirmAction()) {
				t.Fatalf("%dx%d %s confirmation hides command:\n%s", m.width, m.height, confirm, ansi.Strip(s))
			}
		}
	}
	m := consoleFixture(80, 24)
	m.loaded = false
	m.load = loadPending
	m.rows = nil
	m.stack = nil
	if s := ansi.Strip(m.View()); !strings.Contains(s, "checking") || strings.Contains(s, "stopped") {
		t.Fatal("pending discovery must not appear stopped")
	}
	m.load = loadFailed
	if !strings.Contains(ansi.Strip(m.View()), "unknown") {
		t.Fatal("failed discovery must say unknown")
	}
	m.load = loadReady
	if !strings.Contains(ansi.Strip(m.View()), "stopped") {
		t.Fatal("successful empty discovery must say stopped")
	}
	m = consoleFixture(80, 24)
	m.load = loadPending
	if s := ansi.Strip(m.View()); !strings.Contains(s, "refreshing") || !strings.Contains(s, "db") {
		t.Fatal("refresh must preserve snapshot")
	}
}

func TestConsoleSecondaryScreensFit(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}, {40, 12}} {
		m := consoleFixture(size[0], size[1])
		m.logName = "db"
		m.logFollow = true
		for i := 0; i < 50; i++ {
			m.logLines = append(m.logLines, "INFO\tdb\tready")
		}
		m.logOff = m.logMaxOff()
		m.wsOpen = true
		m.wsLoading = true
		for name, view := range map[string]string{"logs": m.logsView(), "top": m.topView(), "nets": m.netView(), "recover": m.hostView(), "workspaces": m.workspaceView(), "actions": m.actionPickerView(), "activity": m.activityView()} {
			t.Run(fmt.Sprintf("%dx%d/%s", m.width, m.height, name), func(t *testing.T) { assertFits(t, m, view) })
		}
	}
	if got := logDisplayText("a\tb\t界\tx"); got != "a       b       界      x" {
		t.Fatalf("tab stops: %q", got)
	}
}

func TestConsoleHelpAndOutputReachable(t *testing.T) {
	m := consoleFixture(40, 12)
	m.more = true
	if !strings.Contains(ansi.Strip(m.View()), "HELP") {
		t.Fatal("minimal help is inaccessible")
	}
	got, _ := m.handleKey("pgdown")
	m = got.(model)
	if m.moreOff == 0 {
		t.Fatal("help does not scroll")
	}
	got, _ = m.handleKey("esc")
	m = got.(model)
	if m.more || m.quitting {
		t.Fatal("Esc should return from help")
	}
	m.status = "disk report\n" + strings.Repeat("row\n", 50) + "END"
	got, _ = m.handleKey("G")
	m = got.(model)
	s := m.View()
	assertFits(t, m, s)
	if !strings.Contains(ansi.Strip(s), "END") {
		t.Fatal("last output line unreachable")
	}
	got, _ = m.handleKey("q")
	m = got.(model)
	if m.status != "" || m.quitting {
		t.Fatal("output q should return to board")
	}
}

// Export the actual renderer's fixtures for the documentation preview script.
// This remains optional: normal tests do not write repository artifacts.
func TestConsolePreviewFixtures(t *testing.T) {
	dir := os.Getenv("DC_TUI_PREVIEW_DIR")
	if dir == "" {
		t.Skip("preview export not requested")
	}
	release, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DC_CLI_VERSION", strings.TrimSpace(string(release)))
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	type fixture struct {
		Name          string
		Width, Height int
		View          string
	}
	var fixtures []fixture
	for _, size := range [][2]int{{120, 36}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m := consoleFixture(size[0], size[1])
		fixtures = append(fixtures, fixture{fmt.Sprintf("board-%dx%d", m.width, m.height), m.width, m.height, m.View()})
	}
	m := consoleFixture(100, 30)
	m.logName = "db"
	m.logFollow = true
	m.logLines = []string{"2026-10-04T10:00:00Z INFO database system is ready to accept connections", "2026-10-04T10:00:01Z INFO connection received"}
	fixtures = append(fixtures, fixture{"logs", m.width, m.height, m.logsView()})
	m.more = true
	fixtures = append(fixtures, fixture{"help", m.width, m.height, m.View()})
	m = consoleFixture(100, 30)
	m.fleet = true
	m.rows = []container{{ID: "a", Status: "running", LocalFolder: "/Users/dev/src/acme-storefront"}, {ID: "b", Status: "exited", LocalFolder: "/Users/dev/src/docs"}}
	fixtures = append(fixtures, fixture{"fleet", m.width, m.height, m.View()})
	m = consoleFixture(100, 30)
	m.topSnap = statsSnapshot{SchemaVersion: 1, Engine: "colima", Guest: statsGuest{Label: "colima", CPUs: 4, MemoryBytes: 8 * 1024 * 1024 * 1024}, Containers: []statsBox{{ID: "app", Name: "app-1", Service: "app", CPUPct: 3.2, MemUsedBytes: 412 * 1024 * 1024, MemLimitBytes: 8 * 1024 * 1024 * 1024, NetRxBytes: 1024 * 1024, NetTxBytes: 512 * 1024}, {ID: "db", Name: "db-1", Service: "db", CPUPct: 0.8, MemUsedBytes: 82 * 1024 * 1024, MemLimitBytes: 8 * 1024 * 1024 * 1024}}}
	fixtures = append(fixtures, fixture{"top", m.width, m.height, m.topView()})
	m.net = netReport{Networks: []netRow{{Name: "acme_default", Kind: "managed", Reason: "compose-managed", Present: true}, {Name: "shared-dev", Kind: "external", Reason: "missing", Creatable: true}}, MissingCreatable: []string{"shared-dev"}}
	fixtures = append(fixtures, fixture{"nets", m.width, m.height, m.netView()})
	m.hostBlock = true
	m.host = hostReport{Code: "docker_engine_stopped", Summary: "Docker engine is not running", EngineHint: "colima", NextID: "start_colima", NextCommand: "colima start", NextApply: "colima_start", ApplyAllowed: true}
	fixtures = append(fixtures, fixture{"recover", m.width, m.height, m.hostView()})
	m = consoleFixture(120, 36)
	m.more = true
	fixtures = append(fixtures, fixture{"help-wide", m.width, m.height, m.View()})
	data, err := json.MarshalIndent(fixtures, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "fixtures.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestConsoleFooterEnterAndKeyboardOnlyConfirm(t *testing.T) {
	m := consoleFixture(80, 24)
	var enter button
	for _, b := range m.consoleLayout().buttons {
		if b.key == "enter" {
			enter = b
		}
	}
	got, cmd := m.handleClick(enter.x0, enter.y0)
	next := got.(model)
	if cmd == nil || next.pending != "stack:1" {
		t.Fatal("Enter shortcut click must shell into selected service")
	}
	m.confirm = "rm"
	for _, b := range m.consoleLayout().buttons {
		if b.key == "y" {
			got, cmd = m.handleClick(b.x0, b.y0)
			if cmd != nil || got.(model).confirm != "" {
				t.Fatal("a mouse click must not confirm removal")
			}
		}
	}
	m = consoleFixture(80, 24)
	m.loaded = false
	m.load = loadPending
	for _, b := range m.consoleLayout().buttons {
		if (b.key == "u" || b.key == "enter") && !b.disabled {
			t.Fatalf("%s must be disabled while checking", b.key)
		}
	}
}

func TestConsoleFeedbackSemantics(t *testing.T) {
	m := consoleFixture(80, 24)
	refused, _ := m.refuse("restart unavailable for the labeled app — use u/s")
	if !strings.HasPrefix(ansi.Strip(refused.boardFeedback()), "! ") {
		t.Fatal("unavailable action must use warning marker")
	}
	failed := m.withErr("start failed — inspect command output")
	if !strings.HasPrefix(ansi.Strip(failed.boardFeedback()), "✗ ") {
		t.Fatal("failure must use failure marker")
	}
	returned := m.withStatus("back from shell")
	if strings.Contains(ansi.Strip(returned.boardFeedback()), "✓") {
		t.Fatal("normal return must not imply command success")
	}
	returned.feedbackSuccess = true
	if !strings.HasPrefix(ansi.Strip(returned.boardFeedback()), "✓ ") {
		t.Fatal("known success must use success marker")
	}
}

func TestConsoleRecoveryMouseDoesNotHitHiddenBoard(t *testing.T) {
	m := consoleFixture(80, 24)
	frame := m.consoleLayout()
	m.hostBlock = true
	for _, b := range frame.buttons {
		got, cmd := m.handleClick(b.x0, b.y0)
		if cmd != nil || got.(model).leaving != "" {
			t.Fatal("recover mouse hit hidden board control")
		}
	}
}
