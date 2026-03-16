package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/pkg/cmdagentclient"
	"cmdagent/sdk/go/examples/exampleutil"
)

func main() {
	shell := flag.String("shell", "/bin/sh", "shell binary")
	usePTY := flag.Bool("pty", false, "run the shell session under a PTY")
	flag.Parse()
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	execution, err := client.StartShellSessionWithOptions(context.Background(), *shell, flag.Args(), cmdagentclient.ShellOptions{UsePTY: *usePTY})
	if err != nil {
		panic(err)
	}
	fmt.Println(execution.GetExecutionId())
}
