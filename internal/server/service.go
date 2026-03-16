package server

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentv1 "cmdagent/gen/agent/v1"
	"cmdagent/internal/auth"
	"cmdagent/internal/execution"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultChunkSize = 32 * 1024

// Service adapts gRPC requests into execution-manager operations.
type Service struct {
	agentv1.UnimplementedAgentServiceServer
	manager *execution.Manager
}

// NewService constructs a gRPC service backed by one execution manager.
func NewService(manager *execution.Manager) *Service {
	return &Service{manager: manager}
}

func (s *Service) StartCommand(ctx context.Context, req *agentv1.StartCommandRequest) (*agentv1.StartCommandResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	spec, err := mapCommandSpec(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	meta, err := s.manager.StartCommand(ownerCN, spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &agentv1.StartCommandResponse{Execution: toProtoExecution(meta)}, nil
}

func (s *Service) StartShell(ctx context.Context, req *agentv1.StartShellRequest) (*agentv1.StartShellResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	meta, err := s.manager.StartShell(ownerCN, req.GetShellBinary(), req.GetShellArgs())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &agentv1.StartShellResponse{Execution: toProtoExecution(meta)}, nil
}

func (s *Service) GetExecution(ctx context.Context, req *agentv1.GetExecutionRequest) (*agentv1.GetExecutionResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	meta, err := s.manager.Get(req.GetExecutionId(), ownerCN)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &agentv1.GetExecutionResponse{Execution: toProtoExecution(meta)}, nil
}

func (s *Service) ListExecutions(ctx context.Context, req *agentv1.ListExecutionsRequest) (*agentv1.ListExecutionsResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	runningOnly := false
	if req != nil && req.RunningOnly != nil {
		runningOnly = req.GetRunningOnly()
	}
	metas := s.manager.List(ownerCN, runningOnly)
	resp := &agentv1.ListExecutionsResponse{Executions: make([]*agentv1.Execution, 0, len(metas))}
	for _, meta := range metas {
		resp.Executions = append(resp.Executions, toProtoExecution(meta))
	}
	return resp, nil
}

func (s *Service) DeleteExecution(ctx context.Context, req *agentv1.DeleteExecutionRequest) (*agentv1.DeleteExecutionResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetExecutionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "execution_id is required")
	}
	if err := s.manager.Delete(req.GetExecutionId(), ownerCN); err != nil {
		return nil, grpcErr(err)
	}
	return &agentv1.DeleteExecutionResponse{ExecutionId: req.GetExecutionId()}, nil
}

func (s *Service) ClearHistory(ctx context.Context, _ *agentv1.ClearHistoryRequest) (*agentv1.ClearHistoryResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	deleted, skipped, err := s.manager.ClearHistory(ownerCN)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &agentv1.ClearHistoryResponse{
		DeletedCount:        uint64(deleted),
		SkippedRunningCount: uint64(skipped),
	}, nil
}

func (s *Service) CancelExecution(ctx context.Context, req *agentv1.CancelExecutionRequest) (*agentv1.CancelExecutionResponse, error) {
	ownerCN, err := ownerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	grace := time.Duration(req.GetGracePeriodSeconds()) * time.Second
	meta, err := s.manager.Cancel(req.GetExecutionId(), ownerCN, grace)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &agentv1.CancelExecutionResponse{Execution: toProtoExecution(meta)}, nil
}

func (s *Service) ReadOutput(req *agentv1.ReadOutputRequest, stream grpc.ServerStreamingServer[agentv1.OutputChunk]) error {
	ownerCN, err := ownerFromContext(stream.Context())
	if err != nil {
		return err
	}
	if req.GetExecutionId() == "" {
		return status.Error(codes.InvalidArgument, "execution_id is required")
	}
	var seq int64
	err = s.manager.StreamOutput(stream.Context(), req.GetExecutionId(), ownerCN, req.GetOffset(), func(evt execution.OutputEvent) error {
		seq++
		return stream.Send(&agentv1.OutputChunk{
			ExecutionId: req.GetExecutionId(),
			Sequence:    seq,
			Offset:      evt.Offset,
			Source:      toProtoSource(evt.Source),
			Timestamp:   timestamppb.New(evt.Timestamp),
			Data:        evt.Data,
		})
	}, req.GetDiscardAfterRead())
	if err != nil {
		return grpcErr(err)
	}
	return stream.Send(&agentv1.OutputChunk{
		ExecutionId: req.GetExecutionId(),
		Sequence:    seq + 1,
		Eof:         true,
	})
}

