package actions

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points config/state at temp dirs and returns a canonical workspace.
func isolate(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	ws, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func writePersonal(t *testing.T, ws, body string) {
	t.Helper()
	p, err := PersonalPath(ws)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeShared(t *testing.T, ws, body string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Join(ws, ".dc"), 0o755)
	if err := os.WriteFile(SharedPath(ws), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseValid(t *testing.T) {
	as, err := Parse("f", []byte(`{"schemaVersion":1,"actions":[
		{"id":"test","label":"Run tests","argv":["go","test","./..."]},
		{"id":"db.shell","label":"DB","argv":["psql","-c","select 'a b'"],"service":"db"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 2 || as[1].Service != "db" || as[1].Argv[2] != "select 'a b'" {
		t.Fatalf("as=%+v", as)
	}
	if as2, err := Parse("f", []byte(`{"schemaVersion":1,"actions":[]}`)); err != nil || len(as2) != 0 {
		t.Fatalf("empty list must be valid: %v", err)
	}
}

func TestParseRejectsWholeFile(t *testing.T) {
	bad := map[string]string{
		"not json":         `{`,
		"no schema":        `{"actions":[]}`,
		"schema 2":         `{"schemaVersion":2,"actions":[]}`,
		"no actions":       `{"schemaVersion":1}`,
		"unknown field":    `{"schemaVersion":1,"actions":[],"x":1}`,
		"unknown in act":   `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"],"shell":true}]}`,
		"empty id":         `{"schemaVersion":1,"actions":[{"id":"","label":"A","argv":["x"]}]}`,
		"NUL in argv":      `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["echo","a\u0000b"]}]}`,
		"NUL in argv0":     `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["ech\u0000o"]}]}`,
		"null in argv":     `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["echo",null]}]}`,
		"null argv0":       `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":[null]}]}`,
		"argv null":        `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":null}]}`,
		"service null":     `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"],"service":null}]}`,
		"service number":   `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"],"service":5}]}`,
		"null after good":  `{"schemaVersion":1,"actions":[{"id":"ok","label":"OK","argv":["x"]},{"id":"b","label":"B","argv":["y",null]}]}`,
		"NUL in service":   `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"],"service":"d\u0000b"}]}`,
		"dup id":           `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"]},{"id":"a","label":"B","argv":["y"]}]}`,
		"empty label":      `{"schemaVersion":1,"actions":[{"id":"a","label":"","argv":["x"]}]}`,
		"empty argv":       `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":[]}]}`,
		"argv0 empty":      `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":[""]}]}`,
		"argv not strings": `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x",1]}]}`,
		"argv string":      `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":"make test"}]}`,
		"empty service":    `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":["x"],"service":""}]}`,
		"trailing":         `{"schemaVersion":1,"actions":[]} {}`,
		"good then bad":    `{"schemaVersion":1,"actions":[{"id":"ok","label":"OK","argv":["x"]},{"id":"","label":"B","argv":["y"]}]}`,
	}
	for name, body := range bad {
		if as, err := Parse("f", []byte(body)); err == nil || !IsConfigError(err) || as != nil {
			t.Errorf("%s: want whole-file ConfigError, got as=%v err=%v", name, as, err)
		}
	}
}

func TestParseOpaqueIDs(t *testing.T) {
	as, err := Parse("f", []byte(`{"schemaVersion":1,"actions":[{"id":"run tests / ../x","label":"A","argv":["x"]},{"id":"é-1","label":"B","argv":["y"]}]}`))
	if err != nil || len(as) != 2 || as[0].ID != "run tests / ../x" {
		t.Fatalf("opaque ids must be accepted: %v %+v", err, as)
	}
}

func TestPersonalPathIsHashOfCanonical(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	p, err := PersonalPath("/w/app")
	if err != nil {
		t.Fatal(err)
	}
	want := "/cfg/dc-cli/actions/" + HashContent([]byte("/w/app")) + ".json"
	if p != want {
		t.Fatalf("p=%s want %s", p, want)
	}
}

func TestPrecedencePersonalOverridesShared(t *testing.T) {
	ws := isolate(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"test","label":"Mine","argv":["make","t"]}]}`)
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"test","label":"Team","argv":["rm","-rf","/"]},{"id":"lint","label":"Lint","argv":["make","lint"]}]}`)
	set := Load(ws, nil)
	es := set.Entries()
	if len(es) != 2 || es[0].Label != "Mine" || es[0].Source != SourcePersonal || !es[0].Enabled {
		t.Fatalf("es=%+v", es)
	}
	if es[1].ID != "lint" || es[1].Enabled {
		t.Fatalf("untrusted shared must be disabled: %+v", es[1])
	}
	if !set.Overridden("test") || set.Overridden("lint") {
		t.Fatal("override detection wrong")
	}
	e, err := set.Find("test")
	if err != nil || e.Label != "Mine" {
		t.Fatalf("find personal: %+v %v", e, err)
	}
	if _, err := set.Find("lint"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled shared: %v", err)
	}
	if _, err := set.Find("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestInvalidSharedNeverBlocksPersonal(t *testing.T) {
	ws := isolate(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"mine","label":"Mine","argv":["true"]}]}`)
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"x"}]}`)
	set := Load(ws, nil)
	if set.Shared.Err == nil {
		t.Fatal("shared must be invalid")
	}
	if e, err := set.Find("mine"); err != nil || !e.Enabled {
		t.Fatalf("valid personal must run: %v", err)
	}
	if _, err := set.Find("x"); !IsConfigError(err) {
		t.Fatalf("shared id lookup must report config error, got %v", err)
	}
	if es := set.Entries(); len(es) != 1 {
		t.Fatalf("es=%+v", es)
	}
}

func TestInvalidPersonalIsConfigError(t *testing.T) {
	ws := isolate(t)
	writePersonal(t, ws, `{"schemaVersion":1,"actions":[{"id":"a","label":"A","argv":"x"}]}`)
	set := Load(ws, nil)
	if !IsConfigError(set.PersonalErr) {
		t.Fatalf("err=%v", set.PersonalErr)
	}
	if _, err := set.Find("a"); !IsConfigError(err) {
		t.Fatalf("find must surface personal config error: %v", err)
	}
}

func TestFormatArgvAndTarget(t *testing.T) {
	got := FormatArgv([]string{"echo", "a b", "it's", "", "$HOME", "x=1", "l1\nl2\x1b[2J"})
	want := `echo 'a b' 'it'\''s' '' '$HOME' x=1 "l1\nl2\x1b[2J"`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if Target(Action{Service: "db"}) != "service db" || Target(Action{}) != "app" {
		t.Fatal("target")
	}
	if !strings.Contains(Usage, "can change data") {
		t.Fatal("usage must warn that actions can change data")
	}
}
