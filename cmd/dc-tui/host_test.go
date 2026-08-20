package main

import (
	"strings"
	"testing"
)

func TestHostViewShowsSetup(t *testing.T) {
	detail := "Cannot connect to the Docker daemon"
	m := model{
		width:     80,
		hostBlock: true,
		host: hostReport{
			Code:        "docker_engine_stopped",
			Summary:     "Docker engine is not running",
			Detail:      &detail,
			EngineHint:  "desktop",
			GuideURL:    "https://docs.docker.com/desktop/",
			Remediation: "Start Docker Desktop",
		},
	}
	s := m.View()
	for _, want := range []string{
		"Docker setup required",
		"Docker engine is not running",
		"Start Docker Desktop",
		"[d] Desktop guide",
		"[c] copy Colima setup",
		"[r] check again",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}

func TestHostKeyRetryClearsViaReloadMsg(t *testing.T) {
	old := runHostDiagnose
	t.Cleanup(func() { runHostDiagnose = old })
	runHostDiagnose = func() (hostReport, error) {
		return hostReport{Code: "ready", Status: "ok", Summary: "Docker engine reachable"}, nil
	}

	m := model{
		workspace: "/tmp/app",
		hostBlock: true,
		host:      hostReport{Code: "docker_cli_missing", Summary: "docker is not on PATH"},
		load:      loadFailed,
		loadGen:   1,
		hoverStack: -1,
	}
	got, cmd := m.handleKey("r")
	mm := got.(model)
	if mm.status != "checking Docker…" {
		t.Fatalf("status=%q", mm.status)
	}
	if cmd == nil {
		t.Fatal("expected reload cmd")
	}
}

func TestHostKeyQuit(t *testing.T) {
	m := model{hostBlock: true, host: hostReport{Code: "docker_cli_missing"}}
	got, cmd := m.handleKey("q")
	mm := got.(model)
	if !mm.quitting || cmd == nil {
		t.Fatal("q must quit")
	}
}

func TestApplyReloadHostBlock(t *testing.T) {
	m := model{workspace: "/tmp/app", loadGen: 2, load: loadPending, hoverStack: -1}
	m = m.applyReload(reloadMsg{
		gen:       2,
		workspace: "/tmp/app",
		host: hostReport{
			Code:    "docker_engine_missing",
			Summary: "No Docker engine is installed",
		},
	})
	if !m.hostBlock || m.host.Code != "docker_engine_missing" {
		t.Fatalf("hostBlock=%v code=%q", m.hostBlock, m.host.Code)
	}
	if m.load != loadFailed {
		t.Fatalf("load=%v", m.load)
	}
}

func TestApplyReloadReadyClearsHostBlock(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		loadGen:    3,
		load:       loadPending,
		hostBlock:  true,
		host:       hostReport{Code: "docker_cli_missing"},
		hoverStack: -1,
	}
	m = m.applyReload(reloadMsg{
		gen:       3,
		workspace: "/tmp/app",
		host:      hostReport{Code: "ready", Status: "ok"},
		rows:      []container{},
	})
	if m.hostBlock {
		t.Fatal("ready must clear hostBlock")
	}
	if m.load != loadReady {
		t.Fatalf("load=%v", m.load)
	}
}
