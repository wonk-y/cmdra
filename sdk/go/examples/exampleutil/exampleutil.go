package exampleutil

import (
	"context"
	"fmt"
	"os"

	"cmdra/pkg/cmdraclient"
)

func NewClient(ctx context.Context) (*cmdraclient.Client, error) {
	address := os.Getenv("CMDRA_ADDRESS")
	ca := os.Getenv("CMDRA_CA")
	cert := os.Getenv("CMDRA_CERT")
	key := os.Getenv("CMDRA_KEY")
	serverName := os.Getenv("CMDRA_SERVER_NAME")
	if address == "" || ca == "" || cert == "" || key == "" {
		return nil, fmt.Errorf("CMDRA_ADDRESS, CMDRA_CA, CMDRA_CERT, and CMDRA_KEY are required")
	}
	return cmdraclient.Dial(ctx, cmdraclient.DialConfig{
		Address:        address,
		CAFile:         ca,
		ClientCertFile: cert,
		ClientKeyFile:  key,
		ServerName:     serverName,
	})
}
