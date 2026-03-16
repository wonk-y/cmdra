package main

import (
	"context"
	"flag"
	"fmt"

	"cmdagent/sdk/go/examples/exampleutil"
	"cmdagent/pkg/cmdagentclient"
)

type multiString []string

func (m *multiString) String() string { return fmt.Sprint([]string(*m)) }
func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	local := flag.String("local", "", "local path")
	var paths multiString
	flag.Var(&paths, "path", "remote path (repeatable)")
	flag.Parse()
	if *local == "" || len(paths) == 0 {
		panic("--local and at least one --path are required")
	}
	client, err := exampleutil.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	resp, err := client.DownloadArchive(context.Background(), []string(paths), *local, cmdagentclient.DownloadOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("transfer_id=%s bytes=%d\n", resp.TransferID, resp.BytesWritten)
}
