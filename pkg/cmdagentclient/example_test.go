package cmdagentclient

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func ExampleClient_StartArgv() {
	if runtime.GOOS == "windows" {
		fmt.Println("skipped")
		// Output:
		// skipped
		return
	}

	dir, err := os.MkdirTemp("", "cmdagentclient-example-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	env, err := startIntegrationEnv(dir)
	if err != nil {
		panic(err)
	}
	defer env.close()

	execMeta, err := env.clientA.StartArgv(context.Background(), "/bin/echo", []string{"example"})
	if err != nil {
		panic(err)
	}
	finished := waitForCompletion(&testingTAdapter{}, env.clientA, execMeta.GetExecutionId())
	details, err := env.clientA.GetExecutionWithOutput(context.Background(), execMeta.GetExecutionId(), false)
	if err != nil {
		panic(err)
	}
	var output bytes.Buffer
	for _, chunk := range details.Output {
		if !chunk.GetEof() {
			output.Write(chunk.GetData())
		}
	}
	fmt.Println(finished.GetExecutionId() != "", output.String() == "example\n")
	// Output:
	// true true
}

func ExampleClient_UploadFileAsync() {
	if runtime.GOOS == "windows" {
		fmt.Println("skipped")
		// Output:
		// skipped
		return
	}

	dir, err := os.MkdirTemp("", "cmdagentclient-example-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	env, err := startIntegrationEnv(dir)
	if err != nil {
		panic(err)
	}
	defer env.close()

	localPath := filepath.Join(env.dir, "example-upload.txt")
	remotePath := filepath.Join(env.dir, "example-remote.txt")
	if err := os.WriteFile(localPath, []byte("example-file"), 0o644); err != nil {
		panic(err)
	}

	resp, err := env.clientA.UploadFileAsync(context.Background(), localPath, remotePath, UploadOptions{}).Wait()
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(remotePath)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.GetTransferId() != "", string(data) == "example-file")
	// Output:
	// true true
}

type testingTAdapter struct{}

func (*testingTAdapter) Helper() {}
func (*testingTAdapter) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}
