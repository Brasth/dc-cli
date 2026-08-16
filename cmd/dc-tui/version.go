package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Set at pack time: -ldflags "-X main.version=0.10.0"
var version = "dev"

func cliVersion() string {
	if v := strings.TrimSpace(os.Getenv("DC_CLI_VERSION")); v != "" {
		return strings.TrimPrefix(v, "v")
	}
	if version != "" && version != "dev" {
		return strings.TrimPrefix(version, "v")
	}
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "VERSION")
		if b, err := os.ReadFile(p); err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}
	return "dev"
}
