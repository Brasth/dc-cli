package actions

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func runCLI(t *testing.T, tty bool, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, IO{Stdin: strings.NewReader(stdin), Stdout: &out, Stderr: &errb, IsTTY: tty})
	return code, out.String(), errb.String()
}

func TestCLIUsage(t *testing.T) {
	if c, _, _ := runCLI(t, false, ""); c != ExitUsage {
		t.Fatalf("no args → 2, got %d", c)
	}
	if c, out, _ := runCLI(t, false, "", "--help"); c != 0 || !strings.Contains(out, "dc-actions list") {
		t.Fatalf("help: %d", c)
	}
	for _, args := range [][]string{{"bogus"}, {"list", "--nope"}, {"run"}, {"list", "a", "b"}, {"trust", "--json"}} {
		if c, _, _ := runCLI(t, false, "", args...); c != ExitUsage {
			t.Errorf("%v → want 2, got %d", args, c)
		}
	}
	if c, _, _ := runCLI(t, false, "", "list", "/definitely/missing/dir"); c != ExitUnavailable {
		t.Fatalf("missing workspace → 1, got %d", c)
	}
}

func TestCLIListJSON(t *testing.T) {
	ws := isolate(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"test","label":"Mine","argv":["make","t"]}]}`)
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"test","label":"Team","argv":["x"]},{"id":"psql","label":"DB","argv":["psql"],"service":"db"}]}`)
	code, out, errs := runCLI(t, false, "", "list", ws, "--json")
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errs)
	}
	var doc struct {
		SchemaVersion int    `json:"schemaVersion"`
		Workspace     string `json:"workspace"`
		Actions       []map[string]any
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || doc.Workspace != ws || len(doc.Actions) != 2 {
		t.Fatalf("doc=%+v", doc)
	}
	a0, a1 := doc.Actions[0], doc.Actions[1]
	if a0["id"] != "test" || a0["source"] != "personal" || a0["enabled"] != true || a0["service"] != "" {
		t.Fatalf("a0=%v", a0)
	}
	if a1["id"] != "psql" || a1["source"] != "shared" || a1["enabled"] != false || a1["service"] != "db" {
		t.Fatalf("a1=%v", a1)
	}
	for _, k := range []string{"id", "label", "argv", "service", "source", "enabled"} {
		if _, ok := a0[k]; !ok {
			t.Fatalf("missing key %s", k)
		}
	}
	if !strings.Contains(errs, "disabled") {
		t.Fatalf("stderr should note disabled shared: %s", errs)
	}
	// Human table.
	if code, out, _ := runCLI(t, false, "", "list", ws); code != 0 || !strings.Contains(out, "psql") || !strings.Contains(out, "disabled") {
		t.Fatalf("table code=%d out=%s", code, out)
	}
}

func TestCLIListConfigErrorExit2ButShowsValid(t *testing.T) {
	ws := isolate(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"mine","label":"Mine","argv":["true"]}]}`)
	writeShared(t, ws, `{"schemaVersion":3}`)
	code, out, errs := runCLI(t, false, "", "list", ws, "--json")
	if code != ExitUsage || !strings.Contains(out, `"mine"`) || !strings.Contains(errs, "shared actions ignored") {
		t.Fatalf("code=%d out=%s err=%s", code, out, errs)
	}
}

func TestCLIRunPropagatesExitAndKeepsOutput(t *testing.T) {
	ws := isolate(t)
	rec := fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"t","label":"T","argv":["make","a b"],"service":"app2"}]}`)
	t.Setenv("FAKE_RC", "42")
	code, out, errs := runCLI(t, false, "", "run", "t", ws)
	if code != 42 {
		t.Fatalf("exit must propagate, got %d (%s)", code, errs)
	}
	if out != "child-out\n" {
		t.Fatalf("child stdout must be untouched, got %q", out)
	}
	if !strings.Contains(errs, "child-err") || !strings.Contains(errs, "exited 42") {
		t.Fatalf("stderr=%q", errs)
	}
	b, _ := os.ReadFile(rec)
	if string(b) != "--no-start\n--service\napp2\n"+ws+"\n--\nmake\na b\n" {
		t.Fatalf("argv=%q", b)
	}
}

func TestCLIRunPersonalWhenSharedInvalidOrDisabled(t *testing.T) {
	ws := isolate(t)
	fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"mine","label":"M","argv":["true"]}]}`)
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"team","label":"T","argv":["x"]}]}`)
	if code, _, _ := runCLI(t, false, "", "run", "mine", ws); code != 0 {
		t.Fatalf("personal with disabled shared → 0, got %d", code)
	}
	if code, _, errs := runCLI(t, false, "", "run", "team", ws); code != ExitUnavailable || !strings.Contains(errs, "trust") {
		t.Fatalf("disabled shared → 1, got %d %s", code, errs)
	}
	writeShared(t, ws, `{oops`)
	if code, _, _ := runCLI(t, false, "", "run", "mine", ws); code != 0 {
		t.Fatalf("personal with invalid shared → 0, got %d", code)
	}
	if code, _, _ := runCLI(t, false, "", "run", "team", ws); code != ExitUsage {
		t.Fatalf("shared id with invalid shared → 2, got %d", code)
	}
	if code, _, _ := runCLI(t, false, "", "run", "ghost", t.TempDir()); code != ExitUnavailable {
		t.Fatalf("missing id → 1, got %d", code)
	}
}

