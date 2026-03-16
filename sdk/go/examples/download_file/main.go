package main

import (
	"context"
	"flag"
	"fmt"

	"cmdra/pkg/cmdraclient"
	"cmdra/sdk/go/examples/exampleutil"
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
	resp, err := client.DownloadFile(context.Background(), *remote, *local, cmdraclient.DownloadOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("transfer_id=%s bytes=%d\n", resp.TransferID, resp.BytesWritten)
}
