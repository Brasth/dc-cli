package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// idleEventProcess is the package-wide test default: a stream that never
// emits and ends when its context is cancelled.
func idleEventProcess(ctx context.Context, engine string) (io.Reader, func() error, error) {
	if _, ok := engineArgs(engine); !ok {
		return nil, nil, errEngineUnknown
	}
	pr, pw := io.Pipe()
	go func() {
		<-ctx.Done()
		_ = pw.Close()
	}()
	return pr, func() error { return nil }, nil
}

func fullID(c byte) string { return strings.Repeat(string(c), 64) }

func evLine(id, action string, attrs map[string]string) string {
	if attrs == nil {
		attrs = map[string]string{}
	}
	b, _ := json.Marshal(map[string]any{
		"Type":     "container",
		"Action":   action,
		"Actor":    map[string]any{"ID": id, "Attributes": attrs},
		"scope":    "local",
		"time":     time.Now().Unix(),
		"timeNano": time.Now().UnixNano(),
	})
	return string(b)
}

func TestParseActivityEvent(t *testing.T) {
	id := fullID('a')
	ev, ok := parseActivityEvent([]byte(evLine(id, "die", map[string]string{
		"name": "/app-1", "exitCode": "137", "com.docker.compose.service": "web",
		"SECRET_TOKEN": "hunter2", "image": "node:22",
	})))
	if !ok || ev.id != id || ev.action != "die" || ev.exit != "137" || ev.name != "app-1" || ev.service != "web" {
		t.Fatalf("ev=%+v ok=%v", ev, ok)
	}
	if strings.Contains(fmt.Sprintf("%+v", ev), "hunter2") {
		t.Fatal("event must not retain arbitrary attributes")
	}
	if describeEvent(ev) != "exited (code 137)" {
		t.Fatalf("desc=%q", describeEvent(ev))
	}
	hs, ok := parseActivityEvent([]byte(evLine(id, "health_status: unhealthy", nil)))
	if !ok || hs.action != "health_status" || hs.health != "unhealthy" {
		t.Fatalf("health=%+v", hs)
	}
	odd, _ := parseActivityEvent([]byte(evLine(id, "health_status: $(rm -rf)", nil)))
	if odd.health != "changed" {
		t.Fatalf("unknown health text must not be kept: %q", odd.health)
	}
	side, _ := parseActivityEvent([]byte(evLine(id, "start", map[string]string{"dc.forward.for": "x"})))
	if !side.sidecar {
		t.Fatal("dc.forward.for must mark a sidecar")
	}
	bad := []string{
		`not json`,
		`{"Type":"network","Action":"create","Actor":{"ID":"` + id + `"},"time":1}`,
		`{"Type":"container","Action":"exec_start: sh","Actor":{"ID":"` + id + `"},"time":1}`,
		`{"Type":"container","Action":"start","Actor":{"ID":"xyz"},"time":1}`,
		`{"Type":"container","Action":"start","Actor":{"ID":"` + id + `"}}`,
		`{"Type":"container","Action":"start","Actor":{"ID":"` + id + `/../x"},"time":1}`,
		``,
	}
	for _, l := range bad {
		if _, ok := parseActivityEvent([]byte(l)); ok {
			t.Errorf("must ignore %q", l)
		}
	}
}

func TestReadEventLinesSkipsMalformedAndOverlong(t *testing.T) {
	id := fullID('b')
	huge := `{"Type":"container","Action":"start","Actor":{"ID":"` + id + `","Attributes":{"x":"` + strings.Repeat("z", activityLineMax+10) + `"}},"time":1}`
	input := strings.Join([]string{"garbage", huge, "{\"half\":", evLine(id, "start", nil), evLine(id, "stop", nil)}, "\n") + "\n"
	out := make(chan activityEvent, 10)
	if err := readEventLines(context.Background(), strings.NewReader(input), out); err != nil {
		t.Fatal(err)
	}
	close(out)
	var got []string
	for ev := range out {
		got = append(got, ev.action)
	}
	if strings.Join(got, ",") != "start,stop" {
		t.Fatalf("got %v", got)
	}
}

