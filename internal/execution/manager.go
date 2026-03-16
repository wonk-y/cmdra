package execution

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"cmdra/internal/audit"
	"cmdra/internal/history"
	"cmdra/internal/platform"
)

var (
	// ErrNotFound is returned when the requested execution or transfer ID does not exist.
	ErrNotFound = errors.New("execution not found")
	// ErrForbidden is returned when a caller tries to access another CN's job.
	ErrForbidden = errors.New("execution is owned by a different identity")
	// ErrAlreadyAttached prevents multiple live attach streams from consuming one execution.
	ErrAlreadyAttached = errors.New("execution already has an attached client")
	// ErrRunningHistoryDelete prevents history cleanup from removing active items.
	ErrRunningHistoryDelete = errors.New("cannot delete running execution or transfer")
	// ErrPTYUnsupported is returned when a caller requests PTY execution on an unsupported platform.
	ErrPTYUnsupported = errors.New("pty execution is not supported on this platform")
	// ErrPTYRequired is returned when a resize is requested for a non-PTY execution.
	ErrPTYRequired = errors.New("execution is not using a PTY")
)

// Kind identifies the high-level job type stored in history.
type Kind int

const (
	KindCommand Kind = iota + 1
	KindShell
	KindUpload
	KindDownload
	KindArchiveDownload
)

// State tracks the lifecycle state of a command or transfer.
type State int

const (
	StateRunning State = iota + 1
	StateExited
	StateCanceled
	StateFailedToStart
)

// Source identifies where an output chunk originated.
type Source int

const (
	SourceStdout Source = iota + 1
	SourceStderr
	SourceSystem
)

// CommandSpec selects either argv execution or shell-string execution.
type CommandSpec struct {
	Argv  *ArgvSpec
	Shell *ShellSpec
}

// ArgvSpec describes a direct binary invocation.
type ArgvSpec struct {
	Binary string
	Args   []string
}

// ShellSpec describes a shell plus a command string to execute through it.
type ShellSpec struct {
	ShellBinary string
	Command     string
	UsePTY      bool
	PTYRows     uint32
	PTYCols     uint32
}

// OutputEvent is the in-memory/live replay representation of one output chunk.
type OutputEvent struct {
	ExecutionID string
	Sequence    int64
	Offset      int64
	Source      Source
	Timestamp   time.Time
	Data        []byte
	EOF         bool
}

// ExecutionView is the public in-memory/domain view returned to RPC and SDK layers.
type ExecutionView struct {
	ID                     string
	Kind                   Kind
	State                  State
	PID                    int64
	OwnerCN                string
	CommandArgv            []string
	CommandShell           string
	StartedAt              time.Time
	EndedAt                *time.Time
	ExitCode               *int32
	Signal                 string
	OutputSizeBytes        int64
	LastUploadLocalPath    string
	LastUploadRemotePath   string
	LastDownloadLocalPath  string
	LastDownloadRemotePath string
	LastUploadTransferID   string
	LastDownloadTransferID string
	TransferDirection      string
	TransferProgressBytes  int64
	TransferTotalBytes     *int64
	ErrorMessage           string
	UsesPTY                bool
	PTYRows                uint32
	PTYCols                uint32
}

// Config configures the execution manager and backing history store.
type Config struct {
	DataDir            string
	ChunkSize          int
	FlushInterval      time.Duration
	DefaultGracePeriod time.Duration
	AuditLogger        *audit.Logger
}

// Manager owns running processes, transfer metadata, and output persistence.
type Manager struct {
	mu                 sync.RWMutex
	executions         map[string]*managedExecution
	chunkSize          int
	flushInterval      time.Duration
	defaultGracePeriod time.Duration
	audit              *audit.Logger
	store              *history.Store
}

type managedExecution struct {
	mu              sync.Mutex
	id              string
	kind            Kind
	state           State
	pid             int64
	ownerCN         string
	commandArgv     []string
	commandShell    string
	startedAt       time.Time
	endedAt         *time.Time
	exitCode        *int32
	signal          string
	outputSizeBytes int64

	cmd             *exec.Cmd
	process         *os.Process
	waitFunc        func() (int32, string, error)
	stdin           io.WriteCloser
	pty             platform.PTY
	sequence        int64
	cancelRequested bool
	usesPTY         bool
	ptyRows         uint32
	ptyCols         uint32

	attached   bool
	subscriber chan OutputEvent
}

