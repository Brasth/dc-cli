package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// finishStay runs an async stay command and applies its result.
func finishStay(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected stay cmd")
	}
	msg := cmd()
	done, ok := msg.(stayDoneMsg)
	if !ok {
		t.Fatalf("want stayDoneMsg, got %T", msg)
	}
	got, _ := m.Update(done)
	return got.(model)
}

// fakeProbes swaps runProbe/probeEngine/runHostDiagnose for the test.
type fakeProbes struct {
	mu     sync.Mutex
	calls  []string
	engine string
	out    map[string]string        // name → stdout
	errs   map[string]error         // name → error
	delay  map[string]time.Duration // name → latency (honours ctx)
	block  map[string]bool          // name → wait for ctx
}

func installFakeProbes(t testing.TB, f *fakeProbes) {
	t.Helper()
	oldRun, oldEngine, oldHost := runProbe, probeEngine, runHostDiagnose
	t.Cleanup(func() {
		runProbe, probeEngine, runHostDiagnose = oldRun, oldEngine, oldHost
		diskCache.reset()
	})
	diskCache.reset()
	runProbe = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		key := name
		if name == "docker" && len(args) > 0 {
			key = "docker " + args[0]
		}
		f.mu.Lock()
		f.calls = append(f.calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
		d := f.delay[key]
		blk := f.block[key]
		out, err := f.out[key], f.errs[key]
		f.mu.Unlock()
		if blk {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		if d > 0 {
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return []byte(out), err
	}
	probeEngine = func(context.Context) string { return f.engine }
	runHostDiagnose = func(ctx context.Context) (hostReport, error) {
		if _, err := probe(ctx, "host-check"); err != nil {
			return hostReport{}, err
		}
		return hostReport{Code: "ready"}, nil
	}
}

func (f *fakeProbes) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func rowsJSON(n int) string {
	var rows []container
	for i := 0; i < n; i++ {
		rows = append(rows, container{ID: fmt.Sprintf("c%02d", i), Name: fmt.Sprintf("svc-%d", i), Status: "running", LocalFolder: "/tmp/app"})
	}
	b, _ := json.Marshal(rows)
	return string(b)
}

func stackJSON(n int) string {
	var rows []stackSvc
	for i := 0; i < n; i++ {
		rows = append(rows, stackSvc{ID: fmt.Sprintf("c%02d", i), Name: fmt.Sprintf("svc-%d", i), Status: "running", Service: fmt.Sprintf("s%d", i)})
	}
	b, _ := json.Marshal(rows)
	return string(b)
}

func defaultFake() *fakeProbes {
	return &fakeProbes{
		engine: "context:colima",
		out: map[string]string{
			"dc-ls":   rowsJSON(1),
			"dc-exec": stackJSON(2),
			"dc-df":   `{"compact":"docker 40%"}`,
			"dc-net":  `{"schemaVersion":1,"networks":[{"name":"shared","present":true}]}`,
		},
		errs:  map[string]error{},
		delay: map[string]time.Duration{},
		block: map[string]bool{},
	}
}

func pendingModel() model {
	return model{workspace: "/tmp/app", hasConfig: true, width: 100, hoverStack: -1, editor: "zed", load: loadPending, loadGen: 1}
}

func TestEssentialsNeverRunOptionalProbes(t *testing.T) {
	f := defaultFake()
	installFakeProbes(t, f)
	msg := runEssentials(context.Background(), "/tmp/app", false, 1)
	if msg.err != nil {
		t.Fatalf("err=%v", msg.err)
	}
	if len(msg.rows) != 1 || len(msg.stack) != 2 || msg.engine != "context:colima" {
		t.Fatalf("msg=%+v", msg)
	}
	for _, name := range []string{"dc-df", "dc-net", "docker ps"} {
		if f.count(name) != 0 {
			t.Fatalf("essentials ran optional probe %s: %v", name, f.calls)
		}
	}
}

func TestEssentialsRenderBeforeOptional(t *testing.T) {
	f := defaultFake()
	f.delay["dc-df"] = time.Hour // optional never lands during this test
	installFakeProbes(t, f)
	m := pendingModel()
	msg := runEssentials(context.Background(), m.workspace, false, m.loadGen)
	got, cmd := m.Update(msg)
	mm := got.(model)
	if mm.load != loadReady || !mm.loaded {
		t.Fatalf("essentials must make the board ready: load=%v", mm.load)
	}
	if cmd == nil {
		t.Fatal("essentials must start optional probes")
	}
	if mm.diskSec.status != sectionLoading || mm.netsSec.status != sectionLoading || mm.portsSec.status != sectionLoading {
		t.Fatalf("sections must be loading: %+v %+v %+v", mm.diskSec, mm.netsSec, mm.portsSec)
	}
	s := ansi.Strip(mm.View())
	if !strings.Contains(s, "running") {
		t.Fatalf("rows must render first:\n%s", s)
	}
	if !strings.Contains(s, "loading…") {
		t.Fatalf("optional sections should say loading:\n%s", s)
	}
	// Board is actionable while optionals load.
	got, c := mm.handleKey("e")
	if c == nil || got.(model).leaving != "shell" {
		t.Fatal("shell must work before optional probes finish")
	}
}

func TestOptionalApplyAndStaleTags(t *testing.T) {
	m := pendingModel()
	m = m.applyReload(reloadMsg{gen: 1, workspace: "/tmp/app", engine: "context:a", rows: []container{{ID: "a", Status: "running"}}})
	m, _ = m.startOptional()
	tag := m.tag()

	stale := []discoveryTag{
		{workspace: "/tmp/other", engine: tag.engine, gen: tag.gen},
		{workspace: tag.workspace, engine: "context:b", gen: tag.gen},
		{workspace: tag.workspace, engine: tag.engine, gen: tag.gen - 1},
		{workspace: tag.workspace, fleet: true, engine: tag.engine, gen: tag.gen},
	}
	for _, st := range stale {
		mm := m.applyOptional(optionalMsg{tag: st, section: sectionDisk, disk: diskInfo{compact: "docker 99%"}})
		if mm.disk != "" || mm.diskSec.status != sectionLoading {
			t.Fatalf("stale tag %+v applied: disk=%q sec=%+v", st, mm.disk, mm.diskSec)
		}
	}
	m = m.applyOptional(optionalMsg{tag: tag, section: sectionDisk, disk: diskInfo{compact: "docker 91%"}})
	if m.disk != "docker 91%" || !m.diskCritical || m.diskSec.status != sectionReady {
		t.Fatalf("disk=%q critical=%v sec=%+v", m.disk, m.diskCritical, m.diskSec)
	}
	m = m.applyOptional(optionalMsg{tag: tag, section: sectionPorts, fwd: []portPair{{Host: 3000, Container: 3000}}})
	if len(m.webLinks()) != 1 {
		t.Fatalf("ports not applied: %+v", m.fwdMaps)
	}
	m = m.applyOptional(optionalMsg{tag: tag, section: sectionNets, nets: netReport{Networks: []netRow{{Name: "x", Present: true}}}})
	if netHeaderLine(m.net) != "ok" {
		t.Fatalf("nets not applied: %+v", m.net)
	}
}

func TestOptionalFailureUnavailableVsStale(t *testing.T) {
	m := pendingModel()
	m = m.applyReload(reloadMsg{gen: 1, workspace: "/tmp/app", rows: []container{{ID: "a", Status: "running"}}})
	m, _ = m.startOptional()
	tag := m.tag()
	m = m.applyOptional(optionalMsg{tag: tag, section: sectionDisk, err: errStr("dc-df: timed out after 5s")})
	if m.diskSec.status != sectionUnavailable {
		t.Fatalf("no value + error must be unavailable: %+v", m.diskSec)
	}
	if !strings.Contains(ansi.Strip(m.View()), "unavailable") {
		t.Fatalf("view must say unavailable:\n%s", ansi.Strip(m.View()))
	}
	m = m.applyOptional(optionalMsg{tag: tag, section: sectionNets, nets: netReport{Networks: []netRow{{Name: "x", Present: true}}}})
	// Soft refresh: values stay, a failed refresh marks them stale.
	m, _ = m.beginSoftReload()
	m = m.applyReload(reloadMsg{gen: m.loadGen, workspace: "/tmp/app", rows: []container{{ID: "a", Status: "running"}}})
	m, _ = m.startOptional()
	m = m.applyOptional(optionalMsg{tag: m.tag(), section: sectionNets, err: errStr("dc-net boom")})
	if m.netsSec.status != sectionStale || netHeaderLine(m.net) != "ok" {
		t.Fatalf("value + error must be stale and kept: %+v %+v", m.netsSec, m.net)
	}
	if !strings.Contains(ansi.Strip(m.View()), "(stale)") {
		t.Fatalf("view must say stale:\n%s", ansi.Strip(m.View()))
	}
}

func TestEngineChangeClearsOptional(t *testing.T) {
	m := pendingModel()
	m = m.applyReload(reloadMsg{gen: 1, workspace: "/tmp/app", engine: "context:a", rows: []container{{ID: "a", Status: "running"}}})
	m, _ = m.startOptional()
	m = m.applyOptional(optionalMsg{tag: m.tag(), section: sectionDisk, disk: diskInfo{compact: "docker 10%"}})
	m, _ = m.beginSoftReload()
	m, _ = m.handleReload(reloadMsg{gen: m.loadGen, workspace: "/tmp/app", engine: "context:b", rows: []container{{ID: "a", Status: "running"}}})
	if m.engine != "context:b" {
		t.Fatalf("engine=%q", m.engine)
	}
	if m.disk != "" {
		t.Fatalf("disk from the old engine must not survive: %q", m.disk)
	}
	if m.diskSec.status != sectionLoading {
		t.Fatalf("new engine disk must reload: %+v", m.diskSec)
	}
}

func TestEssentialDeadline(t *testing.T) {
	f := defaultFake()
	f.block["dc-ls"] = true
	installFakeProbes(t, f)
	oldP, oldE := probeTimeout, essentialDeadline
	t.Cleanup(func() { probeTimeout, essentialDeadline = oldP, oldE })
	probeTimeout = 40 * time.Millisecond
	essentialDeadline = 80 * time.Millisecond
	start := time.Now()
	msg := runEssentials(context.Background(), "/tmp/app", false, 1)
	if time.Since(start) > time.Second {
		t.Fatalf("deadline not enforced: %s", time.Since(start))
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "timed out") {
		t.Fatalf("want per-probe timeout error, got %v", msg.err)
	}
	m := pendingModel()
	got, _ := m.Update(msg)
	mm := got.(model)
	if mm.load != loadFailed || !strings.Contains(ansi.Strip(mm.View()), "unknown") {
		t.Fatalf("timed-out discovery must show unknown, load=%v", mm.load)
	}
}

func TestEssentialAggregateDeadline(t *testing.T) {
	f := defaultFake()
	f.delay["host-check"] = 60 * time.Millisecond
	f.delay["dc-ls"] = 60 * time.Millisecond
	installFakeProbes(t, f)
	oldP, oldE := probeTimeout, essentialDeadline
	t.Cleanup(func() { probeTimeout, essentialDeadline = oldP, oldE })
	probeTimeout = time.Second
	essentialDeadline = 90 * time.Millisecond // each probe fits, the sum does not
	msg := runEssentials(context.Background(), "/tmp/app", false, 1)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "deadline") {
		t.Fatalf("want aggregate deadline error, got %v", msg.err)
	}
}

func TestSwitchCancelsInFlightProbes(t *testing.T) {
	f := defaultFake()
	f.block["dc-ls"] = true
	installFakeProbes(t, f)
	m := pendingModel()
	m.probes = newProbeSession()
	cmd := m.reloadCmd()
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	time.Sleep(20 * time.Millisecond)
	// Context switch: new session, old probe tree is cancelled.
	m = m.renewProbes()
	select {
	case msg := <-done:
		rm := msg.(reloadMsg)
		if rm.err == nil || rm.err != errProbeCancelled {
			t.Fatalf("want cancelled, got %v", rm.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("switch did not cancel the in-flight probe")
	}
	if m.probes.context().Err() != nil {
		t.Fatal("new session must be live")
	}
}

func TestNewGenerationCancelsOld(t *testing.T) {
	s := newProbeSession()
	g1 := s.beginGeneration()
	g2 := s.beginGeneration()
	if g1.Err() == nil {
		t.Fatal("older generation must be cancelled")
	}
	if g2.Err() != nil {
		t.Fatal("current generation must be live")
	}
	s.close()
	if g2.Err() == nil || s.context().Err() == nil {
		t.Fatal("close must cancel everything")
	}
}

func TestQuitClosesProbes(t *testing.T) {
	m := pendingModel()
	m.probes = newProbeSession()
	ctx := m.probes.context()
	got, cmd := m.handleKey("q")
	if cmd == nil || !got.(model).quitting {
		t.Fatal("q must quit")
	}
	if ctx.Err() == nil {
		t.Fatal("quit must cancel probes")
	}
}

func TestDiskCachePerEngine(t *testing.T) {
	f := defaultFake()
	installFakeProbes(t, f)
	tagA := discoveryTag{workspace: "/tmp/app", engine: "context:a", gen: 1}
	tagB := discoveryTag{workspace: "/tmp/app", engine: "context:b", gen: 1}
	for i := 0; i < 3; i++ {
		if msg := fetchDisk(context.Background(), tagA); msg.err != nil || msg.disk.compact != "docker 40%" {
			t.Fatalf("msg=%+v", msg)
		}
	}
	if n := f.count("dc-df"); n != 1 {
		t.Fatalf("cached disk must not re-probe, dc-df calls=%d", n)
	}
	fetchDisk(context.Background(), tagB)
	if n := f.count("dc-df"); n != 2 {
		t.Fatalf("other engine must have its own entry, calls=%d", n)
	}
	m := model{engine: "context:a", workspace: "/tmp/app", hoverStack: -1}
	m, _ = m.explicitRefresh()
	fetchDisk(context.Background(), tagA)
	if n := f.count("dc-df"); n != 3 {
		t.Fatalf("r must invalidate the cache, calls=%d", n)
	}
	// TTL expiry.
	old := diskCacheTTL
	t.Cleanup(func() { diskCacheTTL = old })
	diskCacheTTL = 0
	fetchDisk(context.Background(), tagA)
	if n := f.count("dc-df"); n != 4 {
		t.Fatalf("expired entry must re-probe, calls=%d", n)
	}
}

func TestDiskErrorNotCached(t *testing.T) {
	f := defaultFake()
	f.errs["dc-df"] = errStr("exit status 1")
	installFakeProbes(t, f)
	tag := discoveryTag{engine: "context:a"}
	fetchDisk(context.Background(), tag)
	fetchDisk(context.Background(), tag)
	if n := f.count("dc-df"); n != 2 {
		t.Fatalf("errors must not be cached, calls=%d", n)
	}
}

func TestPruneInvalidatesDisk(t *testing.T) {
	f := defaultFake()
	installFakeProbes(t, f)
	tag := discoveryTag{engine: "context:a"}
	fetchDisk(context.Background(), tag)
	m := model{engine: "context:a", workspace: "/tmp/app", hoverStack: -1, busy: "dc-prune"}
	m, _ = m.applyStayDone(stayDoneMsg{name: "dc-prune", workspace: "/tmp/app", out: "reclaimed"})
	fetchDisk(context.Background(), tag)
	if n := f.count("dc-df"); n != 2 {
		t.Fatalf("prune must invalidate disk cache, calls=%d", n)
	}
}

func TestStayIsAsyncAndSingle(t *testing.T) {
	old := runStay
	t.Cleanup(func() { runStay = old })
	release := make(chan struct{})
	runStay = func(name string, args ...string) (string, error) {
		<-release
		return "done " + name, nil
	}
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1, loaded: true}
	got, cmd := m.handleKey("o")
	mm := got.(model)
	if cmd == nil || mm.busy != "dc-open" {
		t.Fatalf("stay must return immediately with busy, busy=%q", mm.busy)
	}
	if !strings.Contains(mm.status, "running dc-open") {
		t.Fatalf("status=%q", mm.status)
	}
	// Loop keeps working: keys are handled while the command runs.
	got2, _ := mm.handleKey("?")
	if !got2.(model).more {
		t.Fatal("board must stay responsive while a stay command runs")
	}
	got3, cmd3 := mm.handleKey("b")
	if cmd3 != nil || !strings.Contains(got3.(model).status, "still running") {
		t.Fatalf("second stay must wait, status=%q", got3.(model).status)
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	close(release)
	msg := (<-done).(stayDoneMsg)
	got, reload := mm.Update(msg)
	mm = got.(model)
	if mm.busy != "" || mm.status != "done dc-open" {
		t.Fatalf("busy=%q status=%q", mm.busy, mm.status)
	}
	if reload == nil || mm.load != loadPending {
		t.Fatal("finished stay must refresh")
	}
}

func TestStayResultForOldContextDoesNotReload(t *testing.T) {
	m := model{workspace: "/tmp/new", hoverStack: -1, busy: "dc-open", loaded: true}
	got, cmd := m.Update(stayDoneMsg{name: "dc-open", workspace: "/tmp/old", out: "opened"})
	mm := got.(model)
	if cmd != nil || mm.load == loadPending {
		t.Fatal("stay from another context must not reload this one")
	}
	if mm.status != "opened" || mm.busy != "" {
		t.Fatalf("status=%q busy=%q", mm.status, mm.busy)
	}
}

func TestRequestRefreshOneInFlightOnePending(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, loaded: true, load: loadReady, loadGen: 1}
	m, c1 := m.requestRefresh()
	if c1 == nil || m.loadGen != 2 {
		t.Fatalf("first refresh must start, gen=%d", m.loadGen)
	}
	m, c2 := m.requestRefresh()
	m, c3 := m.requestRefresh()
	if c2 != nil || c3 != nil || m.loadGen != 2 || !m.discoveryPending {
		t.Fatalf("refresh while in flight must queue once: gen=%d pending=%v", m.loadGen, m.discoveryPending)
	}
	m, next := m.handleReload(reloadMsg{gen: 2, workspace: "/tmp/app", rows: []container{{ID: "a"}}})
	if next == nil || m.loadGen != 3 || m.discoveryPending {
		t.Fatalf("queued refresh must run once after the in-flight one, gen=%d pending=%v", m.loadGen, m.discoveryPending)
	}
}

func TestPulseFromOtherContextDropped(t *testing.T) {
	m := model{workspace: "/tmp/new", engine: "context:a", hoverStack: -1}
	got, _ := m.Update(pulseMsg{workspace: "/tmp/old", engine: "context:a", line: "cpu 9%"})
	if got.(model).pulse != "" {
		t.Fatal("pulse for another workspace must be dropped")
	}
	got, _ = m.Update(pulseMsg{workspace: "/tmp/new", engine: "context:b", line: "cpu 9%"})
	if got.(model).pulse != "" {
		t.Fatal("pulse for another engine must be dropped")
	}
}

func TestStackExecUsesFreshMembership(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		stack:      []stackSvc{{ID: "app1", Service: "app"}, {ID: "db1", Name: "db-1", Service: "db"}},
	}
	cmd := stackExecCommand(m.workspace, m.stack[1])
	want := []string{"dc-exec", "--service", "db1", "/tmp/app"}
	if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("args=%v want %v", cmd.Args, want)
	}
	if cmd.SysProcAttr != nil {
		t.Fatal("user actions must not run in a probe process group")
	}
	var _ *exec.Cmd = cmd
}

func TestParseDisk(t *testing.T) {
	info, err := parseDisk([]byte(`{"compact":" docker 12% ","colima":{"guest_root":"80%"}}`))
	if err != nil || info.compact != "docker 12%" || info.guest != "80%" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if _, err := parseDisk([]byte(`nope`)); err == nil {
		t.Fatal("bad json must error")
	}
}

func TestFetchPortsAndNetsErrors(t *testing.T) {
	f := defaultFake()
	f.errs["docker ps"] = errStr("cannot connect")
	installFakeProbes(t, f)
	tag := discoveryTag{workspace: "/tmp/app"}
	if msg := fetchPorts(context.Background(), tag, "/tmp/app", nil); msg.err == nil {
		t.Fatal("ports error must surface")
	}
	if msg := fetchNetsSection(context.Background(), tag, "/tmp/app"); msg.err != nil || len(msg.nets.Networks) != 1 {
		t.Fatalf("nets=%+v err=%v", msg.nets, msg.err)
	}
}
