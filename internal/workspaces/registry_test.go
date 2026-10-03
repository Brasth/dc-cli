package workspaces

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Path: filepath.Join(t.TempDir(), "state", "dc-cli", "workspaces.json"), LockWait: 10 * time.Second}
}

// clock returns strictly increasing times.
func clock() func() time.Time {
	base := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	n := 0
	var mu sync.Mutex
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		n++
		return base.Add(time.Duration(n) * time.Second)
	}
}

func TestDefaultPathXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/s")
	if p, _ := DefaultPath(); p != "/s/dc-cli/workspaces.json" {
		t.Fatalf("p=%s", p)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	if p, _ := DefaultPath(); p != filepath.Join(home, ".local", "state", "dc-cli", "workspaces.json") {
		t.Fatalf("p=%s", p)
	}
}

func TestRecordDedupsSymlinks(t *testing.T) {
	s := newStore(t)
	s.Now = clock()
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(real); err != nil {
		t.Fatal(err)
	}
	es, err := s.Record(link + "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 1 {
		t.Fatalf("symlink spelling must dedup, got %+v", es)
	}
	want, _ := filepath.EvalSymlinks(real)
	if es[0].Path != want {
		t.Fatalf("path=%s want canonical %s", es[0].Path, want)
	}
}

func TestRecordMissingFolderStaysAbsolute(t *testing.T) {
	s := newStore(t)
	es, err := s.Record("/definitely/not/here/x")
	if err != nil {
		t.Fatal(err)
	}
	if es[0].Path != "/definitely/not/here/x" || Exists(es[0].Path) {
		t.Fatalf("es=%+v", es)
	}
}

func TestRetention30PlusFavorites(t *testing.T) {
	s := newStore(t)
	s.Now = clock()
	root := t.TempDir()
	var paths []string
	for i := 0; i < 40; i++ {
		p := filepath.Join(root, fmt.Sprintf("p%02d", i))
		paths = append(paths, p)
	}
	// Favorite the three oldest; they must survive retention.
	for i := 0; i < 3; i++ {
		if _, err := s.Record(paths[i]); err != nil {
			t.Fatal(err)
		}
		if _, err := s.SetFavorite(paths[i], true); err != nil {
			t.Fatal(err)
		}
	}
	for i := 3; i < 40; i++ {
		if _, err := s.Record(paths[i]); err != nil {
			t.Fatal(err)
		}
	}
	es, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 33 {
		t.Fatalf("want 3 favorites + 30 recent, got %d", len(es))
	}
	for i := 0; i < 3; i++ {
		if !es[i].Favorite {
			t.Fatalf("favorites must sort first: %+v", es[:4])
		}
	}
	// Newest non-favorite first; oldest 7 recents dropped.
	if es[3].Path != paths[39] || es[32].Path != paths[10] {
		t.Fatalf("order/retention wrong: first=%s last=%s", es[3].Path, es[32].Path)
	}
}

func TestSortedFavoritesThenLastOpen(t *testing.T) {
	t0 := time.Unix(100, 0)
	es := Sorted([]Entry{
		{Path: "/a", LastOpen: t0},
		{Path: "/b", LastOpen: t0.Add(2 * time.Second)},
		{Path: "/c", LastOpen: t0.Add(-time.Hour), Favorite: true},
		{Path: "/d", LastOpen: t0.Add(time.Second), Favorite: true},
	})
	var got []string
	for _, e := range es {
		got = append(got, e.Path)
	}
	if strings.Join(got, ",") != "/d,/c,/b,/a" {
		t.Fatalf("order=%v", got)
	}
}