func (s *Service) Attach(stream grpc.BidiStreamingServer[agentv1.AttachRequest, agentv1.AttachEvent]) error {
	ownerCN, err := ownerFromContext(stream.Context())
	if err != nil {
		return err
	}
	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return status.Error(codes.InvalidArgument, "attach start message is required")
		}
		return err
	}
	start := first.GetStart()
	if start == nil {
		return status.Error(codes.InvalidArgument, "first attach message must be AttachStart")
	}
	if start.GetExecutionId() == "" {
		return status.Error(codes.InvalidArgument, "execution_id is required")
	}

	liveEvents, detach, meta, err := s.manager.Attach(start.GetExecutionId(), ownerCN)
	if err != nil {
		return grpcErr(err)
	}
	defer detach()

	if err := stream.Send(&agentv1.AttachEvent{Payload: &agentv1.AttachEvent_Ack{Ack: &agentv1.AttachAck{Execution: toProtoExecution(meta)}}}); err != nil {
		return err
	}
	if start.GetReplayBuffered() {
		err = s.manager.StreamOutput(stream.Context(), start.GetExecutionId(), ownerCN, start.GetReplayFromOffset(), func(evt execution.OutputEvent) error {
			return stream.Send(&agentv1.AttachEvent{Payload: &agentv1.AttachEvent_Output{Output: &agentv1.OutputChunk{
				ExecutionId: evt.ExecutionID,
				Sequence:    evt.Sequence,
				Offset:      evt.Offset,
				Source:      toProtoSource(evt.Source),
				Timestamp:   timestamppb.New(evt.Timestamp),
				Data:        evt.Data,
			}}})
		}, false)
		if err != nil {
			return grpcErr(err)
		}
	}

	recvErr := make(chan error, 1)
	go func() {
		// Attach reads client control frames concurrently while the main goroutine
		// forwards live output back to the caller.
		for {
			req, err := stream.Recv()
			if err != nil {
				recvErr <- err
				return
			}
			if stdin := req.GetStdin(); stdin != nil {
				if err := s.manager.WriteStdin(start.GetExecutionId(), ownerCN, stdin.GetData(), stdin.GetEof()); err != nil {
					recvErr <- err
					return
				}
				continue
			}
			if control := req.GetControl(); control != nil && control.GetCancelExecution() {
				if _, err := s.manager.Cancel(start.GetExecutionId(), ownerCN, 0); err != nil {
					recvErr <- err
					return
				}
			}
		}
	}()

	for {
		select {
		case err := <-recvErr:
			if err == nil || errors.Is(err, io.EOF) {
				return nil
			}
			return grpcErr(err)
		case evt, ok := <-liveEvents:
			if !ok || evt.EOF {
				latest, err := s.manager.Get(start.GetExecutionId(), ownerCN)
				if err != nil {
					return grpcErr(err)
				}
				return stream.Send(&agentv1.AttachEvent{Payload: &agentv1.AttachEvent_Exit{Exit: &agentv1.ExitEvent{Execution: toProtoExecution(latest)}}})
			}
			if err := stream.Send(&agentv1.AttachEvent{Payload: &agentv1.AttachEvent_Output{Output: &agentv1.OutputChunk{
				ExecutionId: evt.ExecutionID,
				Sequence:    evt.Sequence,
				Offset:      evt.Offset,
				Source:      toProtoSource(evt.Source),
				Timestamp:   timestamppb.New(evt.Timestamp),
				Data:        evt.Data,
			}}}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *Service) UploadFile(stream grpc.ClientStreamingServer[agentv1.UploadFileRequest, agentv1.UploadFileResponse]) (retErr error) {
	ownerCN, err := ownerFromContext(stream.Context())
	if err != nil {
		return err
	}
	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return status.Error(codes.InvalidArgument, "upload metadata is required")
		}
		return err
	}
	meta := first.GetMetadata()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first upload frame must be metadata")
	}
	if meta.GetPath() == "" {
		return status.Error(codes.InvalidArgument, "path is required")
	}

	var total *int64
	transfer, err := s.manager.StartTransfer(ownerCN, execution.KindUpload, meta.GetClientLocalPath(), meta.GetPath(), total)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	flags := os.O_CREATE | os.O_WRONLY
	if meta.Overwrite != nil && !meta.GetOverwrite() {
		flags |= os.O_EXCL
	} else {
		flags |= os.O_TRUNC
	}
	if err := os.MkdirAll(filepath.Dir(meta.GetPath()), 0o755); err != nil {
		_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
		return status.Error(codes.Internal, err.Error())
	}
	f, err := os.OpenFile(meta.GetPath(), flags, os.FileMode(meta.GetFileMode()))
	if err != nil {
		_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
		return status.Error(codes.Internal, err.Error())
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	var written int64
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, true, err.Error())
			return err
		}
		chunk := req.GetChunk()
		if len(chunk) == 0 {
			continue
		}
		if _, err := f.Write(chunk); err != nil {
			_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
			return status.Error(codes.Internal, err.Error())
		}
		_, _ = hasher.Write(chunk)
		written += int64(len(chunk))
		_ = s.manager.UpdateTransferProgress(transfer.ID, written, nil)
	}

	if _, err := s.manager.CompleteTransfer(transfer.ID, ownerCN, false, ""); err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return stream.SendAndClose(&agentv1.UploadFileResponse{
		Path:         meta.GetPath(),
		BytesWritten: uint64(written),
		Sha256:       hex.EncodeToString(hasher.Sum(nil)),
		TransferId:   transfer.ID,
	})
}