// NewManager creates a manager backed by a SQLite history database in DataDir.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("data directory must be set")
	}
	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 32 * 1024
	}
	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond
	}
	grace := cfg.DefaultGracePeriod
	if grace <= 0 {
		grace = 5 * time.Second
	}
	store, err := history.Open(filepath.Join(cfg.DataDir, "history.sqlite"))
	if err != nil {
		return nil, err
	}
	return &Manager{
		executions:         map[string]*managedExecution{},
		chunkSize:          chunkSize,
		flushInterval:      flushInterval,
		defaultGracePeriod: grace,
		audit:              cfg.AuditLogger,
		store:              store,
	}, nil
}

// Close closes the backing history store.
func (m *Manager) Close() error {
	if m == nil || m.store == nil {
		return nil
	}
	return m.store.Close()
}

// StartCommand starts either an argv command or a shell command.
func (m *Manager) StartCommand(ownerCN string, spec CommandSpec) (ExecutionView, error) {
	if ownerCN == "" {
		return ExecutionView{}, errors.New("owner CN is required")
	}
	var cmd *exec.Cmd
	var commandArgv []string
	var commandShell string
	var usePTY bool
	var ptyRows uint32
	var ptyCols uint32
	switch {
	case spec.Argv != nil:
		if spec.Argv.Binary == "" {
			return ExecutionView{}, errors.New("argv binary is required")
		}
		cmd = exec.Command(spec.Argv.Binary, spec.Argv.Args...)
		commandArgv = append([]string{spec.Argv.Binary}, spec.Argv.Args...)
	case spec.Shell != nil:
		if strings.TrimSpace(spec.Shell.Command) == "" {
			return ExecutionView{}, errors.New("shell command is required")
		}
		shellBin := strings.TrimSpace(spec.Shell.ShellBinary)
		if shellBin == "" {
			shellBin = defaultShellBinary()
		}
		shellArgs := shellCommandArgs(shellBin, spec.Shell.Command)
		cmd = exec.Command(shellBin, shellArgs...)
		commandArgv = append([]string{shellBin}, shellArgs...)
		commandShell = spec.Shell.Command
		usePTY = spec.Shell.UsePTY
		ptyRows = spec.Shell.PTYRows
		ptyCols = spec.Shell.PTYCols
	default:
		return ExecutionView{}, errors.New("either argv or shell command must be provided")
	}
	return m.start(ownerCN, KindCommand, commandArgv, commandShell, usePTY, ptyRows, ptyCols, cmd)
}

// StartShell starts a persistent shell session for later attach operations.
func (m *Manager) StartShell(ownerCN, shellBinary string, shellArgs []string) (ExecutionView, error) {
	return m.StartShellWithOptions(ownerCN, shellBinary, shellArgs, false, 0, 0)
}

// StartShellWithOptions starts a persistent shell session with optional PTY backing.
func (m *Manager) StartShellWithOptions(ownerCN, shellBinary string, shellArgs []string, usePTY bool, ptyRows, ptyCols uint32) (ExecutionView, error) {
	if ownerCN == "" {
		return ExecutionView{}, errors.New("owner CN is required")
	}
	shellBinary = strings.TrimSpace(shellBinary)
	if shellBinary == "" {
		return ExecutionView{}, errors.New("shell binary is required")
	}
	cmd := exec.Command(shellBinary, shellArgs...)
	commandArgv := append([]string{shellBinary}, shellArgs...)
	return m.start(ownerCN, KindShell, commandArgv, "", usePTY, ptyRows, ptyCols, cmd)
}

