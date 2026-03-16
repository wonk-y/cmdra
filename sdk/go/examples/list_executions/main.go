package main

import (
	"context"
	"fmt"

	"cmdra/sdk/go/examples/exampleutil"
)

func main() {
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	executions, err := client.ListExecutions(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	for _, item := range executions {
		fmt.Printf("%s %v %v %v\n", item.GetExecutionId(), item.GetKind(), item.GetState(), item.GetCommandArgv())
	}
}
