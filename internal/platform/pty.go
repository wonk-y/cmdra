package platform

import "os"

// PTY is the host-side PTY transport used by the execution manager.
type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
	Resize(rows, cols uint16) error
}

// PTYProcess is the result of starting one PTY-backed child process.
type PTYProcess struct {
	PTY     PTY
	Process *os.Process
	Wait    func() (int32, string, error)
}