func (m *Manager) start(ownerCN string, kind Kind, commandArgv []string, commandShell string, usePTY bool, ptyRows, ptyCols uint32, cmd *exec.Cmd) (ExecutionView, error) {
	id, err := newExecutionID()
	if err != nil {
		return ExecutionView{}, err
	}
	if !usePTY {
		ptyRows = 0
		ptyCols = 0
	}

	var stdin io.WriteCloser
	var stdout io.ReadCloser
	var stderr io.ReadCloser
	var ptyReader io.ReadCloser
	var ptyRW platform.PTY
	if usePTY {
		switch kind {
		case KindCommand, KindShell:
		default:
			return ExecutionView{}, errors.New("pty is only supported for shell-based executions")
		}
	} else {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return ExecutionView{}, err
		}
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return ExecutionView{}, err
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return ExecutionView{}, err
		}
	}

	execState := &managedExecution{
		id:           id,
		kind:         kind,
		state:        StateRunning,
		ownerCN:      ownerCN,
		commandArgv:  append([]string{}, commandArgv...),
		commandShell: commandShell,
		startedAt:    time.Now().UTC(),
		cmd:          cmd,
		stdin:        stdin,
		pty:          ptyRW,
		usesPTY:      usePTY,
		ptyRows:      ptyRows,
		ptyCols:      ptyCols,
	}
	if err := m.store.CreateJob(toHistoryJob(execState.snapshot())); err != nil {
		return ExecutionView{}, err
	}
	if usePTY {
		ptyProc, err := platform.StartProcessWithPTY(cmd, uint16(ptyRows), uint16(ptyCols))
		if err != nil {
			now := time.Now().UTC()
			exitCode := int32(-1)
			execState.state = StateFailedToStart
			execState.endedAt = &now
			execState.exitCode = &exitCode
			_ = m.store.CompleteJob(execState.id, stateToDB(execState.state), now, execState.exitCode, "", err.Error())
			if errors.Is(err, platform.ErrPTYUnsupported) {
				return ExecutionView{}, ErrPTYUnsupported
			}
			return ExecutionView{}, err
		}
		ptyRW = ptyProc.PTY
		ptyReader = io.NopCloser(ptyProc.PTY)
		execState.pty = ptyProc.PTY
		execState.process = ptyProc.Process
		execState.waitFunc = ptyProc.Wait
	} else if err := cmd.Start(); err != nil {
		now := time.Now().UTC()
		exitCode := int32(-1)
		execState.state = StateFailedToStart
		execState.endedAt = &now
		execState.exitCode = &exitCode
		_ = m.store.CompleteJob(execState.id, stateToDB(execState.state), now, execState.exitCode, "", err.Error())
		return ExecutionView{}, err
	} else {
		execState.process = cmd.Process
		execState.waitFunc = func() (int32, string, error) {
			err := cmd.Wait()
			return int32(cmd.ProcessState.ExitCode()), platform.ExitSignal(cmd.ProcessState), err
		}
	}
	if usePTY && execState.process == nil {
		now := time.Now().UTC()
		exitCode := int32(-1)
		execState.state = StateFailedToStart
		execState.endedAt = &now
		execState.exitCode = &exitCode
		_ = m.store.CompleteJob(execState.id, stateToDB(execState.state), now, execState.exitCode, "", "pty-backed process did not start")
		return ExecutionView{}, errors.New("pty-backed process did not start")
	}
	if !usePTY && cmd.Process == nil {
		now := time.Now().UTC()
		exitCode := int32(-1)
		execState.state = StateFailedToStart
		execState.endedAt = &now
		execState.exitCode = &exitCode
		_ = m.store.CompleteJob(execState.id, stateToDB(execState.state), now, execState.exitCode, "", "process did not start")
		return ExecutionView{}, errors.New("process did not start")
	}
	execState.pid = int64(execState.process.Pid)
	if err := m.store.UpdateJobProcess(execState.id, execState.pid); err != nil {
		return ExecutionView{}, err
	}
	m.mu.Lock()
	m.executions[id] = execState
	m.mu.Unlock()
	_ = m.audit.Write("execution.start", execState.snapshot())

	if usePTY {
		go m.captureOutput(execState, ptyReader, SourceStdout)
	} else {
		go m.captureOutput(execState, stdout, SourceStdout)
		go m.captureOutput(execState, stderr, SourceStderr)
	}
	go m.waitForExit(execState)

	return execState.snapshot(), nil
}

// Get returns metadata for one owned execution or transfer.
func (m *Manager) Get(executionID, ownerCN string) (ExecutionView, error) {
	job, err := m.store.GetJobByID(executionID)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ExecutionView{}, ErrNotFound
		}
		return ExecutionView{}, err
	}
	if job.OwnerCN != ownerCN {
		return ExecutionView{}, ErrForbidden
	}
	return fromHistoryJob(job), nil
}

