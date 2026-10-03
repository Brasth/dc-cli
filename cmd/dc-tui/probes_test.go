package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// treeScript starts a grandchild sleeper, records its pid, then waits.
const treeScript = `sleep 30 & echo $! > "$1"; wait`

func waitPidFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("grandchild pid never written")
	return 0
}

func pidGone(pid int) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestProbeCancelKillsWholeTree(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := probe(ctx, "sh", "-c", treeScript, "sh", pidFile)
		done <- err
	}()
	pid := waitPidFile(t, pidFile)
	cancel()
	select {
	case err := <-done:
		if err != errProbeCancelled {
			t.Fatalf("want cancelled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not return the probe")
	}
	if !pidGone(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("grandchild %d survived cancel", pid)
	}
}

func TestProbeTimeoutKillsWholeTree(t *testing.T) {
	old := probeTimeout
	t.Cleanup(func() { probeTimeout = old })
	probeTimeout = 150 * time.Millisecond
	pidFile := filepath.Join(t.TempDir(), "pid")
	start := time.Now()
	_, err := probe(context.Background(), "sh", "-c", treeScript, "sh", pidFile)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("want timeout, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("timeout not enforced: %s", time.Since(start))
	}
	pid := waitPidFile(t, pidFile)
	if !pidGone(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("grandchild %d survived timeout", pid)
	}
}

func TestProbeAlreadyCancelledDoesNotSpawn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	old := runProbe
	t.Cleanup(func() { runProbe = old })
	runProbe = func(context.Context, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}
	if _, err := probe(ctx, "dc-ls"); err != errProbeCancelled {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("cancelled context must not spawn a probe")
	}
}

func TestProbeSuccessAndExitError(t *testing.T) {
	out, err := probe(context.Background(), "sh", "-c", "printf ok")
	if err != nil || string(out) != "ok" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, err := probe(context.Background(), "sh", "-c", "exit 3"); err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("exit error must pass through, got %v", err)
	}
}

func TestNilProbeSessionIsSafe(t *testing.T) {
	var s *probeSession
	if s.context().Err() != nil || s.beginGeneration().Err() != nil || s.generation().Err() != nil {
		t.Fatal("nil session must give live background contexts")
	}
	s.close()
}

func TestDefaultProbeEngineUsesDockerHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/x.sock")
	if got := defaultProbeEngine(context.Background()); got != "host:unix:///tmp/x.sock" {
		t.Fatalf("engine=%q", got)
	}
}
