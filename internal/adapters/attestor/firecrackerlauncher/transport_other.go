//go:build !linux

package firecrackerlauncher

import (
	"context"
	"os"
)

type Client struct{}

type ClientResult struct {
	Response LaunchResponse
	Output   *os.File
}

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
