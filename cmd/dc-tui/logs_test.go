package main

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestColorizeApacheAccessLine(t *testing.T) {
	line := `172.21.0.1 - - [16/Aug/2026:09:37:28 +0000] "GET /wp-json/wp/v2/users/me HTTP/1.1" 200 3037 "http://localhost:8890/wp-admin/" "Mozilla/5.0"`
	got := colorizeLogLine(line)
	plain := ansi.Strip(got)
	if plain != line {
		t.Fatalf("colorize must keep text, got %q", plain)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatal("expected ANSI on apache line")
	}
	if got == line {
		t.Fatal("line should be styled")
	}
}

func TestColorizeStatusClasses(t *testing.T) {
	cases := []struct {
		code string
	}{
		{"200"},
		{"302"},
		{"404"},
		{"500"},
	}
	for _, c := range cases {
		line := `"GET /x HTTP/1.1" ` + c.code + ` 1`
		got := colorizeLogLine(line)
		if ansi.Strip(got) != line {
			t.Fatalf("%s stripped mismatch", c.code)
		}
		if !strings.Contains(got, "\x1b[") {
			t.Fatalf("%s expected color", c.code)
		}
	}
}

func TestColorizeLogLevels(t *testing.T) {
	got := colorizeLogLine("2026-08-16T09:37:28Z ERROR boom")
	if !strings.Contains(got, "\x1b[") {
		t.Fatal("expected level color")
	}
	if ansi.Strip(got) != "2026-08-16T09:37:28Z ERROR boom" {
		t.Fatalf("strip=%q", ansi.Strip(got))
	}
}

func TestOpenLogsUsesStackCursor(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close(); _ = pr.Close() })
	old := startLogFollow
	t.Cleanup(func() { startLogFollow = old })
	var saw string
	startLogFollow = func(id string) (io.ReadCloser, func(), error) {
		saw = id
		return pr, func() { _ = pw.Close() }, nil
	}
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		width:      80,
		height:     24,
		cursor:     1,
		rows:       []container{{ID: "app1", Name: "app-1"}},
		stack: []stackSvc{
			{ID: "app1", Name: "app-1", Service: "app"},
			{ID: "db1", Name: "db-1", Service: "db"},
		},
	}
	got, cmd := m.handleKey("l")
	mm := got.(model)
	if saw != "db1" {
		t.Fatalf("id=%s want db1", saw)
	}
	if mm.logName != "db-1" {
		t.Fatalf("name=%s", mm.logName)
	}
	if mm.leaving != "" {
		t.Fatalf("logs must stay on the board, leaving=%q", mm.leaving)
	}
	if !mm.logOpen {
		t.Fatal("logOpen")
	}
	if cmd == nil {
		t.Fatal("expected follow cmd")
	}
}

func TestOpenLogsEmptyStackFallsBackToRows0(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close(); _ = pr.Close() })
	old := startLogFollow
	t.Cleanup(func() { startLogFollow = old })
	startLogFollow = func(id string) (io.ReadCloser, func(), error) {
		if id != "abc123" {
			t.Fatalf("id=%s", id)
		}
		return pr, func() { _ = pw.Close() }, nil
	}
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		width:      80,
		height:     24,
		cursor:     3,
		rows:       []container{{ID: "abc123", Name: "app-1"}},
	}
	got, _ := m.handleKey("l")
	mm := got.(model)
	if !mm.logOpen || mm.logID != "abc123" {
		t.Fatalf("fallback logID=%q open=%v", mm.logID, mm.logOpen)
	}
}

func TestFleetLogsRefuse(t *testing.T) {
	old := startLogFollow
	t.Cleanup(func() { startLogFollow = old })
	startLogFollow = func(id string) (io.ReadCloser, func(), error) {
		t.Fatalf("fleet must not follow logs, id=%s", id)
		return nil, nil, nil
	}
	m := model{
		fleet:      true,
		hoverStack: -1,
		rows:       []container{{ID: "app1", Name: "app-1"}},
		stack:      []stackSvc{{ID: "db1", Name: "db-1", Service: "db"}},
	}
	got, cmd := m.handleKey("l")
	mm := got.(model)
	if mm.logOpen {
		t.Fatal("fleet l must not open logs")
	}
	if cmd != nil {
		t.Fatal("fleet l must not start follow")
	}
	if mm.status == "" {
		t.Fatal("fleet l must explain itself")
	}
}

func TestLogsButtonEnabledFromStackCursor(t *testing.T) {
	m := model{
		hoverStack: -1,
		stack:      []stackSvc{{ID: "db1", Name: "db-1", Service: "db"}},
	}
	disabled := true
	found := false
	for _, g := range m.buttonGroups() {
		for _, s := range g {
			if s.key == "l" {
				found = true
				disabled = s.disabled
			}
		}
	}
	if !found {
		t.Fatal("logs button missing")
	}
	if disabled {
		t.Fatal("logs should enable from stack cursor id, not only rows[0]")
	}
}

func TestOpenLogsStaysOnBoard(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close(); _ = pr.Close() })
	old := startLogFollow
	t.Cleanup(func() { startLogFollow = old })
	startLogFollow = func(id string) (io.ReadCloser, func(), error) {
		if id != "abc123" {
			t.Fatalf("id=%s", id)
		}
		return pr, func() { _ = pw.Close() }, nil
	}
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		width:      80,
		height:     24,
		rows:       []container{{ID: "abc123", Name: "app-1"}},
	}
	got, cmd := m.handleKey("l")
	mm := got.(model)
	if mm.leaving != "" {
		t.Fatalf("logs must stay on the board, leaving=%q", mm.leaving)
	}
	if !mm.logOpen {
		t.Fatal("logOpen")
	}
	if cmd == nil {
		t.Fatal("expected follow cmd")
	}
	got, _ = mm.handleKey("q")
	mm = got.(model)
	if mm.logOpen {
		t.Fatal("q should close logs")
	}
	if mm.status != "back from logs" {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestLogsViewHighlights(t *testing.T) {
	m := model{
		logOpen:   true,
		logID:     "abc123deadbeef",
		logName:   "app",
		logFollow: true,
		width:     100,
		height:    20,
		logLines:  []string{`172.21.0.1 - - [16/Aug/2026:09:37:28 +0000] "POST /wp-admin/admin-ajax.php HTTP/1.1" 500 581`},
	}
	s := m.logsView()
	if !strings.Contains(ansi.Strip(s), "logs") {
		t.Fatalf("header missing: %s", ansi.Strip(s))
	}
	if !strings.Contains(s, "\x1b[") {
		t.Fatal("logs view should colorize lines")
	}
}
