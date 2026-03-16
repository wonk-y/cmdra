//go:build windows

package platform

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrPTYUnsupported reports that PTY-backed execution is not implemented on this platform.
var ErrPTYUnsupported = errors.New("pty execution is not supported on this platform")

type conPTY struct {
	in      *os.File
	out     *os.File
	console windows.Handle
}

func (p *conPTY) Read(buf []byte) (int, error)  { return p.out.Read(buf) }
func (p *conPTY) Write(buf []byte) (int, error) { return p.in.Write(buf) }

func (p *conPTY) Close() error {
	var firstErr error
	if p.in != nil {
		if err := p.in.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.in = nil
	}
	if p.out != nil {
		if err := p.out.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.out = nil
	}
	if p.console != 0 {
		windows.ClosePseudoConsole(p.console)
		p.console = 0
	}
	return firstErr
}

func (p *conPTY) Resize(rows, cols uint16) error {
	if p == nil || p.console == 0 {
		return errors.New("pty handle is not available")
	}
	return windows.ResizePseudoConsole(p.console, windows.Coord{X: int16(cols), Y: int16(rows)})
}

// StartProcessWithPTY starts cmd attached to a Windows ConPTY pseudo console.
func StartProcessWithPTY(cmd *exec.Cmd, rows, cols uint16) (*PTYProcess, error) {
	if cmd == nil {
		return nil, errors.New("command is required")
	}
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}

	hostRead, ptyWrite, err := createHostReadPipe()
	if err != nil {
		return nil, err
	}
	defer func() {
		if ptyWrite != 0 {
			_ = windows.CloseHandle(ptyWrite)
		}
	}()
	defer func() {
		if hostRead != nil {
			_ = hostRead.Close()
		}
	}()

	hostWrite, ptyRead, err := createHostWritePipe()
	if err != nil {
		return nil, err
	}
	defer func() {
		if ptyRead != 0 {
			_ = windows.CloseHandle(ptyRead)
		}
	}()
	defer func() {
		if hostWrite != nil {
			_ = hostWrite.Close()
		}
	}()

	var console windows.Handle
	if err := windows.CreatePseudoConsole(windows.Coord{X: int16(cols), Y: int16(rows)}, ptyRead, ptyWrite, 0, &console); err != nil {
		return nil, err
	}

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(console)
		return nil, err
	}
	defer attrList.Delete()
	if err := attrList.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(&console), unsafe.Sizeof(console)); err != nil {
		windows.ClosePseudoConsole(console)
		return nil, err
	}

	si := &windows.StartupInfoEx{
		StartupInfo:             windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{}))},
		ProcThreadAttributeList: attrList.List(),
	}

	argv0 := cmd.Path
	if argv0 == "" {
		argv0 = cmd.Args[0]
	}
	appName, err := windows.UTF16PtrFromString(argv0)
	if err != nil {
		windows.ClosePseudoConsole(console)
		return nil, err
	}
	cmdLine, err := windows.UTF16PtrFromString(makeWindowsCommandLine(cmd.Args))
	if err != nil {
		windows.ClosePseudoConsole(console)
		return nil, err
	}

	var dir *uint16
	if cmd.Dir != "" {
		dir, err = windows.UTF16PtrFromString(cmd.Dir)
		if err != nil {
			windows.ClosePseudoConsole(console)
			return nil, err
		}
	}

	var procInfo windows.ProcessInformation
	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT)
	if err := windows.CreateProcess(appName, cmdLine, nil, nil, false, flags, nil, dir, &si.StartupInfo, &procInfo); err != nil {
		windows.ClosePseudoConsole(console)
		return nil, err
	}
	_ = windows.CloseHandle(procInfo.Thread)

	ptyRead = 0
	ptyWrite = 0

	in := hostWrite
	out := hostRead
	hostWrite = nil
	hostRead = nil

	proc, err := os.FindProcess(int(procInfo.ProcessId))
	if err != nil {
		_ = in.Close()
		_ = out.Close()
		windows.CloseHandle(procInfo.Process)
		windows.ClosePseudoConsole(console)
		return nil, err
	}
	transport := &conPTY{
		in:      in,
		out:     out,
		console: console,
	}
	return &PTYProcess{
		PTY:     transport,
		Process: proc,
		Wait: func() (int32, string, error) {
			defer windows.CloseHandle(procInfo.Process)
			_, err := windows.WaitForSingleObject(procInfo.Process, windows.INFINITE)
			if err != nil {
				return -1, "", err
			}
			var exitCode uint32
			if err := windows.GetExitCodeProcess(procInfo.Process, &exitCode); err != nil {
				return -1, "", err
			}
			return int32(exitCode), "", nil
		},
	}, nil
}

func createHostReadPipe() (*os.File, windows.Handle, error) {
	sa := &windows.SecurityAttributes{
		Length:        uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		InheritHandle: 1,
	}
	var readHandle windows.Handle
	var writeHandle windows.Handle
	if err := windows.CreatePipe(&readHandle, &writeHandle, sa, 0); err != nil {
		return nil, 0, err
	}
	if err := windows.SetHandleInformation(readHandle, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		_ = windows.CloseHandle(readHandle)
		_ = windows.CloseHandle(writeHandle)
		return nil, 0, err
	}
	return os.NewFile(uintptr(readHandle), "conpty-pipe"), writeHandle, nil
}

func createHostWritePipe() (*os.File, windows.Handle, error) {
	sa := &windows.SecurityAttributes{
		Length:        uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		InheritHandle: 1,
	}
	var readHandle windows.Handle
	var writeHandle windows.Handle
	if err := windows.CreatePipe(&readHandle, &writeHandle, sa, 0); err != nil {
		return nil, 0, err
	}
	if err := windows.SetHandleInformation(writeHandle, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		_ = windows.CloseHandle(readHandle)
		_ = windows.CloseHandle(writeHandle)
		return nil, 0, err
	}
	return os.NewFile(uintptr(writeHandle), "conpty-pipe"), readHandle, nil
}

func makeWindowsCommandLine(args []string) string {
	if len(args) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(args))
	for _, arg := range args {
		escaped = append(escaped, syscall.EscapeArg(arg))
	}
	return strings.Join(escaped, " ")
}
