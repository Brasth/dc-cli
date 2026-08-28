package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldShowSplashFirstRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DC_TUI_ACTIVATED_FILE", filepath.Join(dir, "activated"))
	t.Setenv("DC_TUI_NO_SPLASH", "")
	if !shouldShowSplash() {
		t.Fatal("first run must splash")
	}
	markActivated()
	if shouldShowSplash() {
		t.Fatal("after activation splash must skip")
	}
}

func TestShouldShowSplashEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DC_TUI_ACTIVATED_FILE", filepath.Join(dir, "activated"))
	t.Setenv("DC_TUI_NO_SPLASH", "1")
	if shouldShowSplash() {
		t.Fatal("DC_TUI_NO_SPLASH must skip even on first run")
	}
}

func TestMarkActivatedWritesFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "activated")
	t.Setenv("DC_TUI_ACTIVATED_FILE", p)
	markActivated()
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		t.Fatalf("expected activation file: %v", err)
	}
}
