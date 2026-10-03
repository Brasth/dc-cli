package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- fakes ---

// fakeEvents replaces the docker events process with in-memory pipes.
type fakeEvents struct {
	mu      sync.Mutex
	engines []string
	writers []*io.PipeWriter
	closed  []bool
	fail    int
}

func installFakeEvents(t *testing.T) *fakeEvents {
	t.Helper()
	f := &fakeEvents{}
	old := startEventProcess
	t.Cleanup(func() { startEventProcess = old })
	startEventProcess = func(ctx context.Context, engine string) (io.Reader, func() error, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.engines = append(f.engines, engine)
		if f.fail > 0 {
			f.fail--
			return nil, nil, errors.New("cannot connect")
		}
		pr, pw := io.Pipe()
		i := len(f.writers)
		f.writers = append(f.writers, pw)
		f.closed = append(f.closed, false)
		go func() {
			<-ctx.Done()
			f.mu.Lock()
			f.closed[i] = true
			f.mu.Unlock()
			_ = pw.Close()
		}()
		return pr, func() error { return nil }, nil
	}
	return f
}

func (f *fakeEvents) starts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.engines)
}

func (f *fakeEvents) emit(t *testing.T, lines ...string) {
	t.Helper()
	f.mu.Lock()
	w := f.writers[len(f.writers)-1]
	f.mu.Unlock()
	// The reader only drains while the pump runs, so write in the background
	// (one writer per call keeps the order).
	go func() {
		for _, l := range lines {
			if _, err := io.WriteString(w, l+"\n"); err != nil {
				return
			}
		}
	}()
}

// drop ends the latest stream as if the daemon went away.
func (f *fakeEvents) drop() {
	f.mu.Lock()
	w := f.writers[len(f.writers)-1]
	f.mu.Unlock()
	_ = w.Close()
}

func (f *fakeEvents) allClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.closed {
		if !c {
			return false
		}
	}
	return true
}

// fakeInspect answers `docker [--context X] inspect --type container ids...`.
type fakeInspect struct {
	mu    sync.Mutex
	docs  map[string]inspectDoc
	calls [][]string
	block bool
}

func installActivityProbes(t *testing.T, ws string) (*fakeProbes, *fakeInspect) {
	t.Helper()
	fp := defaultFake()
	fp.out["dc-ls"] = "[]"
	fp.out["dc-exec"] = "[]"
	installFakeProbes(t, fp)
	fi := &fakeInspect{docs: map[string]inspectDoc{}}
	old := runPinnedDocker
	t.Cleanup(func() { runPinnedDocker = old })
	runPinnedDocker = func(ctx context.Context, args ...string) ([]byte, error) {
		for i, a := range args {
			if a != "inspect" {
				continue
			}
			fi.mu.Lock()
			fi.calls = append(fi.calls, append([]string(nil), args...))
			blk := fi.block
			var out []inspectDoc
			for _, id := range args[i+3:] {
				for full, d := range fi.docs {
					if strings.HasPrefix(full, id) {
						out = append(out, d)
					}
				}
			}
			fi.mu.Unlock()
			if blk {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			b, _ := json.Marshal(out)
			return b, nil
		}
		return nil, errors.New("unexpected pinned docker call")
	}
	return fp, fi
}

func (f *fakeInspect) add(id, name string, labels map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := inspectDoc{ID: id, Name: "/" + name}
	d.Config.Labels = labels
	f.docs[id] = d
}

func (f *fakeInspect) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeInspect) lastArgs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return nil
	}
	return f.calls[len(f.calls)-1]
}

// --- a tiny Bubble Tea runtime: runs cmds in goroutines, feeds Update ---

type pump struct {
	t    *testing.T
	m    model
	msgs chan tea.Msg
	wg   sync.WaitGroup
	// refreshes counts discoveries started through reloadMsg results.
	reloads int
}