// List returns all jobs for one caller, optionally restricted to running jobs.
func (m *Manager) List(ownerCN string, runningOnly bool) []ExecutionView {
	jobs, err := m.store.ListJobs(ownerCN, runningOnly)
	if err != nil {
		return nil
	}
	out := make([]ExecutionView, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, fromHistoryJob(job))
	}
	return out
}

// Delete removes one finished execution or transfer from persisted history.
func (m *Manager) Delete(executionID, ownerCN string) error {
	job, err := m.store.GetJobByID(executionID)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if job.OwnerCN != ownerCN {
		return ErrForbidden
	}
	if job.State == stateToDB(StateRunning) {
		return ErrRunningHistoryDelete
	}
	if err := m.store.DeleteJob(executionID); err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	m.mu.Lock()
	delete(m.executions, executionID)
	m.mu.Unlock()
	_ = m.audit.Write("history.delete", fromHistoryJob(job))
	return nil
}

// ClearHistory removes all finished executions and transfers owned by one CN.
func (m *Manager) ClearHistory(ownerCN string) (deletedCount int, skippedRunningCount int, err error) {
	jobs, err := m.store.ListJobs(ownerCN, false)
	if err != nil {
		return 0, 0, err
	}
	for _, job := range jobs {
		if job.State == stateToDB(StateRunning) {
			skippedRunningCount++
			continue
		}
		if err := m.store.DeleteJob(job.ID); err != nil {
			if errors.Is(err, history.ErrNotFound) {
				continue
			}
			return deletedCount, skippedRunningCount, err
		}
		m.mu.Lock()
		delete(m.executions, job.ID)
		m.mu.Unlock()
		deletedCount++
		_ = m.audit.Write("history.delete", fromHistoryJob(job))
	}
	return deletedCount, skippedRunningCount, nil
}

// StartTransfer records the beginning of an upload/download style job.
func (m *Manager) StartTransfer(ownerCN string, kind Kind, localPath, remotePath string, totalBytes *int64) (ExecutionView, error) {
	if ownerCN == "" {
		return ExecutionView{}, errors.New("owner CN is required")
	}
	transferID, err := newTransferID()
	if err != nil {
		return ExecutionView{}, err
	}
	now := time.Now().UTC()
	view := ExecutionView{
		ID:                    transferID,
		Kind:                  kind,
		State:                 StateRunning,
		OwnerCN:               ownerCN,
		StartedAt:             now,
		TransferProgressBytes: 0,
		TransferTotalBytes:    totalBytes,
	}
	switch kind {
	case KindUpload:
		view.LastUploadLocalPath = localPath
		view.LastUploadRemotePath = remotePath
		view.LastUploadTransferID = transferID
		view.TransferDirection = "upload"
	case KindDownload, KindArchiveDownload:
		view.LastDownloadLocalPath = localPath
		view.LastDownloadRemotePath = remotePath
		view.LastDownloadTransferID = transferID
		view.TransferDirection = "download"
	}
	if err := m.store.CreateJob(toHistoryJob(view)); err != nil {
		return ExecutionView{}, err
	}
	_, _, _ = m.store.AppendOutput(transferID, sourceToDB(SourceSystem), now, []byte("transfer started\n"))
	_ = m.audit.Write("transfer.start", view)
	return view, nil
}

// UpdateTransferProgress updates persisted byte counters for a transfer job.
func (m *Manager) UpdateTransferProgress(id string, progress int64, totalBytes *int64) error {
	return m.store.UpdateTransferProgress(id, progress, totalBytes)
}

// CompleteTransfer finalizes an upload/download style job.
func (m *Manager) CompleteTransfer(id, ownerCN string, canceled bool, errMsg string) (ExecutionView, error) {
	meta, err := m.store.GetJobByID(id)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ExecutionView{}, ErrNotFound
		}
		return ExecutionView{}, err
	}
	if meta.OwnerCN != ownerCN {
		return ExecutionView{}, ErrForbidden
	}
	state := StateExited
	if canceled {
		state = StateCanceled
	}
	now := time.Now().UTC()
	if err := m.store.CompleteJob(id, stateToDB(state), now, nil, "", errMsg); err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ExecutionView{}, ErrNotFound
		}
		return ExecutionView{}, err
	}
	updated, err := m.store.GetJobByID(id)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ExecutionView{}, ErrNotFound
		}
		return ExecutionView{}, err
	}
	view := fromHistoryJob(updated)
	if errMsg == "" {
		_, _, _ = m.store.AppendOutput(id, sourceToDB(SourceSystem), now, []byte("transfer completed\n"))
	} else {
		_, _, _ = m.store.AppendOutput(id, sourceToDB(SourceSystem), now, []byte("transfer failed: "+errMsg+"\n"))
	}
	_ = m.audit.Write("transfer.stop", view)
	return view, nil
}

