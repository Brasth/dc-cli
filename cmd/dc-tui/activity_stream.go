package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// One `docker events` process per board context, pinned to the engine the
// discovery snapshot came from. It runs in its own process group (same helper
// as read-only probes) so closing the stream kills and reaps the whole tree.

// activityEvents are the container actions the timeline keeps.
var activityEvents = []string{"create", "start", "stop", "die", "restart", "destroy", "oom", "health_status"}

// activityLineMax bounds one stream line; longer lines are discarded whole.
const activityLineMax = 64 << 10

// activityStopWait bounds how long closing a stream may wait for the reap.
const activityStopWait = 2 * time.Second

// engineArgs pins a docker CLI call to one engine identity (see probeEngine).
// ok=false: the identity is unknown and nothing may be pinned to it.
func engineArgs(engine string) (args []string, ok bool) {
	switch {
	case strings.HasPrefix(engine, "context:"):
		name := strings.TrimPrefix(engine, "context:")
		if name == "" {
			return nil, false
		}
		return []string{"--context", name}, true
	case strings.HasPrefix(engine, "host:"):
		h := strings.TrimPrefix(engine, "host:")
		if h == "" {
			return nil, false
		}
		return []string{"--host", h}, true
	}
	return nil, false
}

// pinEngineEnv drops DOCKER_HOST / DOCKER_CONTEXT from the child's
// environment so only engineArgs (--context / --host) picks the engine.
func pinEngineEnv(cmd *exec.Cmd) {
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "DOCKER_HOST=") || strings.HasPrefix(kv, "DOCKER_CONTEXT=") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = env
}

func eventsArgs(engine string) ([]string, bool) {
	args, ok := engineArgs(engine)
	if !ok {
		return nil, false
	}
	args = append(args, "events", "--format", "{{json .}}", "--filter", "type=container")
	for _, e := range activityEvents {
		args = append(args, "--filter", "event="+e)
	}
	return args, true
}

// startEventProcess starts the stream and returns its stdout plus a wait
// func. The process must die when ctx is cancelled. Tests replace it.
var startEventProcess = defaultStartEventProcess

var errEngineUnknown = errors.New("engine unknown")

func defaultStartEventProcess(ctx context.Context, engine string) (io.Reader, func() error, error) {
	args, ok := eventsArgs(engine)
	if !ok {
		return nil, nil, errEngineUnknown
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	setProbeProcAttr(cmd)
	pinEngineEnv(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return out, cmd.Wait, nil
}

// activityStream is one running `docker events` process and its reader.
type activityStream struct {
	ctx     context.Context
	cancel  context.CancelFunc
	events  chan activityEvent // closed when the reader stops
	done    chan struct{}      // closed after the process is reaped
	err     error              // valid after done
	started time.Time
	once    sync.Once
}

// openActivityStream starts the process under parent. Malformed lines are
// skipped; the reader never holds more than one line.
func openActivityStream(parent context.Context, engine string) (*activityStream, error) {
	ctx, cancel := context.WithCancel(parent)
	r, wait, err := startEventProcess(ctx, engine)
	if err != nil {
		cancel()
		return nil, err
	}
	s := &activityStream{
		ctx:     ctx,
		cancel:  cancel,
		events:  make(chan activityEvent, 32),
		done:    make(chan struct{}),
		started: time.Now(),
	}
	go func() {
		defer close(s.done)
		rerr := readEventLines(ctx, r, s.events)
		close(s.events)
		werr := wait()
		switch {
		case ctx.Err() != nil:
			s.err = errProbeCancelled
		case werr != nil:
			s.err = werr
		case rerr != nil:
			s.err = rerr
		default:
			s.err = io.EOF
		}
	}()
	return s, nil
}

// close kills the process tree and waits (bounded) for the reap.
func (s *activityStream) close() {
	if s == nil {
		return
	}
	s.once.Do(s.cancel)
	select {
	case <-s.done:
	case <-time.After(activityStopWait):
	}
}

// readEventLines parses one JSON event per line into out until EOF or ctx.
// Lines over activityLineMax are dropped whole.
func readEventLines(ctx context.Context, r io.Reader, out chan<- activityEvent) error {
	br := bufio.NewReaderSize(r, 4096)
	var buf []byte
	tooLong := false
	for {
		chunk, err := br.ReadSlice('\n')
		if !tooLong {
			if len(buf)+len(chunk) > activityLineMax {
				tooLong = true
				buf = buf[:0]
			} else {
				buf = append(buf, chunk...)
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if len(buf) > 0 && !tooLong {
			if ev, ok := parseActivityEvent(buf); ok {
				select {
				case out <- ev:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		buf = buf[:0]
		tooLong = false
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// activityEvent is the parsed, trimmed engine event. Only what the timeline
// may keep: id (for membership), action, time, display name / service from
// attributes (fallback only), exit code, health, sidecar marker.
type activityEvent struct {
	id      string
	action  string // one of activityEvents
	at      time.Time
	name    string
	service string
	exit    string // die only, digits
	health  string // health_status only
	sidecar bool   // carries dc.forward.for
}

type rawEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
	Time     int64 `json:"time"`
	TimeNano int64 `json:"timeNano"`
}

func parseActivityEvent(line []byte) (activityEvent, bool) {
	var raw rawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return activityEvent{}, false
	}
	if raw.Type != "" && raw.Type != "container" {
		return activityEvent{}, false
	}
	action := raw.Action
	if action == "" {
		action = raw.Status
	}
	id := raw.Actor.ID
	if id == "" {
		id = raw.ID
	}
	if !validContainerID(id) {
		return activityEvent{}, false
	}
	ev := activityEvent{id: strings.ToLower(id)}
	base, detail, _ := strings.Cut(action, ":")
	base = strings.TrimSpace(base)
	known := false
	for _, a := range activityEvents {
		if a == base {
			known = true
			break
		}
	}
	if !known {
		return activityEvent{}, false
	}
	ev.action = base
	switch {
	case raw.TimeNano > 0:
		ev.at = time.Unix(0, raw.TimeNano)
	case raw.Time > 0:
		ev.at = time.Unix(raw.Time, 0)
	default:
		return activityEvent{}, false
	}
	attrs := raw.Actor.Attributes
	ev.name = safeLabel(strings.TrimPrefix(attrs["name"], "/"), 64)
	ev.service = safeLabel(attrs["com.docker.compose.service"], 64)
	ev.sidecar = attrs[labelForwardFor] != ""
	if base == "die" {
		if code := attrs["exitCode"]; code != "" {
			if _, err := strconv.Atoi(code); err == nil && len(code) <= 4 {
				ev.exit = code
			}
		}
	}
	if base == "health_status" {
		switch h := strings.TrimSpace(detail); h {
		case "healthy", "unhealthy", "starting":
			ev.health = h
		default:
			ev.health = "changed"
		}
	}
	return ev, true
}

// validContainerID: 12–64 lowercase/uppercase hex.
func validContainerID(id string) bool {
	if len(id) < 12 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// safeLabel keeps a short printable display string (no control characters).
func safeLabel(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= n {
			break
		}
	}
	return b.String()
}
