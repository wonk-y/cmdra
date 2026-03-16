package cmdraclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	agentv1 "cmdra/gen/agent/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// DialConfig defines how the SDK connects to one cmdrad endpoint.
type DialConfig struct {
	Address            string
	CAFile             string
	ClientCertFile     string
	ClientKeyFile      string
	ServerName         string
	InsecureSkipVerify bool
	DialOptions        []grpc.DialOption
}

// UploadOptions controls one upload request.
type UploadOptions struct {
	Overwrite *bool
	FileMode  os.FileMode
}

// DownloadOptions controls one download or archive download request.
type DownloadOptions struct {
	ChunkSize int
}

// ShellOptions controls shell-command and shell-session startup behavior.
type ShellOptions struct {
	UsePTY  bool
	PTYRows uint32
	PTYCols uint32
}

// DownloadResult reports the final transfer metadata returned by streaming downloads.
type DownloadResult struct {
	BytesWritten uint64
	TransferID   string
}

// ExecutionDetails combines metadata with persisted output replay.
type ExecutionDetails struct {
	Execution *agentv1.Execution
	Output    []*agentv1.OutputChunk
}

// ClearHistoryResult reports the outcome of deleting persisted history.
type ClearHistoryResult struct {
	DeletedCount        uint64
	SkippedRunningCount uint64
}

// ExecutionFuture exposes the optional asynchronous execution-start helpers.
type ExecutionFuture struct {
	done chan struct{}
	resp *agentv1.Execution
	err  error
}

// Wait blocks until the asynchronous execution start request finishes.
func (f *ExecutionFuture) Wait() (*agentv1.Execution, error) {
	<-f.done
	return f.resp, f.err
}

// UploadFuture and DownloadFuture expose the optional asynchronous transfer helpers.
type UploadFuture struct {
	done chan struct{}
	resp *agentv1.UploadFileResponse
	err  error
}

// Wait blocks until the asynchronous upload finishes.
func (f *UploadFuture) Wait() (*agentv1.UploadFileResponse, error) {
	<-f.done
	return f.resp, f.err
}

// DownloadFuture is the asynchronous download equivalent of UploadFuture.
type DownloadFuture struct {
	done chan struct{}
	resp *DownloadResult
	err  error
}

// Wait blocks until the asynchronous download finishes.
func (f *DownloadFuture) Wait() (*DownloadResult, error) {
	<-f.done
	return f.resp, f.err
}

// Client wraps one gRPC connection and generated service client.
type Client struct {
	conn *grpc.ClientConn
	api  agentv1.AgentServiceClient
}

// Dial opens a new mTLS gRPC connection to cmdrad.
func Dial(ctx context.Context, cfg DialConfig) (*Client, error) {
	if cfg.Address == "" {
		return nil, errors.New("address is required")
	}
	tlsCfg, err := loadTLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))}
	opts = append(opts, cfg.DialOptions...)
	conn, err := grpc.DialContext(ctx, cfg.Address, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, api: agentv1.NewAgentServiceClient(conn)}, nil
}

