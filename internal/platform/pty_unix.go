//go:build !windows

package platform

import (
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// ErrPTYUnsupported is returned on platforms where PTY support is unavailable.
var ErrPTYUnsupported = errors.New("pty execution is not supported on this platform")

type unixPTY struct {
	file *os.File
}

func (p *unixPTY) Read(buf []byte) (int, error)  { return p.file.Read(buf) }
func (p *unixPTY) Write(buf []byte) (int, error) { return p.file.Write(buf) }
func (p *unixPTY) Close() error                  { return p.file.Close() }

func (p *unixPTY) Resize(rows, cols uint16) error {
	if p == nil || p.file == nil {
		return errors.New("pty handle is not available")
	}
	return pty.Setsize(p.file, &pty.Winsize{Rows: rows, Cols: cols})
}

// StartProcessWithPTY starts cmd attached to a PTY master/slave pair.
func StartProcessWithPTY(cmd *exec.Cmd, rows, cols uint16) (*PTYProcess, error) {
	var (
		f   *os.File
		err error
	)
	if rows > 0 && cols > 0 {
		f, err = pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	} else {
		f, err = pty.Start(cmd)
	}
	if err != nil {
		return nil, err
	}
	return &PTYProcess{
		PTY:     &unixPTY{file: f},
		Process: cmd.Process,
		Wait: func() (int32, string, error) {
			err := cmd.Wait()
			return int32(cmd.ProcessState.ExitCode()), ExitSignal(cmd.ProcessState), err
		},
	}, nil
}