func newPump(t *testing.T, m model) *pump {
	p := &pump{t: t, m: m, msgs: make(chan tea.Msg, 1024)}
	t.Cleanup(func() {
		// Close streams + probes, then let every cmd goroutine finish before
		// the fakes they use are restored.
		p.m.shutdown()
		done := make(chan struct{})
		go func() { p.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("cmd goroutines leaked past shutdown")
		}
	})
	return p
}

func (p *pump) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		msg := cmd()
		if b, ok := msg.(tea.BatchMsg); ok {
			for _, c := range b {
				p.run(c)
			}
			return
		}
		if msg != nil {
			p.msgs <- msg
		}
	}()
}

// do applies a direct model step and runs its cmd.
func (p *pump) do(f func(model) (model, tea.Cmd)) {
	m, cmd := f(p.m)
	p.m = m
	p.run(cmd)
}

func (p *pump) key(k string) {
	p.do(func(m model) (model, tea.Cmd) {
		got, cmd := m.handleKey(k)
		return got.(model), cmd
	})
}

// settle feeds messages until nothing arrives for quiet.
func (p *pump) settle(quiet time.Duration) {
	for {
		select {
		case msg := <-p.msgs:
			if _, ok := msg.(reloadMsg); ok {
				p.reloads++
			}
			got, cmd := p.m.Update(msg)
			p.m = got.(model)
			p.run(cmd)
		case <-time.After(quiet):
			return
		}
	}
}

// until settles until cond holds (or fails after 3s).
func (p *pump) until(what string, cond func(model) bool) {
	p.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond(p.m) {
		if time.Now().After(deadline) {
			p.t.Fatalf("timed out waiting for %s", what)
		}
		p.settle(10 * time.Millisecond)
	}
}

func fastActivity(t *testing.T) {
	t.Helper()
	oldD, oldB, oldH := activityDebounceDelay, activityBackoff, activityHealthyAfter
	activityDebounceDelay = 30 * time.Millisecond
	activityBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond, 100 * time.Millisecond}
	activityHealthyAfter = time.Hour
	t.Cleanup(func() { activityDebounceDelay, activityBackoff, activityHealthyAfter = oldD, oldB, oldH })
}

func devWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".devcontainer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".devcontainer", "devcontainer.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws
}

func composeWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws
}

var (
	appID  = fullID('a')
	dbID   = fullID('d')
	newID  = fullID('e')
	sideID = fullID('5')
	strID  = fullID('9')
)

// liveBoard is a loaded devcontainer board with app + db known and the
// stream started.
func liveBoard(t *testing.T) (*pump, *fakeEvents, *fakeProbes, *fakeInspect) {
	t.Helper()
	fastActivity(t)
	ws := devWorkspace(t)
	fe := installFakeEvents(t)
	fp, fi := installActivityProbes(t, ws)
	m := model{
		workspace: ws, hasConfig: true, hoverStack: -1, width: 100, height: 30,
		load: loadReady, loaded: true, loadGen: 1, engine: "context:colima",
		probes: newProbeSession(),
		rows:   []container{{ID: appID, Name: "app-1", Status: "running", LocalFolder: ws, Compose: "proj"}},
		stack:  []stackSvc{{ID: appID, Name: "app-1", Service: "app", Status: "running"}, {ID: dbID, Name: "db-1", Service: "db", Status: "running"}},
	}
	b, _ := json.Marshal(m.rows)
	fp.out["dc-ls"] = string(b)
	b, _ = json.Marshal(m.stack)
	fp.out["dc-exec"] = string(b)
	p := newPump(t, m)
	p.do(func(m model) (model, tea.Cmd) { return m.syncActivity("") })
	if fe.starts() != 1 || p.m.activity.stream == nil {
		t.Fatalf("stream must start once the board shows a workspace (starts=%d)", fe.starts())
	}
	return p, fe, fp, fi
}