// Cancel requests graceful termination for a running command or shell.
func (m *Manager) Cancel(executionID, ownerCN string, grace time.Duration) (ExecutionView, error) {
	e, err := m.getOwned(executionID, ownerCN)
	if err != nil {
		return ExecutionView{}, err
	}
	e.mu.Lock()
	e.cancelRequested = true
	proc := e.process
	e.mu.Unlock()
	if grace <= 0 {
		grace = m.defaultGracePeriod
	}
	if err := platform.TerminateProcess(proc, grace); err != nil {
		return ExecutionView{}, err
	}
	return e.snapshot(), nil
}

// WriteStdin writes input into an attached command's stdin pipe.
func (m *Manager) WriteStdin(executionID, ownerCN string, data []byte, eof bool) error {
	e, err := m.getOwned(executionID, ownerCN)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pty != nil {
		if len(data) > 0 {
			if _, err := e.pty.Write(data); err != nil {
				return err
			}
		}
		if eof {
			// PTY sessions treat EOF as an interactive EOT byte instead of closing the master side.
			if _, err := e.pty.Write([]byte{0x04}); err != nil {
				return err
			}
		}
		return nil
	}
	if e.stdin == nil {
		return errors.New("stdin is not available")
	}
	if len(data) > 0 {
		if _, err := e.stdin.Write(data); err != nil {
			return err
		}
	}
	if eof {
		if err := e.stdin.Close(); err != nil {
			return err
		}
		e.stdin = nil
	}
	return nil
}

// ResizePTY updates the terminal size for one PTY-backed execution.
func (m *Manager) ResizePTY(executionID, ownerCN string, rows, cols uint32) error {
	if rows == 0 || cols == 0 {
		return nil
	}
	e, err := m.getOwned(executionID, ownerCN)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pty == nil || !e.usesPTY {
		return ErrPTYRequired
	}
	if err := e.pty.Resize(uint16(rows), uint16(cols)); err != nil {
		return err
	}
	e.ptyRows = rows
	e.ptyCols = cols
	return m.store.UpdatePTYSize(executionID, rows, cols)
}

