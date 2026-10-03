package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Inspect counts measured on the pre-batch shell discovery (fake-docker,
// commit fd01797). The batched snapshot must stay at or under half of these.
var inspectBaseline = map[string]map[int]int{
	"devcontainer-stack": {1: 17, 10: 52, 50: 252},
	"compose-stack":      {1: 6, 10: 60, 50: 300},
	"compose-ls":         {1: 5, 10: 50, 50: 250},
}

var benchSizes = []int{1, 10, 50}

// BenchmarkEssentials measures board-side essential discovery (fake probes
// with fixed latency) and reports probe launches per discovery.
func BenchmarkEssentials(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("containers=%d", n), func(b *testing.B) {
			f := defaultFake()
			f.out["dc-ls"] = rowsJSON(1)
			f.out["dc-exec"] = stackJSON(n)
			f.delay["dc-ls"] = time.Millisecond
			f.delay["dc-exec"] = time.Millisecond
			installFakeProbes(b, f)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				msg := runEssentials(context.Background(), "/tmp/app", false, 1)
				if msg.err != nil || len(msg.stack) != n {
					b.Fatalf("err=%v stack=%d", msg.err, len(msg.stack))
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(len(f.calls))/float64(b.N), "probes/op")
		})
	}
}

// shellFixture is a fake-docker state with one workspace of n containers.
type shellFixture struct {
	root  string
	state string
	ws    string
	env   []string
}

