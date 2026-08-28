package main

import (
	"os"
	"path/filepath"
	"time"
)

func activationStatePath() string {
	if p := os.Getenv("DC_TUI_ACTIVATED_FILE"); p != "" {
		return p
	}
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "dc-cli", "tui-activated")
}

func hasActivated() bool {
	p := activationStatePath()
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func markActivated() {
	p := activationStatePath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(p, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)
}

func shouldShowSplash() bool {
	if os.Getenv("DC_TUI_NO_SPLASH") != "" {
		return false
	}
	return !hasActivated()
}
