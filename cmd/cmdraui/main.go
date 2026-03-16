package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"cmdra/internal/buildinfo"
	"cmdra/internal/tui"
	"cmdra/pkg/cmdraclient"
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
	if len(args) > 0 && args[0] == "version" {
		fmt.Fprintln(os.Stdout, buildinfo.Summary("cmdraui"))
		return 0
	}

	cfg, err := parseConnectionFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	client, err := cmdraclient.Dial(ctx, cmdraclient.DialConfig{
		Address:            cfg.address,
		CAFile:             cfg.caFile,
		ClientCertFile:     cfg.clientCertFile,
		ClientKeyFile:      cfg.clientKeyFile,
		ServerName:         cfg.serverName,
		InsecureSkipVerify: cfg.insecureSkipVerify,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = client.Close() }()

	if err := tui.Run(client, cmdraclient.DialConfig{
		Address:            cfg.address,
		CAFile:             cfg.caFile,
		ClientCertFile:     cfg.clientCertFile,
		ClientKeyFile:      cfg.clientKeyFile,
		ServerName:         cfg.serverName,
		InsecureSkipVerify: cfg.insecureSkipVerify,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func parseConnectionFlags(args []string) (connectionFlags, error) {
	fs := flag.NewFlagSet("cmdraui", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	cfg := connectionFlags{address: "127.0.0.1:8443", timeout: 15 * time.Second}
	fs.StringVar(&cfg.address, "address", cfg.address, "cmdrad address")
	fs.StringVar(&cfg.caFile, "ca", "", "CA PEM path")
	fs.StringVar(&cfg.clientCertFile, "cert", "", "client certificate PEM path")
	fs.StringVar(&cfg.clientKeyFile, "key", "", "client private key PEM path")
	fs.StringVar(&cfg.serverName, "server-name", "", "TLS server name override")
	fs.BoolVar(&cfg.insecureSkipVerify, "insecure-skip-verify", false, "skip server certificate hostname verification")
	fs.DurationVar(&cfg.timeout, "timeout", cfg.timeout, "initial dial timeout")
	if err := fs.Parse(args); err != nil {
		return connectionFlags{}, err
	}
	if cfg.caFile == "" || cfg.clientCertFile == "" || cfg.clientKeyFile == "" {
		return connectionFlags{}, errors.New("--ca, --cert, and --key are required")
	}
	return cfg, nil
}

func printUsage() {
	fmt.Fprintln(os.Stdout, `Usage:
  cmdraui [connection flags]
  cmdraui version

Connection flags:
  --address
  --ca
  --cert
  --key
  --server-name
  --insecure-skip-verify
  --timeout`)
}
