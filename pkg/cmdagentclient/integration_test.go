package cmdagentclient

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	agentv1 "cmdagent/gen/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientExecutionLifecycle(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	execMeta, err := env.clientA.StartArgv(context.Background(), "/bin/echo", []string{"hello-go-client"})
	if err != nil {
		t.Fatalf("start argv: %v", err)
	}
	finished := waitForCompletion(t, env.clientA, execMeta.GetExecutionId())
	if finished.GetState() != agentv1.ExecutionState_EXECUTION_STATE_EXITED {
		t.Fatalf("unexpected state: %v", finished.GetState())
	}
	if got := finished.GetCommandArgv(); len(got) != 2 || got[0] != "/bin/echo" || got[1] != "hello-go-client" {
		t.Fatalf("unexpected argv metadata: %v", got)
	}

	details, err := env.clientA.GetExecutionWithOutput(context.Background(), execMeta.GetExecutionId(), false)
	if err != nil {
		t.Fatalf("get execution with output: %v", err)
	}
	var output bytes.Buffer
	for _, chunk := range details.Output {
		if !chunk.GetEof() {
			output.Write(chunk.GetData())
		}
	}
	if got := output.String(); got != "hello-go-client\n" {
		t.Fatalf("unexpected output: %q", got)
	}

	items, err := env.clientA.ListExecutions(context.Background(), nil)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	found := false
	for _, item := range items {
		if item.GetExecutionId() == execMeta.GetExecutionId() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("execution %s not found in list", execMeta.GetExecutionId())
	}
}

func TestClientAsyncExecutionLifecycle(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	argvExec, err := env.clientA.StartArgvAsync(context.Background(), "/bin/echo", []string{"hello-async-argv"}).Wait()
	if err != nil {
		t.Fatalf("start argv async: %v", err)
	}
	argvFinished := waitForCompletion(t, env.clientA, argvExec.GetExecutionId())
	if argvFinished.GetState() != agentv1.ExecutionState_EXECUTION_STATE_EXITED {
		t.Fatalf("unexpected argv async state: %v", argvFinished.GetState())
	}
	argvDetails, err := env.clientA.GetExecutionWithOutput(context.Background(), argvExec.GetExecutionId(), false)
	if err != nil {
		t.Fatalf("get argv async execution with output: %v", err)
	}
	var argvOutput bytes.Buffer
	for _, chunk := range argvDetails.Output {
		if !chunk.GetEof() {
			argvOutput.Write(chunk.GetData())
		}
	}
	if got := argvOutput.String(); got != "hello-async-argv\n" {
		t.Fatalf("unexpected argv async output: %q", got)
	}

	shellExec, err := env.clientA.StartShellCommandAsync(context.Background(), "/bin/sh", "printf 'hello-async-shell\\n'").Wait()
	if err != nil {
		t.Fatalf("start shell command async: %v", err)
	}
	shellFinished := waitForCompletion(t, env.clientA, shellExec.GetExecutionId())
	if shellFinished.GetState() != agentv1.ExecutionState_EXECUTION_STATE_EXITED {
		t.Fatalf("unexpected shell async state: %v", shellFinished.GetState())
	}
	shellDetails, err := env.clientA.GetExecutionWithOutput(context.Background(), shellExec.GetExecutionId(), false)
	if err != nil {
		t.Fatalf("get shell async execution with output: %v", err)
	}
	var shellOutput bytes.Buffer
	for _, chunk := range shellDetails.Output {
		if !chunk.GetEof() {
			shellOutput.Write(chunk.GetData())
		}
	}
	if got := shellOutput.String(); got != "hello-async-shell\n" {
		t.Fatalf("unexpected shell async output: %q", got)
	}
}

func TestClientAttachShellSession(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	execMeta, err := env.clientA.StartShellSession(context.Background(), "/bin/sh", nil)
	if err != nil {
		t.Fatalf("start shell session: %v", err)
	}
	session, err := env.clientA.Attach(context.Background(), execMeta.GetExecutionId(), true, 0)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer func() { _ = session.CloseSend() }()

	ack, err := session.Recv()
	if err != nil {
		t.Fatalf("recv ack: %v", err)
	}
	if ack.GetAck() == nil || ack.GetAck().GetExecution().GetExecutionId() != execMeta.GetExecutionId() {
		t.Fatalf("unexpected attach ack: %+v", ack)
	}
	if err := session.SendStdin([]byte("printf 'attached-output\\n'\nexit\n"), true); err != nil {
		t.Fatalf("send stdin: %v", err)
	}

	var output bytes.Buffer
	for {
		evt, err := session.Recv()
		if err != nil {
			t.Fatalf("recv event: %v", err)
		}
		switch payload := evt.GetPayload().(type) {
		case *agentv1.AttachEvent_Output:
			output.Write(payload.Output.GetData())
		case *agentv1.AttachEvent_Exit:
			if !bytes.Contains(output.Bytes(), []byte("attached-output")) {
				t.Fatalf("missing attached output in %q", output.String())
			}
			return
		}
	}
}