func entryTexts(m model) []string {
	var out []string
	for _, e := range m.activity.entries {
		if e.gap {
			out = append(out, "--"+e.what)
			continue
		}
		out = append(out, e.who+" "+e.what)
	}
	return out
}

func hasEntry(m model, s string) bool {
	for _, e := range entryTexts(m) {
		if e == s {
			return true
		}
	}
	return false
}

// --- tests ---

func TestActivityKnownMembersAcceptedWithoutInspect(t *testing.T) {
	p, fe, _, fi := liveBoard(t)
	fe.emit(t, evLine(dbID, "die", map[string]string{"exitCode": "1", "name": "db-1"}), evLine(appID, "start", nil))
	p.until("two entries", func(m model) bool { return len(m.activity.entries) == 2 })
	if !hasEntry(p.m, "db exited (code 1)") || !hasEntry(p.m, "app started") {
		t.Fatalf("entries=%v", entryTexts(p.m))
	}
	if fi.count() != 0 {
		t.Fatal("known ids need no inspect")
	}
}

func TestActivityDestroyOfKnownIDAttributedAfterGone(t *testing.T) {
	p, fe, fp, fi := liveBoard(t)
	// Discovery no longer lists db (it was removed) …
	fp.mu.Lock()
	fp.out["dc-exec"] = `[{"id":"` + appID + `","name":"app-1","service":"app","status":"running"}]`
	fp.mu.Unlock()
	p.do(func(m model) (model, tea.Cmd) { return m.requestRefresh() })
	p.until("refresh", func(m model) bool { return m.load == loadReady && len(m.stack) == 1 })
	// … and inspect cannot see it either, yet destroy still attributes.
	fe.emit(t, evLine(dbID, "destroy", nil))
	p.until("destroy entry", func(m model) bool { return hasEntry(m, "db removed") })
	if fi.count() != 0 {
		t.Fatal("retained known id must not be inspected")
	}
}

func TestActivityUnknownIDsVerifiedByInspect(t *testing.T) {
	p, fe, _, fi := liveBoard(t)
	ws := p.m.workspace
	other := t.TempDir()
	fi.add(newID, "proj-cache-1", map[string]string{labelProject: "proj", labelService: "cache"})
	fi.add(sideID, "fwd-1", map[string]string{labelForwardFor: appID, labelProject: "proj"})
	fi.add(strID, "intruder", map[string]string{labelProject: "elsewhere", labelFolder: other})
	fe.emit(t,
		evLine(newID, "create", map[string]string{"name": "proj-cache-1"}),
		// Sidecar: labels in the event are enough to exclude, never to include.
		evLine(sideID, "start", map[string]string{"dc.forward.for": appID}),
		// Forged: event claims our project, inspect says otherwise.
		evLine(strID, "start", map[string]string{"com.docker.compose.project": "proj", "devcontainer.local_folder": ws}),
	)
	p.until("cache accepted", func(m model) bool { return hasEntry(m, "cache created") })
	p.settle(80 * time.Millisecond)
	if len(p.m.activity.entries) != 1 {
		t.Fatalf("only the verified sibling may appear: %v", entryTexts(p.m))
	}
	args := strings.Join(fi.lastArgs(), " ")
	if !strings.HasPrefix(args, "--context colima inspect --type container") {
		t.Fatalf("inspect must be pinned + read-only: %q", args)
	}
	if strings.Contains(args, sideID) {
		t.Fatal("event-labelled sidecar must not even be inspected")
	}
	// Verdicts are cached: a second event for the forged id is dropped without inspect.
	n := fi.count()
	fe.emit(t, evLine(strID, "stop", nil), evLine(newID, "start", nil))
	p.until("cache start", func(m model) bool { return hasEntry(m, "cache started") })
	if fi.count() != n || hasEntry(p.m, "intruder stopped") {
		t.Fatalf("cached verdicts: calls %d→%d entries=%v", n, fi.count(), entryTexts(p.m))
	}
}

