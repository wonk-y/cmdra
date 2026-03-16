package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type serviceConfig struct {
	Name        string
	DisplayName string
	Description string
	Binary      string
	StartNow    bool
	ConfigPath  string
	RunConfig   Config
}

func runServiceCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service action is required")
	}
	switch args[0] {
	case "install":
		return serviceInstall(args[1:])
	case "uninstall":
		return serviceUninstall(args[1:])
	case "start":
		return serviceStart(args[1:])
	case "stop":
		return serviceStop(args[1:])
	case "status":
		return serviceStatus(args[1:])
	case "print":
		return servicePrint(args[1:])
	case "run":
		return serviceRun(args[1:])
	default:
		return fmt.Errorf("unknown service action %q", args[0])
	}
}

func serviceInstall(args []string) error {
	cfg, err := parseServiceConfig("install", args, true)
	if err != nil {
		return err
	}
	spec, err := buildServiceSpec(cfg)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		if err := os.WriteFile(spec.Path, []byte(spec.Body), 0o644); err != nil {
			return err
		}
		if err := runCommand("systemctl", "daemon-reload"); err != nil {
			return err
		}
		if err := runCommand("systemctl", "enable", cfg.Name+".service"); err != nil {
			return err
		}
		if cfg.StartNow {
			return runCommand("systemctl", "start", cfg.Name+".service")
		}
		return nil
	case "darwin":
		if err := os.WriteFile(spec.Path, []byte(spec.Body), 0o644); err != nil {
			return err
		}
		if err := runCommand("launchctl", "load", "-w", spec.Path); err != nil {
			return err
		}
		if cfg.StartNow {
			return runCommand("launchctl", "start", spec.Label)
		}
		return nil
	case "windows":
		if err := runCommand("sc.exe", "create", cfg.Name, "binPath=", spec.Body, "start=", "auto", "DisplayName=", cfg.DisplayName); err != nil {
			return err
		}
		if cfg.Description != "" {
			if err := runCommand("sc.exe", "description", cfg.Name, cfg.Description); err != nil {
				return err
			}
		}
		if cfg.StartNow {
			return runCommand("sc.exe", "start", cfg.Name)
		}
		return nil
	default:
		return fmt.Errorf("service install is unsupported on %s", runtime.GOOS)
	}
}

func serviceUninstall(args []string) error {
	cfg, err := parseServiceConfig("uninstall", args, false)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		_ = runCommand("systemctl", "disable", "--now", cfg.Name+".service")
		if err := os.Remove(systemdUnitPath(cfg.Name)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return runCommand("systemctl", "daemon-reload")
	case "darwin":
		path := launchdPlistPath(cfg.Name)
		_ = runCommand("launchctl", "unload", "-w", path)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "windows":
		_ = runCommand("sc.exe", "stop", cfg.Name)
		return runCommand("sc.exe", "delete", cfg.Name)
	default:
		return fmt.Errorf("service uninstall is unsupported on %s", runtime.GOOS)
	}
}

func serviceStart(args []string) error {
	cfg, err := parseServiceConfig("start", args, false)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return runCommand("systemctl", "start", cfg.Name+".service")
	case "darwin":
		return runCommand("launchctl", "start", launchdLabel(cfg.Name))
	case "windows":
		return runCommand("sc.exe", "start", cfg.Name)
	default:
		return fmt.Errorf("service start is unsupported on %s", runtime.GOOS)
	}
}

func serviceStop(args []string) error {
	cfg, err := parseServiceConfig("stop", args, false)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return runCommand("systemctl", "stop", cfg.Name+".service")
	case "darwin":
		return runCommand("launchctl", "stop", launchdLabel(cfg.Name))
	case "windows":
		return runCommand("sc.exe", "stop", cfg.Name)
	default:
		return fmt.Errorf("service stop is unsupported on %s", runtime.GOOS)
	}
}

func serviceStatus(args []string) error {
	cfg, err := parseServiceConfig("status", args, false)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return streamCommand("systemctl", "status", cfg.Name+".service", "--no-pager")
	case "darwin":
		return streamCommand("launchctl", "print", "system/"+launchdLabel(cfg.Name))
	case "windows":
		return streamCommand("sc.exe", "query", cfg.Name)
	default:
		return fmt.Errorf("service status is unsupported on %s", runtime.GOOS)
	}
}