func TestClientAsyncShellSessionAttach(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	execMeta, err := env.clientA.StartShellSessionAsync(context.Background(), "/bin/sh", nil).Wait()
	if err != nil {
		t.Fatalf("start shell session async: %v", err)
	}
	session, err := env.clientA.Attach(context.Background(), execMeta.GetExecutionId(), true, 0)
	if err != nil {
		t.Fatalf("attach async shell session: %v", err)
	}
	defer func() { _ = session.CloseSend() }()

	if _, err := session.Recv(); err != nil {
		t.Fatalf("recv async shell ack: %v", err)
	}
	if err := session.SendStdin([]byte("printf 'attached-async-session\\n'\nexit\n"), true); err != nil {
		t.Fatalf("send stdin async shell session: %v", err)
	}

	var output bytes.Buffer
	for {
		evt, err := session.Recv()
		if err != nil {
			t.Fatalf("recv async shell session event: %v", err)
		}
		switch payload := evt.GetPayload().(type) {
		case *agentv1.AttachEvent_Output:
			output.Write(payload.Output.GetData())
		case *agentv1.AttachEvent_Exit:
			if !bytes.Contains(output.Bytes(), []byte("attached-async-session")) {
				t.Fatalf("missing async attached output in %q", output.String())
			}
			return
		}
	}
}

func TestClientFileTransfersAndAsync(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	localUpload := filepath.Join(env.dir, "upload.txt")
	if err := os.WriteFile(localUpload, []byte("upload body\n"), 0o644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	remotePath := filepath.Join(env.dir, "remote.txt")
	uploadResp, err := env.clientA.UploadFileAsync(context.Background(), localUpload, remotePath, UploadOptions{}).Wait()
	if err != nil {
		t.Fatalf("upload async: %v", err)
	}
	if uploadResp.GetTransferId() == "" {
		t.Fatal("expected upload transfer id")
	}
	uploadMeta, err := env.clientA.GetExecution(context.Background(), uploadResp.GetTransferId())
	if err != nil {
		t.Fatalf("get upload metadata: %v", err)
	}
	if uploadMeta.GetLastUploadLocalPath() != localUpload || uploadMeta.GetLastUploadRemotePath() != remotePath {
		t.Fatalf("unexpected upload metadata: %+v", uploadMeta)
	}

	localDownload := filepath.Join(env.dir, "download.txt")
	downloadResp, err := env.clientA.DownloadFileAsync(context.Background(), remotePath, localDownload, DownloadOptions{}).Wait()
	if err != nil {
		t.Fatalf("download async: %v", err)
	}
	if downloadResp.TransferID == "" {
		t.Fatal("expected download transfer id")
	}
	content, err := os.ReadFile(localDownload)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != "upload body\n" {
		t.Fatalf("unexpected downloaded content: %q", content)
	}
	downloadMeta, err := env.clientA.GetExecution(context.Background(), downloadResp.TransferID)
	if err != nil {
		t.Fatalf("get download metadata: %v", err)
	}
	if downloadMeta.GetLastDownloadLocalPath() != localDownload || downloadMeta.GetLastDownloadRemotePath() != remotePath {
		t.Fatalf("unexpected download metadata: %+v", downloadMeta)
	}

	archiveSource := filepath.Join(env.dir, "archive-source")
	if err := os.MkdirAll(archiveSource, 0o755); err != nil {
		t.Fatalf("create archive source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveSource, "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatalf("write archive file: %v", err)
	}
	archiveLocal := filepath.Join(env.dir, "bundle.zip")
	archiveResp, err := env.clientA.DownloadArchive(context.Background(), []string{archiveSource}, archiveLocal, DownloadOptions{ChunkSize: 8 * 1024})
	if err != nil {
		t.Fatalf("download archive: %v", err)
	}
	if archiveResp.TransferID == "" {
		t.Fatal("expected archive transfer id")
	}
	reader, err := zip.OpenReader(archiveLocal)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = reader.Close() }()
	if len(reader.File) == 0 {
		t.Fatal("expected archive contents")
	}

	asyncArchiveLocal := filepath.Join(env.dir, "bundle-async.zip")
	asyncArchiveResp, err := env.clientA.DownloadArchiveAsync(context.Background(), []string{archiveSource}, asyncArchiveLocal, DownloadOptions{ChunkSize: 8 * 1024}).Wait()
	if err != nil {
		t.Fatalf("download archive async: %v", err)
	}
	if asyncArchiveResp.TransferID == "" {
		t.Fatal("expected async archive transfer id")
	}
	asyncReader, err := zip.OpenReader(asyncArchiveLocal)
	if err != nil {
		t.Fatalf("open async archive: %v", err)
	}
	defer func() { _ = asyncReader.Close() }()
	if len(asyncReader.File) == 0 {
		t.Fatal("expected async archive contents")
	}
}

func TestClientCrossIdentityAuthorization(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	execMeta, err := env.clientA.StartShellCommand(context.Background(), "/bin/sh", "printf 'owned-by-a\\n'")
	if err != nil {
		t.Fatalf("start shell command: %v", err)
	}
	waitForCompletion(t, env.clientA, execMeta.GetExecutionId())

	_, err = env.clientB.GetExecution(context.Background(), execMeta.GetExecutionId())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied from GetExecution, got %v", err)
	}
	_, err = env.clientB.ReadOutput(context.Background(), execMeta.GetExecutionId(), 0, false)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied from ReadOutput, got %v", err)
	}
	err = env.clientB.DeleteExecution(context.Background(), execMeta.GetExecutionId())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied from DeleteExecution, got %v", err)
	}
}