func TestEngineArgsPinning(t *testing.T) {
	args, ok := eventsArgs("context:colima")
	if !ok || args[0] != "--context" || args[1] != "colima" || args[2] != "events" {
		t.Fatalf("args=%v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "--filter event=health_status") {
		t.Fatalf("filters=%v", args)
	}
	if args, ok := engineArgs("host:unix:///x.sock"); !ok || strings.Join(args, " ") != "--host unix:///x.sock" {
		t.Fatalf("host engine must be pinned by --host: %v", args)
	}
	for _, e := range []string{"", "context:", "host:", "weird"} {
		if _, ok := engineArgs(e); ok {
			t.Errorf("%q must not be pinnable", e)
		}
	}
}

// fakeDockerScript records argv + its pid and a grandchild pid, prints one
// event, then blocks like `docker events`.
const fakeDockerScript = `#!/bin/sh
dir="$(dirname "$0")"
printf '%s\n' "$@" > "$dir/args.$$"
echo "${DOCKER_HOST:-}|${DOCKER_CONTEXT:-}" > "$dir/env.$$"
sleep 60 &
echo $! > "$dir/grandchild.$$"
echo $$ > "$dir/pid.$$"
printf '%s\n' '{"Type":"container","Action":"start","Actor":{"ID":"` + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" + `"},"time":1}'
wait
`

// installFakeDocker puts a fake docker first in PATH and the real default
// process starter in place. Returns the directory with the pid files.
func installFakeDocker(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(fakeDockerScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOCKER_CONTEXT", "should-be-dropped")
	old := startEventProcess
	startEventProcess = defaultStartEventProcess
	t.Cleanup(func() { startEventProcess = old })
	return dir
}

// fakeDockerPids waits for n started fake docker processes and returns
// (pid, grandchild) pairs.
func fakeDockerPids(t *testing.T, dir string, n int) [][2]int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join(dir, "pid.*"))
		if len(matches) >= n {
			var out [][2]int
			for _, p := range matches {
				suffix := strings.TrimPrefix(filepath.Base(p), "pid.")
				pid := waitPidFile(t, p)
				gc := waitPidFile(t, filepath.Join(dir, "grandchild."+suffix))
				out = append(out, [2]int{pid, gc})
			}
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %d fake docker processes", n)
	return nil
}

func assertReaped(t *testing.T, pids [][2]int) {
	t.Helper()
	for _, p := range pids {
		for _, pid := range p {
			if !pidGone(pid) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
				t.Fatalf("process %d survived the stream close", pid)
			}
		}
	}
}

func TestRealStreamPinnedAndReapedOnClose(t *testing.T) {
	dir := installFakeDocker(t)
	s, err := openActivityStream(context.Background(), "context:colima")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-s.events:
		if ev.action != "start" {
			t.Fatalf("ev=%+v", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no event from the fake stream")
	}
	pids := fakeDockerPids(t, dir, 1)
	suffix := fmt.Sprint(pids[0][0])
	args, _ := os.ReadFile(filepath.Join(dir, "args."+suffix))
	if !strings.HasPrefix(string(args), "--context\ncolima\nevents\n") {
		t.Fatalf("stream not pinned: %q", args)
	}
	env, _ := os.ReadFile(filepath.Join(dir, "env."+suffix))
	if strings.TrimSpace(string(env)) != "|" {
		t.Fatalf("context engine must not see DOCKER_HOST/DOCKER_CONTEXT: %q", env)
	}
	start := time.Now()
	s.close()
	if time.Since(start) > activityStopWait {
		t.Fatal("close must be bounded")
	}
	select {
	case <-s.done:
	default:
		t.Fatal("close must reap the process")
	}
	assertReaped(t, pids)
}

// A host engine's inspect is pinned by --host; an inherited DOCKER_CONTEXT
// (or a different DOCKER_HOST) never reaches the docker CLI.
func TestPinnedInspectIgnoresInheritedEngineEnv(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$(dirname \"$0\")/args\"\necho \"${DOCKER_HOST:-}|${DOCKER_CONTEXT:-}\" > \"$(dirname \"$0\")/env\"\necho '[]'\n"
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOCKER_CONTEXT", "desktop-linux")
	t.Setenv("DOCKER_HOST", "unix:///other.sock")
	for _, tc := range []struct{ engine, want string }{
		{"host:unix:///pinned.sock", "--host\nunix:///pinned.sock\ninspect\n"},
		{"context:colima", "--context\ncolima\ninspect\n"},
	} {
		if _, err := inspectContainers(context.Background(), tc.engine, []string{fullID('a')}); err != nil {
			t.Fatal(err)
		}
		args, _ := os.ReadFile(filepath.Join(dir, "args"))
		if !strings.HasPrefix(string(args), tc.want) {
			t.Fatalf("%s: args=%q", tc.engine, args)
		}
		env, _ := os.ReadFile(filepath.Join(dir, "env"))
		if strings.TrimSpace(string(env)) != "|" {
			t.Fatalf("%s: inherited engine env leaked: %q", tc.engine, env)
		}
	}
	if _, err := inspectContainers(context.Background(), "", []string{fullID('a')}); err != errEngineUnknown {
		t.Fatalf("unknown engine must not inspect: %v", err)
	}
}

func TestRealStreamEndReportsError(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho 'Cannot connect to the Docker daemon' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := startEventProcess
	startEventProcess = defaultStartEventProcess
	t.Cleanup(func() { startEventProcess = old })
	s, err := openActivityStream(context.Background(), "host:unix:///nope.sock")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
		t.Fatal("ended stream must finish")
	}
	if s.err == nil || s.err == errProbeCancelled {
		t.Fatalf("err=%v", s.err)
	}
}

// The board's own lifecycle paths must kill + reap the real stream process
// tree (fake docker + its grandchild), not just drop the reader.
func TestBoardLifecycleReapsRealStream(t *testing.T) {
	cases := []struct {
		name string
		end  func(model) model
	}{
		{"switch", func(m model) model { m, _ = m.switchContext(t.TempDir(), false); return m }},
		{"fleet", func(m model) model { m, _ = m.switchContext(m.workspace, true); return m }},
		{"leave", func(m model) model { m, _ = m.startLeave("shell", "e"); return m }},
		{"quit", func(m model) model { m, _ = m.quit(); return m }},
		{"engine", func(m model) model {
			m.engine = "context:other"
			m, _ = m.syncActivity("context:colima")
			return m
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := installFakeDocker(t)
			installFakeProbes(t, defaultFake())
			ws := devWorkspace(t)
			m := model{workspace: ws, hasConfig: true, hoverStack: -1, load: loadReady, loaded: true, loadGen: 1,
				engine: "context:colima", probes: newProbeSession()}
			m, _ = m.syncActivity("")
			if m.activity.stream == nil {
				t.Fatal("stream must start")
			}
			first := fakeDockerPids(t, dir, 1)
			m = tc.end(m)
			assertReaped(t, first)
			if tc.name == "engine" {
				if m.activity.stream == nil || m.activity.sess.engine != "context:other" {
					t.Fatal("engine change must start a stream pinned to the new engine")
				}
			} else if m.activity.stream != nil {
				t.Fatal("no stream may survive " + tc.name)
			}
			m.shutdown()
			assertReaped(t, fakeDockerPids(t, dir, 1))
		})
	}
}

// TestLiveActivity runs only from tests/workspace-hub-smoke/run.sh against a
// real engine and a compose-kind workspace that script created. It creates
// extra containers only with the run's dc.smoke.run label (the script
// removes them) and never touches anything else.
func TestLiveActivity(t *testing.T) {
	ws, run, image := os.Getenv("DC_LIVE_ACTIVITY_WS"), os.Getenv("DC_LIVE_RUN"), os.Getenv("DC_LIVE_IMAGE")
	if ws == "" || run == "" || image == "" {
		t.Skip("live smoke only (tests/workspace-hub-smoke/run.sh)")
	}
	old := startEventProcess
	startEventProcess = defaultStartEventProcess
	t.Cleanup(func() { startEventProcess = old })
	oldD := activityDebounceDelay
	activityDebounceDelay = 300 * time.Millisecond
	t.Cleanup(func() { activityDebounceDelay = oldD })

	docker := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	sandbox := os.Getenv("DC_LIVE_ACTIVITY_KIND") == "none"
	m := model{workspace: ws, hasCompose: !sandbox, hoverStack: -1, width: 100, height: 40,
		load: loadPending, loadGen: 1, probes: newProbeSession()}
	p := newPump(t, m)
	p.run(p.m.reloadCmd())
	p.until("live stream", func(m model) bool { return m.loaded && m.activity.stream != nil })
	if p.m.engine == "" || p.m.activity.sess.engine != p.m.engine {
		t.Fatalf("stream must be pinned to the snapshot engine: %q / %+v", p.m.engine, p.m.activity.sess)
	}
	waitEntry := func(what string) {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for !hasEntry(p.m, what) {
			if time.Now().After(deadline) {
				t.Fatalf("no %q in %v", what, entryTexts(p.m))
			}
			p.settle(50 * time.Millisecond)
		}
	}
	label := "dc.smoke.run=" + run
	if sandbox {
		// kind=none: the labeled sandbox app is the only member.
		if len(p.m.rows) != 1 {
			t.Fatalf("sandbox rows=%+v", p.m.rows)
		}
		appRow := p.m.rows[0]
		docker("restart", "-t", "1", appRow.ID)
		waitEntry(strings.TrimPrefix(appRow.Name, "/") + " restarted")
		foreign := docker("run", "-d", "--label", label, "--label", "devcontainer.local_folder="+t.TempDir(),
			"--name", run+"-foreign", image, "sleep", "300")
		p.settle(2 * time.Second)
		docker("rm", "-f", foreign)
		p.settle(500 * time.Millisecond)
		for _, e := range entryTexts(p.m) {
			if strings.HasPrefix(e, run+"-foreign") {
				t.Fatalf("another folder's labeled app must be excluded: %v", entryTexts(p.m))
			}
		}
		t.Logf("live timeline: %v", entryTexts(p.m))
		s := p.m.activity.stream
		p.do(func(m model) (model, tea.Cmd) { return m.switchContext(t.TempDir(), false) })
		select {
		case <-s.done:
		default:
			t.Fatal("switch must reap the docker events process")
		}
		return
	}
	if len(p.m.stack) == 0 {
		t.Fatal("discovery must list the smoke stack")
	}
	app := p.m.stack[0]
	proj := docker("inspect", "-f", `{{index .Config.Labels "com.docker.compose.project"}}`, app.ID)
	wd := docker("inspect", "-f", `{{index .Config.Labels "com.docker.compose.project.working_dir"}}`, app.ID)

	// Known member: restart is attributed.
	docker("restart", "-t", "1", app.ID)
	waitEntry(app.Service + " restarted")

	// New legit sibling (same project + working_dir) vs forged (same project,
	// working_dir elsewhere) vs a dc-cli port sidecar.
	extra := docker("run", "-d", "--label", label, "--label", "com.docker.compose.project="+proj,
		"--label", "com.docker.compose.service=extra", "--label", "com.docker.compose.project.working_dir="+wd,
		image, "sleep", "300")
	forged := docker("run", "-d", "--label", label, "--label", "com.docker.compose.project="+proj,
		"--label", "com.docker.compose.service=forged", "--label", "com.docker.compose.project.working_dir=/tmp",
		image, "sleep", "300")
	side := docker("run", "-d", "--label", label, "--label", "com.docker.compose.project="+proj,
		"--label", "com.docker.compose.service=side", "--label", "com.docker.compose.project.working_dir="+wd,
		"--label", "dc.forward.for="+app.ID, image, "sleep", "300")
	waitEntry("extra started")
	docker("rm", "-f", extra, forged, side)
	waitEntry("extra removed")
	p.settle(500 * time.Millisecond)
	for _, e := range entryTexts(p.m) {
		if strings.HasPrefix(e, "forged") || strings.HasPrefix(e, "side") {
			t.Fatalf("forged / sidecar must be excluded: %v", entryTexts(p.m))
		}
	}
	if len(p.m.activity.entries) > activityCap {
		t.Fatal("bounded")
	}

	t.Logf("live timeline: %v", entryTexts(p.m))
	// Switching away closes and reaps the real stream.
	s := p.m.activity.stream
	p.do(func(m model) (model, tea.Cmd) { return m.switchContext(t.TempDir(), false) })
	select {
	case <-s.done:
	default:
		t.Fatal("switch must reap the docker events process")
	}
}