func servicePrint(args []string) error {
	cfg, err := parseServiceConfig("print", args, true)
	if err != nil {
		return err
	}
	spec, err := buildServiceSpec(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("Path: %s\n\n%s\n", spec.Path, spec.Body)
	return nil
}

func serviceRun(args []string) error {
	fs := flag.NewFlagSet("service run", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	var serviceName string
	var configPath string
	var overrides configOverrides
	fs.StringVar(&serviceName, "service-name", defaultServiceName(), "service name")
	appendRunFlags(fs, &overrides, &configPath)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(configPath, overrides)
	if err != nil {
		return err
	}
	return runServiceHost(serviceName, cfg)
}

type serviceSpec struct {
	Path  string
	Label string
	Body  string
}

func buildServiceSpec(cfg serviceConfig) (serviceSpec, error) {
	args := append([]string{"service", "run", "--service-name", cfg.Name}, runArgsFromConfig(cfg.ConfigPath, cfg.RunConfig)...)
	switch runtime.GOOS {
	case "linux":
		return serviceSpec{
			Path: systemdUnitPath(cfg.Name),
			Body: renderSystemdUnit(cfg, args),
		}, nil
	case "darwin":
		return serviceSpec{
			Path:  launchdPlistPath(cfg.Name),
			Label: launchdLabel(cfg.Name),
			Body:  renderLaunchdPlist(cfg, args),
		}, nil
	case "windows":
		return serviceSpec{
			Path: "",
			Body: windowsCommandLine(append([]string{cfg.Binary}, args...)),
		}, nil
	default:
		return serviceSpec{}, fmt.Errorf("service management is unsupported on %s", runtime.GOOS)
	}
}

func parseServiceConfig(action string, args []string, includeRunFlags bool) (serviceConfig, error) {
	fs := flag.NewFlagSet("service "+action, flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))

	cfg := serviceConfig{
		Name:        defaultServiceName(),
		DisplayName: "Cmdra",
		Description: "Cmdra gRPC daemon",
	}
	var overrides configOverrides
	fs.StringVar(&cfg.Name, "name", cfg.Name, "service name")
	fs.StringVar(&cfg.DisplayName, "display-name", cfg.DisplayName, "service display name")
	fs.StringVar(&cfg.Description, "description", cfg.Description, "service description")
	fs.StringVar(&cfg.Binary, "binary", "", "path to the cmdrad binary")
	fs.BoolVar(&cfg.StartNow, "start-now", false, "start the service immediately after install")
	if includeRunFlags {
		appendRunFlags(fs, &overrides, &cfg.ConfigPath)
	}
	if err := fs.Parse(args); err != nil {
		return serviceConfig{}, err
	}
	if cfg.Binary == "" {
		exe, err := os.Executable()
		if err != nil {
			return serviceConfig{}, err
		}
		cfg.Binary = exe
	}
	if strings.Contains(cfg.Binary, string(filepath.Separator)+"go-build") {
		return serviceConfig{}, fmt.Errorf("service install requires a built cmdrad binary, not go run")
	}
	if includeRunFlags {
		runCfg, err := loadConfig(cfg.ConfigPath, overrides)
		if err != nil {
			return serviceConfig{}, err
		}
		cfg.RunConfig = runCfg
	}
	return cfg, nil
}

func defaultServiceName() string {
	return "cmdrad"
}

func systemdUnitPath(name string) string {
	return filepath.Join("/etc/systemd/system", name+".service")
}

func launchdLabel(name string) string {
	return "com.cmdra." + name
}

func launchdPlistPath(name string) string {
	return filepath.Join("/Library/LaunchDaemons", launchdLabel(name)+".plist")
}

func renderSystemdUnit(cfg serviceConfig, args []string) string {
	return fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
`, cfg.Description, joinShellArgs(append([]string{cfg.Binary}, args...)))
}

func renderLaunchdPlist(cfg serviceConfig, args []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>` + xmlEscape(launchdLabel(cfg.Name)) + `</string>
  <key>ProgramArguments</key>
  <array>
`)
	for _, arg := range append([]string{cfg.Binary}, args...) {
		b.WriteString("    <string>" + xmlEscape(arg) + "</string>\n")
	}
	b.WriteString(`  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
`)
	return b.String()
}

func joinShellArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			quoted = append(quoted, "''")
			continue
		}
		if strings.IndexFunc(arg, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\'' || r == '"' || r == '\\'
		}) == -1 {
			quoted = append(quoted, arg)
			continue
		}
		quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", `'"'"'`)+"'")
	}
	return strings.Join(quoted, " ")
}

func windowsCommandLine(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			quoted = append(quoted, `""`)
			continue
		}
		if strings.IndexAny(arg, " \t\"") == -1 {
			quoted = append(quoted, arg)
			continue
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(arg, `"`, `\"`)+`"`)
	}
	return strings.Join(quoted, " ")
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func streamCommand(name string, args ...string) error {
	return runCommand(name, args...)
}
