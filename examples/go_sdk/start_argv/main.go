package main

import (
	"context"
	"fmt"
	"os"

	"cmdagent/examples/go_sdk/exampleutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/go_sdk/start_argv -- <binary> [args...]")
		os.Exit(1)
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	execution, err := client.StartArgv(context.Background(), os.Args[1], os.Args[2:])
	if err != nil {
		panic(err)
	}
	fmt.Println(execution.GetExecutionId())
}