func repoRoot(tb testing.TB) string {
	tb.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// bash4 finds a Bash >= 4 (mapfile). macOS /bin/bash is 3.2.
func bash4(tb testing.TB) string {
	tb.Helper()
	for _, cand := range []string{"bash", "/opt/homebrew/bin/bash", "/usr/local/bin/bash", "/bin/bash", "/usr/bin/bash"} {
		p, err := exec.LookPath(cand)
		if err != nil {
			continue
		}
		out, err := exec.Command(p, "-c", "echo ${BASH_VERSINFO[0]}").Output()
		if major, convErr := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && convErr == nil && major >= 4 {
			return p
		}
	}
	tb.Skip("needs bash >= 4 for the shell discovery fixture")
	return ""
}

func newShellFixture(tb testing.TB, kind string, n int) *shellFixture {
	tb.Helper()
	bash := bash4(tb)
	root := repoRoot(tb)
	state := tb.TempDir()
	bin := filepath.Join(state, "bin")
	must := func(err error) {
		if err != nil {
			tb.Fatal(err)
		}
	}
	must(os.MkdirAll(bin, 0o755))
	must(os.Symlink(filepath.Join(root, "tests", "lib", "fake-docker"), filepath.Join(bin, "docker")))
	// Put the chosen bash first so `#!/usr/bin/env bash` resolves to it.
	must(os.Symlink(bash, filepath.Join(bin, "bash")))
	ws := filepath.Join(state, "ws-"+kind)
	must(os.MkdirAll(ws, 0o755))
	add := func(id, name string, labels ...string) {
		dir := filepath.Join(state, "containers", id)
		must(os.MkdirAll(dir, 0o755))
		must(os.WriteFile(filepath.Join(dir, "name"), []byte(name+"\n"), 0o644))
		must(os.WriteFile(filepath.Join(dir, "status"), []byte("running\n"), 0o644))
		must(os.WriteFile(filepath.Join(dir, "image"), []byte("alpine\n"), 0o644))
		must(os.WriteFile(filepath.Join(dir, "labels"), []byte(strings.Join(labels, "\n")+"\n"), 0o644))
	}
	switch kind {
	case "devcontainer":
		must(os.MkdirAll(filepath.Join(ws, ".devcontainer"), 0o755))
		must(os.WriteFile(filepath.Join(ws, ".devcontainer", "devcontainer.json"), []byte("{}\n"), 0o644))
		add("app1", "app-1", "devcontainer.local_folder="+ws, "com.docker.compose.project=perf", "com.docker.compose.service=app")
		for i := 2; i <= n; i++ {
			add(fmt.Sprintf("svc%d", i), fmt.Sprintf("svc-%d", i), "com.docker.compose.project=perf", fmt.Sprintf("com.docker.compose.service=svc%d", i))
		}
	case "compose":
		must(os.WriteFile(filepath.Join(ws, "compose.yaml"), []byte("services:\n  app:\n    image: alpine\n"), 0o644))
		proj := filepath.Base(ws)
		for i := 1; i <= n; i++ {
			add(fmt.Sprintf("c%d", i), fmt.Sprintf("c-%d", i), "com.docker.compose.project="+proj,
				fmt.Sprintf("com.docker.compose.service=s%d", i), "com.docker.compose.project.working_dir="+ws)
		}
	}
	env := append(os.Environ(),
		"DC_FAKE_STATE="+state,
		"FAKE_DOCKER_LOG="+filepath.Join(state, "docker.log"),
		"PATH="+bin+":"+filepath.Join(root, "bin")+":"+os.Getenv("PATH"),
		"DOCKER_HOST=",
	)
	return &shellFixture{root: root, state: state, ws: ws, env: env}
}

// run executes a wrapper and returns (inspect calls, docker calls, latency).
func (f *shellFixture) run(tb testing.TB, name string, args ...string) (int, int, time.Duration) {
	tb.Helper()
	log := filepath.Join(f.state, "docker.log")
	_ = os.WriteFile(log, nil, 0o644)
	cmd := exec.Command(filepath.Join(f.root, "bin", name), args...)
	cmd.Env = f.env
	start := time.Now()
	if out, err := cmd.CombinedOutput(); err != nil {
		tb.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	took := time.Since(start)
	b, _ := os.ReadFile(log)
	inspects, calls := 0, 0
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		calls++
		if strings.HasPrefix(line, "inspect") {
			inspects++
		}
	}
	return inspects, calls, took
}

// TestShellDiscoveryInspectBudget records latency + call counts at 1/10/50
// containers and holds stack discovery to ≤ 50% of the pre-batch inspects.
func TestShellDiscoveryInspectBudget(t *testing.T) {
	if testing.Short() || os.Getenv("DC_SHELL_PERF") != "1" {
		t.Skip("shell fixture: DC_SHELL_PERF=1 (tests/performance/run.sh is the canonical suite)")
	}
	for _, n := range benchSizes {
		for _, c := range []struct {
			key, kind, bin string
			args           func(ws string) []string
		}{
			{"devcontainer-stack", "devcontainer", "dc-exec", func(ws string) []string { return []string{"--list", "--json", ws} }},
			{"compose-stack", "compose", "dc-exec", func(ws string) []string { return []string{"--list", "--json", ws} }},
			{"compose-ls", "compose", "dc-ls", func(ws string) []string { return []string{"--json", ws} }},
		} {
			fx := newShellFixture(t, c.kind, n)
			inspects, calls, took := fx.run(t, c.bin, c.args(fx.ws)...)
			base := inspectBaseline[c.key][n]
			t.Logf("%-19s n=%-3d inspects=%-3d (baseline %d) docker-calls=%-3d latency=%s", c.key, n, inspects, base, calls, took.Round(time.Millisecond))
			if inspects*2 > base {
				t.Fatalf("%s n=%d: %d inspects > 50%% of baseline %d", c.key, n, inspects, base)
			}
		}
	}
}

// BenchmarkShellStackDiscovery is the shell side at 1/10/50 containers.
func BenchmarkShellStackDiscovery(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("containers=%d", n), func(b *testing.B) {
			fx := newShellFixture(b, "devcontainer", n)
			var inspects, calls int
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ins, c, _ := fx.run(b, "dc-exec", "--list", "--json", fx.ws)
				inspects += ins
				calls += c
			}
			b.ReportMetric(float64(inspects)/float64(b.N), "inspects/op")
			b.ReportMetric(float64(calls)/float64(b.N), "docker-calls/op")
		})
	}
}
