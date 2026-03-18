package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	agentv1 "cmdra/gen/agent/v1"
	"cmdra/internal/audit"
	"cmdra/internal/auth"
	"cmdra/internal/execution"
	"cmdra/internal/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestCmdractlLifecycle(t *testing.T) {
	requireUnixCommandsCtl(t)
	env := newCtlEnv(t)

	out := env.runCtl(t, "start-argv", "--binary", "/bin/echo", "hello-cli")
	id := mustMatch(t, `ID:\s+(\S+)`, out)

	getOut := env.runCtl(t, "get", "--id", id)
	if !strings.Contains(getOut, "Command Argv: /bin/echo hello-cli") {
		t.Fatalf("missing argv metadata in get output:\n%s", getOut)
	}
	if !strings.Contains(getOut, "[STDOUT] hello-cli") {
		t.Fatalf("missing stdout in get output:\n%s", getOut)
	}

	listOut := env.runCtl(t, "list")
	if !strings.Contains(listOut, "ID: "+id) {
		t.Fatalf("missing execution in list output:\n%s", listOut)
	}
}

func TestCmdractlFileTransferMetadata(t *testing.T) {
	requireUnixCommandsCtl(t)
	env := newCtlEnv(t)

	localUpload := filepath.Join(env.dir, "upload.txt")
	if err := os.WriteFile(localUpload, []byte("upload via cli\n"), 0o644); err != nil {
		t.Fatalf("write upload source: %v", err)
	}
	remotePath := filepath.Join(env.dir, "remote.txt")
	uploadOut := env.runCtl(t, "upload", "--local", localUpload, "--remote", remotePath)
	uploadID := mustMatch(t, `Transfer ID:\s+(\S+)`, uploadOut)

	getUpload := env.runCtl(t, "get", "--id", uploadID)
	if !strings.Contains(getUpload, "Upload Local Path: "+localUpload) || !strings.Contains(getUpload, "Upload Remote Path: "+remotePath) {
		t.Fatalf("missing upload metadata in get output:\n%s", getUpload)
	}

	localDownload := filepath.Join(env.dir, "download.txt")
	downloadOut := env.runCtl(t, "download", "--remote", remotePath, "--local", localDownload)
	downloadID := mustMatch(t, `Transfer ID:\s+(\S+)`, downloadOut)
	if data, err := os.ReadFile(localDownload); err != nil || string(data) != "upload via cli\n" {
		t.Fatalf("unexpected downloaded content: %q err=%v", data, err)
	}

	getDownload := env.runCtl(t, "get", "--id", downloadID)
	if !strings.Contains(getDownload, "Download Local Path: "+localDownload) || !strings.Contains(getDownload, "Download Remote Path: "+remotePath) {
		t.Fatalf("missing download metadata in get output:\n%s", getDownload)
	}
}

func TestCmdractlStartShellWithPTY(t *testing.T) {
	requireUnixCommandsCtl(t)
	env := newCtlEnv(t)

	out := env.runCtl(t, "start-shell", "--shell", "/bin/sh", "--command", "printf 'cli-pty\\n'", "--pty", "--pty-rows", "24", "--pty-cols", "80")
	id := mustMatch(t, `ID:\s+(\S+)`, out)

	getOut := waitForCtlOutput(t, env, id)
	if !strings.Contains(getOut, "Uses PTY: true") {
		t.Fatalf("missing PTY metadata in get output:\n%s", getOut)
	}
	if !strings.Contains(getOut, "PTY Size: 24x80") {
		t.Fatalf("missing PTY size in get output:\n%s", getOut)
	}
	if !strings.Contains(getOut, "[STDOUT] cli-pty") {
		t.Fatalf("missing PTY output in get output:\n%s", getOut)
	}
}

func TestCmdractlWriteStdin(t *testing.T) {
	requireUnixCommandsCtl(t)
	env := newCtlEnv(t)

	startOut := env.runCtl(t, "start-shell", "--shell", "/bin/sh", "--command", "read line; printf '%s\\n' \"$line\"")
	id := mustMatch(t, `ID:\s+(\S+)`, startOut)

	stdinOut := env.runCtl(t, "stdin", "--id", id, "--data", "cli-stdin\n", "--eof")
	if !strings.Contains(stdinOut, "Sent stdin to ID: "+id) {
		t.Fatalf("unexpected stdin output:\n%s", stdinOut)
	}
	if !strings.Contains(stdinOut, "Bytes Sent: 10") {
		t.Fatalf("expected byte count in stdin output:\n%s", stdinOut)
	}
	if !strings.Contains(stdinOut, "EOF: true") {
		t.Fatalf("expected EOF marker in stdin output:\n%s", stdinOut)
	}

	getOut := waitForCtlOutput(t, env, id)
	if !strings.Contains(getOut, "[STDOUT] cli-stdin") {
		t.Fatalf("missing stdin-fed output in get output:\n%s", getOut)
	}
}