func TestActivityInspectSidecarLabelRejected(t *testing.T) {
	p, fe, _, fi := liveBoard(t)
	// Event without the label (forged / stripped); inspect shows the sidecar.
	fi.add(sideID, "fwd", map[string]string{labelForwardFor: appID, labelProject: "proj"})
	fe.emit(t, evLine(sideID, "start", nil))
	p.until("inspected", func(m model) bool { return fi.count() == 1 && len(m.activity.checking) == 0 })
	p.settle(50 * time.Millisecond)
	if len(p.m.activity.entries) != 0 {
		t.Fatalf("sidecar must be excluded: %v", entryTexts(p.m))
	}
}

func TestActivityNewAppVerifiesItsProject(t *testing.T) {
	p, fe, _, fi := liveBoard(t)
	ws := p.m.workspace
	app2, sib := fullID('1'), fullID('2')
	fi.add(app2, "app-2", map[string]string{labelFolder: ws + "/.devcontainer", labelProject: "proj2"})
	fi.add(sib, "proj2-redis-1", map[string]string{labelProject: "proj2", labelService: "redis"})
	// Sibling first in the same batch: the app still verifies proj2 for it.
	fe.emit(t, evLine(sib, "create", nil), evLine(app2, "create", nil))
	p.until("both", func(m model) bool { return hasEntry(m, "redis created") && hasEntry(m, "app-2 created") })
}

func TestActivityAwaitResolvedByLaterDiscovery(t *testing.T) {
	p, fe, fp, fi := liveBoard(t)
	late, stranger := fullID('3'), fullID('4')
	fi.add(late, "p9-web-1", map[string]string{labelProject: "p9", labelService: "web"})
	fi.add(stranger, "p8-web-1", map[string]string{labelProject: "p8", labelService: "web"})
	// The next discovery (started after the events) lists `late` only.
	fp.mu.Lock()
	fp.out["dc-exec"] = `[{"id":"` + appID + `","service":"app"},{"id":"` + late[:12] + `","name":"p9-web-1","service":"web"}]`
	fp.mu.Unlock()
	fe.emit(t, evLine(late, "start", nil), evLine(stranger, "start", nil))
	p.until("late accepted", func(m model) bool { return hasEntry(m, "web started") })
	p.until("pending settled", func(m model) bool { return len(m.activity.pending) == 0 })
	if n := len(p.m.activity.entries); n != 1 {
		t.Fatalf("stranger must be dropped after discovery: %v", entryTexts(p.m))
	}
	if p.m.activity.verdict[stranger] != memberReject {
		t.Fatal("unlisted id must be rejected after the discovery")
	}
}

func TestActivityShortIDsNormalizedNotPrefixMatched(t *testing.T) {
	p, fe, _, fi := liveBoard(t)
	short := fullID('7')[:12]
	full := fullID('7')
	fi.add(full, "short-svc-1", map[string]string{labelProject: "proj", labelService: "short"})
	p.do(func(m model) (model, tea.Cmd) {
		m.stack = append(m.stack, stackSvc{ID: short, Name: "short-svc-1", Service: "short"})
		return m.syncActivity(m.engine)
	})
	if _, ok := lookupKnown(p.m.activity.known, full); ok {
		t.Fatal("a short id must not match before normalization")
	}
	p.until("normalized", func(m model) bool { _, ok := m.activity.known[full]; return ok })
	if _, ok := p.m.activity.known[short]; ok {
		t.Fatal("known keys must be full ids")
	}
	n := fi.count()
	fe.emit(t, evLine(full, "restart", nil))
	p.until("restart", func(m model) bool { return hasEntry(m, "short restarted") })
	if fi.count() != n {
		t.Fatal("normalized id must match without another inspect")
	}
}