func TestClientCancelExecution(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	execMeta, err := env.clientA.StartShellCommand(context.Background(), "/bin/sh", "sleep 30")
	if err != nil {
		t.Fatalf("start long-running shell command: %v", err)
	}
	if _, err := env.clientA.CancelExecution(context.Background(), execMeta.GetExecutionId(), time.Second); err != nil {
		t.Fatalf("cancel execution: %v", err)
	}
	finished := waitForCompletion(t, env.clientA, execMeta.GetExecutionId())
	if finished.GetState() != agentv1.ExecutionState_EXECUTION_STATE_CANCELED {
		t.Fatalf("expected canceled state, got %v", finished.GetState())
	}
}

func TestClientDeleteExecutionAndClearHistory(t *testing.T) {
	requireUnixCommands(t)
	env := newIntegrationEnv(t)

	deletedExec, err := env.clientA.StartArgv(context.Background(), "/bin/echo", []string{"delete-me"})
	if err != nil {
		t.Fatalf("start argv for delete: %v", err)
	}
	waitForCompletion(t, env.clientA, deletedExec.GetExecutionId())
	if err := env.clientA.DeleteExecution(context.Background(), deletedExec.GetExecutionId()); err != nil {
		t.Fatalf("delete execution: %v", err)
	}
	_, err = env.clientA.GetExecution(context.Background(), deletedExec.GetExecutionId())
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}

	finishedExec, err := env.clientA.StartArgv(context.Background(), "/bin/echo", []string{"clear-me"})
	if err != nil {
		t.Fatalf("start argv for clear-history: %v", err)
	}
	waitForCompletion(t, env.clientA, finishedExec.GetExecutionId())

	localUpload := filepath.Join(env.dir, "clear-upload.txt")
	if err := os.WriteFile(localUpload, []byte("clear-history upload\n"), 0o644); err != nil {
		t.Fatalf("write upload source: %v", err)
	}
	remotePath := filepath.Join(env.dir, "clear-remote.txt")
	uploadResp, err := env.clientA.UploadFile(context.Background(), localUpload, remotePath, UploadOptions{})
	if err != nil {
		t.Fatalf("upload for clear-history: %v", err)
	}

	runningExec, err := env.clientA.StartShellCommand(context.Background(), "/bin/sh", "sleep 30")
	if err != nil {
		t.Fatalf("start running execution: %v", err)
	}

	result, err := env.clientA.ClearHistory(context.Background())
	if err != nil {
		t.Fatalf("clear history: %v", err)
	}
	if result.DeletedCount != 2 {
		t.Fatalf("expected deleted_count=2, got %d", result.DeletedCount)
	}
	if result.SkippedRunningCount != 1 {
		t.Fatalf("expected skipped_running_count=1, got %d", result.SkippedRunningCount)
	}

	_, err = env.clientA.GetExecution(context.Background(), finishedExec.GetExecutionId())
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected finished execution to be deleted, got %v", err)
	}
	_, err = env.clientA.GetExecution(context.Background(), uploadResp.GetTransferId())
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected upload transfer to be deleted, got %v", err)
	}
	runningMeta, err := env.clientA.GetExecution(context.Background(), runningExec.GetExecutionId())
	if err != nil {
		t.Fatalf("get running execution after clear-history: %v", err)
	}
	if runningMeta.GetState() != agentv1.ExecutionState_EXECUTION_STATE_RUNNING {
		t.Fatalf("expected running execution to remain, got %v", runningMeta.GetState())
	}
	if err := env.clientA.DeleteExecution(context.Background(), runningExec.GetExecutionId()); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition deleting running execution, got %v", err)
	}

	if _, err := env.clientA.CancelExecution(context.Background(), runningExec.GetExecutionId(), time.Second); err != nil {
		t.Fatalf("cancel running execution: %v", err)
	}
	waitForCompletion(t, env.clientA, runningExec.GetExecutionId())
}
