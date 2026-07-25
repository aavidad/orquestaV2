//go:build !linux

package firecrackerlauncher

import (
	"context"
	"os"

	launchercontract "orquesta/internal/testattestorprotocol/launcher"
)

type Client struct{}

type ClientResult = launchercontract.ClientResult

func NewClient(string) (*Client, error) {
	return nil, launcherError(CodeUnavailable)
}

func (*Client) Launch(
	context.Context,
	LaunchRequest,
	*os.File,
	int64,
) (ClientResult, error) {
	return ClientResult{}, launcherError(CodeUnavailable)
}

var _ launchercontract.Client = (*Client)(nil)
