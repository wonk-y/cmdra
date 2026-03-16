package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config controls the cmdrad runtime.
type Config struct {
	ListenAddress   string
	ServerCertFile  string
	ServerKeyFile   string
	ClientCAFile    string
	AllowedClientCN string
	DataDir         string
	AuditLogPath    string
	ChunkSize       int
	FlushInterval   time.Duration
	GracePeriod     time.Duration
}

type configFile struct {
	ListenAddress   string `json:"listen_address"`
	ServerCertFile  string `json:"server_cert_file"`
	ServerKeyFile   string `json:"server_key_file"`
	ClientCAFile    string `json:"client_ca_file"`
	AllowedClientCN string `json:"allowed_client_cn"`
	DataDir         string `json:"data_dir"`
	AuditLogPath    string `json:"audit_log_path"`
	ChunkSize       int    `json:"chunk_size"`
	FlushInterval   string `json:"flush_interval"`
	GracePeriod     string `json:"grace_period"`
}

type configOverrides struct {
	listenAddress   string
	serverCertFile  string
	serverKeyFile   string
	clientCAFile    string
	allowedClientCN string
	dataDir         string
	auditLogPath    string
	chunkSize       int
	flushInterval   time.Duration
	gracePeriod     time.Duration
}

func defaultConfig() Config {
	return Config{
		ListenAddress: "127.0.0.1:8443",
		DataDir:       "./data",
		ChunkSize:     32 * 1024,
		FlushInterval: 100 * time.Millisecond,
		GracePeriod:   5 * time.Second,
	}
}

