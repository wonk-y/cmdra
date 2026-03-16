package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/examples/go_sdk/exampleutil"
)

func main() {
	shell := flag.String("shell", "/bin/sh", "shell binary")
	flag.Parse()
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	execution, err := client.StartShellSession(context.Background(), *shell, flag.Args())
	if err != nil {
		panic(err)
	}
	fmt.Println(execution.GetExecutionId())
}
