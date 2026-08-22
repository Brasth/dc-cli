package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestParseUpdateCheck(t *testing.T) {
	msg := parseUpdateCheck("installed: 0.1.0\nlatest:    0.2.0\nchannel:   source\nstate:     available\n")
	if !msg.available || msg.installed != "0.1.0" || msg.latest != "0.2.0" {
		t.Fatalf("%+v", msg)
	}
	msg = parseUpdateCheck("installed: 0.2.0\nlatest:    0.2.0\nchannel:   source\nstate:     current\n")
	if msg.available {
		t.Fatalf("current should not be available: %+v", msg)
	}
}

func TestUpdateBanner(t *testing.T) {
	m := model{updateAvail: true, updateInstalled: "0.1.0", updateLatest: "0.2.0"}
	got := m.updateBanner()
	if !strings.Contains(got, "0.1.0") || !strings.Contains(got, "0.2.0") || !strings.Contains(got, "U") {
		t.Fatalf("banner=%q", got)
	}
	if (model{}).updateBanner() != "" {
		t.Fatal("empty when no update")
	}
}

func TestHeaderShowsUpdateBanner(t *testing.T) {
	t.Setenv("DC_CLI_VERSION", "0.1.0")
	m := model{
		workspace:       "/tmp/app",
		hasConfig:       true,
		width:           100,
		hoverStack:      -1,
		editor:          "zed",
		updateAvail:     true,
		updateInstalled: "0.1.0",
		updateLatest:    "0.2.0",
	}
	s := ansi.Strip(m.View())
	if !strings.Contains(s, "update 0.1.0 → 0.2.0") {
		t.Fatalf("missing banner:\n%s", s)
	}
}

func TestUpgradeKeyConfirms(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, updateAvail: true, updateLatest: "0.2.0"}
	got, cmd := m.handleKey("U")
	mm := got.(model)
	if cmd != nil {
		t.Fatal("U should only enter confirm")
	}
	if mm.confirm != "upgrade" {
		t.Fatalf("confirm=%q", mm.confirm)
	}
	if mm.confirmAction() != "dc-upgrade --yes" {
		t.Fatalf("confirmAction=%q", mm.confirmAction())
	}
}

func TestUpgradeKeyWithoutUpdate(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1}
	got, _ := m.handleKey("U")
	mm := got.(model)
	if mm.confirm != "" {
		t.Fatalf("confirm=%q", mm.confirm)
	}
	if !strings.Contains(mm.status, "no update") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestUpgradeConfirmYesLeaves(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, confirm: "upgrade", updateLatest: "0.2.0"}
	got, cmd := m.handleConfirmKey("y")
	mm := got.(model)
	if mm.confirm != "" {
		t.Fatalf("confirm=%q", mm.confirm)
	}
	if mm.leaving != "upgrade" || mm.pending != "upgrade" {
		t.Fatalf("leaving=%q pending=%q", mm.leaving, mm.pending)
	}
	if cmd == nil {
		t.Fatal("expected leave tick")
	}
}

func TestUpgradeConfirmCancel(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, confirm: "upgrade", updateLatest: "0.2.0"}
	got, cmd := m.handleConfirmKey("n")
	mm := got.(model)
	if cmd != nil || mm.confirm != "" {
		t.Fatal("n must cancel")
	}
	if !strings.Contains(mm.status, "cancelled") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestUpgradeDoneQuits(t *testing.T) {
	m := model{workspace: "/tmp/app", hoverStack: -1, leaving: "upgrade"}
	got, cmd := m.Update(execDoneMsg{action: "upgrade"})
	mm := got.(model)
	if !mm.quitting {
		t.Fatal("upgrade done must quit")
	}
	if cmd == nil {
		t.Fatal("expected Quit")
	}
}
