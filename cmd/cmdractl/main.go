package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	agentv1 "cmdra/gen/agent/v1"
	"cmdra/internal/buildinfo"
	"cmdra/pkg/cmdraclient"
	"golang.org/x/term"
)

type connectionFlags struct {
	address            string
	caFile             string
	clientCertFile     string
	clientKeyFile      string
	serverName         string
	insecureSkipVerify bool
	timeout            time.Duration
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage()
		return 0
	}
	if args[0] == "version" {
		fmt.Fprintln(os.Stdout, buildinfo.Summary("cmdractl"))
		return 0
	}

	connFlags, subArgs, err := parseConnectionFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(subArgs) == 0 {
		printUsage()
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), connFlags.timeout)
	defer cancel()
	client, err := cmdraclient.Dial(ctx, cmdraclient.DialConfig{
		Address:            connFlags.address,
		CAFile:             connFlags.caFile,
		ClientCertFile:     connFlags.clientCertFile,
		ClientKeyFile:      connFlags.clientKeyFile,
		ServerName:         connFlags.serverName,
		InsecureSkipVerify: connFlags.insecureSkipVerify,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = client.Close() }()

	if err := runSubcommand(client, subArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func parseConnectionFlags(args []string) (connectionFlags, []string, error) {
	fs := flag.NewFlagSet("cmdractl", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	cfg := connectionFlags{address: "127.0.0.1:8443", timeout: 30 * time.Second}
	fs.StringVar(&cfg.address, "address", cfg.address, "cmdrad address")
	fs.StringVar(&cfg.caFile, "ca", "", "CA PEM path")
	fs.StringVar(&cfg.clientCertFile, "cert", "", "client certificate PEM path")
	fs.StringVar(&cfg.clientKeyFile, "key", "", "client private key PEM path")
	fs.StringVar(&cfg.serverName, "server-name", "", "TLS server name override")
	fs.BoolVar(&cfg.insecureSkipVerify, "insecure-skip-verify", false, "skip server certificate hostname verification")
	fs.DurationVar(&cfg.timeout, "timeout", cfg.timeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return connectionFlags{}, nil, err
	}
	if cfg.caFile == "" || cfg.clientCertFile == "" || cfg.clientKeyFile == "" {
		return connectionFlags{}, nil, errors.New("--ca, --cert, and --key are required")
	}
	return cfg, fs.Args(), nil
}

func runSubcommand(client *cmdraclient.Client, args []string) error {
	switch args[0] {
	case "start-argv":
		return runStartArgv(client, args[1:])
	case "start-shell":
		return runStartShell(client, args[1:])
	case "start-session":
		return runStartSession(client, args[1:])
	case "list":
		return runList(client, args[1:])
	case "get":
		return runGet(client, args[1:])
	case "delete":
		return runDelete(client, args[1:])
	case "clear-history":
		return runClearHistory(client, args[1:])
	case "cancel":
		return runCancel(client, args[1:])
	case "output":
		return runOutput(client, args[1:])
	case "attach":
		return runAttach(client, args[1:])
	case "upload":
		return runUpload(client, args[1:])
	case "download":
		return runDownload(client, args[1:])
	case "download-archive":
		return runDownloadArchive(client, args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runStartArgv(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("start-argv", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var binary string
	fs.StringVar(&binary, "binary", "", "remote binary path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if binary == "" {
		return errors.New("--binary is required")
	}
	execMeta, err := client.StartArgv(context.Background(), binary, fs.Args())
	if err != nil {
		return err
	}
	printExecution(execMeta)
	return nil
}

func runStartShell(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("start-shell", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var shell, command string
	var usePTY bool
	var ptyRows, ptyCols uint
	fs.StringVar(&shell, "shell", "", "shell binary")
	fs.StringVar(&command, "command", "", "shell command")
	fs.BoolVar(&usePTY, "pty", false, "run the shell command under a PTY")
	fs.UintVar(&ptyRows, "pty-rows", 0, "initial PTY row count")
	fs.UintVar(&ptyCols, "pty-cols", 0, "initial PTY column count")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if command == "" {
		return errors.New("--command is required")
	}
	execMeta, err := client.StartShellCommandWithOptions(context.Background(), shell, command, cmdraclient.ShellOptions{
		UsePTY:  usePTY,
		PTYRows: uint32(ptyRows),
		PTYCols: uint32(ptyCols),
	})
	if err != nil {
		return err
	}
	printExecution(execMeta)
	return nil
}

func runStartSession(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("start-session", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var shell string
	var usePTY bool
	var ptyRows, ptyCols uint
	fs.StringVar(&shell, "shell", "", "shell binary")
	fs.BoolVar(&usePTY, "pty", false, "run the shell session under a PTY")
	fs.UintVar(&ptyRows, "pty-rows", 0, "initial PTY row count")
	fs.UintVar(&ptyCols, "pty-cols", 0, "initial PTY column count")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if shell == "" {
		return errors.New("--shell is required")
	}
	execMeta, err := client.StartShellSessionWithOptions(context.Background(), shell, fs.Args(), cmdraclient.ShellOptions{
		UsePTY:  usePTY,
		PTYRows: uint32(ptyRows),
		PTYCols: uint32(ptyCols),
	})
	if err != nil {
		return err
	}
	printExecution(execMeta)
	return nil
}

func runList(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var runningOnly bool
	fs.BoolVar(&runningOnly, "running-only", false, "only show running executions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var running *bool
	if fs.Lookup("running-only") != nil {
		running = &runningOnly
	}
	items, err := client.ListExecutions(context.Background(), running)
	if err != nil {
		return err
	}
	for i, item := range items {
		if i > 0 {
			fmt.Println()
		}
		printExecution(item)
	}
	return nil
}

func runGet(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var id string
	var discard bool
	var jsonOutput bool
	fs.StringVar(&id, "id", "", "execution or transfer id")
	fs.BoolVar(&discard, "discard-output", false, "discard persisted output after replay if the job is finished")
	fs.BoolVar(&jsonOutput, "json", false, "print combined metadata and output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if id == "" {
		return errors.New("--id is required")
	}
	details, err := client.GetExecutionWithOutput(context.Background(), id, discard)
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(details)
	}
	printExecution(details.Execution)
	if len(details.Output) > 0 {
		fmt.Println()
		fmt.Println("Output:")
		for _, chunk := range details.Output {
			if chunk.GetEof() {
				continue
			}
			source := strings.TrimPrefix(chunk.GetSource().String(), "OUTPUT_SOURCE_")
			fmt.Printf("[%s] %s", source, string(chunk.GetData()))
			if len(chunk.GetData()) == 0 || chunk.GetData()[len(chunk.GetData())-1] != '\n' {
				fmt.Println()
			}
		}
	}
	return nil
}

func runDelete(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var id string
	fs.StringVar(&id, "id", "", "execution or transfer id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if id == "" {
		return errors.New("--id is required")
	}
	if err := client.DeleteExecution(context.Background(), id); err != nil {
		return err
	}
	fmt.Printf("Deleted ID: %s\n", id)
	return nil
}

func runClearHistory(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("clear-history", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var yes bool
	fs.BoolVar(&yes, "yes", false, "confirm clearing finished history for the authenticated identity")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !yes {
		return errors.New("--yes is required to clear history")
	}
	result, err := client.ClearHistory(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Deleted Count: %d\n", result.DeletedCount)
	fmt.Printf("Skipped Running Count: %d\n", result.SkippedRunningCount)
	return nil
}

func runCancel(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var id string
	var grace time.Duration
	fs.StringVar(&id, "id", "", "execution id")
	fs.DurationVar(&grace, "grace", 5*time.Second, "grace period before force kill")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if id == "" {
		return errors.New("--id is required")
	}
	execMeta, err := client.CancelExecution(context.Background(), id, grace)
	if err != nil {
		return err
	}
	printExecution(execMeta)
	return nil
}

func runOutput(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("output", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var id string
	var offset int64
	var discard bool
	fs.StringVar(&id, "id", "", "execution id")
	fs.Int64Var(&offset, "offset", 0, "starting output byte offset")
	fs.BoolVar(&discard, "discard-output", false, "discard persisted output after replay if the job is finished")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if id == "" {
		return errors.New("--id is required")
	}
	output, err := client.ReadOutput(context.Background(), id, offset, discard)
	if err != nil {
		return err
	}
	for _, chunk := range output {
		if chunk.GetEof() {
			continue
		}
		_, _ = os.Stdout.Write(chunk.GetData())
	}
	return nil
}

func runAttach(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var id string
	var replayBuffered bool
	var replayFromOffset int64
	fs.StringVar(&id, "id", "", "execution id")
	fs.BoolVar(&replayBuffered, "replay-buffered", true, "replay buffered output before live attach")
	fs.Int64Var(&replayFromOffset, "replay-from-offset", 0, "replay output from byte offset")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if id == "" {
		return errors.New("--id is required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	session, err := client.Attach(ctx, id, replayBuffered, replayFromOffset)
	if err != nil {
		return err
	}

	stdinStarted := false
	restoreRaw := func() {}
	defer restoreRaw()

	for {
		evt, err := session.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		switch payload := evt.GetPayload().(type) {
		case *agentv1.AttachEvent_Ack:
			printExecution(payload.Ack.GetExecution())
			if payload.Ack.GetExecution().GetUsesPty() {
				restoreRaw = enterRawMode()
				go watchTerminalSize(ctx, session)
			}
			if !stdinStarted {
				stdinStarted = true
				go pumpAttachInput(session)
			}
		case *agentv1.AttachEvent_Output:
			_, _ = os.Stdout.Write(payload.Output.GetData())
		case *agentv1.AttachEvent_Exit:
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "Execution exited:")
			printExecution(payload.Exit.GetExecution())
			return nil
		case *agentv1.AttachEvent_Error:
			return errors.New(payload.Error.GetMessage())
		}
	}
}

func pumpAttachInput(session *cmdraclient.AttachSession) {
	buf := make([]byte, 32*1024)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			payload := make([]byte, n)
			copy(payload, buf[:n])
			if sendErr := session.SendStdin(payload, false); sendErr != nil {
				return
			}
		}
		if err != nil {
			_ = session.SendStdin(nil, true)
			return
		}
	}
}

func enterRawMode() func() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return func() {}
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return func() {}
	}
	return func() {
		_ = term.Restore(fd, state)
	}
}

func watchTerminalSize(ctx context.Context, session *cmdraclient.AttachSession) {
	rows, cols, ok := currentTerminalSize()
	if ok {
		_ = session.ResizePTY(rows, cols)
	}
	lastRows, lastCols := rows, cols
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, cols, ok = currentTerminalSize()
			if !ok {
				continue
			}
			if rows == lastRows && cols == lastCols {
				continue
			}
			if err := session.ResizePTY(rows, cols); err != nil {
				return
			}
			lastRows, lastCols = rows, cols
		}
	}
}

func currentTerminalSize() (uint32, uint32, bool) {
	for _, f := range []*os.File{os.Stdout, os.Stdin} {
		fd := int(f.Fd())
		if !term.IsTerminal(fd) {
			continue
		}
		cols, rows, err := term.GetSize(fd)
		if err != nil || rows <= 0 || cols <= 0 {
			continue
		}
		return uint32(rows), uint32(cols), true
	}
	return 0, 0, false
}

func runUpload(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var localPath, remotePath string
	var overwrite bool
	var fileMode uint
	var async bool
	fs.StringVar(&localPath, "local", "", "local file path")
	fs.StringVar(&remotePath, "remote", "", "remote file path")
	fs.BoolVar(&overwrite, "overwrite", true, "overwrite existing remote path")
	fs.UintVar(&fileMode, "mode", 0, "file mode in octal or decimal")
	fs.BoolVar(&async, "async", false, "run the transfer in a background goroutine")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if localPath == "" || remotePath == "" {
		return errors.New("--local and --remote are required")
	}
	opts := cmdraclient.UploadOptions{Overwrite: &overwrite, FileMode: os.FileMode(fileMode)}
	if async {
		resp, err := client.UploadFileAsync(context.Background(), localPath, remotePath, opts).Wait()
		if err != nil {
			return err
		}
		printUploadResponse(resp)
		return nil
	}
	resp, err := client.UploadFile(context.Background(), localPath, remotePath, opts)
	if err != nil {
		return err
	}
	printUploadResponse(resp)
	return nil
}

func runDownload(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var localPath, remotePath string
	var chunkSize int
	var async bool
	fs.StringVar(&localPath, "local", "", "local file path")
	fs.StringVar(&remotePath, "remote", "", "remote file path")
	fs.IntVar(&chunkSize, "chunk-size", 32*1024, "download chunk size")
	fs.BoolVar(&async, "async", false, "run the transfer in a background goroutine")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if localPath == "" || remotePath == "" {
		return errors.New("--local and --remote are required")
	}
	opts := cmdraclient.DownloadOptions{ChunkSize: chunkSize}
	if async {
		resp, err := client.DownloadFileAsync(context.Background(), remotePath, localPath, opts).Wait()
		if err != nil {
			return err
		}
		printDownloadResponse(localPath, remotePath, resp)
		return nil
	}
	resp, err := client.DownloadFile(context.Background(), remotePath, localPath, opts)
	if err != nil {
		return err
	}
	printDownloadResponse(localPath, remotePath, resp)
	return nil
}

func runDownloadArchive(client *cmdraclient.Client, args []string) error {
	fs := flag.NewFlagSet("download-archive", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var localPath string
	var chunkSize int
	var async bool
	var paths multiStringFlag
	fs.StringVar(&localPath, "local", "", "local archive path")
	fs.IntVar(&chunkSize, "chunk-size", 32*1024, "download chunk size")
	fs.BoolVar(&async, "async", false, "run the transfer in a background goroutine")
	fs.Var(&paths, "path", "remote path to include in the archive (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if localPath == "" || len(paths) == 0 {
		return errors.New("--local and at least one --path are required")
	}
	opts := cmdraclient.DownloadOptions{ChunkSize: chunkSize}
	if async {
		resp, err := client.DownloadArchiveAsync(context.Background(), paths, localPath, opts).Wait()
		if err != nil {
			return err
		}
		printArchiveDownloadResponse(localPath, paths, resp)
		return nil
	}
	resp, err := client.DownloadArchive(context.Background(), paths, localPath, opts)
	if err != nil {
		return err
	}
	printArchiveDownloadResponse(localPath, paths, resp)
	return nil
}

type multiStringFlag []string

func (m *multiStringFlag) String() string { return strings.Join(*m, ",") }
func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func printExecution(execMeta *agentv1.Execution) {
	if execMeta == nil {
		return
	}
	fmt.Printf("ID: %s\n", execMeta.GetExecutionId())
	fmt.Printf("Kind: %s\n", strings.TrimPrefix(execMeta.GetKind().String(), "EXECUTION_KIND_"))
	fmt.Printf("State: %s\n", strings.TrimPrefix(execMeta.GetState().String(), "EXECUTION_STATE_"))
	fmt.Printf("Owner CN: %s\n", execMeta.GetOwnerCn())
	if execMeta.GetPid() != 0 {
		fmt.Printf("PID: %d\n", execMeta.GetPid())
	}
	fmt.Printf("Started At: %s\n", execMeta.GetStartedAt().AsTime().Format(time.RFC3339Nano))
	if execMeta.GetEndedAt() != nil {
		fmt.Printf("Ended At: %s\n", execMeta.GetEndedAt().AsTime().Format(time.RFC3339Nano))
	}
	if argv := execMeta.GetCommandArgv(); len(argv) > 0 {
		fmt.Printf("Command Argv: %s\n", strings.Join(argv, " "))
	}
	if execMeta.GetCommandShell() != "" {
		fmt.Printf("Command Shell: %s\n", execMeta.GetCommandShell())
	}
	if execMeta.GetUsesPty() {
		fmt.Printf("Uses PTY: %t\n", execMeta.GetUsesPty())
		if execMeta.GetPtyRows() > 0 && execMeta.GetPtyCols() > 0 {
			fmt.Printf("PTY Size: %dx%d\n", execMeta.GetPtyRows(), execMeta.GetPtyCols())
		}
	}
	if execMeta.GetLastUploadLocalPath() != "" {
		fmt.Printf("Upload Local Path: %s\n", execMeta.GetLastUploadLocalPath())
	}
	if execMeta.GetLastUploadRemotePath() != "" {
		fmt.Printf("Upload Remote Path: %s\n", execMeta.GetLastUploadRemotePath())
	}
	if execMeta.GetLastUploadTransferId() != "" {
		fmt.Printf("Upload Transfer ID: %s\n", execMeta.GetLastUploadTransferId())
	}
	if execMeta.GetLastDownloadLocalPath() != "" {
		fmt.Printf("Download Local Path: %s\n", execMeta.GetLastDownloadLocalPath())
	}
	if execMeta.GetLastDownloadRemotePath() != "" {
		fmt.Printf("Download Remote Path: %s\n", execMeta.GetLastDownloadRemotePath())
	}
	if execMeta.GetLastDownloadTransferId() != "" {
		fmt.Printf("Download Transfer ID: %s\n", execMeta.GetLastDownloadTransferId())
	}
	if execMeta.GetTransferDirection() != "" {
		fmt.Printf("Transfer Direction: %s\n", execMeta.GetTransferDirection())
		fmt.Printf("Transfer Progress Bytes: %d\n", execMeta.GetTransferProgressBytes())
		if execMeta.GetTransferTotalBytes() != 0 {
			fmt.Printf("Transfer Total Bytes: %d\n", execMeta.GetTransferTotalBytes())
		}
	}
	fmt.Printf("Output Size Bytes: %d\n", execMeta.GetOutputSizeBytes())
	fmt.Printf("Exit Code: %d\n", execMeta.GetExitCode())
	if execMeta.GetSignal() != "" {
		fmt.Printf("Signal: %s\n", execMeta.GetSignal())
	}
	if execMeta.GetErrorMessage() != "" {
		fmt.Printf("Error: %s\n", execMeta.GetErrorMessage())
	}
}

func printUploadResponse(resp *agentv1.UploadFileResponse) {
	fmt.Printf("Path: %s\n", resp.GetPath())
	fmt.Printf("Bytes Written: %d\n", resp.GetBytesWritten())
	fmt.Printf("SHA256: %s\n", resp.GetSha256())
	fmt.Printf("Transfer ID: %s\n", resp.GetTransferId())
}

func printDownloadResponse(localPath, remotePath string, resp *cmdraclient.DownloadResult) {
	fmt.Printf("Local Path: %s\n", localPath)
	fmt.Printf("Remote Path: %s\n", remotePath)
	fmt.Printf("Bytes Written: %d\n", resp.BytesWritten)
	fmt.Printf("Transfer ID: %s\n", resp.TransferID)
}

func printArchiveDownloadResponse(localPath string, remotePaths []string, resp *cmdraclient.DownloadResult) {
	fmt.Printf("Local Path: %s\n", localPath)
	fmt.Printf("Remote Paths: %s\n", strings.Join(remotePaths, ","))
	fmt.Printf("Bytes Written: %d\n", resp.BytesWritten)
	fmt.Printf("Transfer ID: %s\n", resp.TransferID)
}

func printUsage() {
	fmt.Fprintln(os.Stdout, `Usage:
  cmdractl [connection flags] <subcommand> [flags]

Connection flags:
  --address
  --ca
  --cert
  --key
  --server-name
  --insecure-skip-verify
  --timeout

Subcommands:
  version
  start-argv
  start-shell
  start-session
  list
  get
  delete
  clear-history
  cancel
  output
  attach
  upload
  download
  download-archive`)
}
