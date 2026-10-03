package actions

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// DCExecPath finds dc-exec: next to the running binary first (release kit
// and generation payload keep them together), then PATH.
var DCExecPath = func() string {
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), "dc-exec")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return cand
		}
	}
	if p, err := exec.LookPath("dc-exec"); err == nil {
		return p
	}
	return "dc-exec"
}

// Args is the exact dc-exec invocation for an action. --no-start: the target
// must already be running; dc-exec re-checks ownership right before exec.
// The app goes through the official `devcontainer exec`; a service through
// `docker exec` on the current stack member. argv follows `--` untouched.
func Args(ws string, a Action) []string {
	args := []string{"--no-start"}
	if a.Service != "" {
		args = append(args, "--service", a.Service)
	}
	args = append(args, ws, "--")
	return append(args, a.Argv...)
}

// Command builds the exec.Cmd (stdio left for the caller).
func Command(ws string, a Action) *exec.Cmd {
	return exec.Command(DCExecPath(), Args(ws, a)...)
}

// ExitCode maps a finished child to a shell-style status: its exit code,
// 128+signal when killed by a signal, 1 when it could not start.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		if c := ee.ExitCode(); c >= 0 {
			return c
		}
	}
	return 1
}

// FormatArgv shows argv for review, quoting anything a reader could misread.
// Display only: execution never goes through a shell.
func FormatArgv(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = quoteArg(a)
	}
	return strings.Join(parts, " ")
}

func quoteArg(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		// Control characters (newline, tab, ESC…) are shown escaped so a
		// review screen can never be redrawn or hidden by argv content.
		if r < 0x20 || r == 0x7f {
			return strconv.Quote(s)
		}
	}
	safe := true
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("@%+=:,./_-", r)) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// DisplayText makes an id / label / path safe to print on a review screen:
// anything with control characters is shown Go-quoted.
func DisplayText(s string) string {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return strconv.Quote(s)
		}
	}
	return s
}

// Target names where an action runs, for review screens.
func Target(a Action) string {
	if a.Service != "" {
		return "service " + DisplayText(a.Service)
	}
	return "app"
}