// Attach reserves the single live subscriber channel for one execution.
func (m *Manager) Attach(executionID, ownerCN string) (<-chan OutputEvent, func(), ExecutionView, error) {
	e, err := m.getOwned(executionID, ownerCN)
	if err != nil {
		return nil, nil, ExecutionView{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.attached {
		return nil, nil, ExecutionView{}, ErrAlreadyAttached
	}
	ch := make(chan OutputEvent, 256)
	e.attached = true
	e.subscriber = ch
	detach := func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.subscriber == ch {
			e.subscriber = nil
			e.attached = false
		}
	}
	return ch, detach, e.snapshotLocked(), nil
}

// StreamOutput replays persisted output to the caller and optionally discards it afterward.
func (m *Manager) StreamOutput(ctx context.Context, executionID, ownerCN string, offset int64, send func(OutputEvent) error, discard bool) error {
	meta, err := m.store.GetJobByID(executionID)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if meta.OwnerCN != ownerCN {
		return ErrForbidden
	}
	if err := m.store.StreamOutput(ctx, executionID, offset, func(rec history.OutputRecord) error {
		return send(OutputEvent{
			ExecutionID: executionID,
			Sequence:    rec.Sequence,
			Offset:      rec.Offset,
			Source:      sourceFromDB(rec.Source),
			Timestamp:   rec.Timestamp,
			Data:        rec.Data,
		})
	}); err != nil {
		return err
	}
	if discard && meta.State != "running" {
		_ = m.store.DeleteOutput(executionID)
	}
	return nil
}

func (m *Manager) captureOutput(e *managedExecution, r io.ReadCloser, source Source) {
	defer r.Close()

	chunks := make(chan []byte, 32)
	go func() {
		defer close(chunks)
		buf := make([]byte, 4*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				payload := make([]byte, n)
				copy(payload, buf[:n])
				chunks <- payload
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	var pending bytes.Buffer
	flush := func() {
		if pending.Len() == 0 {
			return
		}
		data := make([]byte, pending.Len())
		copy(data, pending.Bytes())
		pending.Reset()
		m.appendOutput(e, source, data)
	}

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				flush()
				return
			}
			pending.Write(chunk)
			// Full chunks are flushed immediately while short chunks are flushed by the ticker.
			for pending.Len() >= m.chunkSize {
				data := make([]byte, m.chunkSize)
				_, _ = pending.Read(data)
				m.appendOutput(e, source, data)
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (m *Manager) appendOutput(e *managedExecution, source Source, data []byte) {
	if len(data) == 0 {
		return
	}
	ts := time.Now().UTC()
	offset, seq, err := m.store.AppendOutput(e.id, sourceToDB(source), ts, data)
	if err != nil {
		return
	}
	e.mu.Lock()
	e.sequence = seq
	e.outputSizeBytes += int64(len(data))
	e.mu.Unlock()
	e.publish(OutputEvent{
		ExecutionID: e.id,
		Sequence:    seq,
		Offset:      offset,
		Source:      source,
		Timestamp:   ts,
		Data:        data,
	})
}

func (m *Manager) waitForExit(e *managedExecution) {
	exitCode, signal, err := e.waitFunc()
	if err != nil && exitCode == 0 {
		exitCode = -1
	}
	now := time.Now().UTC()

	e.mu.Lock()
	if e.cancelRequested {
		e.state = StateCanceled
	} else {
		e.state = StateExited
	}
	e.endedAt = &now
	e.exitCode = &exitCode
	e.signal = signal
	if e.stdin != nil {
		_ = e.stdin.Close()
		e.stdin = nil
	}
	if e.pty != nil {
		_ = e.pty.Close()
		e.pty = nil
	}
	snapshot := e.snapshotLocked()
	e.mu.Unlock()

	_ = m.store.CompleteJob(e.id, stateToDB(snapshot.State), now, snapshot.ExitCode, snapshot.Signal, "")
	_ = m.audit.Write("execution.stop", snapshot)
	// EOF is published after final state is persisted so reconnecting clients observe stable metadata.
	e.publish(OutputEvent{ExecutionID: e.id, Source: SourceSystem, Timestamp: now, EOF: true})
}

func (m *Manager) getOwned(executionID, ownerCN string) (*managedExecution, error) {
	job, err := m.store.GetJobByID(executionID)
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if job.OwnerCN != ownerCN {
		return nil, ErrForbidden
	}

	m.mu.RLock()
	e, ok := m.executions[executionID]
	m.mu.RUnlock()
	if ok {
		return e, nil
	}
	return nil, ErrNotFound
}

func (e *managedExecution) snapshot() ExecutionView {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *managedExecution) snapshotLocked() ExecutionView {
	view := ExecutionView{
		ID:              e.id,
		Kind:            e.kind,
		State:           e.state,
		PID:             e.pid,
		OwnerCN:         e.ownerCN,
		CommandArgv:     append([]string{}, e.commandArgv...),
		CommandShell:    e.commandShell,
		StartedAt:       e.startedAt,
		EndedAt:         e.endedAt,
		ExitCode:        e.exitCode,
		Signal:          e.signal,
		OutputSizeBytes: e.outputSizeBytes,
		UsesPTY:         e.usesPTY,
		PTYRows:         e.ptyRows,
		PTYCols:         e.ptyCols,
	}
	return view
}

func (e *managedExecution) publish(evt OutputEvent) {
	e.mu.Lock()
	ch := e.subscriber
	if evt.EOF {
		e.subscriber = nil
		e.attached = false
	}
	e.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- evt:
	default:
	}
	if evt.EOF {
		close(ch)
	}
}

func newExecutionID() (string, error) { return newID("exec") }

func newTransferID() (string, error) { return newID("xfer") }

func newID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buf)), nil
}

func defaultShellBinary() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/sh"
}

func shellCommandArgs(shellBinary, command string) []string {
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(filepath.Base(shellBinary))
		if lower == "powershell.exe" || lower == "pwsh.exe" {
			return []string{"-Command", command}
		}
		return []string{"/C", command}
	}
	return []string{"-lc", command}
}