// New wraps an existing gRPC client connection.
func New(conn *grpc.ClientConn) *Client {
	return &Client{conn: conn, api: agentv1.NewAgentServiceClient(conn)}
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// StartArgv starts a direct argv execution on the remote host.
func (c *Client) StartArgv(ctx context.Context, binary string, args []string) (*agentv1.Execution, error) {
	resp, err := c.api.StartCommand(ctx, &agentv1.StartCommandRequest{
		CommandSpec: &agentv1.StartCommandRequest_Argv{
			Argv: &agentv1.ArgvCommand{Binary: binary, Args: args},
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.GetExecution(), nil
}

// StartArgvAsync runs StartArgv in a background goroutine.
func (c *Client) StartArgvAsync(ctx context.Context, binary string, args []string) *ExecutionFuture {
	future := &ExecutionFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.StartArgv(ctx, binary, args)
	}()
	return future
}

// StartShellCommand runs one shell command string.
func (c *Client) StartShellCommand(ctx context.Context, shellBinary, command string) (*agentv1.Execution, error) {
	return c.StartShellCommandWithOptions(ctx, shellBinary, command, ShellOptions{})
}

// StartShellCommandWithOptions runs one shell command string with optional PTY backing.
func (c *Client) StartShellCommandWithOptions(ctx context.Context, shellBinary, command string, opts ShellOptions) (*agentv1.Execution, error) {
	resp, err := c.api.StartCommand(ctx, &agentv1.StartCommandRequest{
		CommandSpec: &agentv1.StartCommandRequest_Shell{
			Shell: &agentv1.ShellCommand{
				ShellBinary: shellBinary,
				Command:     command,
				UsePty:      opts.UsePTY,
				PtyRows:     opts.PTYRows,
				PtyCols:     opts.PTYCols,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.GetExecution(), nil
}

// StartShellCommandAsync runs StartShellCommand in a background goroutine.
func (c *Client) StartShellCommandAsync(ctx context.Context, shellBinary, command string) *ExecutionFuture {
	return c.StartShellCommandAsyncWithOptions(ctx, shellBinary, command, ShellOptions{})
}

// StartShellCommandAsyncWithOptions runs StartShellCommandWithOptions in a background goroutine.
func (c *Client) StartShellCommandAsyncWithOptions(ctx context.Context, shellBinary, command string, opts ShellOptions) *ExecutionFuture {
	future := &ExecutionFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.StartShellCommandWithOptions(ctx, shellBinary, command, opts)
	}()
	return future
}

// StartShellSession starts a persistent shell process that clients can later attach to.
func (c *Client) StartShellSession(ctx context.Context, shellBinary string, shellArgs []string) (*agentv1.Execution, error) {
	return c.StartShellSessionWithOptions(ctx, shellBinary, shellArgs, ShellOptions{})
}

// StartShellSessionWithOptions starts a persistent shell process with optional PTY backing.
func (c *Client) StartShellSessionWithOptions(ctx context.Context, shellBinary string, shellArgs []string, opts ShellOptions) (*agentv1.Execution, error) {
	resp, err := c.api.StartShell(ctx, &agentv1.StartShellRequest{
		ShellBinary: shellBinary,
		ShellArgs:   shellArgs,
		UsePty:      opts.UsePTY,
		PtyRows:     opts.PTYRows,
		PtyCols:     opts.PTYCols,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetExecution(), nil
}

// StartShellSessionAsync runs StartShellSession in a background goroutine.
func (c *Client) StartShellSessionAsync(ctx context.Context, shellBinary string, shellArgs []string) *ExecutionFuture {
	return c.StartShellSessionAsyncWithOptions(ctx, shellBinary, shellArgs, ShellOptions{})
}

// StartShellSessionAsyncWithOptions runs StartShellSessionWithOptions in a background goroutine.
func (c *Client) StartShellSessionAsyncWithOptions(ctx context.Context, shellBinary string, shellArgs []string, opts ShellOptions) *ExecutionFuture {
	future := &ExecutionFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.StartShellSessionWithOptions(ctx, shellBinary, shellArgs, opts)
	}()
	return future
}

// GetExecution returns metadata for one execution or transfer.
func (c *Client) GetExecution(ctx context.Context, executionID string) (*agentv1.Execution, error) {
	resp, err := c.api.GetExecution(ctx, &agentv1.GetExecutionRequest{ExecutionId: executionID})
	if err != nil {
		return nil, err
	}
	return resp.GetExecution(), nil
}

// GetExecutionWithOutput returns metadata plus persisted output replay.
func (c *Client) GetExecutionWithOutput(ctx context.Context, executionID string, discard bool) (*ExecutionDetails, error) {
	execMeta, err := c.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	output, err := c.ReadOutput(ctx, executionID, 0, discard)
	if err != nil {
		return nil, err
	}
	return &ExecutionDetails{Execution: execMeta, Output: output}, nil
}

// ListExecutions lists the caller's executions and transfers.
func (c *Client) ListExecutions(ctx context.Context, runningOnly *bool) ([]*agentv1.Execution, error) {
	req := &agentv1.ListExecutionsRequest{}
	if runningOnly != nil {
		req.RunningOnly = runningOnly
	}
	resp, err := c.api.ListExecutions(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetExecutions(), nil
}

// DeleteExecution removes one finished execution or transfer from persisted history.
func (c *Client) DeleteExecution(ctx context.Context, executionID string) error {
	_, err := c.api.DeleteExecution(ctx, &agentv1.DeleteExecutionRequest{ExecutionId: executionID})
	return err
}

// ClearHistory deletes all finished executions and transfers owned by the caller.
func (c *Client) ClearHistory(ctx context.Context) (*ClearHistoryResult, error) {
	resp, err := c.api.ClearHistory(ctx, &agentv1.ClearHistoryRequest{})
	if err != nil {
		return nil, err
	}
	return &ClearHistoryResult{
		DeletedCount:        resp.GetDeletedCount(),
		SkippedRunningCount: resp.GetSkippedRunningCount(),
	}, nil
}

// CancelExecution requests graceful termination of a running execution.
func (c *Client) CancelExecution(ctx context.Context, executionID string, grace time.Duration) (*agentv1.Execution, error) {
	resp, err := c.api.CancelExecution(ctx, &agentv1.CancelExecutionRequest{
		ExecutionId:        executionID,
		GracePeriodSeconds: uint32(grace / time.Second),
	})
	if err != nil {
		return nil, err
	}
	return resp.GetExecution(), nil
}

// ReadOutput replays persisted output starting at the given byte offset.
func (c *Client) ReadOutput(ctx context.Context, executionID string, offset int64, discard bool) ([]*agentv1.OutputChunk, error) {
	stream, err := c.api.ReadOutput(ctx, &agentv1.ReadOutputRequest{
		ExecutionId:      executionID,
		Offset:           offset,
		DiscardAfterRead: discard,
	})
	if err != nil {
		return nil, err
	}
	var chunks []*agentv1.OutputChunk
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return chunks, nil
			}
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
}

// Attach opens a bidirectional interactive session to one running execution.
func (c *Client) Attach(ctx context.Context, executionID string, replayBuffered bool, replayFromOffset int64) (*AttachSession, error) {
	stream, err := c.api.Attach(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&agentv1.AttachRequest{
		Payload: &agentv1.AttachRequest_Start{
			Start: &agentv1.AttachStart{
				ExecutionId:      executionID,
				ReplayBuffered:   replayBuffered,
				ReplayFromOffset: replayFromOffset,
			},
		},
	}); err != nil {
		return nil, err
	}
	return &AttachSession{stream: stream}, nil
}

// UploadFile uploads one local file to the remote host.
func (c *Client) UploadFile(ctx context.Context, localPath, remotePath string, opts UploadOptions) (*agentv1.UploadFileResponse, error) {
	stream, err := c.api.UploadFile(ctx)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	mode := opts.FileMode
	if mode == 0 {
		mode = info.Mode().Perm()
	}
	if err := stream.Send(&agentv1.UploadFileRequest{
		Payload: &agentv1.UploadFileRequest_Metadata{
			Metadata: &agentv1.UploadFileMetadata{
				Path:            remotePath,
				Overwrite:       opts.Overwrite,
				FileMode:        uint32(mode.Perm()),
				ClientLocalPath: stringPtr(localPath),
			},
		},
	}); err != nil {
		return nil, err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			payload := make([]byte, n)
			copy(payload, buf[:n])
			if err := stream.Send(&agentv1.UploadFileRequest{
				Payload: &agentv1.UploadFileRequest_Chunk{Chunk: payload},
			}); err != nil {
				return nil, err
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
	}
	return stream.CloseAndRecv()
}

// UploadFileAsync runs UploadFile in a background goroutine.
func (c *Client) UploadFileAsync(ctx context.Context, localPath, remotePath string, opts UploadOptions) *UploadFuture {
	future := &UploadFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.UploadFile(ctx, localPath, remotePath, opts)
	}()
	return future
}

// DownloadFile downloads one remote file to the local filesystem.
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string, opts DownloadOptions) (*DownloadResult, error) {
	stream, err := c.api.DownloadFile(ctx, &agentv1.DownloadFileRequest{
		Path:            remotePath,
		ChunkSize:       uint32(opts.ChunkSize),
		ClientLocalPath: stringPtr(localPath),
	})
	if err != nil {
		return nil, err
	}
	return receiveFileStream(localPath, stream)
}

// DownloadFileAsync runs DownloadFile in a background goroutine.
func (c *Client) DownloadFileAsync(ctx context.Context, remotePath, localPath string, opts DownloadOptions) *DownloadFuture {
	future := &DownloadFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.DownloadFile(ctx, remotePath, localPath, opts)
	}()
	return future
}

// DownloadArchive downloads a zip archive containing the requested remote paths.
func (c *Client) DownloadArchive(ctx context.Context, paths []string, localPath string, opts DownloadOptions) (*DownloadResult, error) {
	stream, err := c.api.DownloadArchive(ctx, &agentv1.DownloadArchiveRequest{
		Paths:           paths,
		Format:          agentv1.ArchiveFormat_ARCHIVE_FORMAT_ZIP,
		ChunkSize:       uint32(opts.ChunkSize),
		ClientLocalPath: stringPtr(localPath),
	})
	if err != nil {
		return nil, err
	}
	return receiveFileStream(localPath, stream)
}

// DownloadArchiveAsync runs DownloadArchive in a background goroutine.
func (c *Client) DownloadArchiveAsync(ctx context.Context, paths []string, localPath string, opts DownloadOptions) *DownloadFuture {
	future := &DownloadFuture{done: make(chan struct{})}
	go func() {
		defer close(future.done)
		future.resp, future.err = c.DownloadArchive(ctx, paths, localPath, opts)
	}()
	return future
}

// AttachSession wraps the bidirectional attach RPC.
type AttachSession struct {
	stream grpc.BidiStreamingClient[agentv1.AttachRequest, agentv1.AttachEvent]
}

// Recv receives the next attach event from the server.
func (s *AttachSession) Recv() (*agentv1.AttachEvent, error) {
	return s.stream.Recv()
}

// SendStdin writes raw stdin bytes to the remote execution.
func (s *AttachSession) SendStdin(data []byte, eof bool) error {
	return s.stream.Send(&agentv1.AttachRequest{
		Payload: &agentv1.AttachRequest_Stdin{
			Stdin: &agentv1.StdinChunk{Data: data, Eof: eof},
		},
	})
}

// CancelExecution asks the server to terminate the attached execution.
func (s *AttachSession) CancelExecution() error {
	return s.stream.Send(&agentv1.AttachRequest{
		Payload: &agentv1.AttachRequest_Control{
			Control: &agentv1.AttachControl{CancelExecution: true},
		},
	})
}

// ResizePTY sends an updated terminal size for a PTY-backed execution.
func (s *AttachSession) ResizePTY(rows, cols uint32) error {
	return s.stream.Send(&agentv1.AttachRequest{
		Payload: &agentv1.AttachRequest_Control{
			Control: &agentv1.AttachControl{PtyRows: rows, PtyCols: cols},
		},
	})
}

// CloseSend closes the client->server side of the attach stream.
func (s *AttachSession) CloseSend() error {
	return s.stream.CloseSend()
}

func receiveFileStream(localPath string, stream grpc.ServerStreamingClient[agentv1.FileChunk]) (*DownloadResult, error) {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(localPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	result := &DownloadResult{}
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			return nil, err
		}
		if chunk.GetTransferId() != "" {
			result.TransferID = chunk.GetTransferId()
		}
		if len(chunk.GetData()) > 0 {
			n, err := f.Write(chunk.GetData())
			if err != nil {
				return nil, err
			}
			result.BytesWritten += uint64(n)
		}
		if chunk.GetEof() {
			return result, nil
		}
	}
}

func loadTLSConfig(cfg DialConfig) (*tls.Config, error) {
	caPEM, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	rootPool := x509.NewCertPool()
	if !rootPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("CA PEM did not contain any certificates")
	}
	cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		RootCAs:            rootPool,
		Certificates:       []tls.Certificate{cert},
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
