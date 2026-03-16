//go:build !windows

package platform

import (
	"errors"
	"os"
	"syscall"
	"time"
)

// TerminateProcess sends SIGTERM, waits for the grace period, then force kills if needed.
func TerminateProcess(proc *os.Process, grace time.Duration) error {
	if proc == nil {
		return nil
	}
	if grace <= 0 {
		grace = 5 * time.Second
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	time.Sleep(grace)
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		return proc.Kill()
	}
	return nil
}

// ExitSignal reports the terminating signal name when the process ended due to a signal.
func ExitSignal(state *os.ProcessState) string {
	if state == nil {
		return ""
	}
	waitStatus, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return ""
	}
	if waitStatus.Signaled() {
		return waitStatus.Signal().String()
	}
	return ""
}