func toHistoryJob(view ExecutionView) history.Job {
	return history.Job{
		ID:                    view.ID,
		OwnerCN:               view.OwnerCN,
		Kind:                  kindToDB(view.Kind),
		State:                 stateToDB(view.State),
		PID:                   view.PID,
		CommandArgv:           append([]string{}, view.CommandArgv...),
		CommandShell:          view.CommandShell,
		StartedAt:             view.StartedAt,
		EndedAt:               view.EndedAt,
		ExitCode:              view.ExitCode,
		Signal:                view.Signal,
		OutputSizeBytes:       view.OutputSizeBytes,
		TransferLocalPath:     firstNonEmpty(view.LastUploadLocalPath, view.LastDownloadLocalPath),
		TransferRemotePath:    firstNonEmpty(view.LastUploadRemotePath, view.LastDownloadRemotePath),
		TransferDirection:     view.TransferDirection,
		TransferProgressBytes: view.TransferProgressBytes,
		TransferTotalBytes:    view.TransferTotalBytes,
		ErrorMessage:          view.ErrorMessage,
		UsesPTY:               view.UsesPTY,
		PTYRows:               view.PTYRows,
		PTYCols:               view.PTYCols,
	}
}

func fromHistoryJob(job history.Job) ExecutionView {
	view := ExecutionView{
		ID:                    job.ID,
		Kind:                  kindFromDB(job.Kind),
		State:                 stateFromDB(job.State),
		PID:                   job.PID,
		OwnerCN:               job.OwnerCN,
		CommandArgv:           append([]string{}, job.CommandArgv...),
		CommandShell:          job.CommandShell,
		StartedAt:             job.StartedAt,
		EndedAt:               job.EndedAt,
		ExitCode:              job.ExitCode,
		Signal:                job.Signal,
		OutputSizeBytes:       job.OutputSizeBytes,
		TransferDirection:     job.TransferDirection,
		TransferProgressBytes: job.TransferProgressBytes,
		TransferTotalBytes:    job.TransferTotalBytes,
		ErrorMessage:          job.ErrorMessage,
		UsesPTY:               job.UsesPTY,
		PTYRows:               job.PTYRows,
		PTYCols:               job.PTYCols,
	}
	switch view.Kind {
	case KindUpload:
		view.LastUploadLocalPath = job.TransferLocalPath
		view.LastUploadRemotePath = job.TransferRemotePath
		view.LastUploadTransferID = job.ID
	case KindDownload, KindArchiveDownload:
		view.LastDownloadLocalPath = job.TransferLocalPath
		view.LastDownloadRemotePath = job.TransferRemotePath
		view.LastDownloadTransferID = job.ID
	}
	return view
}

func kindToDB(kind Kind) string {
	switch kind {
	case KindCommand:
		return "command"
	case KindShell:
		return "shell"
	case KindUpload:
		return "upload"
	case KindDownload:
		return "download"
	case KindArchiveDownload:
		return "archive_download"
	default:
		return "unknown"
	}
}

func kindFromDB(raw string) Kind {
	switch raw {
	case "command":
		return KindCommand
	case "shell":
		return KindShell
	case "upload":
		return KindUpload
	case "download":
		return KindDownload
	case "archive_download":
		return KindArchiveDownload
	default:
		return 0
	}
}

func stateToDB(state State) string {
	switch state {
	case StateRunning:
		return "running"
	case StateExited:
		return "exited"
	case StateCanceled:
		return "canceled"
	case StateFailedToStart:
		return "failed_to_start"
	default:
		return "unknown"
	}
}

func stateFromDB(raw string) State {
	switch raw {
	case "running":
		return StateRunning
	case "exited":
		return StateExited
	case "canceled":
		return StateCanceled
	case "failed_to_start":
		return StateFailedToStart
	default:
		return 0
	}
}

func sourceToDB(source Source) int {
	switch source {
	case SourceStdout:
		return 1
	case SourceStderr:
		return 2
	case SourceSystem:
		return 3
	default:
		return 0
	}
}

func sourceFromDB(raw int) Source {
	switch raw {
	case 1:
		return SourceStdout
	case 2:
		return SourceStderr
	case 3:
		return SourceSystem
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