func TestActivityComposeKindNeedsWorkdirProofAndProject(t *testing.T) {
	fastActivity(t)
	ws := composeWorkspace(t)
	other := t.TempDir()
	fe := installFakeEvents(t)
	_, fi := installActivityProbes(t, ws)
	good, wrongDir, wrongProj := fullID('1'), fullID('2'), fullID('3')
	fi.add(good, "cw-api-1", map[string]string{labelProject: "cw", labelService: "api", labelWorkdir: ws})
	fi.add(wrongDir, "cw-x-1", map[string]string{labelProject: "cw", labelService: "x", labelWorkdir: other})
	fi.add(wrongProj, "zz-y-1", map[string]string{labelProject: "zz", labelService: "y", labelConfigFiles: filepath.Join(ws, "compose.yaml")})
	m := model{
		workspace: ws, hasCompose: true, hoverStack: -1, load: loadReady, loaded: true, loadGen: 1,
		engine: "context:colima", probes: newProbeSession(),
		rows: []container{{ID: fullID('0'), Compose: "cw"}},
	}
	p := newPump(t, m)
	p.do(func(m model) (model, tea.Cmd) { return m.syncActivity("") })
	fe.emit(t, evLine(good, "start", nil), evLine(wrongDir, "start", nil), evLine(wrongProj, "start", nil))
	p.until("good", func(m model) bool { return hasEntry(m, "api started") })
	p.until("settled", func(m model) bool { return len(m.activity.pending) == 0 })
	if len(p.m.activity.entries) != 1 {
		t.Fatalf("compose-kind must prove working_dir and project: %v", entryTexts(p.m))
	}
}

func TestActivityBoundedTo200(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	var lines []string
	for i := 0; i < 260; i++ {
		lines = append(lines, evLine(appID, "start", nil))
	}
	fe.emit(t, lines...)
	p.until("cap", func(m model) bool { return len(m.activity.entries) == activityCap })
	p.settle(50 * time.Millisecond)
	if len(p.m.activity.entries) != activityCap {
		t.Fatalf("len=%d", len(p.m.activity.entries))
	}
	for i := 1; i < len(p.m.activity.entries); i++ {
		if p.m.activity.entries[i].at.Before(p.m.activity.entries[i-1].at) {
			t.Fatal("entries must stay in time order")
		}
	}
}

func TestActivityDebounceCoalescesDiscovery(t *testing.T) {
	p, fe, fp, _ := liveBoard(t)
	before := fp.count("dc-ls")
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, evLine(dbID, "restart", nil))
	}
	fe.emit(t, lines...)
	p.until("20 entries", func(m model) bool { return len(m.activity.entries) == 20 })
	p.settle(150 * time.Millisecond)
	if got := fp.count("dc-ls") - before; got < 1 || got > 2 {
		t.Fatalf("20 events → one discovery (+ at most one queued), got %d", got)
	}
	// health changes go through the same coalesced refresh.
	p.until("idle", func(m model) bool { return m.load == loadReady && !m.discoveryPending && !m.activity.armed })
	before = fp.count("dc-ls")
	var health []string
	for i := 0; i < 10; i++ {
		health = append(health, evLine(appID, "health_status: healthy", nil))
	}
	fe.emit(t, health...)
	p.until("health", func(m model) bool { return len(m.activity.entries) == 30 })
	p.settle(150 * time.Millisecond)
	if got := fp.count("dc-ls") - before; got < 1 || got > 2 {
		t.Fatalf("10 health events → one coalesced discovery (+ at most one queued), got %d", got)
	}
}

