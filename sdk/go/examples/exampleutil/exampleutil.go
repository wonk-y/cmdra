package exampleutil

import (
	"context"
	"fmt"
	"os"

	"cmdagent/pkg/cmdagentclient"
)

func NewClient(ctx context.Context) (*cmdagentclient.Client, error) {
	address := os.Getenv("CMDAGENT_ADDRESS")
	ca := os.Getenv("CMDAGENT_CA")
	cert := os.Getenv("CMDAGENT_CERT")
	key := os.Getenv("CMDAGENT_KEY")
	serverName := os.Getenv("CMDAGENT_SERVER_NAME")
	if address == "" || ca == "" || cert == "" || key == "" {
		return nil, fmt.Errorf("CMDAGENT_ADDRESS, CMDAGENT_CA, CMDAGENT_CERT, and CMDAGENT_KEY are required")
	}
	return cmdagentclient.Dial(ctx, cmdagentclient.DialConfig{
		Address:        address,
		CAFile:         ca,
		ClientCertFile: cert,
		ClientKeyFile:  key,
		ServerName:     serverName,
	})
}
