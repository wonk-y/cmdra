package main

import (
	"strings"
	"testing"
	"time"
)

func TestRunArgsFromConfig(t *testing.T) {
	cfg := Config{
		ListenAddress:   "127.0.0.1:8443",
		ServerCertFile:  "/tmp/server.crt",
		ServerKeyFile:   "/tmp/server.key",
		ClientCAFile:    "/tmp/ca.crt",
		AllowedClientCN: "client-a,client-b",
		DataDir:         "/tmp/data",
		AuditLogPath:    "/tmp/data/audit.log",
		ChunkSize:       32768,
		FlushInterval:   100 * time.Millisecond,
		GracePeriod:     5 * time.Second,
	}
	args := runArgsFromConfig("", cfg)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--listen-address 127.0.0.1:8443") {
		t.Fatalf("unexpected args: %s", joined)
	}
	if !strings.Contains(joined, "--data-dir /tmp/data") {
		t.Fatalf("unexpected args: %s", joined)
	}
}

func TestRenderSystemdUnit(t *testing.T) {
	cfg := serviceConfig{Name: "cmdrad", Description: "Cmdra daemon", Binary: "/usr/local/bin/cmdrad"}
	body := renderSystemdUnit(cfg, []string{"service", "run", "--config", "/etc/cmdrad.json"})
	if !strings.Contains(body, "Description=Cmdra daemon") {
		t.Fatalf("missing description: %s", body)
	}
	if !strings.Contains(body, "ExecStart=/usr/local/bin/cmdrad service run --config /etc/cmdrad.json") {
		t.Fatalf("missing exec start: %s", body)
	}
}

func TestRenderLaunchdPlist(t *testing.T) {
	cfg := serviceConfig{Name: "cmdrad", Binary: "/usr/local/bin/cmdrad"}
	body := renderLaunchdPlist(cfg, []string{"service", "run", "--config", "/usr/local/etc/cmdrad.json"})
	if !strings.Contains(body, "<string>com.cmdra.cmdrad</string>") {
		t.Fatalf("missing launchd label: %s", body)
	}
	if !strings.Contains(body, "<string>/usr/local/bin/cmdrad</string>") {
		t.Fatalf("missing binary path: %s", body)
	}
}

func TestWindowsCommandLine(t *testing.T) {
	got := windowsCommandLine([]string{`C:\Program Files\Cmdra\cmdrad.exe`, "service", "run", "--config", `C:\Program Files\Cmdra\cmdrad.json`})
	if !strings.Contains(got, `"C:\Program Files\Cmdra\cmdrad.exe"`) {
		t.Fatalf("missing quoted binary path: %s", got)
	}
	if !strings.Contains(got, `"C:\Program Files\Cmdra\cmdrad.json"`) {
		t.Fatalf("missing quoted config path: %s", got)
	}
}
