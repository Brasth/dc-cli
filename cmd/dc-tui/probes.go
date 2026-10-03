package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Read-only probe budget. User actions (start, shell, stay commands) never
// get a deadline and are never cancelled by the board.
var (
	probeTimeout      = 5 * time.Second
	essentialDeadline = 10 * time.Second
)

var errProbeCancelled = errors.New("cancelled")

// probeSession scopes read-only probes to one board context (workspace +
// fleet). Closing it kills every probe tree still running for that context.
// Each discovery generation gets a child context so a newer reload cancels
// the older one's leftovers.
type probeSession struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	genCtx    context.Context
	genCancel context.CancelFunc
}

func newProbeSession() *probeSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &probeSession{ctx: ctx, cancel: cancel}
}

// context is the session context; nil sessions (tests) never cancel.
func (s *probeSession) context() context.Context {
	if s == nil {
		return context.Background()
	}
	return s.ctx
}

// beginGeneration cancels the previous generation's probes and returns a
// fresh child context for the new one.
func (s *probeSession) beginGeneration() context.Context {
	if s == nil {
		return context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.genCancel != nil {
		s.genCancel()
	}
	s.genCtx, s.genCancel = context.WithCancel(s.ctx)
	return s.genCtx
}

// generation is the current generation context (session context before the
// first discovery).
func (s *probeSession) generation() context.Context {
	if s == nil {
		return context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.genCtx == nil {
		return s.ctx
	}
	return s.genCtx
}

func (s *probeSession) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.genCancel != nil {
		s.genCancel()
	}
	s.cancel()
}

// runProbe runs one read-only command and returns stdout. Tests replace it;
// replacements must honour ctx.
var runProbe = defaultRunProbe

func defaultRunProbe(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	setProbeProcAttr(cmd)
	return cmd.Output()
}

// probe applies the per-probe timeout and turns context errors into readable ones.
func probe(ctx context.Context, name string, args ...string) ([]byte, error) {
	return probeVia(ctx, name, func(pctx context.Context) ([]byte, error) {
		return runProbe(pctx, name, args...)
	})
}

// probeVia applies the probe budget and error mapping to any read-only run.
func probeVia(ctx context.Context, name string, run func(context.Context) ([]byte, error)) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, probeCtxErr(name, ctx, err)
	}
	pctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	out, err := run(pctx)
	if err != nil {
		if cerr := pctx.Err(); cerr != nil {
			return out, probeCtxErr(name, ctx, cerr)
		}
		return out, err
	}
	return out, nil
}

func probeCtxErr(name string, parent context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(parent.Err(), context.Canceled) {
		return errProbeCancelled
	}
	if errors.Is(parent.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s: discovery deadline (%s) passed", name, essentialDeadline)
	}
	return fmt.Errorf("%s: timed out after %s", name, probeTimeout)
}

// probeEngine names the engine the docker CLI targets right now:
// DOCKER_HOST wins, else the current context. "" when unknown.
var probeEngine = defaultProbeEngine

func defaultProbeEngine(ctx context.Context) string {
	if h := strings.TrimSpace(os.Getenv("DOCKER_HOST")); h != "" {
		return "host:" + h
	}
	out, err := probe(ctx, "docker", "context", "show")
	if err != nil {
		return ""
	}
	if name := strings.TrimSpace(string(out)); name != "" {
		return "context:" + name
	}
	return ""
}
