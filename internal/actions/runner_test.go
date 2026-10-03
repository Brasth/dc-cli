package actions

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDCExec installs a dc-exec stand-in that records argv (one per line)
// and exits with $FAKE_RC. Returns the record file.
func fakeDCExec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	rec := filepath.Join(dir, "argv")
	script := filepath.Join(dir, "dc-exec")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done >\"" + rec + "\"\necho child-out\necho child-err >&2\nexit ${FAKE_RC:-0}\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	old := DCExecPath
	t.Cleanup(func() { DCExecPath = old })
	DCExecPath = func() string { return script }
	return rec
}

func TestArgsNoShellAndNoStart(t *testing.T) {
	got := Args("/w/app", Action{Argv: []string{"sh", "-c", "echo $HOME; rm x"}, Service: "db"})
	want := []string{"--no-start", "--service", "db", "/w/app", "--", "sh", "-c", "echo $HOME; rm x"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("got %q", got)
	}
	app := Args("/w/app", Action{Argv: []string{"make", "a b"}})
	if strings.Join(app, "\x00") != strings.Join([]string{"--no-start", "/w/app", "--", "make", "a b"}, "\x00") {
		t.Fatalf("app=%q", app)
	}
	for _, a := range append(got, app...) {
		if a == "--id" || a == "--restart" {
			t.Fatal("actions must never target --id or --restart")
		}
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != 0 {
		t.Fatal("nil")
	}
	err := exec.Command("sh", "-c", "exit 7").Run()
	if ExitCode(err) != 7 {
		t.Fatalf("got %d", ExitCode(err))
	}
	err = exec.Command("sh", "-c", "kill -INT $$").Run()
	if ExitCode(err) != 130 {
		t.Fatalf("signal → 128+n, got %d", ExitCode(err))
	}
	if ExitCode(errors.New("no such file")) != 1 {
		t.Fatal("start failure → 1")
	}
}

func TestCommandQuotingReachesChildExactly(t *testing.T) {
	rec := fakeDCExec(t)
	argv := []string{"printf", "%s|", "a b", "it's", "$(touch pwned)", "*"}
	if err := Command("/w/app", Action{Argv: argv}).Run(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(rec)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	want := append([]string{"--no-start", "/w/app", "--"}, argv...)
	if strings.Join(lines, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("child argv=%q want %q", lines, want)
	}
	if _, err := os.Stat("pwned"); err == nil {
		t.Fatal("argv must never pass through a shell")
	}
}
