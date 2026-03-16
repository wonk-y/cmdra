package main

import (
	"context"
	"flag"
	"fmt"

	"cmdra/pkg/cmdraclient"
	"cmdra/sdk/go/examples/exampleutil"
)

func main() {
	command := flag.String("command", "", "shell command")
	shell := flag.String("shell", "", "shell binary")
	usePTY := flag.Bool("pty", false, "run the shell command under a PTY")
	flag.Parse()
	if *command == "" {
		panic("--command is required")
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	execution, err := client.StartShellCommandWithOptions(context.Background(), *shell, *command, cmdraclient.ShellOptions{UsePTY: *usePTY})
	if err != nil {
		panic(err)
	}
	fmt.Println(execution.GetExecutionId())
}