func TestActivityKnownBoundedKeepsNewest(t *testing.T) {
	known := map[string]memberInfo{}
	var newest string
	for i := 0; i < activityKnownCap+40; i++ {
		id := strings.Repeat("0", 56) + fmt.Sprintf("%08x", i)
		known[id] = stamped(memberInfo{name: id})
		newest = id
	}
	got := capKnown(known)
	if len(got) != activityKnownCap {
		t.Fatalf("len=%d", len(got))
	}
	if _, ok := got[newest]; !ok {
		t.Fatal("the most recently verified id must be kept")
	}
	if _, ok := got[strings.Repeat("0", 64)]; ok {
		t.Fatal("the oldest retained id goes first")
	}
	// Verified additions from inspect are bounded too.
	p, fe, _, fi := liveBoard(t)
	p.do(func(m model) (model, tea.Cmd) { m.activity.known = known; return m, nil })
	fi.add(newID, "proj-x-1", map[string]string{labelProject: "proj", labelService: "x"})
	fe.emit(t, evLine(newID, "create", nil))
	p.until("x", func(m model) bool { return hasEntry(m, "x created") })
	if n := len(p.m.activity.known); n > activityKnownCap {
		t.Fatalf("known=%d over cap", n)
	}
	if _, ok := p.m.activity.known[newID]; !ok {
		t.Fatal("newly verified id must survive the cap")
	}
}

func TestActivityReconnectBackoffGapAndRescan(t *testing.T) {
	p, fe, fp, _ := liveBoard(t)
	fe.mu.Lock()
	fe.fail = 3
	fe.mu.Unlock()
	before := fp.count("dc-ls")
	fe.drop()
	var waits []time.Duration
	p.until("reconnected", func(m model) bool {
		if m.activity.retrying && (len(waits) == 0 || waits[len(waits)-1] != m.activity.retryIn) {
			waits = append(waits, m.activity.retryIn)
		}
		return m.activity.stream != nil && fe.starts() == 5
	})
	want := activityBackoff[:4]
	if len(waits) != 4 {
		t.Fatalf("waits=%v want %v", waits, want)
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("waits=%v want %v", waits, want)
		}
	}
	gaps := 0
	for _, e := range entryTexts(p.m) {
		if strings.HasPrefix(e, "--event stream lost") {
			gaps++
		}
	}
	if gaps != 1 || !hasEntry(p.m, "--event stream reconnected — rescanning") {
		t.Fatalf("one loss marker + reconnect marker: %v", entryTexts(p.m))
	}
	p.until("rescan", func(m model) bool { return fp.count("dc-ls") > before })
	// An event on the healthy stream resets the backoff.
	fe.emit(t, evLine(appID, "start", nil))
	p.until("event", func(m model) bool { return hasEntry(m, "app started") })
	if p.m.activity.retry != 0 {
		t.Fatalf("retry=%d", p.m.activity.retry)
	}
}

