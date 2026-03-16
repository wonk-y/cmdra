package cmdraclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
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

type integrationEnv struct {
	t        testing.TB
	dir      string
	server   *grpc.Server
	listener net.Listener
	manager  *execution.Manager
	audit    *audit.Logger
	clientA  *Client
	clientB  *Client
}

func newIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()
	env, err := startIntegrationEnv(t.TempDir())
	if err != nil {
		t.Fatalf("start integration env: %v", err)
	}
	env.t = t
	t.Cleanup(env.close)
	return env
}

func (e *integrationEnv) close() {
	if e.clientA != nil {
		_ = e.clientA.Close()
	}
	if e.clientB != nil {
		_ = e.clientB.Close()
	}
	if e.server != nil {
		e.server.Stop()
	}
	if e.listener != nil {
		_ = e.listener.Close()
	}
	if e.manager != nil {
		_ = e.manager.Close()
	}
	if e.audit != nil {
		_ = e.audit.Close()
	}
}

type fatalHelper interface {
	Helper()
	Fatalf(string, ...any)
}

func waitForCompletion(t fatalHelper, client *Client, executionID string) *agentv1.Execution {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		execMeta, err := client.GetExecution(context.Background(), executionID)
		if err != nil {
			t.Fatalf("get execution %s: %v", executionID, err)
		}
		if execMeta.GetState() != agentv1.ExecutionState_EXECUTION_STATE_RUNNING {
			return execMeta
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("execution %s did not finish", executionID)
	return nil
}

func startIntegrationEnv(dir string) (*integrationEnv, error) {
	certs, err := writeTestCertificates(dir)
	if err != nil {
		return nil, err
	}
	auditLog, err := audit.New(filepath.Join(dir, "audit.log"))
	if err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}
	manager, err := execution.NewManager(execution.Config{
		DataDir:            filepath.Join(dir, "data"),
		ChunkSize:          32 * 1024,
		FlushInterval:      100 * time.Millisecond,
		DefaultGracePeriod: 2 * time.Second,
		AuditLogger:        auditLog,
	})
	if err != nil {
		_ = auditLog.Close()
		return nil, fmt.Errorf("create manager: %w", err)
	}
	serverTLS, err := loadServerTestTLS(certs.serverCertFile, certs.serverKeyFile, certs.caFile)
	if err != nil {
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, fmt.Errorf("load server TLS: %w", err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLS)),
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(auth.ParseAllowList("client-a,client-b"))),
		grpc.StreamInterceptor(auth.StreamAuthInterceptor(auth.ParseAllowList("client-a,client-b"))),
	)
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewService(manager))
	go func() {
		_ = grpcServer.Serve(lis)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientA, err := Dial(ctx, DialConfig{
		Address:        lis.Addr().String(),
		CAFile:         certs.caFile,
		ClientCertFile: certs.clientACertFile,
		ClientKeyFile:  certs.clientAKeyFile,
		ServerName:     "localhost",
		DialOptions:    []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		grpcServer.Stop()
		_ = lis.Close()
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, fmt.Errorf("dial client-a: %w", err)
	}
	clientB, err := Dial(ctx, DialConfig{
		Address:        lis.Addr().String(),
		CAFile:         certs.caFile,
		ClientCertFile: certs.clientBCertFile,
		ClientKeyFile:  certs.clientBKeyFile,
		ServerName:     "localhost",
		DialOptions:    []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		_ = clientA.Close()
		grpcServer.Stop()
		_ = lis.Close()
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, fmt.Errorf("dial client-b: %w", err)
	}

	return &integrationEnv{
		dir:      dir,
		server:   grpcServer,
		listener: lis,
		manager:  manager,
		audit:    auditLog,
		clientA:  clientA,
		clientB:  clientB,
	}, nil
}

func requireUnixCommands(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("these integration tests currently target Unix-like command paths")
	}
}

type testCertFiles struct {
	caFile          string
	serverCertFile  string
	serverKeyFile   string
	clientACertFile string
	clientAKeyFile  string
	clientBCertFile string
	clientBKeyFile  string
}

func writeTestCertificates(dir string) (testCertFiles, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return testCertFiles{}, fmt.Errorf("generate CA key: %w", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Cmdra Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return testCertFiles{}, fmt.Errorf("create CA cert: %w", err)
	}

	writePair := func(prefix string, cn string, eku []x509.ExtKeyUsage, dnsNames []string, ips []net.IP) (string, string, error) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", "", fmt.Errorf("generate key for %s: %w", prefix, err)
		}
		template := &x509.Certificate{
			SerialNumber: big.NewInt(int64(time.Now().UnixNano())),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  eku,
			DNSNames:     dnsNames,
			IPAddresses:  ips,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caTemplate, &key.PublicKey, caKey)
		if err != nil {
			return "", "", fmt.Errorf("create cert for %s: %w", prefix, err)
		}
		certFile := filepath.Join(dir, prefix+".crt")
		keyFile := filepath.Join(dir, prefix+".key")
		if err := writePEMFile(certFile, "CERTIFICATE", der); err != nil {
			return "", "", err
		}
		if err := writePEMFile(keyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)); err != nil {
			return "", "", err
		}
		return certFile, keyFile, nil
	}

	caFile := filepath.Join(dir, "ca.crt")
	if err := writePEMFile(caFile, "CERTIFICATE", caDER); err != nil {
		return testCertFiles{}, err
	}
	serverCertFile, serverKeyFile, err := writePair("server", "localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	if err != nil {
		return testCertFiles{}, err
	}
	clientACertFile, clientAKeyFile, err := writePair("client-a", "client-a", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil, nil)
	if err != nil {
		return testCertFiles{}, err
	}
	clientBCertFile, clientBKeyFile, err := writePair("client-b", "client-b", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil, nil)
	if err != nil {
		return testCertFiles{}, err
	}

	return testCertFiles{
		caFile:          caFile,
		serverCertFile:  serverCertFile,
		serverKeyFile:   serverKeyFile,
		clientACertFile: clientACertFile,
		clientAKeyFile:  clientAKeyFile,
		clientBCertFile: clientBCertFile,
		clientBKeyFile:  clientBKeyFile,
	}, nil
}

func writePEMFile(path string, blockType string, der []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func loadServerTestTLS(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append CA PEM")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	}, nil
}