func TestCLITrustFlow(t *testing.T) {
	ws := isolate(t)
	fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"lint","label":"Mine","argv":["true"]}]}`)
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"lint","label":"Team lint","argv":["make","lint"]},{"id":"db","label":"DB","argv":["psql","-c","x y"],"service":"db"}]}`)

	// Non-TTY without --yes: preview, refuse, nothing trusted.
	code, out, errs := runCLI(t, false, "", "trust", ws)
	if code != ExitUsage || !strings.Contains(errs, "--yes") {
		t.Fatalf("code=%d err=%s", code, errs)
	}
	for _, want := range []string{"lint — Team lint", "overridden", "argv:   make lint", "target: service db", "'x y'", "modify their data"} {
		if !strings.Contains(out, want) {
			t.Fatalf("preview missing %q:\n%s", want, out)
		}
	}
	if code, _, _ := runCLI(t, false, "", "run", "db", ws); code != ExitUnavailable {
		t.Fatal("refused trust must leave shared disabled")
	}
	// TTY answering no.
	if code, _, _ := runCLI(t, true, "n\n", "trust", ws); code != ExitUnavailable {
		t.Fatalf("declined → 1, got %d", code)
	}
	// TTY yes.
	if code, _, _ := runCLI(t, true, "y\n", "trust", ws); code != 0 {
		t.Fatalf("approved → 0, got %d", code)
	}
	if code, _, _ := runCLI(t, false, "", "run", "db", ws); code != 0 {
		t.Fatal("trusted shared must run")
	}
	// Untrust.
	if code, _, _ := runCLI(t, false, "", "untrust", ws); code != 0 {
		t.Fatal("untrust")
	}
	if code, _, _ := runCLI(t, false, "", "run", "db", ws); code != ExitUnavailable {
		t.Fatal("untrusted must be disabled")
	}
	// --yes works without a TTY.
	if code, _, _ := runCLI(t, false, "", "trust", ws, "--yes"); code != 0 {
		t.Fatal("--yes")
	}
	// Edit → revoked.
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"db","label":"DB","argv":["psql"],"service":"db"}]}`)
	if code, _, _ := runCLI(t, false, "", "run", "db", ws); code != ExitUnavailable {
		t.Fatal("changed shared must be disabled again")
	}
}

func TestCLITrustMissingAndInvalid(t *testing.T) {
	ws := isolate(t)
	if code, _, _ := runCLI(t, false, "", "trust", ws, "--yes"); code != ExitUnavailable {
		t.Fatalf("no shared file → 1, got %d", code)
	}
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"x","label":"X","argv":[]}]}`)
	if code, _, _ := runCLI(t, false, "", "trust", ws, "--yes"); code != ExitUsage {
		t.Fatalf("invalid shared → 2, got %d", code)
	}
}

func TestCLIOpaqueIDWithSpaceRuns(t *testing.T) {
	ws := isolate(t)
	rec := fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"run all tests","label":"T","argv":["make"]}]}`)
	if code, _, errs := runCLI(t, false, "", "run", "run all tests", ws); code != 0 {
		t.Fatalf("opaque id must run: %d %s", code, errs)
	}
	if b, _ := os.ReadFile(rec); !strings.Contains(string(b), "make") {
		t.Fatal("child not run")
	}
}

func TestCLIRejectsNULBeforeExec(t *testing.T) {
	ws := isolate(t)
	rec := fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"x","label":"X","argv":["echo","a\u0000b"]}]}`)
	if code, _, _ := runCLI(t, false, "", "run", "x", ws); code != ExitUsage {
		t.Fatalf("NUL argv → config error 2, got %d", code)
	}
	if _, err := os.Stat(rec); err == nil {
		t.Fatal("nothing may execute")
	}
}

func TestCLIDoubleDashEndsOptions(t *testing.T) {
	ws := isolate(t)
	rec := fakeDCExec(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"-x","label":"Dash","argv":["make","dash"]}]}`)
	// Without -- an id that starts with - is an unknown flag (usage), nothing runs.
	if code, _, _ := runCLI(t, false, "", "run", "-x", ws); code != ExitUsage {
		t.Fatalf("run -x → want usage 2, got %d", code)
	}
	if _, err := os.Stat(rec); err == nil {
		t.Fatal("nothing may execute on a usage error")
	}
	if code, _, errs := runCLI(t, false, "", "run", "--", "-x", ws); code != 0 {
		t.Fatalf("run -- -x must run: %d %s", code, errs)
	}
	if b, _ := os.ReadFile(rec); !strings.Contains(string(b), "dash") {
		t.Fatalf("child argv=%q", b)
	}
	// Flags before -- still work; after -- everything is positional.
	if code, out, errs := runCLI(t, false, "", "list", "--json", "--", ws); code != 0 || !strings.Contains(out, `"-x"`) {
		t.Fatalf("list --json -- ws: %d %s %s", code, out, errs)
	}
	if code, _, _ := runCLI(t, false, "", "list", "--", ws, "--json"); code != ExitUsage {
		t.Fatalf("--json after -- is a positional → too many args, got %d", code)
	}
	if code, _, _ := runCLI(t, false, "", "run", "--"); code != ExitUsage {
		t.Fatalf("run -- without id → usage, got %d", code)
	}
}
