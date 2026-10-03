//go:build unix

package main

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

// probeWaitDelay bounds how long Wait lingers on pipes after the group is
// killed (a grandchild that inherited stdout must not hold the board).
const probeWaitDelay = 500 * time.Millisecond

// setProbeProcAttr puts a read-only probe in its own process group so a
// cancel kills the whole tree (dc-ls → bash → docker), not just the direct
// child. Only probes use this; user actions keep the board's group so the
// terminal (and Ctrl+C) reaches them normally.
func setProbeProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return killProcessGroup(cmd)
	}
	cmd.WaitDelay = probeWaitDelay
}

// killProcessGroup SIGKILLs the probe's group (negative pid). Falls back to
// the direct child when the group is already gone.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return cmd.Process.Kill()
}
