package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/examples/go_sdk/exampleutil"
)

func main() {
	command := flag.String("command", "", "shell command")
	shell := flag.String("shell", "", "shell binary")
	flag.Parse()
	if *command == "" {
		panic("--command is required")
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	execution, err := client.StartShellCommand(context.Background(), *shell, *command)
	if err != nil {
		panic(err)
	}
	fmt.Println(execution.GetExecutionId())
}
