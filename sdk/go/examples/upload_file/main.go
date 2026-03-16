package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/sdk/go/examples/exampleutil"
	"cmdagent/pkg/cmdagentclient"
)

func main() {
	local := flag.String("local", "", "local path")
	remote := flag.String("remote", "", "remote path")
	flag.Parse()
	if *local == "" || *remote == "" {
		panic("--local and --remote are required")
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	resp, err := client.UploadFile(context.Background(), *local, *remote, cmdagentclient.UploadOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("transfer_id=%s bytes=%d sha256=%s\n", resp.GetTransferId(), resp.GetBytesWritten(), resp.GetSha256())
}
