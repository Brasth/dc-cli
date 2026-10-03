package localstate

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStateAndConfigDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if d, _ := StateDir(); d != filepath.Join(home, ".local", "state", "dc-cli") {
		t.Fatalf("state=%s", d)
	}
	if d, _ := ConfigDir(); d != filepath.Join(home, ".config", "dc-cli") {
		t.Fatalf("config=%s", d)
	}
	t.Setenv("XDG_STATE_HOME", "/x/state")
	t.Setenv("XDG_CONFIG_HOME", "/x/config")
	if d, _ := StateDir(); d != "/x/state/dc-cli" {
		t.Fatalf("state=%s", d)
	}
	if d, _ := ConfigDir(); d != "/x/config/dc-cli" {
		t.Fatalf("config=%s", d)
	}
	t.Setenv("XDG_STATE_HOME", "relative/state")
	if d, _ := StateDir(); d != filepath.Join(home, ".local", "state", "dc-cli") {
		t.Fatalf("relative XDG must be ignored: %s", d)
	}
}

func TestAtomicWriteReplacesAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "f.json")
	if err := AtomicWrite(p, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(p, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "two" {
		t.Fatalf("got %q", b)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm=%v", st.Mode().Perm())
	}
	ents, _ := os.ReadDir(filepath.Dir(p))
	if len(ents) != 1 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

func TestAtomicWriteUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := AtomicWrite(filepath.Join(dir, "f"), []byte("x"), 0o600); err == nil {
		t.Fatal("unwritable dir must error")
	}
}

func TestWithLockSerializes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "counter")
	var wg sync.WaitGroup
	inside := 0
	maxInside := 0
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithLock(p, 5*time.Second, func() error {
				mu.Lock()
				inside++
				if inside > maxInside {
					maxInside = inside
				}
				mu.Unlock()
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				inside--
				mu.Unlock()
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if maxInside != 1 {
		t.Fatalf("lock not exclusive: %d holders at once", maxInside)
	}
}

func TestWithLockTimeout(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = WithLock(p, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	err := WithLock(p, 50*time.Millisecond, func() error { return nil })
	close(release)
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("want ErrLockTimeout, got %v", err)
	}
}
