package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"cmdra/internal/buildinfo"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		if err := runRunCommand(nil); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	switch args[0] {
	case "run":
		if err := runRunCommand(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "service":
		if err := runServiceCommand(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "version":
		fmt.Fprintln(os.Stdout, buildinfo.Summary("cmdrad"))
		return 0
	case "-h", "--help", "help":
		printUsage()
		return 0
	default:
		// Backward compatibility keeps direct-flag invocation equivalent to `run`.
		if strings.HasPrefix(args[0], "-") {
			if err := runRunCommand(args); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", args[0])
		printUsage()
		return 1
	}
}

func runRunCommand(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	buf := new(bytes.Buffer)
	fs.SetOutput(buf)

	var overrides configOverrides
	var configPath string
	appendRunFlags(fs, &overrides, &configPath)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(os.Stdout, buf.String())
			return nil
		}
		return err
	}
	cfg, err := loadConfig(configPath, overrides)
	if err != nil {
		return err
	}
	return runForeground(cfg)
}

func printUsage() {
	fmt.Fprintln(os.Stdout, `Usage:
  cmdrad run [flags]
  cmdrad version
  cmdrad service <install|uninstall|start|stop|status|print|run> [flags]

Direct flag invocation remains supported and is treated as:
  cmdrad run [flags]`)
}
