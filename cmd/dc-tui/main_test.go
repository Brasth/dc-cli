package main

import (
	"strings"
	"testing"
)

func TestRenderButtonsHitboxesCoverGlyphs(t *testing.T) {
	line, buttons := renderButtons(false, 80, "", 3)
	if len(buttons) < 8 {
		t.Fatalf("expected many buttons, got %d", len(buttons))
	}
	for _, b := range buttons {
		if b.x1 <= b.x0 || b.y1 <= b.y0 {
			t.Fatalf("empty hitbox %+v", b)
		}
		if b.y1-b.y0 < 2 {
			t.Fatalf("hitbox too short (need padding): %+v", b)
		}
	}
	if !strings.Contains(line, "start") || !strings.Contains(line, "shell") {
		t.Fatalf("missing labels in:\n%s", line)
	}
}

func TestRenderButtonsWraps(t *testing.T) {
	_, wide := renderButtons(false, 200, "", 0)
	_, narrow := renderButtons(false, 40, "", 0)
	if len(wide) == 0 || len(narrow) == 0 {
		t.Fatal("no buttons")
	}
	maxY := 0
	for _, b := range wide {
		if b.y0 > maxY {
			maxY = b.y0
		}
	}
	maxYn := 0
	for _, b := range narrow {
		if b.y0 > maxYn {
			maxYn = b.y0
		}
	}
	if maxYn <= maxY {
		t.Fatalf("narrow should wrap to more rows: wide y0=%d narrow y0=%d", maxY, maxYn)
	}
}

func TestBenignExecErr(t *testing.T) {
	if !benignExecErr(nil) {
		t.Fatal("nil should be benign")
	}
	if !benignExecErr(errStr("exit status 1")) {
		t.Fatal("exit 1 (shell) should resume TUI")
	}
	if !benignExecErr(errStr("could not restore terminal: something")) {
		t.Fatal("restore-terminal should resume")
	}
	if benignExecErr(errStr("dc-ls: command not found")) {
		t.Fatal("real errors must still surface")
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