func TestCmdractlVersion(t *testing.T) {
	env := newCtlEnv(t)
	out := env.runCtl(t, "version")
	if !strings.Contains(out, "cmdractl") {
		t.Fatalf("unexpected version output: %s", out)
	}
}

func TestCmdractlDeleteAndClearHistory(t *testing.T) {
	requireUnixCommandsCtl(t)
	env := newCtlEnv(t)

	deleteOut := env.runCtl(t, "start-argv", "--binary", "/bin/echo", "delete-cli")
	deleteID := mustMatch(t, `ID:\s+(\S+)`, deleteOut)
	env.runCtl(t, "delete", "--id", deleteID)
	if output, err := env.runCtlErr("get", "--id", deleteID); err == nil || !strings.Contains(output, "execution not found") {
		t.Fatalf("expected deleted CLI execution to be missing, err=%v output=%s", err, output)
	}

	finishedOut := env.runCtl(t, "start-argv", "--binary", "/bin/echo", "clear-cli")
	finishedID := mustMatch(t, `ID:\s+(\S+)`, finishedOut)

	localUpload := filepath.Join(env.dir, "clear-upload.txt")
	if err := os.WriteFile(localUpload, []byte("clear-history via cli\n"), 0o644); err != nil {
		t.Fatalf("write clear-history upload source: %v", err)
	}
	remotePath := filepath.Join(env.dir, "clear-remote.txt")
	uploadOut := env.runCtl(t, "upload", "--local", localUpload, "--remote", remotePath)
	uploadID := mustMatch(t, `Transfer ID:\s+(\S+)`, uploadOut)

	runningOut := env.runCtl(t, "start-shell", "--shell", "/bin/sh", "--command", "sleep 30")
	runningID := mustMatch(t, `ID:\s+(\S+)`, runningOut)

	clearOut := env.runCtl(t, "clear-history", "--yes")
	if !strings.Contains(clearOut, "Deleted Count: 2") || !strings.Contains(clearOut, "Skipped Running Count: 1") {
		t.Fatalf("unexpected clear-history output:\n%s", clearOut)
	}

	if output, err := env.runCtlErr("get", "--id", finishedID); err == nil || !strings.Contains(output, "execution not found") {
		t.Fatalf("expected cleared finished execution to be missing, err=%v output=%s", err, output)
	}
	if output, err := env.runCtlErr("get", "--id", uploadID); err == nil || !strings.Contains(output, "execution not found") {
		t.Fatalf("expected cleared transfer to be missing, err=%v output=%s", err, output)
	}

	getRunning := env.runCtl(t, "get", "--id", runningID)
	if !strings.Contains(getRunning, "State: RUNNING") {
		t.Fatalf("expected running execution to remain after clear-history:\n%s", getRunning)
	}
	if output, err := env.runCtlErr("delete", "--id", runningID); err == nil || !strings.Contains(output, "cannot delete running execution or transfer") {
		t.Fatalf("expected delete of running execution to fail, err=%v output=%s", err, output)
	}
	env.runCtl(t, "cancel", "--id", runningID)
}

type ctlEnv struct {
	t          *testing.T
	dir        string
	address    string
	caFile     string
	certFile   string
	keyFile    string
	server     *grpc.Server
	listener   net.Listener
	manager    *execution.Manager
	auditLog   *audit.Logger
	binaryPath string
}

