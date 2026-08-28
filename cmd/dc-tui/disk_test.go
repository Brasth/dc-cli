package main

import (
	"strings"
	"testing"
)

func TestDiskLooksCritical(t *testing.T) {
	if diskLooksCritical("img 12GB  cache 4GB", "") {
		t.Fatal("no percent should not be critical")
	}
	if !diskLooksCritical("", "12G used of 20G (91%)") {
		t.Fatal("91% guest must be critical")
	}
	if diskLooksCritical("docker 42%", "") {
		t.Fatal("42% is not critical")
	}
	if !diskLooksCritical("docker 85% · colima 61%", "") {
		t.Fatal("85% must be critical")
	}
}

func TestPruneConfirmKey(t *testing.T) {
	old := runStay
	t.Cleanup(func() { runStay = old })
	var gotName string
	var gotArgs []string
	runStay = func(name string, args ...string) (string, error) {
		gotName = name
		gotArgs = args
		return "pruned", nil
	}
	m := model{workspace: "/tmp/app", confirm: "prune", hoverStack: -1, loaded: true, load: loadReady}
	got, _ := m.handleConfirmKey("y")
	mm := got.(model)
	if gotName != "dc-prune" || len(gotArgs) != 1 || gotArgs[0] != "--yes" {
		t.Fatalf("prune y ran %s %v", gotName, gotArgs)
	}
	if mm.confirm != "" {
		t.Fatalf("confirm leftover %q", mm.confirm)
	}
}

func TestPruneKeyRequiresCritical(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1}
	got, _ := m.handleKey("P")
	mm := got.(model)
	if mm.confirm == "prune" {
		t.Fatal("P must not prune when disk is ok")
	}
	if !strings.Contains(mm.status, "disk ok") {
		t.Fatalf("status=%q", mm.status)
	}
	m.diskCritical = true
	got, _ = m.handleKey("P")
	mm = got.(model)
	if mm.confirm != "prune" {
		t.Fatalf("confirm=%q", mm.confirm)
	}
}

func TestViewShowsCriticalPruneHint(t *testing.T) {
	m := model{
		workspace:    "/tmp/app",
		hasConfig:    true,
		width:        80,
		hoverStack:   -1,
		editor:       "zed",
		disk:         "img 12GB",
		diskCritical: true,
		load:         loadReady,
		loaded:       true,
	}
	s := m.View()
	if !strings.Contains(s, "CRITICAL") || !strings.Contains(s, "P=prune") {
		t.Fatalf("missing critical hint:\n%s", s)
	}
}
