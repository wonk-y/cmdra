//go:build windows

package platform

import (
	"os"
	"time"
)

// TerminateProcess is the Windows approximation of graceful terminate followed by force kill.
func TerminateProcess(proc *os.Process, grace time.Duration) error {
	if proc == nil {
		return nil
	}
	if grace <= 0 {
		grace = 5 * time.Second
	}
	if err := proc.Signal(os.Interrupt); err != nil && err != os.ErrProcessDone {
		_ = proc.Kill()
		return nil
	}
	time.Sleep(grace)
	return proc.Kill()
}

// ExitSignal returns an empty string on Windows where Go does not expose a Unix-style signal name.
func ExitSignal(state *os.ProcessState) string {
	return ""
}
