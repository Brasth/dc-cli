package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestNetRefusesFleet(t *testing.T) {
	got, cmd := model{fleet: true, hoverStack: -1}.handleKey("n")
	mm := got.(model)
	if cmd != nil || mm.netOpen {
		t.Fatal("fleet n must not open nets")
	}
	if mm.status != "nets is this folder only." {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestNetOpensAndQuits(t *testing.T) {
	old := runNet
	t.Cleanup(func() { runNet = old })
	runNet = func(args ...string) ([]byte, error) {
		return []byte(`{"schemaVersion":1,"command":"dc-net","workspace":"/tmp/app","networks":[{"name":"shared","kind":"external","present":false,"creatable":true,"driver":"","reason":"missing"}],"missingCreatable":["shared"],"missingBlocked":[]}`), nil
	}
	m := model{workspace: "/tmp/app", hoverStack: -1}
	got, cmd := m.handleKey("n")
	mm := got.(model)
	if !mm.netOpen {
		t.Fatal("n must open nets")
	}
	if cmd == nil {
		t.Fatal("expected fetchNets")
	}
	got, _ = mm.Update(netMsg{rep: netReport{
		Networks:         []netRow{{Name: "shared", Kind: "external", Creatable: true, Reason: "missing"}},
		MissingCreatable: []string{"shared"},
	}})
	mm = got.(model)
	s := ansi.Strip(mm.View())
	if !strings.Contains(s, "shared") {
		t.Fatalf("missing net name:\n%s", s)
	}
	if !strings.Contains(s, "create missing") {
		t.Fatalf("missing create prompt:\n%s", s)
	}
	got, _ = mm.handleKey("q")
	mm = got.(model)
	if mm.netOpen {
		t.Fatal("q must close nets")
	}
}

func TestNetYesStartsCreateNets(t *testing.T) {
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		hoverStack: -1,
		netOpen:    true,
		net: netReport{
			MissingCreatable: []string{"shared"},
		},
	}
	got, cmd := m.handleKey("y")
	mm := got.(model)
	if mm.netOpen {
		t.Fatal("y should close overlay")
	}
	if mm.pending != "create-nets" {
		t.Fatalf("pending=%q", mm.pending)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
}

func TestNetNDoesNotCreate(t *testing.T) {
	m := model{
		hoverStack: -1,
		netOpen:    true,
		net:        netReport{MissingCreatable: []string{"shared"}},
	}
	got, cmd := m.handleKey("n")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("n must not start")
	}
	if mm.netOpen {
		t.Fatal("n must close overlay")
	}
	if mm.pending != "" {
		t.Fatalf("pending=%q", mm.pending)
	}
}

func TestConfirmRmNStillCancels(t *testing.T) {
	m := model{workspace: "/tmp/app", hasConfig: true, hoverStack: -1, confirm: "rm"}
	got, cmd := m.handleKey("n")
	mm := got.(model)
	if cmd != nil || mm.confirm != "" {
		t.Fatal("rm confirm n must cancel")
	}
	if mm.netOpen {
		t.Fatal("rm cancel must not open nets")
	}
}

func TestMoreDocumentsNets(t *testing.T) {
	more := morePanel("zed", 120)
	if !strings.Contains(more, "nets") {
		t.Fatalf("more must document nets:\n%s", more)
	}
}
