package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	agentv1 "cmdra/gen/agent/v1"
	"cmdra/internal/audit"
	"cmdra/internal/auth"
	"cmdra/internal/execution"
	"cmdra/internal/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type runtimeHandle struct {
	listener   net.Listener
	grpcServer *grpc.Server
	manager    *execution.Manager
	auditLog   *audit.Logger
	serveErr   chan error
}

func runForeground(cfg Config) error {
	handle, err := startRuntime(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = handle.close() }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GracePeriod)
		defer cancel()
		if err := handle.stop(ctx); err != nil {
			return fmt.Errorf("stop after signal %s: %w", sig, err)
		}
		return nil
	case err := <-handle.serveErr:
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
}

func startRuntime(cfg Config) (*runtimeHandle, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}

	auditPath := cfg.AuditLogPath
	if auditPath == "" {
		auditPath = filepath.Join(cfg.DataDir, "audit.log")
	}
	auditLog, err := audit.New(auditPath)
	if err != nil {
		return nil, err
	}

	manager, err := execution.NewManager(execution.Config{
		DataDir:            cfg.DataDir,
		ChunkSize:          cfg.ChunkSize,
		FlushInterval:      cfg.FlushInterval,
		DefaultGracePeriod: cfg.GracePeriod,
		AuditLogger:        auditLog,
	})
	if err != nil {
		_ = auditLog.Close()
		return nil, err
	}

	tlsCfg, err := loadServerTLSConfig(cfg)
	if err != nil {
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, err
	}

	lis, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		_ = manager.Close()
		_ = auditLog.Close()
		return nil, err
	}

	allowed := auth.ParseAllowList(cfg.AllowedClientCN)
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(allowed)),
		grpc.StreamInterceptor(auth.StreamAuthInterceptor(allowed)),
	)
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewService(manager))

	handle := &runtimeHandle{
		listener:   lis,
		grpcServer: grpcServer,
		manager:    manager,
		auditLog:   auditLog,
		serveErr:   make(chan error, 1),
	}
	go func() {
		handle.serveErr <- grpcServer.Serve(lis)
	}()
	return handle, nil
}

func (h *runtimeHandle) stop(ctx context.Context) error {
	if h == nil || h.grpcServer == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.grpcServer.GracefulStop()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		h.grpcServer.Stop()
	}
	return nil
}

func (h *runtimeHandle) close() error {
	if h == nil {
		return nil
	}
	var firstErr error
	if h.listener != nil {
		if err := h.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) && firstErr == nil {
			firstErr = err
		}
	}
	if h.manager != nil {
		if err := h.manager.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if h.auditLog != nil {
		if err := h.auditLog.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func loadServerTLSConfig(cfg Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.ServerCertFile, cfg.ServerKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server certificate: %w", err)
	}
	caPEM, err := os.ReadFile(cfg.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read client CA: %w", err)
	}
	clientPool := x509.NewCertPool()
	if !clientPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("client CA PEM did not contain any certificates")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientPool,
	}, nil
}
