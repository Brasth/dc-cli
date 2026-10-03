package main

import (
	"os"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	t.Setenv("DC_CLI_VERSION", "v9.9.9")
	if got := cliVersion(); got != "9.9.9" {
		t.Fatalf("version=%q", got)
	}
	if code := run([]string{"--version"}); code != 0 {
		t.Fatalf("code=%d", code)
	}
}

func TestDelegatesToActionsMain(t *testing.T) {
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("help code=%d", code)
	}
	if code := run([]string{"bogus"}); code != 2 {
		t.Fatalf("usage code=%d", code)
	}
}

func TestIsTerminalOnFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Fatal("regular file is not a terminal")
	}
}