func (s *Service) DownloadFile(req *agentv1.DownloadFileRequest, stream grpc.ServerStreamingServer[agentv1.FileChunk]) error {
	ownerCN, err := ownerFromContext(stream.Context())
	if err != nil {
		return err
	}
	if req.GetPath() == "" {
		return status.Error(codes.InvalidArgument, "path is required")
	}
	info, err := os.Stat(req.GetPath())
	if err != nil {
		if os.IsNotExist(err) {
			return status.Error(codes.NotFound, "file not found")
		}
		return status.Error(codes.Internal, err.Error())
	}
	if info.IsDir() {
		return status.Error(codes.InvalidArgument, "path must be a file")
	}
	total := info.Size()
	transfer, err := s.manager.StartTransfer(ownerCN, execution.KindDownload, req.GetClientLocalPath(), req.GetPath(), &total)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	f, err := os.Open(req.GetPath())
	if err != nil {
		_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
		return status.Error(codes.Internal, err.Error())
	}
	defer func() { _ = f.Close() }()
	chunkSize := int(req.GetChunkSize())
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	buf := make([]byte, chunkSize)
	var offset uint64
	for {
		n, err := f.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := stream.Send(&agentv1.FileChunk{
				Data:       chunk,
				Offset:     offset,
				TransferId: transfer.ID,
			}); err != nil {
				_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, true, err.Error())
				return err
			}
			offset += uint64(n)
			_ = s.manager.UpdateTransferProgress(transfer.ID, int64(offset), &total)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
			return status.Error(codes.Internal, err.Error())
		}
	}
	_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, "")
	return stream.Send(&agentv1.FileChunk{Offset: offset, Eof: true, TransferId: transfer.ID})
}

func (s *Service) DownloadArchive(req *agentv1.DownloadArchiveRequest, stream grpc.ServerStreamingServer[agentv1.FileChunk]) error {
	ownerCN, err := ownerFromContext(stream.Context())
	if err != nil {
		return err
	}
	if len(req.GetPaths()) == 0 {
		return status.Error(codes.InvalidArgument, "paths are required")
	}
	transfer, err := s.manager.StartTransfer(ownerCN, execution.KindArchiveDownload, req.GetClientLocalPath(), strings.Join(req.GetPaths(), ","), nil)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	pr, pw := io.Pipe()
	go func() {
		zw := zip.NewWriter(pw)
		var writeErr error
		for _, root := range req.GetPaths() {
			writeErr = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				rel := filepath.Base(root)
				if root != path {
					if r, err := filepath.Rel(filepath.Dir(root), path); err == nil {
						rel = r
					}
				}
				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}
				header.Name = filepath.ToSlash(rel)
				w, err := zw.CreateHeader(header)
				if err != nil {
					return err
				}
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(w, f)
				return err
			})
			if writeErr != nil {
				break
			}
		}
		if closeErr := zw.Close(); writeErr == nil {
			writeErr = closeErr
		}
		_ = pw.CloseWithError(writeErr)
	}()

	chunkSize := int(req.GetChunkSize())
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	buf := make([]byte, chunkSize)
	var offset uint64
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := stream.Send(&agentv1.FileChunk{Data: chunk, Offset: offset, TransferId: transfer.ID}); err != nil {
				_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, true, err.Error())
				return err
			}
			offset += uint64(n)
			_ = s.manager.UpdateTransferProgress(transfer.ID, int64(offset), nil)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, err.Error())
			return status.Error(codes.Internal, err.Error())
		}
	}
	_, _ = s.manager.CompleteTransfer(transfer.ID, ownerCN, false, "")
	return stream.Send(&agentv1.FileChunk{Offset: offset, Eof: true, TransferId: transfer.ID})
}