func TestActivityBackoffCapsAtTen(t *testing.T) {
	oldB := activityBackoff
	t.Cleanup(func() { activityBackoff = oldB })
	activityBackoff = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 10 * time.Second}
	m := model{activity: activityState{sess: &activitySession{id: 1}}}
	var got []time.Duration
	for i := 0; i < 6; i++ {
		m, _ = m.scheduleActivityRetry()
		got = append(got, m.activity.retryIn)
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 10 * time.Second, 10 * time.Second, 10 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestActivityStaleMessagesDropped(t *testing.T) {
	p, _, _, _ := liveBoard(t)
	sess := p.m.activity.sess.id
	gen := p.m.activity.streamGen
	ev := activityEvent{id: appID, action: "start", at: time.Now()}
	for _, msg := range []tea.Msg{
		activityEventMsg{sess: sess + 99, gen: gen, ev: ev},
		activityEventMsg{sess: sess, gen: gen + 1, ev: ev},
		activityVerifiedMsg{sess: sess + 99, ids: []string{newID}},
		activityRefreshMsg{sess: sess + 99},
		activityRetryMsg{sess: sess + 99},
		activityEndMsg{sess: sess, gen: gen + 7},
		activityNormalizedMsg{sess: sess + 99},
	} {
		got, cmd := p.m.Update(msg)
		gm := got.(model)
		if len(gm.activity.entries) != 0 || cmd != nil || gm.activity.stream == nil || gm.load != loadReady {
			t.Fatalf("%T must be dropped", msg)
		}
	}
	// Engine label moved without a sync yet: the old session no longer matches.
	m := p.m
	m.engine = "context:other"
	got, _ := m.Update(activityEventMsg{sess: sess, gen: gen, ev: ev})
	if len(got.(model).activity.entries) != 0 {
		t.Fatal("result for the old engine must not land on the new one")
	}
}

func TestActivitySwitchClearsAndReaps(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	fe.emit(t, evLine(appID, "start", nil))
	p.until("entry", func(m model) bool { return len(m.activity.entries) == 1 })
	oldSess := p.m.activity.sess.id
	p.do(func(m model) (model, tea.Cmd) { return m.switchContext(t.TempDir(), false) })
	if p.m.activity.sess != nil || len(p.m.activity.entries) != 0 || !fe.allClosed() {
		t.Fatal("switch must close the stream and clear the timeline")
	}
	got, _ := p.m.Update(activityEventMsg{sess: oldSess, gen: 1, ev: activityEvent{id: appID, action: "stop", at: time.Now()}})
	if len(got.(model).activity.entries) != 0 {
		t.Fatal("old workspace events must be dropped")
	}
}

func TestActivityFleetHasNoTimeline(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	fe.emit(t, evLine(appID, "start", nil))
	p.until("entry", func(m model) bool { return len(m.activity.entries) == 1 })
	p.key("f")
	p.settle(50 * time.Millisecond)
	if !p.m.fleet || p.m.activity.sess != nil || len(p.m.activity.entries) != 0 || !fe.allClosed() {
		t.Fatal("fleet: no stream, no timeline")
	}
	starts := fe.starts()
	p.key("r") // manual fleet refresh
	p.settle(80 * time.Millisecond)
	if fe.starts() != starts || p.m.activity.sess != nil {
		t.Fatal("fleet refresh must not start a timeline")
	}
	p.key("v")
	if p.m.activity.open || !strings.Contains(p.m.status, "per workspace") {
		t.Fatalf("v in fleet must refuse: %q", p.m.status)
	}
}

func TestActivityEngineChangeRestartsPinned(t *testing.T) {
	p, fe, fp, _ := liveBoard(t)
	fe.emit(t, evLine(appID, "start", nil))
	p.until("entry", func(m model) bool { return len(m.activity.entries) == 1 })
	fp.mu.Lock()
	fp.engine = "context:desktop-linux"
	fp.mu.Unlock()
	p.do(func(m model) (model, tea.Cmd) { return m.requestRefresh() })
	p.until("new engine", func(m model) bool { return m.engine == "context:desktop-linux" && m.activity.sess != nil })
	fe.mu.Lock()
	engines := append([]string(nil), fe.engines...)
	firstClosed := fe.closed[0]
	fe.mu.Unlock()
	if len(engines) != 2 || engines[1] != "context:desktop-linux" || !firstClosed {
		t.Fatalf("engines=%v firstClosed=%v", engines, firstClosed)
	}
	if hasEntry(p.m, "app started") {
		t.Fatal("old engine entries must be cleared")
	}
}

func TestActivityForegroundLeaveReapsAndResumes(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	fe.emit(t, evLine(appID, "start", nil))
	p.until("entry", func(m model) bool { return len(m.activity.entries) == 1 })
	p.do(func(m model) (model, tea.Cmd) { return m.startLeave("shell", "e") })
	if p.m.activity.sess != nil || !fe.allClosed() {
		t.Fatal("leaving must close + reap the stream before the terminal is handed over")
	}
	// A discovery landing during the leave tick must not restart it.
	p.do(func(m model) (model, tea.Cmd) { return m.syncActivity(m.engine) })
	if p.m.activity.sess != nil {
		t.Fatal("no stream while leaving")
	}
	p.do(func(m model) (model, tea.Cmd) {
		got, cmd := m.Update(execDoneMsg{action: "e"})
		return got.(model), cmd
	})
	if p.m.activity.stream == nil || fe.starts() != 2 {
		t.Fatal("return must resume the stream")
	}
	if p.m.load != loadPending {
		t.Fatal("return must rescan")
	}
	if !hasEntry(p.m, "app started") || !strings.HasPrefix(entryTexts(p.m)[1], "--board was away") {
		t.Fatalf("timeline kept with a gap marker: %v", entryTexts(p.m))
	}
}

func TestActivityProjectActionLeaveAndReturn(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	p.do(func(m model) (model, tea.Cmd) { return m.startLeave("action", "action") })
	if p.m.activity.sess != nil || !fe.allClosed() {
		t.Fatal("action leave must reap the stream")
	}
	p.do(func(m model) (model, tea.Cmd) {
		return m.applyActionDone(actionDoneMsg{workspace: m.workspace, label: "t"})
	})
	if p.m.activity.stream == nil {
		t.Fatal("action return must resume")
	}
}

func TestActivityQuitReaps(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	p.key("q")
	if !fe.allClosed() || p.m.activity.sess != nil {
		t.Fatal("quit must close the stream")
	}
}

func TestActivityHostBlockPausesAndRescansOnRecovery(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	p.do(func(m model) (model, tea.Cmd) {
		return m.handleReload(reloadMsg{gen: m.loadGen, workspace: m.workspace, engine: m.engine, host: hostReport{Code: "docker_daemon_down", Summary: "down"}})
	})
	if !p.m.hostBlock || p.m.activity.sess != nil || !fe.allClosed() {
		t.Fatalf("docker down must pause the stream (block=%v)", p.m.hostBlock)
	}
	p.do(func(m model) (model, tea.Cmd) { return m.beginHardReload() })
	p.until("recovered", func(m model) bool { return !m.hostBlock && m.activity.stream != nil })
	if fe.starts() != 2 {
		t.Fatalf("starts=%d", fe.starts())
	}
}

func TestActivityViewKeys(t *testing.T) {
	p, fe, _, _ := liveBoard(t)
	var lines []string
	for i := 0; i < 60; i++ {
		lines = append(lines, evLine(appID, "start", nil))
	}
	fe.emit(t, lines...)
	p.until("60", func(m model) bool { return len(m.activity.entries) == 60 })
	p.key("v")
	if !p.m.activity.open || p.m.activity.off != p.m.activityMaxOff() {
		t.Fatal("v opens following the newest entry")
	}
	if v := p.m.View(); !strings.Contains(v, "activity — ") || !strings.Contains(v, "app") || !strings.Contains(v, "c clear") {
		t.Fatalf("view=%s", v)
	}
	p.key("g")
	if p.m.activity.off != 0 || p.m.activity.follow {
		t.Fatal("g goes to the top")
	}
	p.key("j")
	if p.m.activity.off != 1 {
		t.Fatal("j scrolls")
	}
	p.key("G")
	if !p.m.activity.follow {
		t.Fatal("G follows")
	}
	p.key("c")
	if len(p.m.activity.entries) != 0 || p.m.activity.stream == nil {
		t.Fatal("c clears the timeline, stream keeps running")
	}
	p.key("esc")
	if p.m.activity.open {
		t.Fatal("esc closes")
	}
	if p.m.quitting {
		t.Fatal("esc in the view must not quit the board")
	}
}

func TestActivityInspectBoundedByProbeTimeout(t *testing.T) {
	old := probeTimeout
	t.Cleanup(func() { probeTimeout = old })
	probeTimeout = 50 * time.Millisecond
	p, fe, _, fi := liveBoard(t)
	fi.mu.Lock()
	fi.block = true
	fi.mu.Unlock()
	fe.emit(t, evLine(newID, "create", nil))
	p.until("inspect gave up", func(m model) bool { return fi.count() == 1 && len(m.activity.checking) == 0 })
	if len(p.m.activity.entries) != 0 || len(p.m.activity.pending) != 0 {
		t.Fatalf("unverifiable events are dropped: %v", entryTexts(p.m))
	}
}