func newCtlEnv(t *testing.T) *ctlEnv {
	t.Helper()
	dir := t.TempDir()
	caFile, serverCert, serverKey, clientCert, clientKey := writeCtlCerts(t, dir)
	auditLog, err := audit.New(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	manager, err := execution.NewManager(execution.Config{
		DataDir:            filepath.Join(dir, "data"),
		ChunkSize:          32 * 1024,
		FlushInterval:      100 * time.Millisecond,
		DefaultGracePeriod: 2 * time.Second,
		AuditLogger:        auditLog,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	tlsCfg, err := loadCtlServerTLS(serverCert, serverKey, caFile)
	if err != nil {
		t.Fatalf("load server TLS: %v", err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(auth.ParseAllowList("client-a"))),
		grpc.StreamInterceptor(auth.StreamAuthInterceptor(auth.ParseAllowList("client-a"))),
	)
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewService(manager))
	go func() { _ = grpcServer.Serve(lis) }()

	binPath := filepath.Join(dir, "cmdractl")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", binPath, "./cmd/cmdractl")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build cmdractl: %v\n%s", err, string(output))
	}

	env := &ctlEnv{
		t:          t,
		dir:        dir,
		address:    lis.Addr().String(),
		caFile:     caFile,
		certFile:   clientCert,
		keyFile:    clientKey,
		server:     grpcServer,
		listener:   lis,
		manager:    manager,
		auditLog:   auditLog,
		binaryPath: binPath,
	}
	t.Cleanup(env.close)
	return env
}

func (e *ctlEnv) close() {
	if e.server != nil {
		e.server.Stop()
	}
	if e.listener != nil {
		_ = e.listener.Close()
	}
	if e.manager != nil {
		_ = e.manager.Close()
	}
	if e.auditLog != nil {
		_ = e.auditLog.Close()
	}
}

func (e *ctlEnv) runCtl(t *testing.T, args ...string) string {
	t.Helper()
	if len(args) == 1 && args[0] == "version" {
		cmd := exec.Command(e.binaryPath, "version")
		cmd.Dir = filepath.Join("..", "..")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run cmdractl version: %v\n%s", err, string(output))
		}
		return string(output)
	}
	base := []string{
		"--address", e.address,
		"--ca", e.caFile,
		"--cert", e.certFile,
		"--key", e.keyFile,
		"--server-name", "localhost",
	}
	cmd := exec.Command(e.binaryPath, append(base, args...)...)
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run cmdractl %v: %v\n%s", args, err, string(output))
	}
	return string(output)
}

func (e *ctlEnv) runCtlErr(args ...string) (string, error) {
	if len(args) == 1 && args[0] == "version" {
		cmd := exec.Command(e.binaryPath, "version")
		cmd.Dir = filepath.Join("..", "..")
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
	base := []string{
		"--address", e.address,
		"--ca", e.caFile,
		"--cert", e.certFile,
		"--key", e.keyFile,
		"--server-name", "localhost",
	}
	cmd := exec.Command(e.binaryPath, append(base, args...)...)
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func mustMatch(t *testing.T, pattern, input string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		t.Fatalf("pattern %q not found in output:\n%s", pattern, input)
	}
	return matches[1]
}

func waitForCtlOutput(t *testing.T, env *ctlEnv, id string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out := env.runCtl(t, "get", "--id", id)
		if !strings.Contains(out, "State: RUNNING") {
			return out
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("execution %s did not finish", id)
	return ""
}

func requireUnixCommandsCtl(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("cmdractl integration tests currently target Unix-like command paths")
	}
}

func writeCtlCerts(t *testing.T, dir string) (string, string, string, string, string) {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          bigInt(1),
		Subject:               pkix.Name{CommonName: "Cmdra CLI Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caFile := filepath.Join(dir, "ca.crt")
	writeCtlPEM(t, caFile, "CERTIFICATE", caDER)

	writePair := func(prefix, cn string, eku []x509.ExtKeyUsage, dns []string, ips []net.IP) (string, string) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate %s key: %v", prefix, err)
		}
		template := &x509.Certificate{
			SerialNumber: bigInt(time.Now().UnixNano()),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  eku,
			DNSNames:     dns,
			IPAddresses:  ips,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caTemplate, &key.PublicKey, caKey)
		if err != nil {
			t.Fatalf("create %s cert: %v", prefix, err)
		}
		certFile := filepath.Join(dir, prefix+".crt")
		keyFile := filepath.Join(dir, prefix+".key")
		writeCtlPEM(t, certFile, "CERTIFICATE", der)
		writeCtlPEM(t, keyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
		return certFile, keyFile
	}

	serverCert, serverKey := writePair("server", "localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	clientCert, clientKey := writePair("client-a", "client-a", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil, nil)
	return caFile, serverCert, serverKey, clientCert, clientKey
}

func writeCtlPEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	if err := pem.Encode(f, &pem.Block{Type: typ, Bytes: der}); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func loadCtlServerTLS(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, os.ErrInvalid
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}, nil
}

func bigInt(v int64) *big.Int {
	return big.NewInt(v)
}