func ownerFromContext(ctx context.Context) (string, error) {
	ownerCN, ok := auth.IdentityFromContext(ctx)
	if !ok || ownerCN == "" {
		return "", status.Error(codes.Unauthenticated, "missing authenticated identity")
	}
	return ownerCN, nil
}

func mapCommandSpec(req *agentv1.StartCommandRequest) (execution.CommandSpec, error) {
	switch spec := req.GetCommandSpec().(type) {
	case *agentv1.StartCommandRequest_Argv:
		return execution.CommandSpec{Argv: &execution.ArgvSpec{Binary: spec.Argv.GetBinary(), Args: spec.Argv.GetArgs()}}, nil
	case *agentv1.StartCommandRequest_Shell:
		return execution.CommandSpec{Shell: &execution.ShellSpec{ShellBinary: spec.Shell.GetShellBinary(), Command: spec.Shell.GetCommand()}}, nil
	default:
		return execution.CommandSpec{}, errors.New("command spec is required")
	}
}

func grpcErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, execution.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, execution.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, execution.ErrAlreadyAttached):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, execution.ErrRunningHistoryDelete):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoExecution(view execution.ExecutionView) *agentv1.Execution {
	msg := &agentv1.Execution{
		ExecutionId:            view.ID,
		Kind:                   toProtoKind(view.Kind),
		State:                  toProtoState(view.State),
		Pid:                    view.PID,
		OwnerCn:                view.OwnerCN,
		CommandArgv:            append([]string{}, view.CommandArgv...),
		CommandShell:           view.CommandShell,
		StartedAt:              timestamppb.New(view.StartedAt),
		ExitCode:               valueOrZero(view.ExitCode),
		Signal:                 view.Signal,
		OutputSizeBytes:        view.OutputSizeBytes,
		LastUploadLocalPath:    view.LastUploadLocalPath,
		LastUploadRemotePath:   view.LastUploadRemotePath,
		LastDownloadLocalPath:  view.LastDownloadLocalPath,
		LastDownloadRemotePath: view.LastDownloadRemotePath,
		LastUploadTransferId:   view.LastUploadTransferID,
		LastDownloadTransferId: view.LastDownloadTransferID,
		TransferProgressBytes:  view.TransferProgressBytes,
		TransferDirection:      view.TransferDirection,
		ErrorMessage:           view.ErrorMessage,
	}
	if view.EndedAt != nil {
		msg.EndedAt = timestamppb.New(*view.EndedAt)
	}
	if view.TransferTotalBytes != nil {
		msg.TransferTotalBytes = *view.TransferTotalBytes
	}
	return msg
}

func toProtoKind(kind execution.Kind) agentv1.ExecutionKind {
	switch kind {
	case execution.KindCommand:
		return agentv1.ExecutionKind_EXECUTION_KIND_COMMAND
	case execution.KindShell:
		return agentv1.ExecutionKind_EXECUTION_KIND_SHELL_SESSION
	case execution.KindUpload:
		return agentv1.ExecutionKind_EXECUTION_KIND_UPLOAD
	case execution.KindDownload:
		return agentv1.ExecutionKind_EXECUTION_KIND_DOWNLOAD
	case execution.KindArchiveDownload:
		return agentv1.ExecutionKind_EXECUTION_KIND_ARCHIVE_DOWNLOAD
	default:
		return agentv1.ExecutionKind_EXECUTION_KIND_UNSPECIFIED
	}
}

func toProtoState(state execution.State) agentv1.ExecutionState {
	switch state {
	case execution.StateRunning:
		return agentv1.ExecutionState_EXECUTION_STATE_RUNNING
	case execution.StateExited:
		return agentv1.ExecutionState_EXECUTION_STATE_EXITED
	case execution.StateCanceled:
		return agentv1.ExecutionState_EXECUTION_STATE_CANCELED
	case execution.StateFailedToStart:
		return agentv1.ExecutionState_EXECUTION_STATE_FAILED_TO_START
	default:
		return agentv1.ExecutionState_EXECUTION_STATE_UNSPECIFIED
	}
}

func toProtoSource(source execution.Source) agentv1.OutputSource {
	switch source {
	case execution.SourceStdout:
		return agentv1.OutputSource_OUTPUT_SOURCE_STDOUT
	case execution.SourceStderr:
		return agentv1.OutputSource_OUTPUT_SOURCE_STDERR
	case execution.SourceSystem:
		return agentv1.OutputSource_OUTPUT_SOURCE_SYSTEM
	default:
		return agentv1.OutputSource_OUTPUT_SOURCE_UNSPECIFIED
	}
}

func valueOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}
