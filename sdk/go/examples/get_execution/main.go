package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/sdk/go/examples/exampleutil"
)

func main() {
	id := flag.String("id", "", "execution id")
	flag.Parse()
	if *id == "" {
		panic("--id is required")
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	details, err := client.GetExecutionWithOutput(context.Background(), *id, false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ID: %s\n", details.Execution.GetExecutionId())
	for _, chunk := range details.Output {
		if chunk.GetEof() {
			continue
		}
		fmt.Print(string(chunk.GetData()))
	}
}
