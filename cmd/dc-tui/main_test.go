package main

import (
	"strings"
	"testing"
)

func TestRenderButtonsEqualWidth(t *testing.T) {
	_, buttons := renderButtons(false, 80, "", 0)
	if len(buttons) < 8 {
		t.Fatalf("expected many buttons, got %d", len(buttons))
	}
	w0 := buttons[0].x1 - buttons[0].x0
	if w0 < 6 {
		t.Fatalf("tile too narrow: %d", w0)
	}
	for _, b := range buttons {
		if b.x1-b.x0 != w0 {
			t.Fatalf("uneven tile %+v want width %d", b, w0)
		}
		if b.y1 <= b.y0 {
			t.Fatalf("empty hitbox %+v", b)
		}
	}
}

func TestRenderButtonsTwoRows(t *testing.T) {
	line, buttons := renderButtons(false, 80, "", 0)
	ys := map[int]int{}
	for _, b := range buttons {
		ys[b.y0]++
	}
	if len(ys) < 2 {
		t.Fatalf("expected wrap to 2 rows, got %v\n%s", ys, line)
	}
	if !strings.Contains(line, "start") || !strings.Contains(line, "shell") {
		t.Fatalf("missing labels in:\n%s", line)
	}
}

func TestRenderButtonsNarrowWrapsMore(t *testing.T) {
	_, wide := renderButtons(false, 200, "", 0)
	_, narrow := renderButtons(false, 24, "", 0)
	max := func(bs []button) int {
		m := 0
		for _, b := range bs {
			if b.y0 > m {
				m = b.y0
			}
		}
		return m
	}
	if max(narrow) <= max(wide) {
		t.Fatalf("narrow should wrap more: wide=%d narrow=%d", max(wide), max(narrow))
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