func TestForgetIsStateOnly(t *testing.T) {
	s := newStore(t)
	dir := t.TempDir()
	if _, err := s.Record(dir); err != nil {
		t.Fatal(err)
	}
	es, err := s.Forget(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 0 {
		t.Fatalf("es=%+v", es)
	}
	if !Exists(dir) {
		t.Fatal("forget must not touch the folder")
	}
}

func TestMalformedPreservedNonfatal(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		t.Fatal(err)
	}
	bad := []byte("{not json")
	if err := os.WriteFile(s.Path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	es, err := s.Load()
	var me *MalformedError
	if !errors.As(err, &me) || len(es) != 0 {
		t.Fatalf("want MalformedError, got es=%v err=%v", es, err)
	}
	if _, err := s.Record(t.TempDir()); !errors.As(err, &me) {
		t.Fatalf("record on malformed must refuse, got %v", err)
	}
	b, _ := os.ReadFile(s.Path)
	if string(b) != string(bad) {
		t.Fatalf("malformed file must be preserved, got %q", b)
	}
	for _, body := range []string{"", `{"schemaVersion":0,"workspaces":[]}`, `{"schemaVersion":1,"workspaces":[{"path":"rel"}]}`} {
		_ = os.WriteFile(s.Path, []byte(body), 0o600)
		if _, err := s.Load(); !errors.As(err, &me) {
			t.Fatalf("body %q: want MalformedError, got %v", body, err)
		}
	}
}

func TestNewerSchemaReadOnly(t *testing.T) {
	s := newStore(t)
	_ = os.MkdirAll(filepath.Dir(s.Path), 0o755)
	body := `{"schemaVersion":2,"workspaces":[{"path":"/x","lastOpen":"2026-01-01T00:00:00Z"}]}`
	_ = os.WriteFile(s.Path, []byte(body), 0o600)
	es, err := s.Load()
	if !errors.Is(err, ErrNewerSchema) || len(es) != 1 {
		t.Fatalf("es=%v err=%v", es, err)
	}
	if _, err := s.Record("/y"); !errors.Is(err, ErrNewerSchema) {
		t.Fatalf("newer schema must stay read-only, got %v", err)
	}
	b, _ := os.ReadFile(s.Path)
	if string(b) != body {
		t.Fatal("newer file must not be rewritten")
	}
}

func TestUnwritableNonfatal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "ro", "workspaces.json"), LockWait: time.Second}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(s.Path), 0o755) })
	if _, err := s.Record(t.TempDir()); err == nil {
		t.Fatal("unwritable state dir must return an error")
	}
	if es, err := s.Load(); err != nil || len(es) != 0 {
		t.Fatalf("load must still work: %v %v", es, err)
	}
}

func TestConcurrentGoroutinesNoLostUpdate(t *testing.T) {
	s := newStore(t)
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := s.Record(filepath.Join(root, fmt.Sprintf("w%02d", i))); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	es, err := s.Load()
	if err != nil || len(es) != 20 {
		t.Fatalf("lost updates: %d entries err=%v", len(es), err)
	}
}

// Cross-process: re-exec this test binary as several writers.
func TestConcurrentProcessesNoLostUpdate(t *testing.T) {
	if os.Getenv("DC_REG_CHILD") != "" {
		t.Skip("child")
	}
	s := newStore(t)
	root := t.TempDir()
	var cmds []*exec.Cmd
	for i := 0; i < 6; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=TestRegistryChildWriter")
		cmd.Env = append(os.Environ(), "DC_REG_CHILD=1", "DC_REG_PATH="+s.Path, fmt.Sprintf("DC_REG_PREFIX=%s/p%d", root, i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, cmd)
	}
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatalf("child: %v", err)
		}
	}
	es, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 30 {
		t.Fatalf("6 writers × 5 records: want 30, got %d", len(es))
	}
}

func TestRegistryChildWriter(t *testing.T) {
	if os.Getenv("DC_REG_CHILD") == "" {
		t.Skip("helper for TestConcurrentProcessesNoLostUpdate")
	}
	s := &Store{Path: os.Getenv("DC_REG_PATH"), LockWait: 20 * time.Second}
	for i := 0; i < 5; i++ {
		if _, err := s.Record(fmt.Sprintf("%s-%d", os.Getenv("DC_REG_PREFIX"), i)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLabelsDisambiguateWithParent(t *testing.T) {
	home, _ := os.UserHomeDir()
	es := []Entry{
		{Path: filepath.Join(home, "work", "api")},
		{Path: "/srv/other/api"},
		{Path: "/srv/web"},
	}
	got := Labels(es)
	if got[0] != "api  (~/work)" || got[1] != "api  (/srv/other)" || got[2] != "web" {
		t.Fatalf("labels=%q", got)
	}
}
