// Package localstate holds dc-cli's per-user files: XDG locations, atomic
// writes, and a short cross-process lock for read-modify-write updates.
package localstate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ErrLockTimeout means another dc-cli process held the lock past the wait.
var ErrLockTimeout = errors.New("state file is locked by another dc-cli process")

// StateDir is ${XDG_STATE_HOME:-~/.local/state}/dc-cli.
func StateDir() (string, error) {
	return xdgDir("XDG_STATE_HOME", filepath.Join(".local", "state"))
}

// ConfigDir is ${XDG_CONFIG_HOME:-~/.config}/dc-cli.
func ConfigDir() (string, error) {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

// xdgDir honours an absolute XDG variable (relative values are ignored, per
// the XDG spec) and falls back to a path under $HOME.
func xdgDir(env, homeRel string) (string, error) {
	if v := os.Getenv(env); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, "dc-cli"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no home directory for %s: %w", env, err)
	}
	return filepath.Join(home, homeRel, "dc-cli"), nil
}

// AtomicWrite replaces path with data: temp file in the same directory,
// fsync, rename. Readers see the old or the new file, never a partial one.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// WithLock runs fn while holding an exclusive flock on path+".lock". The
// kernel drops the lock if the process dies, so a crash never wedges it.
// Waits at most wait; then ErrLockTimeout.
func WithLock(path string, wait time.Duration, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	deadline := time.Now().Add(wait)
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) && !errors.Is(err, syscall.EINTR) {
			return err
		}
		if time.Now().After(deadline) {
			return ErrLockTimeout
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}