func loadConfig(configPath string, overrides configOverrides) (Config, error) {
	cfg := defaultConfig()
	if configPath != "" {
		fileCfg, err := readConfigFile(configPath)
		if err != nil {
			return Config{}, err
		}
		cfg = fileCfg
	}
	cfg = applyOverrides(cfg, overrides)
	if cfg.AuditLogPath == "" {
		cfg.AuditLogPath = filepath.Join(cfg.DataDir, "audit.log")
	}
	if err := validateConfig(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func readConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw configFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg := defaultConfig()
	if raw.ListenAddress != "" {
		cfg.ListenAddress = raw.ListenAddress
	}
	if raw.ServerCertFile != "" {
		cfg.ServerCertFile = raw.ServerCertFile
	}
	if raw.ServerKeyFile != "" {
		cfg.ServerKeyFile = raw.ServerKeyFile
	}
	if raw.ClientCAFile != "" {
		cfg.ClientCAFile = raw.ClientCAFile
	}
	if raw.AllowedClientCN != "" {
		cfg.AllowedClientCN = raw.AllowedClientCN
	}
	if raw.DataDir != "" {
		cfg.DataDir = raw.DataDir
	}
	if raw.AuditLogPath != "" {
		cfg.AuditLogPath = raw.AuditLogPath
	}
	if raw.ChunkSize > 0 {
		cfg.ChunkSize = raw.ChunkSize
	}
	if raw.FlushInterval != "" {
		flush, err := time.ParseDuration(raw.FlushInterval)
		if err != nil {
			return Config{}, fmt.Errorf("parse flush_interval: %w", err)
		}
		cfg.FlushInterval = flush
	}
	if raw.GracePeriod != "" {
		grace, err := time.ParseDuration(raw.GracePeriod)
		if err != nil {
			return Config{}, fmt.Errorf("parse grace_period: %w", err)
		}
		cfg.GracePeriod = grace
	}

	baseDir := filepath.Dir(path)
	cfg.ServerCertFile = resolveConfigPath(baseDir, cfg.ServerCertFile)
	cfg.ServerKeyFile = resolveConfigPath(baseDir, cfg.ServerKeyFile)
	cfg.ClientCAFile = resolveConfigPath(baseDir, cfg.ClientCAFile)
	cfg.DataDir = resolveConfigPath(baseDir, cfg.DataDir)
	cfg.AuditLogPath = resolveConfigPath(baseDir, cfg.AuditLogPath)
	return cfg, nil
}

func applyOverrides(cfg Config, overrides configOverrides) Config {
	if overrides.listenAddress != "" {
		cfg.ListenAddress = overrides.listenAddress
	}
	if overrides.serverCertFile != "" {
		cfg.ServerCertFile = overrides.serverCertFile
	}
	if overrides.serverKeyFile != "" {
		cfg.ServerKeyFile = overrides.serverKeyFile
	}
	if overrides.clientCAFile != "" {
		cfg.ClientCAFile = overrides.clientCAFile
	}
	if overrides.allowedClientCN != "" {
		cfg.AllowedClientCN = overrides.allowedClientCN
	}
	if overrides.dataDir != "" {
		cfg.DataDir = overrides.dataDir
	}
	if overrides.auditLogPath != "" {
		cfg.AuditLogPath = overrides.auditLogPath
	}
	if overrides.chunkSize > 0 {
		cfg.ChunkSize = overrides.chunkSize
	}
	if overrides.flushInterval > 0 {
		cfg.FlushInterval = overrides.flushInterval
	}
	if overrides.gracePeriod > 0 {
		cfg.GracePeriod = overrides.gracePeriod
	}
	return cfg
}

func validateConfig(cfg *Config) error {
	switch {
	case strings.TrimSpace(cfg.ListenAddress) == "":
		return errors.New("listen address is required")
	case strings.TrimSpace(cfg.ServerCertFile) == "":
		return errors.New("server cert file is required")
	case strings.TrimSpace(cfg.ServerKeyFile) == "":
		return errors.New("server key file is required")
	case strings.TrimSpace(cfg.ClientCAFile) == "":
		return errors.New("client CA file is required")
	case strings.TrimSpace(cfg.DataDir) == "":
		return errors.New("data dir is required")
	}
	if cfg.ChunkSize <= 0 {
		return errors.New("chunk size must be positive")
	}
	if cfg.FlushInterval <= 0 {
		return errors.New("flush interval must be positive")
	}
	if cfg.GracePeriod <= 0 {
		return errors.New("grace period must be positive")
	}
	return nil
}

func resolveConfigPath(baseDir, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func appendRunFlags(fs *flag.FlagSet, overrides *configOverrides, configPath *string) {
	fs.StringVar(configPath, "config", "", "path to JSON config file")
	fs.StringVar(&overrides.listenAddress, "listen-address", "", "gRPC listen address")
	fs.StringVar(&overrides.serverCertFile, "server-cert", "", "server certificate PEM path")
	fs.StringVar(&overrides.serverKeyFile, "server-key", "", "server private key PEM path")
	fs.StringVar(&overrides.clientCAFile, "client-ca", "", "client CA PEM path")
	fs.StringVar(&overrides.allowedClientCN, "allowed-client-cn", "", "comma-separated client certificate CN allowlist")
	fs.StringVar(&overrides.dataDir, "data-dir", "", "directory for sqlite history and audit data")
	fs.StringVar(&overrides.auditLogPath, "audit-log", "", "audit log JSONL path")
	fs.IntVar(&overrides.chunkSize, "chunk-size", 0, "output/file transfer chunk size in bytes")
	fs.DurationVar(&overrides.flushInterval, "flush-interval", 0, "maximum delay before flushing short output chunks")
	fs.DurationVar(&overrides.gracePeriod, "grace-period", 0, "grace period before force-killing canceled commands")
}

func runArgsFromConfig(configPath string, cfg Config) []string {
	if configPath != "" {
		return []string{"--config", configPath}
	}
	return []string{
		"--listen-address", cfg.ListenAddress,
		"--server-cert", cfg.ServerCertFile,
		"--server-key", cfg.ServerKeyFile,
		"--client-ca", cfg.ClientCAFile,
		"--allowed-client-cn", cfg.AllowedClientCN,
		"--data-dir", cfg.DataDir,
		"--audit-log", cfg.AuditLogPath,
		"--chunk-size", fmt.Sprintf("%d", cfg.ChunkSize),
		"--flush-interval", cfg.FlushInterval.String(),
		"--grace-period", cfg.GracePeriod.String(),
	}
}
